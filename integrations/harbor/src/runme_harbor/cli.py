from __future__ import annotations

import argparse
import importlib
import importlib.metadata
import re
import sys
from typing import Sequence


ENVIRONMENT_IMPORT_PATH = "runme_harbor.environment:RunmeEnvironment"
CODEX_IMPORT_PATH = "runme_harbor.runme_agents:RunmeCodex"
CLAUDE_IMPORT_PATH = "runme_harbor.runme_agents:RunmeClaudeCode"
CURSOR_IMPORT_PATH = "runme_harbor.runme_agents:RunmeCursorCli"
OPENCLAW_IMPORT_PATH = "runme_harbor.runme_agents:RunmeOpenClaw"
MIN_HARBOR_VERSION = (0, 15, 0)
MAX_HARBOR_VERSION = (0, 16, 0)
AGENT_ARGUMENTS = {
    "oracle": ("--agent", "oracle"),
    "codex": ("--agent-import-path", CODEX_IMPORT_PATH),
    "claude-code": ("--agent-import-path", CLAUDE_IMPORT_PATH),
    "cursor-cli": ("--agent-import-path", CURSOR_IMPORT_PATH),
    "openclaw": ("--agent-import-path", OPENCLAW_IMPORT_PATH),
}


def main(argv: Sequence[str] | None = None) -> int:
    try:
        args = _parse_args(list(argv) if argv is not None else sys.argv[1:])
        if args.command == "sync-metadata":
            return sync_metadata(args)
    except SystemExit as exc:
        if isinstance(exc.code, int):
            return exc.code
        print(exc.code, file=sys.stderr)
        return 1
    return 1


def sync_metadata(args: argparse.Namespace) -> int:
    _preflight_harbor_package()

    from runme_harbor.metadata_sync import sync_jobs_metadata

    synced = sync_jobs_metadata(args.jobs_dir)
    print(f"Synced Harbor metadata for {synced} job(s).")
    return 0


def _parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(prog="runme-harbor", allow_abbrev=False)
    subparsers = parser.add_subparsers(dest="command", required=True)

    sync_parser = subparsers.add_parser("sync-metadata", allow_abbrev=False)
    sync_parser.add_argument("--jobs-dir", default=".runme/evals/jobs")

    return parser.parse_args(argv)


def _preflight_harbor_package() -> None:
    try:
        importlib.import_module("harbor")
    except ModuleNotFoundError as exc:
        raise SystemExit("Runme Harbor requires the `harbor` Python package.") from exc

    version = _version_tuple(importlib.metadata.version("harbor"))
    if version < MIN_HARBOR_VERSION or version >= MAX_HARBOR_VERSION:
        raise SystemExit("Runme Harbor requires harbor>=0.15,<0.16.")


def _version_tuple(value: str) -> tuple[int, int, int]:
    parts = []
    for part in value.split(".")[:3]:
        match = re.match(r"\d+", part)
        parts.append(int(match.group(0)) if match else 0)
    while len(parts) < 3:
        parts.append(0)
    return tuple(parts)  # type: ignore[return-value]
