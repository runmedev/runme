import asyncio
from pathlib import Path
from typing import Any

import pytest

from runme_harbor.runme_agents import RunmeAntigravityCli


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_runme_antigravity_cli_name() -> None:
    assert RunmeAntigravityCli.name() == "runme-antigravity-cli"


def test_runme_antigravity_cli_skips_install_and_setup(tmp_path: Path) -> None:
    agent = RunmeAntigravityCli(logs_dir=tmp_path, model_name="google/gemini-3-pro-preview")
    environment = FakeEnvironment()

    assert asyncio.run(agent.install(environment)) is None
    assert asyncio.run(agent.setup(environment)) is None


def test_runme_antigravity_cli_uses_ambient_user_auth(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    for var in (
        "GEMINI_API_KEY",
        "GOOGLE_APPLICATION_CREDENTIALS",
        "GOOGLE_CLOUD_PROJECT",
        "GOOGLE_CLOUD_LOCATION",
        "GOOGLE_GENAI_USE_VERTEXAI",
        "GOOGLE_API_KEY",
    ):
        monkeypatch.delenv(var, raising=False)
    monkeypatch.setenv("GEMINI_API_KEY", "ambient-key")
    monkeypatch.setenv("GOOGLE_CLOUD_PROJECT", "ambient-project")

    environment = FakeEnvironment()
    agent = RunmeAntigravityCli(logs_dir=tmp_path, model_name="google/gemini-3-pro-preview")
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
    assert len(calls) == 3
    assert "settings.json" in calls[0][0]
    assert 'export PATH="$HOME/.local/bin:$PATH"' in calls[1][0]
    assert "\nagy --dangerously-skip-permissions " in calls[1][0]
    assert "--prompt='write result.txt'" in calls[1][0]
    assert "/logs/agent/antigravity-cli.txt" in calls[1][0]
    assert "find ~/.agy/antigravity-cli/tmp" in calls[2][0]
    assert calls[0][1] == {
        "GEMINI_API_KEY": "ambient-key",
        "GEMINI_CLI_TRUST_WORKSPACE": "true",
        "GOOGLE_CLOUD_PROJECT": "ambient-project",
    }
    assert calls[1][1] == calls[0][1]
    assert all("curl -fsSL" not in command for command, _ in calls)
    assert all("apt-get" not in command for command, _ in calls)


def test_runme_antigravity_cli_can_use_local_default_model(tmp_path: Path) -> None:
    environment = FakeEnvironment()
    agent = RunmeAntigravityCli(logs_dir=tmp_path, model_name=None)
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

    assert 'export PATH="$HOME/.local/bin:$PATH"' in calls[1][0]
    assert "\nagy --dangerously-skip-permissions " in calls[1][0]
    assert "defaultModel" not in calls[0][0]
    assert "--prompt='write result.txt'" in calls[1][0]


def test_runme_antigravity_cli_rejects_unqualified_model(tmp_path: Path) -> None:
    agent = RunmeAntigravityCli(logs_dir=tmp_path, model_name="gemini-3-pro-preview")

    with pytest.raises(ValueError, match="provider/model_name"):
        asyncio.run(agent.run("write result.txt", FakeEnvironment(), object()))


def test_runme_antigravity_cli_uses_runme_name_in_atif(tmp_path: Path) -> None:
    agent = RunmeAntigravityCli(logs_dir=tmp_path, model_name="google/gemini-3-pro-preview")

    trajectory = agent._convert_gemini_to_atif(
        {
            "sessionId": "session-1",
            "messages": [
                {
                    "type": "user",
                    "timestamp": "2026-07-01T17:00:00Z",
                    "content": "write result.txt",
                },
                {
                    "type": "gemini",
                    "timestamp": "2026-07-01T17:00:01Z",
                    "content": "done",
                    "model": "gemini-3-pro-preview",
                    "tokens": {
                        "input": 10,
                        "output": 4,
                        "cached": 1,
                    },
                },
            ],
        }
    )

    assert trajectory is not None
    assert trajectory.agent.name == "runme-antigravity-cli"
    assert trajectory.agent.model_name == "gemini-3-pro-preview"
