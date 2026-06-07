import asyncio
from pathlib import Path
from typing import Any

import pytest
from google.protobuf import json_format
from harbor.environments.base import BaseEnvironment
from harbor.models.task.config import EnvironmentConfig
from harbor.models.trial.paths import TrialPaths

from runme_harbor import environment as env_module
from runme_harbor.environment import HarborProtocolError, RunmeEnvironment


class FakeClient:
    instances: list["FakeClient"] = []

    def __init__(self, command: list[str]) -> None:
        self.command = command
        self.requests: list[dict[str, Any]] = []
        self.closed = False
        FakeClient.instances.append(self)

    async def start(self) -> None:
        return None

    async def close(self) -> None:
        self.closed = True

    async def request(self, payload: dict[str, Any]):
        self.requests.append(payload)
        if "preflight" in payload:
            return _response({"preflight": {"protocol": "runme.harbor.stdio"}})
        if "start" in payload:
            return _response({"start": {"root": payload["start"]["root"]}})
        if "stop" in payload:
            return _response({"stop": {}})
        if "exec" in payload:
            return _response(
                {
                    "exec": {
                        "stdout": "b2sK",
                        "stderr": "",
                        "exit_code": 0,
                    }
                }
            )
        return _response({})


def _response(payload: dict[str, Any]):
    response = env_module.harbor_pb2.Response(id="1")
    json_format.ParseDict({"id": "1", **payload}, response)
    return response


def _make_env(tmp_path: Path, **kwargs: Any) -> RunmeEnvironment:
    workspace_root = kwargs.pop("workspace_root", tmp_path)
    trial_paths = TrialPaths(tmp_path / "trial")
    return RunmeEnvironment(
        environment_dir=tmp_path / "environment",
        environment_name="local-agent",
        session_id="trial-1",
        trial_paths=trial_paths,
        task_env_config=EnvironmentConfig(workdir="/app", env={"TASK_ENV": "yes"}),
        workspace_root=str(workspace_root),
        **kwargs,
    )


def test_runme_environment_is_harbor_base_environment(tmp_path: Path) -> None:
    environment = _make_env(tmp_path)
    assert isinstance(environment, BaseEnvironment)
    assert environment.type() == "runme"


def test_environment_starts_runme_harbor_stdio(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))

    client = FakeClient.instances[0]
    assert client.command == ["runme", "harbor", "stdio"]
    assert client.requests[0] == {
        "start": {
            "root": str(tmp_path),
            "env": ["TASK_ENV=yes"],
        }
    }


def test_exec_uses_workspace_root_when_workdir_is_not_uploaded(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    result = asyncio.run(
        environment.exec(
            "printf ok > /app/result.txt && cp /app/result.txt /logs/verifier/result.txt",
            cwd="/app",
        )
    )

    client = FakeClient.instances[0]
    request = client.requests[-1]["exec"]
    assert request["cwd"] == "."
    assert f"{env_module._shell_quote(str(tmp_path))}/result.txt" in request["command"]
    assert str(tmp_path / "trial" / "verifier") in request["command"]
    assert result.stdout == "ok\n"
    assert result.return_code == 0


def test_upload_to_app_uses_workspace_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "setup.sh").write_text("printf setup\n")

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "/app"))
    asyncio.run(environment.exec("printf ok > /app/result.txt", cwd="/app"))

    client = FakeClient.instances[0]
    upload_request = client.requests[-2]["upload_directory"]
    exec_request = client.requests[-1]["exec"]
    assert upload_request["path"] == "."
    assert exec_request["cwd"] == "."
    assert f"{env_module._shell_quote(str(tmp_path))}/result.txt" in exec_request["command"]


def test_relative_upload_uses_workspace_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "setup.sh").write_text("printf setup\n")

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "workdir"))
    asyncio.run(environment.exec("bash workdir/setup.sh", cwd="/app"))

    client = FakeClient.instances[0]
    upload_request = client.requests[-2]["upload_directory"]
    exec_request = client.requests[-1]["exec"]
    assert upload_request["path"] == "workdir"
    assert exec_request["cwd"] == "."
    assert exec_request["command"] == "bash workdir/setup.sh"


def test_upload_dir_rewrites_shell_scripts(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "test.sh").write_text("printf 1.0 > /logs/verifier/reward.txt\n")

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "/tests"))

    request = FakeClient.instances[0].requests[-1]["upload_directory"]
    assert request["path"] == "trial/tests"
    decoded = env_module.base64.b64decode(request["files"][0]["data"]).decode()
    assert str(tmp_path / "trial" / "verifier") in decoded
    assert "/logs/verifier" not in decoded


def test_upload_dir_rewrites_python_verifier_paths(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "llm_judge.py").write_text(
        'poem = Path("/app/poem.txt").read_text()\n'
        'Path("/logs/verifier/reward.json").write_text("{}")\n'
    )

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "/tests"))

    request = FakeClient.instances[0].requests[-1]["upload_directory"]
    decoded = env_module.base64.b64decode(request["files"][0]["data"]).decode()
    assert str(tmp_path / "poem.txt") in decoded
    assert str(tmp_path / "trial" / "verifier" / "reward.json") in decoded
    assert 'Path("/app/poem.txt")' not in decoded
    assert 'Path("/logs/verifier/reward.json")' not in decoded


def test_protocol_error_raises_runtime_error() -> None:
    client = env_module._StdioClient(["runme", "harbor", "stdio"])
    response = env_module.harbor_pb2.Response(
        id="other",
        preflight=env_module.harbor_pb2.PreflightResponse(protocol="runme.harbor.stdio"),
    )
    with pytest.raises(HarborProtocolError) as exc:
        if response.id != "expected":
            raise HarborProtocolError(
                "request_id_mismatch",
                f"expected response id 'expected', got {response.id!r}",
            )

    assert client.command == ["runme", "harbor", "stdio"]
    assert exc.value.code == "request_id_mismatch"
