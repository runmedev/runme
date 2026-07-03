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
    task_env_config = kwargs.pop(
        "task_env_config",
        EnvironmentConfig(workdir="/app", env={"TASK_ENV": "yes"}),
    )
    trial_paths = TrialPaths(tmp_path / "trial")
    return RunmeEnvironment(
        environment_dir=tmp_path / "environment",
        environment_name="simple-agent",
        session_id="trial-1",
        trial_paths=trial_paths,
        task_env_config=task_env_config,
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
            "env": [
                f"RUNME_AGENT_LOG_DIR={tmp_path / 'trial' / 'agent'}",
                f"RUNME_ARTIFACTS_DIR={tmp_path / 'trial' / 'artifacts'}",
                f"RUNME_LOGS_DIR={tmp_path / 'trial'}",
                "RUNME_REWARD_DETAILS_PATH="
                f"{tmp_path / 'trial' / 'verifier' / 'reward-details.json'}",
                f"RUNME_REWARD_PATH={tmp_path / 'trial' / 'verifier' / 'reward.json'}",
                f"RUNME_TASK_DIR={tmp_path}",
                "RUNME_TASK_NAME=simple-agent",
                f"RUNME_TASK_WORKDIR={tmp_path}",
                f"RUNME_TESTS_DIR={tmp_path / 'trial' / 'tests'}",
                f"RUNME_VERIFIER_DIR={tmp_path / 'trial' / 'verifier'}",
                "TASK_ENV=yes",
            ],
        }
    }


def test_stdio_client_allows_large_protojson_lines(monkeypatch: pytest.MonkeyPatch) -> None:
    captured: dict[str, Any] = {}

    async def create_subprocess_exec(*command: str, **kwargs: Any):
        captured["command"] = command
        captured["kwargs"] = kwargs
        return object()

    monkeypatch.setattr(
        env_module.asyncio,
        "create_subprocess_exec",
        create_subprocess_exec,
    )

    client = env_module._StdioClient(["runme", "harbor", "stdio"])
    asyncio.run(client.start())

    assert captured["command"] == ("runme", "harbor", "stdio")
    assert captured["kwargs"]["limit"] == 32 * 1024 * 1024


def test_path_mappings_are_most_specific_first(tmp_path: Path) -> None:
    environment = _make_env(tmp_path)

    remote_paths = [remote.as_posix() for remote, _ in environment._path_mappings()]

    assert remote_paths.index("/logs/verifier") < remote_paths.index("/app")


def test_nested_configured_workdir_is_staged_per_trial(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)
    monkeypatch.setattr(
        env_module,
        "_is_git_ignored",
        lambda _root, path: path.name in {"__pycache__", "results.json"},
    )

    workspace = tmp_path / "repo"
    source = workspace / "examples" / "harbor" / "task" / "workdir"
    source.mkdir(parents=True)
    (source / ".gitkeep").write_text("")
    (source / "results.json").write_text("{}")
    (source / "__pycache__").mkdir()
    (source / "__pycache__" / "textstats.pyc").write_bytes(b"cache")
    sample_target = source.parent / "environment" / "sample.txt"
    sample_target.parent.mkdir()
    sample_target.write_text("sample")
    (source / "sample.txt").symlink_to("../environment/sample.txt")

    configured_workdir = "/app/examples/harbor/task/workdir"
    environment = _make_env(
        tmp_path,
        workspace_root=workspace,
        task_env_config=EnvironmentConfig(
            workdir=configured_workdir,
            env={"TASK_ENV": "yes"},
        ),
    )

    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.exec("cat /app/examples/harbor/task/workdir/sample.txt"))

    staged = tmp_path / "trial" / "workdir"
    assert (staged / ".gitkeep").exists()
    assert not (staged / "sample.txt").is_symlink()
    assert (staged / "sample.txt").read_text() == "sample"
    assert not (staged / "results.json").exists()
    assert not (staged / "__pycache__").exists()

    client = FakeClient.instances[0]
    assert client.requests[0]["start"]["root"] == str(tmp_path)
    assert f"RUNME_TASK_WORKDIR={staged}" in client.requests[0]["start"]["env"]
    request = client.requests[-1]["exec"]
    assert request["cwd"] == "trial/workdir"
    assert str(staged) in request["command"]
    assert "/app/examples/harbor/task/workdir" not in request["command"]


def test_missing_nested_configured_workdir_stages_workspace_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    workspace = tmp_path / "repo"
    workspace.mkdir()
    (workspace / "go.mod").write_text("module example.com/repo\n")

    configured_workdir = "/app/evals/tasks/update-minor-deps/workdir"
    environment = _make_env(
        tmp_path,
        workspace_root=workspace,
        task_env_config=EnvironmentConfig(
            workdir=configured_workdir,
            env={"TASK_ENV": "yes"},
        ),
    )

    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.exec("git status --short"))

    staged = tmp_path / "trial" / "workdir"
    assert (staged / "go.mod").read_text() == "module example.com/repo\n"

    client = FakeClient.instances[0]
    assert f"RUNME_TASK_WORKDIR={staged}" in client.requests[0]["start"]["env"]
    request = client.requests[-1]["exec"]
    assert request["cwd"] == "trial/workdir"
    assert configured_workdir not in request["command"]


def test_absolute_workdir_outside_app_falls_back_to_workspace_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    environment = _make_env(
        tmp_path,
        task_env_config=EnvironmentConfig(
            workdir="/workspace",
            env={"TASK_ENV": "yes"},
        ),
    )

    # An absolute workdir outside /app cannot be mapped into the Harbor root and
    # must not abort start(); RUNME_TASK_WORKDIR falls back to the workspace root.
    asyncio.run(environment.start(force_build=False))

    env = FakeClient.instances[0].requests[0]["start"]["env"]
    assert f"RUNME_TASK_WORKDIR={tmp_path}" in env


def test_missing_configured_workdir_does_not_recurse_into_trial_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    # Trial root lives under the workspace, so staging the whole workspace (the
    # not-a-dir fallback) must not copy the trial dir into its own destination.
    (tmp_path / "go.mod").write_text("module example.com/repo\n")

    configured_workdir = "/app/does-not-exist"
    environment = _make_env(
        tmp_path,
        workspace_root=tmp_path,
        task_env_config=EnvironmentConfig(
            workdir=configured_workdir,
            env={"TASK_ENV": "yes"},
        ),
    )

    asyncio.run(environment.start(force_build=False))

    staged = tmp_path / "trial" / "workdir"
    assert (staged / "go.mod").read_text() == "module example.com/repo\n"
    assert not (staged / "trial").exists()


def test_runtime_task_workdir_overrides_configured_env(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    environment = _make_env(
        tmp_path,
        task_env_config=EnvironmentConfig(
            workdir="/app",
            env={
                "RUNME_ARTIFACTS_DIR": "/wrong",
                "RUNME_REWARD_PATH": "/wrong",
                "RUNME_TASK_WORKDIR": "/wrong",
                "TASK_ENV": "yes",
            },
        ),
    )

    asyncio.run(environment.start(force_build=False))

    env = FakeClient.instances[0].requests[0]["start"]["env"]
    assert f"RUNME_ARTIFACTS_DIR={tmp_path / 'trial' / 'artifacts'}" in env
    assert f"RUNME_REWARD_PATH={tmp_path / 'trial' / 'verifier' / 'reward.json'}" in env
    assert f"RUNME_TASK_WORKDIR={tmp_path}" in env
    assert "RUNME_ARTIFACTS_DIR=/wrong" not in env
    assert "RUNME_REWARD_PATH=/wrong" not in env
    assert "RUNME_TASK_WORKDIR=/wrong" not in env
    assert "TASK_ENV=yes" in env


def test_upload_dir_rewrites_configured_workdir_paths_to_staged_workdir(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    workspace = tmp_path / "repo"
    source_workdir = workspace / "examples" / "harbor" / "task" / "workdir"
    source_workdir.mkdir(parents=True)
    (source_workdir / ".gitkeep").write_text("")

    tests = tmp_path / "tests"
    tests.mkdir()
    (tests / "test.sh").write_text(
        "rewardkit --workspace /app/examples/harbor/task/workdir\n"
    )

    environment = _make_env(
        tmp_path,
        workspace_root=workspace,
        task_env_config=EnvironmentConfig(
            workdir="/app/examples/harbor/task/workdir",
            env={"TASK_ENV": "yes"},
        ),
    )

    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(tests, "/tests"))

    request = FakeClient.instances[0].requests[-1]["upload_directory"]
    decoded = env_module.base64.b64decode(request["files"][0]["data"]).decode()
    assert "rewardkit --workspace " in decoded
    assert str(tmp_path / "trial" / "workdir") in decoded
    assert "/app/examples/harbor/task/workdir" not in decoded


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


def test_upload_dir_rewrites_artifact_paths_in_shell_scripts(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "test.sh").write_text(
        "mkdir -p /logs/artifacts\n"
        "cp results.json /logs/artifacts/results.json\n"
    )

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "/tests"))

    request = FakeClient.instances[0].requests[-1]["upload_directory"]
    decoded = env_module.base64.b64decode(request["files"][0]["data"]).decode()
    assert str(tmp_path / "trial" / "artifacts") in decoded
    assert "/logs/artifacts" not in decoded


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


def test_upload_dir_rewrites_python_artifact_paths(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    FakeClient.instances.clear()
    monkeypatch.setattr(env_module, "_StdioClient", FakeClient)

    source = tmp_path / "source"
    source.mkdir()
    (source / "artifact.py").write_text(
        'Path("/logs/artifacts/results.json").write_text("{}")\n'
    )

    environment = _make_env(tmp_path)
    asyncio.run(environment.start(force_build=False))
    asyncio.run(environment.upload_dir(source, "/tests"))

    request = FakeClient.instances[0].requests[-1]["upload_directory"]
    decoded = env_module.base64.b64decode(request["files"][0]["data"]).decode()
    assert str(tmp_path / "trial" / "artifacts" / "results.json") in decoded
    assert 'Path("/logs/artifacts/results.json")' not in decoded


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
