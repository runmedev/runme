import asyncio
from pathlib import Path
from typing import Any

from runme_harbor.runme_agents import RunmeClaudeCode


class FakeEnvironment:
    def __init__(self) -> None:
        self.uploads: list[tuple[Path | str, str]] = []

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        self.uploads.append((source_path, target_path))


def test_runme_claude_code_name() -> None:
    assert RunmeClaudeCode.name() == "runme-claude-code"


def test_runme_claude_code_uses_ambient_user_config(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("ANTHROPIC_API_KEY", "ambient-key")
    monkeypatch.setenv("CLAUDE_CODE_MAX_OUTPUT_TOKENS", "8192")
    monkeypatch.setenv("CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING", "1")
    monkeypatch.delenv("CLAUDE_CONFIG_DIR", raising=False)
    monkeypatch.delenv("CLAUDE_CODE_OAUTH_TOKEN", raising=False)

    environment = FakeEnvironment()
    agent = RunmeClaudeCode(logs_dir=tmp_path, model_name="anthropic/haiku")
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
    assert "\nclaude --verbose --output-format=stream-json " in calls[0][0]
    assert "--permission-mode=bypassPermissions " in calls[0][0]
    assert "--model haiku " in calls[0][0]
    assert "--print -- 'write result.txt'" in calls[0][0]
    assert calls[0][1] == {
        "CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING": "1",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
        "CLAUDE_CODE_MAX_OUTPUT_TOKENS": "8192",
        "ENABLE_BACKGROUND_TASKS": "1",
        "FORCE_AUTO_BACKGROUND_TASKS": "1",
        "IS_SANDBOX": "1",
    }
    assert all("ANTHROPIC_API_KEY" not in (env or {}) for _, env in calls)
    assert all("CLAUDE_CODE_OAUTH_TOKEN" not in (env or {}) for _, env in calls)
    assert all("CLAUDE_CONFIG_DIR" not in (env or {}) for _, env in calls)
    assert all("register" not in command for command, _ in calls)


def test_runme_claude_code_preserves_model_name_with_custom_base_url(
    tmp_path: Path,
    monkeypatch,
) -> None:
    monkeypatch.setenv("ANTHROPIC_BASE_URL", "https://example.test")

    agent = RunmeClaudeCode(logs_dir=tmp_path, model_name="openrouter/anthropic/haiku")

    assert agent._model_arg() == "--model openrouter/anthropic/haiku "


def test_runme_claude_code_collects_only_new_sessions(
    tmp_path: Path,
    monkeypatch,
) -> None:
    claude_config_dir = tmp_path / "claude-config"
    projects_dir = claude_config_dir / "projects" / "-app"
    old_session = projects_dir / "old.jsonl"
    new_session = projects_dir / "new.jsonl"
    old_session.parent.mkdir(parents=True)
    old_session.write_text('{"type":"old"}\n')
    monkeypatch.setenv("CLAUDE_CONFIG_DIR", str(claude_config_dir))

    agent = RunmeClaudeCode(logs_dir=tmp_path / "logs")
    before = agent._snapshot_session_files()

    new_session.write_text('{"type":"new"}\n')
    agent._collect_new_sessions(before)

    copied_old = agent.logs_dir / "sessions" / old_session.relative_to(claude_config_dir)
    copied_new = agent.logs_dir / "sessions" / new_session.relative_to(claude_config_dir)

    assert not copied_old.exists()
    assert copied_new.read_text() == '{"type":"new"}\n'
