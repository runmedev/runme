import importlib.util
import json
import sys
import types
from pathlib import Path

import pytest

CRITERIA_PATH = (
    Path(__file__).parents[3]
    / "examples/harbor/datasets/runme-rewardkit/text-stats-reward/tests/criteria.py"
)


@pytest.fixture
def criteria(monkeypatch):
    rewardkit = types.ModuleType("rewardkit")
    rewardkit.criterion = lambda **_kwargs: lambda function: function
    monkeypatch.setitem(sys.modules, "rewardkit", rewardkit)

    spec = importlib.util.spec_from_file_location("text_stats_criteria", CRITERIA_PATH)
    assert spec is not None
    assert spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _step(
    step_id: int,
    function_name: str,
    arguments: dict[str, object],
    content: str,
) -> dict[str, object]:
    call_id = f"call-{step_id}"
    return {
        "step_id": step_id,
        "source": "agent",
        "tool_calls": [
            {
                "tool_call_id": call_id,
                "function_name": function_name,
                "arguments": arguments,
            }
        ],
        "observation": {
            "results": [{"source_call_id": call_id, "content": content}]
        },
    }


def _score(criteria, monkeypatch, tmp_path: Path, steps: list[dict[str, object]]) -> float:
    trajectory = tmp_path / "trajectory.json"
    trajectory.write_text(json.dumps({"schema_version": "ATIF-v1.7", "steps": steps}))
    monkeypatch.setenv("RUNME_AGENT_TRAJECTORY", str(trajectory))
    return criteria._lint_validation_score(tmp_path)


@pytest.mark.parametrize(
    "terminal_status",
    [
        "Task lint exited with code 0",
        "Process exited with code 0",
        "Command exited with code 0",
        "Exit code: 0",
    ],
)
def test_lint_validation_accepts_tty_and_non_tty_success(
    criteria,
    monkeypatch,
    tmp_path: Path,
    terminal_status: str,
) -> None:
    steps = [_step(1, "arbitrary-shell-tool", {"command": "runme run lint"}, terminal_status)]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_lint_validation_follows_async_session(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec_command",
            {"cmd": "runme run lint"},
            "Process running with session ID 63887",
        ),
        _step(
            2,
            "write_stdin",
            {"session_id": 63887, "chars": ""},
            "Process exited with code 0",
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


@pytest.mark.parametrize("status", ["Process exited with code 1", "Exit code: 2"])
def test_lint_validation_rejects_failure(
    criteria,
    monkeypatch,
    tmp_path: Path,
    status: str,
) -> None:
    steps = [_step(1, "shell", {"cmd": "runme run lint"}, status)]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_lint_validation_rejects_incomplete_session(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "shell",
            {"cmd": "runme run lint"},
            "Process running with session ID lint-session",
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_lint_validation_rejects_unrelated_command_success(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "shell",
            {"cmd": "runme run lint"},
            "Process running with session ID lint-session",
        ),
        _step(
            2,
            "shell",
            {"command": "echo unrelated"},
            "Process exited with code 0",
        ),
        _step(
            3,
            "poll",
            {"session_id": "lint-session"},
            "Process exited with code 0",
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_lint_validation_accepts_successful_retry(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(1, "shell", {"cmd": "runme run lint"}, "Process exited with code 1"),
        _step(2, "shell", {"cmd": "runme run lint"}, "Process exited with code 0"),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_lint_validation_ignores_agent_claims(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        {
            "step_id": 1,
            "source": "agent",
            "message": "runme run lint completed. Process exited with code 0",
        }
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_lint_validation_rejects_malformed_trajectory(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    trajectory = tmp_path / "trajectory.json"
    trajectory.write_text("[]")
    monkeypatch.setenv("RUNME_AGENT_TRAJECTORY", str(trajectory))

    assert criteria._lint_validation_score(tmp_path) == 0.0
