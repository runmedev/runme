from __future__ import annotations

import asyncio
import base64
import os
import shlex
import shutil
import stat
import subprocess
import sys
from pathlib import Path, PurePosixPath
from typing import Any

from google.protobuf import json_format
from harbor.environments.base import BaseEnvironment, ExecResult
from harbor.environments.capabilities import EnvironmentCapabilities
from harbor.models.task.config import EnvironmentConfig
from harbor.models.trial.paths import EnvironmentPaths, TrialPaths


def _load_harbor_pb2():
    try:
        from runme.harbor.v1 import harbor_pb2
    except ModuleNotFoundError:
        repo_proto_path = Path(__file__).resolve().parents[4] / "api" / "gen" / "proto" / "python"
        if repo_proto_path.exists():
            sys.path.insert(0, str(repo_proto_path))
        from runme.harbor.v1 import harbor_pb2
    return harbor_pb2


harbor_pb2 = _load_harbor_pb2()


class HarborProtocolError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message


class RunmeEnvironment(BaseEnvironment):
    """Harbor environment implemented by a long-lived `runme harbor stdio` process."""

    def __init__(
        self,
        environment_dir: Path,
        environment_name: str,
        session_id: str,
        trial_paths: TrialPaths,
        task_env_config: EnvironmentConfig,
        workspace_root: str | None = None,
        runme_bin: str | None = None,
        runme_args: str | list[str] | tuple[str, ...] | None = None,
        command: list[str] | tuple[str, ...] | None = None,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        self._workspace_root = (
            Path(workspace_root).expanduser() if workspace_root else Path.cwd()
        ).resolve()
        self._runme_bin = runme_bin or os.environ.get("RUNME_BIN") or "runme"
        self._runme_args = _parse_runme_args(runme_args)
        self._command = list(command) if command is not None else None
        self._client: _StdioClient | None = None
        self._root = trial_paths.trial_dir.resolve().absolute()
        self._workdir = self._root / "app"
        self._protocol_root = _common_root(self._workspace_root, self._root)
        self._use_workspace_app = True

        super().__init__(
            environment_dir=environment_dir,
            environment_name=environment_name,
            session_id=session_id,
            trial_paths=trial_paths,
            task_env_config=task_env_config,
            *args,
            **kwargs,
        )

    @staticmethod
    def type() -> str:
        return "runme"

    @classmethod
    def preflight(cls) -> None:
        command = _default_command()
        if not shutil.which(command[0]):
            raise SystemExit(
                "Runme Harbor requires the `runme` CLI. Set RUNME_BIN to a Runme "
                "binary or add `runme` to PATH."
            )
        request = _message_to_line({"id": "preflight", "preflight": {}})
        try:
            result = subprocess.run(
                command,
                input=request,
                capture_output=True,
                text=True,
                timeout=5,
                check=False,
            )
        except subprocess.TimeoutExpired as exc:
            raise SystemExit("`runme harbor stdio` did not respond to preflight.") from exc

        if result.returncode != 0 and not result.stdout:
            raise SystemExit(result.stderr or "`runme harbor stdio` preflight failed.")
        first_line = result.stdout.splitlines()[0] if result.stdout else ""
        response = harbor_pb2.Response()
        try:
            json_format.Parse(first_line, response)
        except Exception as exc:
            raise SystemExit("`runme harbor stdio` returned invalid protobuf JSON.") from exc
        if response.HasField("error"):
            raise SystemExit(response.error.message or "`runme harbor stdio` failed.")

    @property
    def capabilities(self) -> EnvironmentCapabilities:
        return EnvironmentCapabilities(mounted=True)

    def _validate_definition(self) -> None:
        return None

    async def start(self, force_build: bool) -> None:
        self.trial_paths.mkdir()
        self._workdir.mkdir(parents=True, exist_ok=True)
        for path in (
            self._root / "agent",
            self._root / "verifier",
            self._root / "artifacts",
            self._root / "tests",
            self._root / "solution",
        ):
            path.mkdir(parents=True, exist_ok=True)

        client = _StdioClient(
            self._command or [self._runme_bin, *self._runme_args, "harbor", "stdio"]
        )
        await client.start()
        self._client = client
        env = _env_map_to_list(self._persistent_env, self.task_env_config.env)
        await self._request(
            {"start": {"root": str(self._protocol_root), "env": env}},
        )

    async def stop(self, delete: bool) -> None:
        if self._client is None:
            return
        try:
            await self._request({"stop": {}})
        finally:
            await self._close_client()
            if delete:
                shutil.rmtree(self._workdir, ignore_errors=True)
                shutil.rmtree(self._root / "tests", ignore_errors=True)
                shutil.rmtree(self._root / "solution", ignore_errors=True)

    async def upload_file(self, source_path: Path | str, target_path: str) -> None:
        source = Path(source_path)
        self._activate_trial_workdir_if_needed(target_path)
        data = self._rewrite_uploaded_bytes(source, source.read_bytes())
        await self._request(
            {
                "upload_file": {
                    "path": self._map_protocol_path(target_path),
                    "data": base64.b64encode(data).decode(),
                    "mode": _file_mode(source),
                }
            }
        )

    async def upload_dir(self, source_dir: Path | str, target_dir: str) -> None:
        source = Path(source_dir)
        self._activate_trial_workdir_if_needed(target_dir)
        files: list[dict[str, Any]] = []
        for path in sorted(source.rglob("*")):
            if not path.is_file():
                continue
            data = self._rewrite_uploaded_bytes(path, path.read_bytes())
            files.append(
                {
                    "path": path.relative_to(source).as_posix(),
                    "data": base64.b64encode(data).decode(),
                    "mode": _file_mode(path),
                }
            )
        await self._request(
            {
                "upload_directory": {
                    "path": self._map_protocol_path(target_dir),
                    "files": files,
                }
            }
        )

    async def download_file(self, source_path: str, target_path: Path | str) -> None:
        response = await self._request(
            {"download_file": {"path": self._map_protocol_path(source_path)}}
        )
        payload = response.download_file
        target = Path(target_path)
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(payload.data)
        if payload.mode:
            target.chmod(payload.mode)

    async def download_dir(self, source_dir: str, target_dir: Path | str) -> None:
        response = await self._request(
            {"download_directory": {"path": self._map_protocol_path(source_dir)}}
        )
        target = Path(target_dir)
        for file in response.download_directory.files:
            path = target / file.path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(file.data)
            if file.mode:
                path.chmod(file.mode)

    async def exec(
        self,
        command: str,
        cwd: str | None = None,
        env: dict[str, str] | None = None,
        timeout_sec: int | None = None,
        user: str | int | None = None,
    ) -> ExecResult:
        request = {
            "exec": {
                "command": self._rewrite_command(command),
                "cwd": self._map_protocol_path(cwd or self.task_env_config.workdir or "/app"),
                "env": _env_map_to_list(env or {}),
            }
        }
        try:
            response = await asyncio.wait_for(
                self._request(request),
                timeout=timeout_sec,
            )
        except TimeoutError:
            await self._close_client()
            return ExecResult(stdout="", stderr="command timed out", return_code=124)
        payload = response.exec
        return ExecResult(
            stdout=payload.stdout.decode(errors="replace"),
            stderr=payload.stderr.decode(errors="replace"),
            return_code=int(payload.exit_code),
        )

    async def _request(self, payload: dict[str, Any]):
        if self._client is None:
            raise RuntimeError("Runme Harbor process is not running.")
        return await self._client.request(payload)

    async def _close_client(self) -> None:
        if self._client is None:
            return
        client = self._client
        self._client = None
        await client.close()

    def _map_protocol_path(self, path: str | PurePosixPath | None) -> str:
        mapped = self._map_remote_path(path)
        try:
            return str(mapped.relative_to(self._protocol_root))
        except ValueError as exc:
            raise ValueError(
                f"path {mapped} escapes Runme Harbor root {self._protocol_root}"
            ) from exc

    def _map_remote_path(self, path: str | PurePosixPath | None) -> Path:
        if path is None or str(path).strip() == "":
            return self._app_path()
        remote = PurePosixPath(str(path))
        mappings = self._protocol_path_mappings()
        for prefix, target in mappings:
            if remote == prefix:
                return target
            try:
                rel = remote.relative_to(prefix)
            except ValueError:
                continue
            return target / Path(rel.as_posix())
        if remote.is_absolute():
            host_path = Path(str(remote)).resolve()
            if (
                not self._use_workspace_app
                and _is_relative_to(host_path, self._workspace_root)
                and not _is_relative_to(host_path, self._root)
            ):
                rel = host_path.relative_to(self._workspace_root)
                return self._workdir / rel
            if _is_relative_to(host_path, self._protocol_root):
                return host_path
            raise ValueError(f"unsupported absolute Harbor path: {path}")
        return self._app_path() / Path(remote.as_posix())

    def _path_mappings(self) -> list[tuple[PurePosixPath, Path]]:
        mappings: list[tuple[PurePosixPath, Path]] = []
        if not self._use_workspace_app:
            mappings.append((PurePosixPath(str(self._workspace_root)), self._workdir))
        mappings.extend(
            [
                (PurePosixPath("/app"), self._app_path()),
                (EnvironmentPaths.tests_dir, self._root / "tests"),
                (EnvironmentPaths.solution_dir, self._root / "solution"),
                (EnvironmentPaths.agent_dir, self._root / "agent"),
                (EnvironmentPaths.verifier_dir, self._root / "verifier"),
                (EnvironmentPaths.artifacts_dir, self._root / "artifacts"),
            ]
        )
        return mappings

    def _protocol_path_mappings(self) -> list[tuple[PurePosixPath, Path]]:
        return [
            (PurePosixPath("/app"), self._app_path()),
            (EnvironmentPaths.tests_dir, self._root / "tests"),
            (EnvironmentPaths.solution_dir, self._root / "solution"),
            (EnvironmentPaths.agent_dir, self._root / "agent"),
            (EnvironmentPaths.verifier_dir, self._root / "verifier"),
            (EnvironmentPaths.artifacts_dir, self._root / "artifacts"),
        ]

    def _app_path(self) -> Path:
        if self._use_workspace_app:
            return self._workspace_root
        return self._workdir

    def _activate_trial_workdir_if_needed(self, target_path: str) -> None:
        if self._targets_app_path(target_path):
            self._use_workspace_app = False
            self._workdir.mkdir(parents=True, exist_ok=True)

    def _targets_app_path(self, path: str | PurePosixPath | None) -> bool:
        if path is None or str(path).strip() == "":
            return True
        remote = PurePosixPath(str(path))
        if not remote.is_absolute():
            return True
        try:
            remote.relative_to(PurePosixPath("/app"))
            return True
        except ValueError:
            pass
        host_path = Path(str(remote)).resolve()
        return _is_relative_to(host_path, self._workspace_root)

    def _rewrite_command(self, command: str) -> str:
        rewritten = command
        for remote, host in self._path_mappings():
            rewritten = _replace_remote_path_tokens(
                rewritten, remote.as_posix(), _shell_quote(str(host))
            )
        return rewritten

    def _rewrite_uploaded_bytes(self, path: Path, data: bytes) -> bytes:
        try:
            text = data.decode()
        except UnicodeDecodeError:
            return data
        if path.suffix == ".sh":
            rewritten = self._rewrite_command(text)
        else:
            rewritten = self._rewrite_text_paths(text)
        return rewritten.encode()

    def _rewrite_text_paths(self, text: str) -> str:
        rewritten = text
        for remote, host in self._path_mappings():
            rewritten = _replace_remote_path_tokens(rewritten, remote.as_posix(), str(host))
        return rewritten


class _StdioClient:
    def __init__(self, command: list[str]) -> None:
        self.command = command
        self._ids = 0
        self._process: asyncio.subprocess.Process | None = None

    async def start(self) -> None:
        if self._process is not None:
            return
        self._process = await asyncio.create_subprocess_exec(
            *self.command,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

    async def close(self) -> None:
        if self._process is None:
            return
        process = self._process
        self._process = None
        if process.stdin is not None and not process.stdin.is_closing():
            process.stdin.close()
        try:
            await asyncio.wait_for(process.wait(), timeout=5)
        except TimeoutError:
            process.kill()
            await process.wait()

    async def request(self, payload: dict[str, Any]):
        await self.start()
        process = self._required_process()
        if process.stdin is None or process.stdout is None:
            raise RuntimeError("runme harbor stdio missing stdin/stdout")

        self._ids += 1
        request_id = str(self._ids)
        process.stdin.write(_message_to_line({"id": request_id, **payload}).encode())
        await process.stdin.drain()

        line = await process.stdout.readline()
        if not line:
            stderr = await self._read_stderr()
            raise RuntimeError(f"runme harbor stdio exited unexpectedly. {stderr}")

        response = harbor_pb2.Response()
        json_format.Parse(line.decode(), response)
        if response.id != request_id:
            raise HarborProtocolError(
                "request_id_mismatch",
                f"expected response id {request_id!r}, got {response.id!r}",
            )
        if response.HasField("error"):
            raise HarborProtocolError(response.error.code or "unknown", response.error.message)
        return response

    def _required_process(self) -> asyncio.subprocess.Process:
        if self._process is None:
            raise RuntimeError("runme harbor stdio has not been started")
        return self._process

    async def _read_stderr(self) -> str:
        process = self._required_process()
        if process.stderr is None:
            return ""
        try:
            data = await asyncio.wait_for(process.stderr.read(), timeout=0.1)
        except TimeoutError:
            return ""
        return data.decode(errors="replace")


def _default_command() -> list[str]:
    return [os.environ.get("RUNME_BIN") or "runme", *_parse_runme_args(None), "harbor", "stdio"]


def _parse_runme_args(value: str | list[str] | tuple[str, ...] | None) -> list[str]:
    if value is None:
        value = os.environ.get("RUNME_ARGS", "")
    if isinstance(value, str):
        return shlex.split(value)
    return [str(arg) for arg in value]


def _message_to_line(payload: dict[str, Any]) -> str:
    request = harbor_pb2.Request()
    json_format.ParseDict(payload, request)
    return (
        json_format.MessageToJson(
            request,
            preserving_proto_field_name=True,
            indent=None,
        )
        + "\n"
    )


def _env_map_to_list(*maps: dict[str, str] | None) -> list[str]:
    env: dict[str, str] = {}
    for values in maps:
        if values:
            env.update({str(key): str(value) for key, value in values.items()})
    return [f"{key}={value}" for key, value in sorted(env.items())]


def _file_mode(path: Path) -> int:
    try:
        return stat.S_IMODE(path.stat().st_mode)
    except OSError:
        return 0o644


def _common_root(*paths: Path) -> Path:
    return Path(os.path.commonpath([str(path) for path in paths]))


def _is_relative_to(path: Path, root: Path) -> bool:
    try:
        path.relative_to(root)
    except ValueError:
        return False
    return True


def _replace_remote_path_tokens(command: str, remote: str, replacement: str) -> str:
    out: list[str] = []
    i = 0
    while i < len(command):
        idx = command.find(remote, i)
        if idx < 0:
            out.append(command[i:])
            break
        end = idx + len(remote)
        if not _path_boundary_before(command, idx) or not _path_boundary_after(command, end):
            out.append(command[i:end])
            i = end
            continue
        out.append(command[i:idx])
        out.append(replacement)
        i = end
    return "".join(out)


def _path_boundary_before(value: str, idx: int) -> bool:
    if idx == 0:
        return True
    return value[idx - 1] in "'\"` \t\n=:("


def _path_boundary_after(value: str, idx: int) -> bool:
    if idx == len(value):
        return True
    return value[idx] in "/'\"` \t\n;&|><):"


def _shell_quote(value: str) -> str:
    return "'" + value.replace("'", "'\"'\"'") + "'"
