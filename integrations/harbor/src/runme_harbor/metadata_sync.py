from __future__ import annotations

import shutil
from dataclasses import dataclass
from pathlib import Path

from harbor.models.job.config import JobConfig
from harbor.models.trajectories.trajectory import Trajectory
from harbor.models.trial.config import AgentConfig
from harbor.models.trial.result import TrialResult


ORIGINAL_CONFIG_BACKUP = "config.original.json"


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

    observed_agents = _observed_agents(job_dir)
    if not observed_agents:
        return False

    synced_config = config.model_copy(
        update={
            "agents": [
                AgentConfig(name=agent.name, model_name=agent.model_name)
                for agent in observed_agents
            ]
        }
    )
    synced_json = synced_config.model_dump_json(indent=4)
    current_json = config_path.read_text()
    if _same_json(current_json, synced_json):
        return False

    backup_path = job_dir / ORIGINAL_CONFIG_BACKUP
    if not backup_path.exists():
        shutil.copy2(config_path, backup_path)
    config_path.write_text(synced_json)
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


def _same_json(left: str, right: str) -> bool:
    try:
        left_config = JobConfig.model_validate_json(left)
        right_config = JobConfig.model_validate_json(right)
    except Exception:
        return left == right
    return left_config == right_config
