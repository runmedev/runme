from __future__ import annotations

from pathlib import Path

import pytest

from runme_harbor import cli
from runme_harbor import metadata_sync


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
        ".runme/evals/jobs",
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
    assert "runme_harbor.runme_agents:RunmeCodex" in command
    assert "--agent" not in command


def test_build_harbor_command_claude(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "claude-code"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.runme_agents:RunmeClaudeCode" in command
    assert "--agent" not in command


def test_build_harbor_command_openclaw(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "openclaw"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.runme_agents:RunmeOpenClaw" in command
    assert "--agent" not in command


def test_build_harbor_command_task_yes_jobs_and_passthrough(tmp_path: Path) -> None:
    args = parse(
        [
            "run",
            str(tmp_path),
            "--task-dir",
            "simple-agent",
            "--jobs-dir",
            "jobs",
            "-y",
            "--",
            "--model",
            "gpt-5",
        ]
    )

    command = cli.build_harbor_command(args)

    assert ["--include-task-name", "simple-agent"] == command[
        command.index("--include-task-name") : command.index("--include-task-name") + 2
    ]
    assert ["--jobs-dir", "jobs"] == command[command.index("--jobs-dir") : command.index("--jobs-dir") + 2]
    assert "-y" in command
    assert command[-2:] == ["--model", "gpt-5"]


@pytest.mark.parametrize(
    "task_flag",
    ["--task", "--task=simple-agent", "--task-name", "--task-name=simple-agent"],
)
def test_parse_rejects_task_flag(tmp_path: Path, task_flag: str) -> None:
    args = ["run", str(tmp_path), task_flag]
    if task_flag in {"--task", "--task-name"}:
        args.append("simple-agent")

    with pytest.raises(SystemExit):
        parse(args)


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


def test_main_syncs_metadata_after_harbor_run(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight", lambda _agent: None)
    monkeypatch.setattr(cli.subprocess, "call", lambda _command: 7)
    monkeypatch.setattr(metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1)

    assert cli.main(["run", str(tmp_path), "--jobs-dir", "jobs"]) == 7

    assert synced == ["jobs"]


def test_main_skips_metadata_sync_when_disabled(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setenv(cli.SKIP_METADATA_SYNC_ENV, "1")
    monkeypatch.setattr(cli, "_preflight", lambda _agent: None)
    monkeypatch.setattr(cli.subprocess, "call", lambda _command: 0)
    monkeypatch.setattr(metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1)

    assert cli.main(["run", str(tmp_path), "--jobs-dir", "jobs"]) == 0

    assert synced == []


def test_main_sync_metadata_command_uses_default_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 3)

    assert cli.main(["sync-metadata"]) == 0

    assert synced == [".runme/evals/jobs"]
    assert capsys.readouterr().out == "Synced Harbor metadata for 3 job(s).\n"


def test_main_sync_metadata_command_accepts_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1)

    assert cli.main(["sync-metadata", "--jobs-dir", "jobs"]) == 0

    assert synced == ["jobs"]


@pytest.mark.parametrize(
    ("agent", "missing", "message"),
    [
        ("codex", "codex", "`--agent codex` requires the `codex` CLI"),
        ("claude-code", "claude", "`--agent claude-code` requires the `claude` CLI"),
        ("openclaw", "openclaw", "`--agent openclaw` requires the `openclaw` CLI"),
    ],
)
def test_preflight_requires_runme_agent_cli(
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
