from __future__ import annotations

import itertools
import json
import subprocess
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import Any, TextIO


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
        message = {"id": rid, **payload}
        stdin = _required_stream(self.process.stdin, "stdin")
        stdout = _required_stream(self.process.stdout, "stdout")

        stdin.write(json.dumps(message, separators=(",", ":")) + "\n")
        stdin.flush()

        line = stdout.readline()
        if line == "":
            raise RuntimeError("runme harbor stdio closed stdout")
        response = json.loads(line)
        if response.get("id") != rid:
            raise HarborProtocolError(
                "request_id_mismatch",
                f"expected response id {rid!r}, got {response.get('id')!r}",
            )
        error = response.get("error")
        if error:
            raise HarborProtocolError(error.get("code", "unknown"), error.get("message", ""))
        return response


def _required_stream(stream: TextIO | None, name: str) -> TextIO:
    if stream is None:
        raise RuntimeError(f"runme harbor stdio missing {name}")
    return stream
