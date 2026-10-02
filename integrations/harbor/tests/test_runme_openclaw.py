import asyncio
import json
from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pytest

from runme_harbor.runme_agents import RunmeOpenClaw


class FakeEnvironment:
    def __init__(self, workspace_path: Path | None = None) -> None:
        self.uploads: list[tuple[Path | str, str]] = []
        self.workspace_path = workspace_path
        self.task_env_config = SimpleNamespace(workdir="/app/task/workdir")

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))

    def _map_remote_path(self, path: str) -> Path:
        assert path == "/app/task/workdir"
        if self.workspace_path is None:
            raise ValueError("workspace path is not configured")
        return self.workspace_path


def write_openclaw_config(home: Path, workspace: Path) -> Path:
    config_path = home / ".openclaw" / "openclaw.json"
    config_path.parent.mkdir(parents=True)
    config_path.write_text(
        json.dumps(
            {
                "models": {"providers": {}},
                "agents": {
                    "defaults": {
                        "workspace": str(workspace),
                        "model": {"primary": "openai/gpt-5"},
                    }
                },
            }
        )
    )
    return config_path


def test_runme_openclaw_name() -> None:
    assert RunmeOpenClaw.name() == "runme-openclaw"


def test_runme_openclaw_rejects_unknown_options(tmp_path: Path) -> None:
    with pytest.raises(ValueError, match="Unknown option 'unknown_option'"):
        RunmeOpenClaw(logs_dir=tmp_path, unknown_option=True)


def test_runme_openclaw_uses_ambient_user_config(
    tmp_path: Path,
    monkeypatch,
) -> None:
    home = tmp_path / "home"
    original_workspace = tmp_path / "original-workspace"
    staged_workspace = tmp_path / "trial" / "workdir"
    write_openclaw_config(home, original_workspace)
    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setenv("OPENAI_API_KEY", "ambient-key")
    monkeypatch.setenv("OPENAI_BASE_URL", "https://example.test/v1")

    environment = FakeEnvironment(staged_workspace)
    agent = RunmeOpenClaw(logs_dir=tmp_path, model_name="openai/gpt-5")
    calls: list[tuple[str, dict[str, str] | None]] = []

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        calls.append((command, env))

    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

    assert environment.uploads == []
    assert "\nopenclaw agent exec --json " in calls[0][0]
    assert "--thinking high " in calls[0][0]
    assert "--agent " not in calls[0][0]
    assert "--session-" not in calls[0][0]
    assert f"--config {home / '.openclaw' / 'openclaw.json'} " in calls[0][0]
    assert f"--cwd {staged_workspace} " in calls[0][0]
    assert "--model openai/gpt-5 " in calls[0][0]
    assert "'write result.txt'" in calls[0][0]
    assert calls[0][1] == {
        "OPENAI_API_KEY": "ambient-key",
        "OPENAI_BASE_URL": "https://example.test/v1",
    }
    ambient_defaults = json.loads((home / ".openclaw" / "openclaw.json").read_text())[
        "agents"
    ]["defaults"]
    assert ambient_defaults["workspace"] == str(original_workspace)
    assert "skipBootstrap" not in ambient_defaults
    assert all("openclaw setup" not in command for command, _ in calls)


def test_runme_openclaw_can_use_local_config_model(
    tmp_path: Path,
    monkeypatch,
) -> None:
    home = tmp_path / "home"
    write_openclaw_config(home, tmp_path / "original-workspace")
    monkeypatch.setenv("HOME", str(home))

    environment = FakeEnvironment(tmp_path / "trial" / "workdir")
    agent = RunmeOpenClaw(logs_dir=tmp_path, model_name=None)
    calls: list[tuple[str, dict[str, str] | None]] = []

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        calls.append((command, env))

    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

    assert "--model " not in calls[0][0]
    assert "'write result.txt'" in calls[0][0]
    assert "--config " in calls[0][0]
    assert "--cwd " in calls[0][0]
    assert calls[0][1] == {}


def test_runme_openclaw_exec_isolates_state_without_ambient_config(
    tmp_path: Path,
    monkeypatch,
) -> None:
    home = tmp_path / "home"
    monkeypatch.setenv("HOME", str(home))

    workspace = tmp_path / "trial" / "workdir"
    environment = FakeEnvironment(workspace)
    agent = RunmeOpenClaw(logs_dir=tmp_path, model_name=None)

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        assert "openclaw agent exec --json " in command
        assert "--config " not in command
        assert f"--cwd {workspace} " in command
        assert env == {}

    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

def test_runme_openclaw_normalizes_agent_exec_envelope(tmp_path: Path) -> None:
    agent = RunmeOpenClaw(logs_dir=tmp_path / "logs", model_name="openai/gpt-5")
    agent.logs_dir.mkdir()
    (agent.logs_dir / "openclaw.txt").write_text(
        json.dumps(
            {
                "payloads": [{"text": "ok"}],
                "sessionId": "s1",
                "usage": {"input": 10, "output": 2, "total": 12},
            }
        )
    )

    envelope = agent._parse_stdout()

    assert envelope is not None
    assert envelope["meta"]["agentMeta"] == {
        "sessionId": "s1",
        "usage": {"input": 10, "output": 2, "total": 12},
    }
