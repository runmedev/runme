from __future__ import annotations

import itertools
import subprocess
import sys
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import Any, TextIO

from google.protobuf import json_format


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


class RunmeEnvironment:
    def __init__(
        self,
        command: Sequence[str] | None = None,
        cwd: str | Path | None = None,
        env: Mapping[str, str] | None = None,
    ) -> None:
        self.command = list(command or ("runme", "harbor", "stdio"))
        self.cwd = None if cwd is None else str(cwd)
        self.env = None if env is None else dict(env)
        self._ids = itertools.count(1)
        self._process: subprocess.Popen[str] | None = None

    def __enter__(self) -> RunmeEnvironment:
        self.start()
        return self

    def __exit__(self, *_exc: object) -> None:
        self.close()

    @property
    def process(self) -> subprocess.Popen[str]:
        if self._process is None:
            raise RuntimeError("RunmeEnvironment has not been started")
        return self._process

    def start(self) -> None:
        if self._process is not None:
            return
        self._process = subprocess.Popen(
            self.command,
            cwd=self.cwd,
            env=self.env,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding="utf-8",
        )

    def close(self) -> None:
        if self._process is None:
            return
        process = self._process
        if process.stdin is not None and not process.stdin.closed:
            try:
                self.request({"stop": {}})
            except Exception:
                pass
            process.stdin.close()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
        self._process = None

    def preflight(self) -> dict[str, Any]:
        return self.request({"preflight": {}}).get("preflight", {})

    def request(self, payload: Mapping[str, Any], request_id: str | None = None) -> dict[str, Any]:
        self.start()
        rid = request_id or str(next(self._ids))
        request = harbor_pb2.Request()
        json_format.ParseDict({"id": rid, **payload}, request)

        stdin = _required_stream(self.process.stdin, "stdin")
        stdout = _required_stream(self.process.stdout, "stdout")

        stdin.write(_message_to_line(request))
        stdin.flush()

        line = stdout.readline()
        if line == "":
            raise RuntimeError("runme harbor stdio closed stdout")
        response = harbor_pb2.Response()
        json_format.Parse(line, response)

        if response.id != rid:
            raise HarborProtocolError(
                "request_id_mismatch",
                f"expected response id {rid!r}, got {response.id!r}",
            )
        if response.HasField("error"):
            raise HarborProtocolError(response.error.code or "unknown", response.error.message)
        return json_format.MessageToDict(response, preserving_proto_field_name=True)


def _message_to_line(message: Any) -> str:
    return (
        json_format.MessageToJson(
            message,
            preserving_proto_field_name=True,
            indent=None,
        )
        + "\n"
    )


def _required_stream(stream: TextIO | None, name: str) -> TextIO:
    if stream is None:
        raise RuntimeError(f"runme harbor stdio missing {name}")
    return stream
