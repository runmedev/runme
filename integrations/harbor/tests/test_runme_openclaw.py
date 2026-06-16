import asyncio
import json
import re
from pathlib import Path
from types import SimpleNamespace
from typing import Any

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
    runtime_config: dict[str, Any] = {}
    runtime_config_path: Path | None = None

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        nonlocal runtime_config_path
        calls.append((command, env))
        assert env is not None
        runtime_config_path = Path(env["OPENCLAW_CONFIG_PATH"])
        runtime_config.update(json.loads(runtime_config_path.read_text()))

    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

    assert environment.uploads == []
    assert "\nopenclaw agent --local --json " in calls[0][0]
    assert "--agent main --thinking high " in calls[0][0]
    assert re.search(r"--session-key runme-harbor-[0-9a-f]{16} ", calls[0][0])
    assert "--model openai/gpt-5 " in calls[0][0]
    assert "--message 'write result.txt'" in calls[0][0]
    assert calls[0][1] == {
        "OPENCLAW_CONFIG_PATH": str(runtime_config_path),
        "OPENAI_API_KEY": "ambient-key",
        "OPENAI_BASE_URL": "https://example.test/v1",
    }
    assert runtime_config["agents"]["defaults"]["workspace"] == str(staged_workspace)
    assert runtime_config["agents"]["defaults"]["skipBootstrap"] is True
    ambient_defaults = json.loads((home / ".openclaw" / "openclaw.json").read_text())[
        "agents"
    ]["defaults"]
    assert ambient_defaults["workspace"] == str(original_workspace)
    assert "skipBootstrap" not in ambient_defaults
    assert runtime_config_path is not None
    assert not runtime_config_path.exists()
    assert all("openclaw setup" not in command for command, _ in calls)
    assert all("~/.openclaw/openclaw.json" not in command for command, _ in calls)


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
    assert "--message 'write result.txt'" in calls[0][0]
    assert calls[0][1] is not None
    assert "OPENCLAW_CONFIG_PATH" in calls[0][1]


def test_runme_openclaw_allows_explicit_session_key(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    environment = FakeEnvironment()
    agent = RunmeOpenClaw(
        logs_dir=tmp_path,
        model_name=None,
        session_key="agent:ops:incident-42",
        openclaw_agent_id="ops",
    )
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

    assert "--agent ops " in calls[0][0]
    assert "--session-key agent:ops:incident-42 " in calls[0][0]
    assert "runme-harbor-" not in calls[0][0]


def test_runme_openclaw_allows_explicit_session_id(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    environment = FakeEnvironment()
    agent = RunmeOpenClaw(logs_dir=tmp_path, model_name=None, session_id="1234")
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

    assert "--session-id 1234 " in calls[0][0]
    assert "runme-harbor-" not in calls[0][0]


def test_runme_openclaw_collects_session_file(tmp_path: Path) -> None:
    session_file = tmp_path / "session.jsonl"
    session_file.write_text('{"type":"message"}\n')

    agent = RunmeOpenClaw(logs_dir=tmp_path / "logs", model_name="openai/gpt-5")
    agent.logs_dir.mkdir()
    (agent.logs_dir / "openclaw.txt").write_text(
        json.dumps(
            {
                "payloads": [{"text": "ok"}],
                "meta": {
                    "agentMeta": {
                        "sessionId": "s1",
                        "sessionFile": str(session_file),
                    },
                },
            }
        )
    )

    agent._collect_session_file()

    assert (agent.logs_dir / "openclaw.session.jsonl").read_text() == '{"type":"message"}\n'
