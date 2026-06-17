from __future__ import annotations

from pathlib import Path

from harbor.models.job.config import JobConfig
from harbor.models.trial.config import AgentConfig
from harbor.models.trial.result import TrialResult

from runme_harbor.cli import (
    CLAUDE_IMPORT_PATH,
    CODEX_IMPORT_PATH,
    ENVIRONMENT_IMPORT_PATH,
)
from runme_harbor.metadata_sync import (
    ORIGINAL_CONFIG_BACKUP,
    ORIGINAL_RESULT_BACKUP,
    sync_jobs_metadata,
)


def test_sync_jobs_metadata_uses_observed_runme_agent_import_paths(tmp_path: Path) -> None:
    job_dir = _write_job_config(
        tmp_path,
        agents=[
            AgentConfig(
                import_path=CODEX_IMPORT_PATH,
                model_name="planned-provider/planned-model",
            )
        ],
    )
    _write_trial_result(
        job_dir,
        "trial-1",
        agent_name="runme-codex",
        provider="openai",
        model_name="gpt-5",
    )

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, "openai/gpt-5")
    ]
    assert config.environment.import_path == ENVIRONMENT_IMPORT_PATH

    assert ORIGINAL_CONFIG_BACKUP == "config.original.json"
    backup = JobConfig.model_validate_json((job_dir / "config.original.json").read_text())
    assert _agent_summaries(backup) == [
        (
            None,
            CODEX_IMPORT_PATH,
            "planned-provider/planned-model",
        )
    ]


def test_sync_jobs_metadata_sorts_and_deduplicates_observed_agents(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-z", agent_name="runme-codex", provider=None, model_name="gpt-5")
    _write_trial_result(
        job_dir,
        "trial-a",
        agent_name="runme-claude-code",
        provider="anthropic",
        model_name="sonnet",
    )
    _write_trial_result(job_dir, "trial-b", agent_name="runme-codex", provider=None, model_name="gpt-5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-claude-code", CLAUDE_IMPORT_PATH, "anthropic/sonnet"),
        ("runme-codex", CODEX_IMPORT_PATH, "gpt-5"),
    ]


def test_sync_jobs_metadata_sorts_same_agent_with_missing_model_info(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-a", agent_name="runme-codex")
    _write_trial_result(job_dir, "trial-b", agent_name="runme-codex", model_name="gpt-5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, None),
        ("runme-codex", CODEX_IMPORT_PATH, "gpt-5"),
    ]


def test_sync_jobs_metadata_omits_missing_model_info(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-1", agent_name="oracle")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [("oracle", None, None)]


def test_sync_jobs_metadata_keeps_builtin_agent_names(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-1", agent_name="oracle", model_name="gpt-5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [("oracle", None, "gpt-5")]


def test_sync_jobs_metadata_uses_atif_model_when_trial_model_info_missing(
    tmp_path: Path,
) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-1", agent_name="runme-codex")
    _write_trajectory(job_dir / "trial-1", agent_name="runme-codex", model_name="gpt-5.5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, "gpt-5.5")
    ]
    result = TrialResult.model_validate_json((job_dir / "trial-1" / "result.json").read_text())
    assert result.agent_info.model_info is not None
    assert result.agent_info.model_info.name == "gpt-5.5"
    assert result.agent_info.model_info.provider is None
    result_backup = TrialResult.model_validate_json(
        (job_dir / "trial-1" / ORIGINAL_RESULT_BACKUP).read_text()
    )
    assert result_backup.agent_info.model_info is None

    assert sync_jobs_metadata(tmp_path) == 0
    assert result_backup == TrialResult.model_validate_json(
        (job_dir / "trial-1" / ORIGINAL_RESULT_BACKUP).read_text()
    )


def test_sync_jobs_metadata_preserves_trial_model_info_over_atif(
    tmp_path: Path,
) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(
        job_dir,
        "trial-1",
        agent_name="runme-codex",
        provider="openai",
        model_name="gpt-5",
    )
    _write_trajectory(job_dir / "trial-1", agent_name="runme-codex", model_name="gpt-5.5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, "openai/gpt-5")
    ]
    result = TrialResult.model_validate_json((job_dir / "trial-1" / "result.json").read_text())
    assert result.agent_info.model_info is not None
    assert result.agent_info.model_info.name == "gpt-5"
    assert result.agent_info.model_info.provider == "openai"
    assert not (job_dir / "trial-1" / ORIGINAL_RESULT_BACKUP).exists()


def test_sync_jobs_metadata_ignores_unreadable_atif(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-1", agent_name="runme-codex")
    trajectory_path = job_dir / "trial-1" / "agent" / "trajectory.json"
    trajectory_path.parent.mkdir()
    trajectory_path.write_text("{")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, None)
    ]


def test_sync_jobs_metadata_skips_jobs_without_readable_trials(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])

    assert sync_jobs_metadata(tmp_path) == 0

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [("planned", None, None)]
    assert not (job_dir / ORIGINAL_CONFIG_BACKUP).exists()


def test_sync_jobs_metadata_ignores_unreadable_trials(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    unreadable_dir = job_dir / "bad-trial"
    unreadable_dir.mkdir()
    (unreadable_dir / "result.json").write_text("{")
    _write_trial_result(
        job_dir,
        "good-trial",
        agent_name="runme-claude-code",
        provider="anthropic",
        model_name="sonnet",
    )

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-claude-code", CLAUDE_IMPORT_PATH, "anthropic/sonnet")
    ]


def test_sync_jobs_metadata_preserves_original_backup(tmp_path: Path) -> None:
    job_dir = _write_job_config(tmp_path, agents=[AgentConfig(name="planned")])
    _write_trial_result(job_dir, "trial-1", agent_name="runme-codex", model_name="gpt-5")

    assert sync_jobs_metadata(tmp_path) == 1
    backup_path = job_dir / ORIGINAL_CONFIG_BACKUP
    backup_text = backup_path.read_text()

    assert sync_jobs_metadata(tmp_path) == 0
    assert backup_path.read_text() == backup_text


def test_sync_jobs_metadata_corrects_prior_runme_agent_name_sync(tmp_path: Path) -> None:
    job_dir = _write_job_config(
        tmp_path,
        agents=[AgentConfig(import_path=CODEX_IMPORT_PATH)],
    )
    backup_text = (job_dir / "config.json").read_text()
    (job_dir / ORIGINAL_CONFIG_BACKUP).write_text(backup_text)
    config = JobConfig.model_validate_json(backup_text).model_copy(
        update={"agents": [AgentConfig(name="runme-codex", model_name="gpt-5.5")]}
    )
    (job_dir / "config.json").write_text(config.model_dump_json(indent=4))
    _write_trial_result(job_dir, "trial-1", agent_name="runme-codex")
    _write_trajectory(job_dir / "trial-1", agent_name="runme-codex", model_name="gpt-5.5")

    assert sync_jobs_metadata(tmp_path) == 1

    config = JobConfig.model_validate_json((job_dir / "config.json").read_text())
    assert _agent_summaries(config) == [
        ("runme-codex", CODEX_IMPORT_PATH, "gpt-5.5")
    ]
    assert (job_dir / ORIGINAL_CONFIG_BACKUP).read_text() == backup_text


def _agent_summaries(config: JobConfig) -> list[tuple[str | None, str | None, str | None]]:
    return [(agent.name, agent.import_path, agent.model_name) for agent in config.agents]


def _write_job_config(tmp_path: Path, *, agents: list[AgentConfig]) -> Path:
    job_dir = tmp_path / "job-1"
    job_dir.mkdir()
    config = JobConfig(
        job_name="job-1",
        jobs_dir=tmp_path,
        agents=agents,
        environment={"import_path": ENVIRONMENT_IMPORT_PATH},
    )
    (job_dir / "config.json").write_text(config.model_dump_json(indent=4))
    return job_dir


def _write_trial_result(
    job_dir: Path,
    trial_name: str,
    *,
    agent_name: str,
    provider: str | None = None,
    model_name: str | None = None,
) -> None:
    trial_dir = job_dir / trial_name
    trial_dir.mkdir()
    agent_info: dict[str, object] = {"name": agent_name, "version": "1.0"}
    if model_name is not None:
        agent_info["model_info"] = {"name": model_name, "provider": provider}
    result = TrialResult.model_validate(
        {
            "task_name": "task-a",
            "trial_name": trial_name,
            "trial_uri": f"file:///{trial_name}",
            "task_id": {"path": "task-a"},
            "source": "dataset",
            "task_checksum": "checksum",
            "config": {"task": {"path": "task-a"}},
            "agent_info": agent_info,
        }
    )
    (trial_dir / "result.json").write_text(result.model_dump_json(indent=4))


def _write_trajectory(trial_dir: Path, *, agent_name: str, model_name: str) -> None:
    trajectory_dir = trial_dir / "agent"
    trajectory_dir.mkdir()
    (trajectory_dir / "trajectory.json").write_text(
        f"""{{
    "schema_version": "ATIF-v1.5",
    "agent": {{
        "name": "{agent_name}",
        "version": "1.0",
        "model_name": "{model_name}"
    }},
    "steps": [
        {{
            "step_id": 1,
            "source": "user",
            "message": "hello"
        }}
    ]
}}"""
    )
