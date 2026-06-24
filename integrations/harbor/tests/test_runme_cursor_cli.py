import asyncio
from pathlib import Path
from typing import Any

from runme_harbor.runme_agents import RunmeCursorCli


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_runme_cursor_cli_name() -> None:
    assert RunmeCursorCli.name() == "runme-cursor-cli"


def test_runme_cursor_cli_uses_ambient_user_auth(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("CURSOR_API_KEY", raising=False)

    environment = FakeEnvironment()
    agent = RunmeCursorCli(logs_dir=tmp_path, model_name="cursor/composer-2")
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
    assert "\ncursor-agent --yolo --print --output-format=stream-json " in calls[0][0]
    assert "--model=composer-2 -- 'write result.txt'" in calls[0][0]
    assert "cursor-cli.txt" in calls[0][0]
    assert calls[0][1] == {}
    assert all("CURSOR_API_KEY" not in (env or {}) for _, env in calls)
    assert all("register" not in command for command, _ in calls)


def test_runme_cursor_cli_passes_cursor_api_key_when_present(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("CURSOR_API_KEY", "ambient-key")

    environment = FakeEnvironment()
    agent = RunmeCursorCli(logs_dir=tmp_path, model_name="cursor/composer-2")
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

    assert calls[0][1] == {"CURSOR_API_KEY": "ambient-key"}


def test_runme_cursor_cli_can_use_cursor_default_model(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.delenv("CURSOR_API_KEY", raising=False)

    environment = FakeEnvironment()
    agent = RunmeCursorCli(logs_dir=tmp_path, model_name=None)
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

    assert "--model=" not in calls[0][0]
    assert "-- 'write result.txt'" in calls[0][0]
