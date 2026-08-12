import asyncio
from pathlib import Path
from typing import Any

from runme_harbor.runme_agents import RunmeCodex


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_runme_codex_name() -> None:
    assert RunmeCodex.name() == "runme-codex"


def test_runme_codex_uses_ambient_user_auth(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("OPENAI_API_KEY", "ambient-key")
    monkeypatch.delenv("CODEX_HOME", raising=False)
    monkeypatch.delenv("CODEX_AUTH_JSON_PATH", raising=False)
    monkeypatch.delenv("CODEX_FORCE_AUTH_JSON", raising=False)

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path, model_name="openai/gpt-5")
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
    assert "if [ -s ~/.nvm/nvm.sh ]; then . ~/.nvm/nvm.sh; fi" in calls[0][0]
    assert "\ncodex exec " in calls[0][0]
    assert "--model gpt-5 " in calls[0][0]
    assert all("CODEX_HOME" not in (env or {}) for _, env in calls)
    assert all("OPENAI_API_KEY" not in (env or {}) for _, env in calls)
    assert all("register" not in command for command, _ in calls)


def test_runme_codex_routes_through_process_local_base_url(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv(
        "OPENAI_BASE_URL",
        'https://router.test/v1?tenant=runme&note="quoted"',
    )

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path, model_name="openai/gpt-5")
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

    command = calls[0][0]
    assert "codex exec " in command
    assert "-c 'openai_base_url=" in command
    assert '\\"quoted\\"' in command
    assert "OPENAI_BASE_URL" not in command


def test_runme_codex_uses_explicit_openai_key_when_native_login_is_absent(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("OPENAI_API_KEY", "explicit-key")
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path)

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> bool:
        return False

    agent._has_native_auth = fake_has_native_auth

    assert asyncio.run(agent._agent_env_and_router_state(environment)) == ({}, True)


def test_runme_codex_native_login_wins_over_fallback(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path)

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> bool:
        return True

    agent._has_native_auth = fake_has_native_auth

    assert asyncio.run(agent._agent_env_and_router_state(environment)) == ({}, False)


def test_runme_codex_native_login_bypasses_router_base_url(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("OPENAI_API_KEY", "judge-key")
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")
    monkeypatch.setenv("OPENAI_BASE_URL", "http://localhost:4000/router/v1/openai/v1")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path, model_name="openai/gpt-5")
    calls: list[tuple[str, dict[str, str] | None]] = []

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> bool:
        return True

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        calls.append((command, env))

    agent._has_native_auth = fake_has_native_auth
    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

    command, env = calls[0]
    assert "openai_base_url" not in command
    assert "unset OPENAI_API_KEY OPENAI_BASE_URL\n" in command
    assert "OPENAI_API_KEY" not in (env or {})


def test_runme_codex_promoted_fallback_keeps_router_base_url(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")
    monkeypatch.setenv("OPENAI_BASE_URL", "http://localhost:4000/router/v1/openai/v1")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path, model_name="openai/gpt-5")
    calls: list[tuple[str, dict[str, str] | None]] = []

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> bool:
        return False

    async def fake_exec_as_agent(
        _environment: Any,
        command: str,
        env: dict[str, str] | None = None,
    ) -> None:
        calls.append((command, env))

    agent._has_native_auth = fake_has_native_auth
    agent.exec_as_agent = fake_exec_as_agent
    agent.populate_context_post_run = lambda _context: None

    asyncio.run(agent.run("write result.txt", environment, object()))

    command, env = calls[0]
    assert "openai_base_url" in command
    assert "unset OPENAI_API_KEY OPENAI_BASE_URL\n" not in command
    assert (env or {}).get("OPENAI_API_KEY") == "fallback-key"


def test_runme_codex_promotes_fallback_when_auth_gap_exists(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path)

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> bool:
        return False

    agent._has_native_auth = fake_has_native_auth

    assert asyncio.run(agent._agent_env_and_router_state(environment)) == (
        {"OPENAI_API_KEY": "fallback-key"},
        True,
    )


def test_runme_codex_fails_closed_when_login_detection_is_indeterminate(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("OPENAI_API_KEY", raising=False)
    monkeypatch.setenv("RUNME_ROUTER_FALLBACK_OPENAI_API_KEY", "fallback-key")

    environment = FakeEnvironment()
    agent = RunmeCodex(logs_dir=tmp_path)

    async def fake_has_native_auth(_environment: Any, _env: dict[str, str]) -> None:
        return None

    agent._has_native_auth = fake_has_native_auth

    assert asyncio.run(agent._agent_env_and_router_state(environment)) == ({}, False)


def test_runme_codex_collects_only_new_sessions(
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

    agent = RunmeCodex(logs_dir=tmp_path / "logs")
    before = agent._snapshot_session_files()

    new_session.parent.mkdir(parents=True)
    new_session.write_text('{"type":"new"}\n')
    agent._collect_new_sessions(before)

    copied_old = agent.logs_dir / "sessions" / old_session.relative_to(sessions_dir)
    copied_new = agent.logs_dir / "sessions" / new_session.relative_to(sessions_dir)

    assert not copied_old.exists()
    assert copied_new.read_text() == '{"type":"new"}\n'


def test_runme_codex_collects_session_matching_thread_id(
    tmp_path: Path,
    monkeypatch,
) -> None:
    codex_home = tmp_path / "codex-home"
    sessions_dir = codex_home / "sessions"
    old_session = sessions_dir / "2026" / "08" / "05" / "old.jsonl"
    session_a = sessions_dir / "2026" / "08" / "05" / "session-a.jsonl"
    session_b = sessions_dir / "2026" / "08" / "05" / "session-b.jsonl"
    old_session.parent.mkdir(parents=True)
    old_session.write_text('{"type":"session_meta","payload":{"id":"old"}}\n')
    monkeypatch.setenv("CODEX_HOME", str(codex_home))

    agent = RunmeCodex(logs_dir=tmp_path / "logs")
    before = agent._snapshot_session_files()
    agent.logs_dir.mkdir(parents=True)
    (agent.logs_dir / "codex.txt").write_text(
        "Reading additional input from stdin...\n"
        '{"type":"thread.started","thread_id":"session-b"}\n'
    )

    session_a.write_text('{"type":"session_meta","payload":{"id":"session-a"}}\n')
    session_b.write_text('{"type":"session_meta","payload":{"session_id":"session-b"}}\n')

    assert agent._collect_new_sessions(before)

    copied_a = agent.logs_dir / "sessions" / session_a.relative_to(sessions_dir)
    copied_b = agent.logs_dir / "sessions" / session_b.relative_to(sessions_dir)

    assert not copied_a.exists()
    assert copied_b.read_text() == (
        '{"type":"session_meta","payload":{"session_id":"session-b"}}\n'
    )


def test_runme_codex_fails_closed_when_new_sessions_are_ambiguous(
    tmp_path: Path,
    monkeypatch,
) -> None:
    codex_home = tmp_path / "codex-home"
    sessions_dir = codex_home / "sessions" / "2026" / "08" / "05"
    session_a = sessions_dir / "session-a.jsonl"
    session_b = sessions_dir / "session-b.jsonl"
    sessions_dir.mkdir(parents=True)
    monkeypatch.setenv("CODEX_HOME", str(codex_home))

    agent = RunmeCodex(logs_dir=tmp_path / "logs")
    before = agent._snapshot_session_files()
    stale_session = agent.logs_dir / "sessions" / "stale.jsonl"
    stale_session.parent.mkdir(parents=True)
    stale_session.write_text('{"type":"stale"}\n')

    session_a.write_text('{"type":"session_meta","payload":{"id":"session-a"}}\n')
    session_b.write_text('{"type":"session_meta","payload":{"id":"session-b"}}\n')

    assert not agent._collect_new_sessions(before)
    assert not (agent.logs_dir / "sessions").exists()
