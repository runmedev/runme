from __future__ import annotations

import shutil
from dataclasses import dataclass
from importlib import import_module
from pathlib import Path

from harbor.models.job.config import JobConfig
from harbor.models.trajectories.trajectory import Trajectory
from harbor.models.trial.config import AgentConfig
from harbor.models.trial.result import ModelInfo, TrialResult

from runme_harbor.cli import AGENT_ARGUMENTS, ENVIRONMENT_IMPORT_PATH


ORIGINAL_CONFIG_BACKUP = "config.original.json"
ORIGINAL_RESULT_BACKUP = "result.original.json"


@dataclass(frozen=True, order=True)
class ObservedAgent:
    name: str
    model_name: str | None = None


def sync_jobs_metadata(jobs_dir: str | Path) -> int:
    """Sync Harbor job config metadata from completed trial results.

    Harbor's local viewer derives job agents/models from job config. Runme's
    integration can execute a different concrete agent/model than the original
    planned config records, so update local job configs after trials finish.
    """
    jobs_path = Path(jobs_dir).expanduser()
    if not jobs_path.exists():
        return 0

    synced = 0
    for job_dir in sorted(path for path in jobs_path.iterdir() if path.is_dir()):
        if _sync_job_metadata(job_dir):
            synced += 1
    return synced


def _sync_job_metadata(job_dir: Path) -> bool:
    config_path = job_dir / "config.json"
    if not config_path.exists():
        return False

    try:
        config = JobConfig.model_validate_json(config_path.read_text())
    except Exception:
        return False

    results_synced = _sync_trial_results(job_dir)
    original_config = _original_config(job_dir)
    observed_agents = _observed_agents(job_dir)
    if not observed_agents:
        return results_synced

    synced_agents = [
        _synced_agent_config(agent, original_config=original_config)
        for agent in observed_agents
    ]
    updates: dict[str, object] = {"agents": synced_agents}
    if _has_runme_agent(synced_agents):
        updates["environment"] = _synced_environment(config, original_config=original_config)

    synced_config = config.model_copy(
        update=updates
    )
    synced_json = synced_config.model_dump_json(indent=4)
    current_json = config_path.read_text()
    if _same_json(current_json, synced_json):
        return results_synced

    backup_path = job_dir / ORIGINAL_CONFIG_BACKUP
    if not backup_path.exists():
        shutil.copy2(config_path, backup_path)
    config_path.write_text(synced_json)
    return True


def _sync_trial_results(job_dir: Path) -> bool:
    synced = False
    for result_path in sorted(job_dir.glob("*/result.json")):
        if _sync_trial_result(result_path):
            synced = True
    return synced


def _sync_trial_result(result_path: Path) -> bool:
    try:
        result = TrialResult.model_validate_json(result_path.read_text())
    except Exception:
        return False

    if result.agent_info.model_info is not None:
        return False

    model_name = _trajectory_model_name(result_path)
    if not model_name:
        return False

    synced_result = result.model_copy(
        update={
            "agent_info": result.agent_info.model_copy(
                update={"model_info": ModelInfo(name=model_name)}
            )
        }
    )
    synced_json = synced_result.model_dump_json(indent=4)
    current_json = result_path.read_text()
    if _same_trial_result_json(current_json, synced_json):
        return False

    backup_path = result_path.with_name(ORIGINAL_RESULT_BACKUP)
    if not backup_path.exists():
        shutil.copy2(result_path, backup_path)
    result_path.write_text(synced_json)
    return True


def _observed_agents(job_dir: Path) -> list[ObservedAgent]:
    observed: set[ObservedAgent] = set()
    for result_path in sorted(job_dir.glob("*/result.json")):
        try:
            result = TrialResult.model_validate_json(result_path.read_text())
        except Exception:
            continue

        model_name = None
        model_info = result.agent_info.model_info
        if model_info is not None:
            model_name = model_info.name
            if model_info.provider:
                model_name = f"{model_info.provider}/{model_info.name}"
        else:
            model_name = _trajectory_model_name(result_path)
        observed.add(ObservedAgent(name=result.agent_info.name, model_name=model_name))
    return sorted(observed)


def _trajectory_model_name(result_path: Path) -> str | None:
    trajectory_path = result_path.parent / "agent" / "trajectory.json"
    if not trajectory_path.exists():
        return None
    try:
        trajectory = Trajectory.model_validate_json(trajectory_path.read_text())
    except Exception:
        return None
    return trajectory.agent.model_name


def _original_config(job_dir: Path) -> JobConfig | None:
    backup_path = job_dir / ORIGINAL_CONFIG_BACKUP
    if not backup_path.exists():
        return None
    try:
        return JobConfig.model_validate_json(backup_path.read_text())
    except Exception:
        return None


def _synced_agent_config(
    observed: ObservedAgent,
    *,
    original_config: JobConfig | None,
) -> AgentConfig:
    original_agent = _matching_original_agent(observed, original_config=original_config)
    import_path = original_agent.import_path if original_agent else _runme_agent_import_paths().get(observed.name)
    if import_path:
        return AgentConfig(name=observed.name, import_path=import_path, model_name=observed.model_name)
    return AgentConfig(name=observed.name, model_name=observed.model_name)


def _matching_original_agent(
    observed: ObservedAgent,
    *,
    original_config: JobConfig | None,
) -> AgentConfig | None:
    if original_config is None:
        return None

    expected_import_path = _runme_agent_import_paths().get(observed.name)
    for agent in original_config.agents:
        if expected_import_path and agent.import_path == expected_import_path:
            return agent
        if agent.name == observed.name:
            return agent
    return None


def _has_runme_agent(agents: list[AgentConfig]) -> bool:
    import_paths = set(_runme_agent_import_paths().values())
    return any(agent.import_path in import_paths for agent in agents)


def _synced_environment(
    config: JobConfig,
    *,
    original_config: JobConfig | None,
):
    if original_config is not None and original_config.environment.import_path:
        return original_config.environment
    if config.environment.import_path:
        return config.environment
    return config.environment.model_copy(update={"import_path": ENVIRONMENT_IMPORT_PATH})


def _runme_agent_import_paths() -> dict[str, str]:
    agent_import_paths: dict[str, str] = {}
    for args in AGENT_ARGUMENTS.values():
        if len(args) != 2 or args[0] != "--agent-import-path":
            continue
        import_path = args[1]
        agent_name = _agent_name_from_import_path(import_path)
        if agent_name:
            agent_import_paths[agent_name] = import_path
    return agent_import_paths


def _agent_name_from_import_path(import_path: str) -> str | None:
    module_name, _, class_name = import_path.partition(":")
    if not module_name or not class_name:
        return None
    try:
        agent_class = getattr(import_module(module_name), class_name)
        name = agent_class.name()
    except Exception:
        return None
    if isinstance(name, str) and name:
        return name
    return None


def _same_json(left: str, right: str) -> bool:
    try:
        left_config = JobConfig.model_validate_json(left)
        right_config = JobConfig.model_validate_json(right)
    except Exception:
        return left == right
    return left_config == right_config


def _same_trial_result_json(left: str, right: str) -> bool:
    try:
        left_result = TrialResult.model_validate_json(left)
        right_result = TrialResult.model_validate_json(right)
    except Exception:
        return left == right
    return left_result == right_result
