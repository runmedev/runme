from __future__ import annotations

import argparse
import importlib
import importlib.metadata
import json
import re
import sys
import tomllib
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Sequence
from urllib.parse import urlparse
from urllib.request import url2pathname


PACKAGE_NAME = "runme-harbor"
ENVIRONMENT_IMPORT_PATH = "runme_harbor.environment:RunmeEnvironment"
CODEX_IMPORT_PATH = "runme_harbor.runme_agents:RunmeCodex"
CLAUDE_IMPORT_PATH = "runme_harbor.runme_agents:RunmeClaudeCode"
CURSOR_IMPORT_PATH = "runme_harbor.runme_agents:RunmeCursorCli"
OPENCLAW_IMPORT_PATH = "runme_harbor.runme_agents:RunmeOpenClaw"
MIN_HARBOR_VERSION = (0, 16, 0)
MAX_HARBOR_VERSION = (0, 17, 0)
AGENT_ARGUMENTS = {
    "oracle": ("--agent", "oracle"),
    "codex": ("--agent", CODEX_IMPORT_PATH),
    "claude-code": ("--agent", CLAUDE_IMPORT_PATH),
    "cursor-cli": ("--agent", CURSOR_IMPORT_PATH),
    "openclaw": ("--agent", OPENCLAW_IMPORT_PATH),
}


def main(argv: Sequence[str] | None = None) -> int:
    try:
        args = _parse_args(list(argv) if argv is not None else sys.argv[1:])
        if args.command == "version":
            print(_version_text())
            return 0
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
    parser.add_argument(
        "--version",
        action=_VersionAction,
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("version", allow_abbrev=False)

    sync_parser = subparsers.add_parser("sync-metadata", allow_abbrev=False)
    sync_parser.add_argument("--jobs-dir", default=".runme/evals/jobs")

    return parser.parse_args(argv)


class _VersionAction(argparse.Action):
    def __init__(self, option_strings: Sequence[str], dest: str, **kwargs: object) -> None:
        super().__init__(option_strings=option_strings, dest=dest, nargs=0, **kwargs)

    def __call__(
        self,
        parser: argparse.ArgumentParser,
        namespace: argparse.Namespace,
        values: str | Sequence[str] | None,
        option_string: str | None = None,
    ) -> None:
        print(_version_text())
        parser.exit()


def _package_version() -> str:
    try:
        return importlib.metadata.version(PACKAGE_NAME)
    except importlib.metadata.PackageNotFoundError:
        pyproject = Path(__file__).parents[2] / "pyproject.toml"
        return tomllib.loads(pyproject.read_text())["project"]["version"]


def _version_text() -> str:
    return VersionInfo.detect().format()


@dataclass(frozen=True)
class VersionInfo:
    version: str
    install_source: str
    venv: str
    python: str

    @classmethod
    def detect(cls) -> "VersionInfo":
        return cls(
            version=_package_version(),
            install_source=InstallSource.detect().format(),
            venv=sys.prefix,
            python=sys.executable,
        )

    def format(self) -> str:
        return "\n".join(
            [
                f"runme-harbor {self.version}",
                "",
                f"install: {self.install_source}",
                f"venv: {self.venv}",
                f"python: {self.python}",
            ]
        )


@dataclass(frozen=True)
class InstallSource:
    kind: str
    location: str | None = None
    editable: bool = False
    commit: str | None = None

    @classmethod
    def detect(cls) -> "InstallSource":
        try:
            distribution = importlib.metadata.distribution(PACKAGE_NAME)
        except importlib.metadata.PackageNotFoundError:
            return cls(kind="source", location=str(_project_root()))

        direct_url = distribution.read_text("direct_url.json")
        if not direct_url:
            return cls(kind="package")

        return cls.from_direct_url(direct_url)

    @classmethod
    def from_direct_url(cls, value: str) -> "InstallSource":
        try:
            data = json.loads(value)
        except json.JSONDecodeError:
            return cls(kind="unknown")

        url = data.get("url")
        if not isinstance(url, str):
            return cls(kind="unknown")

        vcs_info = data.get("vcs_info")
        if isinstance(vcs_info, dict):
            return cls(kind="vcs", location=url, commit=_str_value(vcs_info, "commit_id"))

        dir_info = data.get("dir_info")
        if isinstance(dir_info, dict):
            return cls(
                kind="source",
                location=cls._path_from_file_url(url),
                editable=dir_info.get("editable") is True,
            )

        return cls(kind="package")

    def format(self) -> str:
        if self.kind == "package" or self.kind == "unknown":
            return self.kind

        if self.kind == "vcs":
            commit = f"@{self.commit}" if self.commit else ""
            return f"source vcs {self.location}{commit}"

        mode = "source editable" if self.editable else "source"
        if self.location:
            return f"{mode} {self.location}"
        return mode

    @staticmethod
    def _path_from_file_url(url: str) -> str:
        parsed = urlparse(url)
        if parsed.scheme != "file":
            return url
        return url2pathname(parsed.path)


def _str_value(data: dict[str, Any], key: str) -> str | None:
    value = data.get(key)
    return value if isinstance(value, str) else None


def _project_root() -> Path:
    return Path(__file__).parents[2]


def _preflight_harbor_package() -> None:
    try:
        importlib.import_module("harbor")
    except ModuleNotFoundError as exc:
        raise SystemExit("Runme Harbor requires the `harbor` Python package.") from exc

    version = _version_tuple(importlib.metadata.version("harbor"))
    if version < MIN_HARBOR_VERSION or version >= MAX_HARBOR_VERSION:
        raise SystemExit("Runme Harbor requires harbor>=0.16,<0.17.")


def _version_tuple(value: str) -> tuple[int, int, int]:
    parts = []
    for part in value.split(".")[:3]:
        match = re.match(r"\d+", part)
        parts.append(int(match.group(0)) if match else 0)
    while len(parts) < 3:
        parts.append(0)
    return tuple(parts)  # type: ignore[return-value]
