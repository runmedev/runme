from __future__ import annotations

from pathlib import Path

import pytest

from runme_harbor import cli


def parse(argv: list[str]):
    return cli._parse_args(argv)


def test_build_harbor_command_oracle_defaults(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path)])

    assert cli.build_harbor_command(args) == [
        "harbor",
        "run",
        "--path",
        str(tmp_path.resolve()),
        "--jobs-dir",
        ".runme/harbor/jobs",
        "--environment-import-path",
        "runme_harbor.environment:RunmeEnvironment",
        "--agent",
        "oracle",
        "--n-concurrent",
        "1",
    ]


def test_build_harbor_command_codex(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "codex"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.local_agents:LocalCodex" in command
    assert "--agent" not in command


def test_build_harbor_command_claude(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "claude-code"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.local_agents:LocalClaudeCode" in command
    assert "--agent" not in command


def test_build_harbor_command_openclaw(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "openclaw"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.local_agents:LocalOpenClaw" in command
    assert "--agent" not in command


def test_build_harbor_command_task_yes_jobs_and_passthrough(tmp_path: Path) -> None:
    args = parse(
        [
            "run",
            str(tmp_path),
            "--task",
            "local-agent",
            "--jobs-dir",
            "jobs",
            "-y",
            "--",
            "--model",
            "gpt-5",
        ]
    )

    command = cli.build_harbor_command(args)

    assert ["--include-task-name", "local-agent"] == command[
        command.index("--include-task-name") : command.index("--include-task-name") + 2
    ]
    assert ["--jobs-dir", "jobs"] == command[command.index("--jobs-dir") : command.index("--jobs-dir") + 2]
    assert "-y" in command
    assert command[-2:] == ["--model", "gpt-5"]


@pytest.mark.parametrize(
    "passthrough",
    [
        ["--", "--n-concurrent", "3"],
        ["--", "--n-concurrent=3"],
        ["--", "-n", "3"],
        ["--", "-n3"],
    ],
)
def test_build_harbor_command_does_not_duplicate_concurrency(
    tmp_path: Path,
    passthrough: list[str],
) -> None:
    args = parse(["run", str(tmp_path), *passthrough])

    command = cli.build_harbor_command(args)

    assert command.count("--n-concurrent") == passthrough.count("--n-concurrent")


def test_main_runs_harbor_and_prints_debug(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    calls: list[list[str]] = []
    monkeypatch.setattr(cli, "_preflight", lambda _agent: None)
    monkeypatch.setattr(cli.subprocess, "call", lambda command: calls.append(command) or 7)

    assert cli.main(["run", str(tmp_path), "--debug"]) == 7

    assert calls[0][0:2] == ["harbor", "run"]
    assert "harbor run" in capsys.readouterr().err


@pytest.mark.parametrize(
    ("agent", "missing", "message"),
    [
        ("codex", "codex", "`--agent codex` requires the `codex` CLI"),
        ("claude-code", "claude", "`--agent claude-code` requires the `claude` CLI"),
        ("openclaw", "openclaw", "`--agent openclaw` requires the `openclaw` CLI"),
    ],
)
def test_preflight_requires_local_agent_cli(
    monkeypatch: pytest.MonkeyPatch,
    agent: str,
    missing: str,
    message: str,
) -> None:
    monkeypatch.setattr(cli.importlib, "import_module", lambda _name: object())
    monkeypatch.setattr(cli.importlib.metadata, "version", lambda _name: "0.13.1")
    monkeypatch.setattr(cli.shutil, "which", lambda name: None if name == missing else f"/bin/{name}")

    with pytest.raises(SystemExit, match=message):
        cli._preflight(agent)
