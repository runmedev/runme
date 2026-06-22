from __future__ import annotations

import sys
import tomllib
from pathlib import Path

import pytest

from runme_harbor import cli
from runme_harbor import metadata_sync


def parse(argv: list[str]):
    return cli._parse_args(argv)


def make_executable(path: Path) -> Path:
    path.write_text("#!/bin/sh\nexit 0\n")
    path.chmod(path.stat().st_mode | 0o111)
    return path


def test_build_harbor_command_oracle_defaults(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path)])

    assert cli.build_harbor_command(args) == [
        "runme-harbor-harbor",
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


def test_build_harbor_command_runme_env_alias(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--env", "runme"])

    assert cli.build_harbor_command(args) == [
        "runme-harbor-harbor",
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


def test_build_harbor_command_builtin_env_uses_harbor_agent(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--env", "docker", "--agent", "codex"])

    command = cli.build_harbor_command(args)

    assert command == [
        "runme-harbor-harbor",
        "run",
        "--path",
        str(tmp_path.resolve()),
        "--jobs-dir",
        ".runme/evals/jobs",
        "--env",
        "docker",
        "--agent",
        "codex",
        "--n-concurrent",
        "1",
    ]


def test_build_harbor_command_builtin_env_accepts_unknown_agent(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "-e", "docker", "--agent", "goose"])

    command = cli.build_harbor_command(args)

    assert command[command.index("--env") : command.index("--env") + 2] == ["--env", "docker"]
    assert command[command.index("--agent") : command.index("--agent") + 2] == ["--agent", "goose"]


def test_build_harbor_command_accepts_resolved_harbor_bin(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path)])

    command = cli.build_harbor_command(args, harbor_bin=tmp_path / "bin" / "runme-harbor-harbor")

    assert command[0] == str(tmp_path / "bin" / "runme-harbor-harbor")


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


def test_build_harbor_command_cursor_cli(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "cursor-cli"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.runme_agents:RunmeCursorCli" in command
    assert "--agent" not in command


def test_build_harbor_command_openclaw(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "openclaw"])

    command = cli.build_harbor_command(args)

    assert "--agent-import-path" in command
    assert "runme_harbor.runme_agents:RunmeOpenClaw" in command
    assert "--agent" not in command


def test_build_harbor_command_runme_env_rejects_unknown_agent(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "goose"])

    with pytest.raises(SystemExit, match="invalid --agent 'goose'"):
        cli.build_harbor_command(args)


def test_build_harbor_command_runme_env_rejects_cursor_alias(tmp_path: Path) -> None:
    args = parse(["run", str(tmp_path), "--agent", "cursor"])

    with pytest.raises(SystemExit, match="invalid --agent 'cursor'"):
        cli.build_harbor_command(args)


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
    assert ["--jobs-dir", "jobs"] == command[
        command.index("--jobs-dir") : command.index("--jobs-dir") + 2
    ]
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


@pytest.mark.parametrize(
    "passthrough",
    [
        ["--", "--env", "docker"],
        ["--", "--env=docker"],
        ["--", "-e", "docker"],
        ["--", "-edocker"],
        ["--", "--environment-import-path", "pkg:Env"],
        ["--", "--environment-import-path=pkg:Env"],
    ],
)
def test_build_harbor_command_rejects_environment_passthrough(
    tmp_path: Path,
    passthrough: list[str],
) -> None:
    args = parse(["run", str(tmp_path), *passthrough])

    with pytest.raises(SystemExit, match="use runme-harbor run --env"):
        cli.build_harbor_command(args)


def test_main_runs_bundled_harbor_and_prints_debug(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    calls: list[list[str]] = []
    harbor_bin = tmp_path / "bin" / "runme-harbor-harbor"
    monkeypatch.setattr(cli, "_preflight", lambda _agent: harbor_bin)
    monkeypatch.setattr(cli.subprocess, "call", lambda command: calls.append(command) or 7)

    assert cli.main(["run", str(tmp_path), "--debug"]) == 7

    assert calls[0][0:2] == [str(harbor_bin), "run"]
    assert f"{harbor_bin} run" in capsys.readouterr().err


def test_main_syncs_metadata_after_harbor_run(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight", lambda _agent: tmp_path / "runme-harbor-harbor")
    monkeypatch.setattr(cli.subprocess, "call", lambda _command: 7)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["run", str(tmp_path), "--jobs-dir", "jobs"]) == 7

    assert synced == ["jobs"]


def test_main_skips_metadata_sync_when_disabled(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setenv(cli.SKIP_METADATA_SYNC_ENV, "1")
    monkeypatch.setattr(cli, "_preflight", lambda _agent: tmp_path / "runme-harbor-harbor")
    monkeypatch.setattr(cli.subprocess, "call", lambda _command: 0)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["run", str(tmp_path), "--jobs-dir", "jobs"]) == 0

    assert synced == []


def test_main_sync_metadata_command_uses_default_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    synced: list[str] = []
    preflighted: list[bool] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", lambda: preflighted.append(True))
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 3
    )

    assert cli.main(["sync-metadata"]) == 0

    assert preflighted == [True]
    assert synced == [".runme/evals/jobs"]
    assert capsys.readouterr().out == "Synced Harbor metadata for 3 job(s).\n"


def test_main_sync_metadata_command_accepts_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", lambda: None)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["sync-metadata", "--jobs-dir", "jobs"]) == 0

    assert synced == ["jobs"]


def test_main_sync_metadata_command_reports_preflight_errors(
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    def fail_preflight() -> None:
        raise SystemExit("Runme Harbor requires the `harbor` Python package.")

    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", fail_preflight)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["sync-metadata"]) == 1

    assert synced == []
    assert capsys.readouterr().err == "Runme Harbor requires the `harbor` Python package.\n"


def test_pyproject_exposes_runme_owned_scripts_only() -> None:
    pyproject = Path(__file__).parents[1] / "pyproject.toml"
    scripts = tomllib.loads(pyproject.read_text())["project"]["scripts"]

    assert scripts["runme-harbor"] == "runme_harbor.cli:main"
    assert scripts["runme-harbor-harbor"] == "harbor.cli.main:app"
    assert "harbor" not in scripts
    assert "hr" not in scripts
    assert "hb" not in scripts


def test_preflight_uses_bundled_harbor_executable(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    runme_harbor = make_executable(bin_dir / "runme-harbor")
    bundled_harbor = make_executable(bin_dir / "runme-harbor-harbor")

    monkeypatch.setattr(sys, "argv", [str(runme_harbor), "run"])
    monkeypatch.setattr(cli.importlib, "import_module", lambda _name: object())
    monkeypatch.setattr(cli.importlib.metadata, "version", lambda _name: "0.13.1")
    monkeypatch.setattr(cli.shutil, "which", lambda name: f"/usr/local/bin/{name}")

    assert cli._preflight("oracle") == bundled_harbor.resolve()


def test_preflight_rejects_global_harbor_without_bundled_sibling(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    runme_harbor = make_executable(bin_dir / "runme-harbor")
    global_bin = tmp_path / "global-bin"
    global_bin.mkdir()
    make_executable(global_bin / "harbor")

    monkeypatch.setenv("PATH", str(global_bin))
    monkeypatch.setattr(sys, "argv", [str(runme_harbor), "run"])
    monkeypatch.setattr(cli.importlib, "import_module", lambda _name: object())
    monkeypatch.setattr(cli.importlib.metadata, "version", lambda _name: "0.13.1")

    with pytest.raises(SystemExit, match="bundled `runme-harbor-harbor` executable"):
        cli._preflight("oracle")


@pytest.mark.parametrize(
    ("agent", "missing", "message"),
    [
        ("codex", "codex", "`--agent codex` requires the `codex` CLI"),
        ("claude-code", "claude", "`--agent claude-code` requires the `claude` CLI"),
        ("cursor-cli", "cursor-agent", "`--agent cursor-cli` requires the `cursor-agent` CLI"),
        ("openclaw", "openclaw", "`--agent openclaw` requires the `openclaw` CLI"),
    ],
)
def test_preflight_requires_runme_agent_cli(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
    agent: str,
    missing: str,
    message: str,
) -> None:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    runme_harbor = make_executable(bin_dir / "runme-harbor")
    make_executable(bin_dir / "runme-harbor-harbor")

    monkeypatch.setattr(sys, "argv", [str(runme_harbor), "run"])
    monkeypatch.setattr(cli.importlib, "import_module", lambda _name: object())
    monkeypatch.setattr(cli.importlib.metadata, "version", lambda _name: "0.13.1")
    monkeypatch.setattr(
        cli.shutil, "which", lambda name: None if name == missing else f"/bin/{name}"
    )

    with pytest.raises(SystemExit, match=message):
        cli._preflight(agent)
