import asyncio
import re
import json
from pathlib import Path
from typing import Any

from runme_harbor.local_agents import LocalOpenClaw


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_local_openclaw_uses_ambient_user_config(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("OPENAI_API_KEY", "ambient-key")
    monkeypatch.setenv("OPENAI_BASE_URL", "https://example.test/v1")

    environment = FakeEnvironment()
    agent = LocalOpenClaw(logs_dir=tmp_path, model_name="openai/gpt-5")
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
    assert "\nopenclaw agent --local --json " in calls[0][0]
    assert "--agent main --thinking high " in calls[0][0]
    assert re.search(r"--session-key runme-harbor-[0-9a-f]{16} ", calls[0][0])
    assert "--model openai/gpt-5 " in calls[0][0]
    assert "--message 'write result.txt'" in calls[0][0]
    assert calls[0][1] == {
        "OPENAI_API_KEY": "ambient-key",
        "OPENAI_BASE_URL": "https://example.test/v1",
    }
    assert all("openclaw setup" not in command for command, _ in calls)
    assert all("~/.openclaw/openclaw.json" not in command for command, _ in calls)


def test_local_openclaw_can_use_local_config_model(tmp_path: Path) -> None:
    environment = FakeEnvironment()
    agent = LocalOpenClaw(logs_dir=tmp_path, model_name=None)
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
    assert calls[0][1] == {}


def test_local_openclaw_allows_explicit_session_key(tmp_path: Path) -> None:
    environment = FakeEnvironment()
    agent = LocalOpenClaw(
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


def test_local_openclaw_allows_explicit_session_id(tmp_path: Path) -> None:
    environment = FakeEnvironment()
    agent = LocalOpenClaw(logs_dir=tmp_path, model_name=None, session_id="1234")
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


def test_local_openclaw_collects_session_file(tmp_path: Path) -> None:
    session_file = tmp_path / "session.jsonl"
    session_file.write_text('{"type":"message"}\n')

    agent = LocalOpenClaw(logs_dir=tmp_path / "logs", model_name="openai/gpt-5")
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
