import asyncio
from pathlib import Path
from typing import Any

from runme_harbor.codex_agent import RunmeCodexAgent


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_codex_agent_uses_ambient_user_auth(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("OPENAI_API_KEY", "ambient-key")
    monkeypatch.delenv("CODEX_HOME", raising=False)
    monkeypatch.delenv("CODEX_AUTH_JSON_PATH", raising=False)
    monkeypatch.delenv("CODEX_FORCE_AUTH_JSON", raising=False)

    environment = FakeEnvironment()
    agent = RunmeCodexAgent(logs_dir=tmp_path)
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
    assert "\ncodex exec " in calls[0][0]
    assert all("CODEX_HOME" not in (env or {}) for _, env in calls)
    assert all("OPENAI_API_KEY" not in (env or {}) for _, env in calls)
    assert all("register" not in command for command, _ in calls)


def test_codex_agent_collects_only_new_sessions(
    tmp_path: Path,
    monkeypatch,
) -> None:
    codex_home = tmp_path / "codex-home"
    sessions_dir = codex_home / "sessions"
    old_session = sessions_dir / "2026" / "06" / "04" / "old.jsonl"
    new_session = sessions_dir / "2026" / "06" / "05" / "new.jsonl"
    old_session.parent.mkdir(parents=True)
    old_session.write_text('{"type":"old"}\n')
    monkeypatch.setenv("CODEX_HOME", str(codex_home))

    agent = RunmeCodexAgent(logs_dir=tmp_path / "logs")
    before = agent._snapshot_session_files()

    new_session.parent.mkdir(parents=True)
    new_session.write_text('{"type":"new"}\n')
    agent._collect_new_sessions(before)

    copied_old = agent.logs_dir / "sessions" / old_session.relative_to(sessions_dir)
    copied_new = agent.logs_dir / "sessions" / new_session.relative_to(sessions_dir)

    assert not copied_old.exists()
    assert copied_new.read_text() == '{"type":"new"}\n'
