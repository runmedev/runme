from __future__ import annotations

import argparse
import importlib
import importlib.metadata
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Sequence


ENVIRONMENT_IMPORT_PATH = "runme_harbor.environment:RunmeEnvironment"
CODEX_IMPORT_PATH = "runme_harbor.runme_agents:RunmeCodex"
CLAUDE_IMPORT_PATH = "runme_harbor.runme_agents:RunmeClaudeCode"
OPENCLAW_IMPORT_PATH = "runme_harbor.runme_agents:RunmeOpenClaw"
MIN_HARBOR_VERSION = (0, 13, 1)
MAX_HARBOR_VERSION = (0, 14, 0)
SKIP_METADATA_SYNC_ENV = "RUNME_HARBOR_SKIP_METADATA_SYNC"
AGENT_ARGUMENTS = {
    "oracle": ("--agent", "oracle"),
    "codex": ("--agent-import-path", CODEX_IMPORT_PATH),
    "claude-code": ("--agent-import-path", CLAUDE_IMPORT_PATH),
    "openclaw": ("--agent-import-path", OPENCLAW_IMPORT_PATH),
}


def main(argv: Sequence[str] | None = None) -> int:
    try:
        args = _parse_args(list(argv) if argv is not None else sys.argv[1:])
        if args.command == "run":
            return run(args)
    except SystemExit as exc:
        if isinstance(exc.code, int):
            return exc.code
        print(exc.code, file=sys.stderr)
        return 1
    return 1


def run(args: argparse.Namespace) -> int:
    _preflight(args.agent)
    command = build_harbor_command(args)
    if args.debug:
        print(_command_string(command), file=sys.stderr)
    exit_code = subprocess.call(command)
    if not _skip_metadata_sync():
        from runme_harbor.metadata_sync import sync_jobs_metadata

        try:
            sync_jobs_metadata(args.jobs_dir)
        except Exception as exc:
            print(f"warning: failed to sync Harbor job metadata: {exc}", file=sys.stderr)
    return exit_code


def build_harbor_command(args: argparse.Namespace) -> list[str]:
    dataset_path = str(Path(args.dataset_path).expanduser().resolve())
    command = [
        "harbor",
        "run",
        "--path",
        dataset_path,
        "--jobs-dir",
        args.jobs_dir,
        "--environment-import-path",
        ENVIRONMENT_IMPORT_PATH,
    ]

    try:
        command.extend(AGENT_ARGUMENTS[args.agent])
    except KeyError as exc:
        valid_agents = ", ".join(AGENT_ARGUMENTS)
        raise SystemExit(f"invalid --agent {args.agent!r}: expected {valid_agents}") from exc

    if args.task_dir:
        command.extend(["--include-task-name", args.task_dir])
    if args.yes:
        command.append("-y")
    passthrough = list(args.passthrough)
    if not _contains_concurrency_flag(passthrough):
        command.extend(["--n-concurrent", "1"])
    command.extend(passthrough)
    return command


def _parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(prog="runme-harbor", allow_abbrev=False)
    subparsers = parser.add_subparsers(dest="command", required=True)

    run_parser = subparsers.add_parser("run", allow_abbrev=False)
    run_parser.add_argument("dataset_path", metavar="dataset-path")
    run_parser.add_argument("--agent", choices=tuple(AGENT_ARGUMENTS), default="oracle")
    run_parser.add_argument("--task-dir")
    run_parser.add_argument("--jobs-dir", default=".runme/evals/jobs")
    run_parser.add_argument("-y", "--yes", action="store_true")
    run_parser.add_argument("--debug", action="store_true")

    args, passthrough = parser.parse_known_args(argv)
    removed_task_flag = _find_removed_task_flag(passthrough)
    if removed_task_flag:
        parser.error(f"unrecognized arguments: {removed_task_flag}")
    if passthrough and passthrough[0] == "--":
        passthrough = passthrough[1:]
    args.passthrough = passthrough
    return args


def _find_removed_task_flag(args: list[str]) -> str | None:
    for arg in args:
        if arg in {"--task", "--task-name"}:
            return arg
        if arg.startswith("--task="):
            return "--task"
        if arg.startswith("--task-name="):
            return "--task-name"
    return None


def _preflight(agent: str) -> None:
    try:
        importlib.import_module("harbor")
    except ModuleNotFoundError as exc:
        raise SystemExit("Runme Harbor requires the `harbor` Python package.") from exc
    if not shutil.which("harbor"):
        raise SystemExit("Runme Harbor requires the `harbor` CLI on PATH.")

    version = _version_tuple(importlib.metadata.version("harbor"))
    if version < MIN_HARBOR_VERSION or version >= MAX_HARBOR_VERSION:
        raise SystemExit("Runme Harbor requires harbor>=0.13.1,<0.14.")

    try:
        importlib.import_module("runme_harbor")
    except ModuleNotFoundError as exc:
        raise SystemExit("Runme Harbor could not import `runme_harbor`.") from exc

    if agent == "codex" and not shutil.which("codex"):
        raise SystemExit("`--agent codex` requires the `codex` CLI on PATH.")
    if agent == "claude-code" and not shutil.which("claude"):
        raise SystemExit("`--agent claude-code` requires the `claude` CLI on PATH.")
    if agent == "openclaw" and not shutil.which("openclaw"):
        raise SystemExit("`--agent openclaw` requires the `openclaw` CLI on PATH.")


def _skip_metadata_sync() -> bool:
    return os.environ.get(SKIP_METADATA_SYNC_ENV, "").lower() in {"1", "true", "yes"}


def _contains_concurrency_flag(args: list[str]) -> bool:
    for index, arg in enumerate(args):
        if arg == "-n" or arg == "--n-concurrent":
            return True
        if arg.startswith("--n-concurrent="):
            return True
        if arg.startswith("-n") and len(arg) > 2:
            return True
        if index > 0 and args[index - 1] in {"-n", "--n-concurrent"}:
            continue
    return False


def _version_tuple(value: str) -> tuple[int, int, int]:
    parts = []
    for part in value.split(".")[:3]:
        match = re.match(r"\d+", part)
        parts.append(int(match.group(0)) if match else 0)
    while len(parts) < 3:
        parts.append(0)
    return tuple(parts)  # type: ignore[return-value]


def _command_string(command: list[str]) -> str:
    import shlex

    return shlex.join(command)


if __name__ == "__main__":
    raise SystemExit(main())
