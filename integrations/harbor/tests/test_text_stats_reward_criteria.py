import importlib.util
import json
import os
import subprocess
import sys
import types
from pathlib import Path

import pytest

CRITERIA_PATH = (
    Path(__file__).parents[3]
    / "examples/harbor/datasets/runme-rewardkit/text-stats-reward/tests/criteria.py"
)
SOLUTION_PATH = CRITERIA_PATH.parents[1] / "solution/solve.sh"


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
    result_extra: dict[str, object] | None = None,
) -> dict[str, object]:
    call_id = f"call-{step_id}"
    result = {"source_call_id": call_id, "content": content}
    if result_extra is not None:
        result["extra"] = result_extra
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
        "observation": {"results": [result]},
    }


def _score(criteria, monkeypatch, tmp_path: Path, steps: list[dict[str, object]]) -> float:
    trajectory = tmp_path / "trajectory.json"
    trajectory.write_text(json.dumps({"schema_version": "ATIF-v1.7", "steps": steps}))
    monkeypatch.setenv("RUNME_AGENT_TRAJECTORY", str(trajectory))
    return criteria._compile_validation_score(tmp_path)


def _background_compile_step(
    step_id: int = 1,
    task_id: str = "compile-task",
) -> dict[str, object]:
    return _step(
        step_id,
        "Bash",
        {"command": "python3 -m py_compile textstats.py analyze.py", "timeout": 120000},
        (
            "Command did not complete within its 120s timeout and was moved "
            f"to the background (ID: {task_id})."
        ),
        {
            "tool_result_metadata": {
                "tool_use_result": {
                    "interrupted": False,
                    "backgroundTaskId": task_id,
                    "timedOutAfterMs": 120000,
                },
                "raw_tool_result": {"is_error": False},
            },
            "tool_result_is_error": False,
        },
    )


def _task_notification_step(
    step_id: int = 2,
    task_id: str = "compile-task",
    tool_use_id: str = "call-1",
    status: str = "completed",
    exit_code: int | None = 0,
    source: str = "user",
) -> dict[str, object]:
    exit_summary = "" if exit_code is None else f" (exit code {exit_code})"
    return {
        "step_id": step_id,
        "source": source,
        "message": (
            "<task-notification>\n"
            f"<task-id>{task_id}</task-id>\n"
            f"<tool-use-id>{tool_use_id}</tool-use-id>\n"
            f"<status>{status}</status>\n"
            f'<summary>Background command "python3 -m py_compile textstats.py analyze.py" '
            f"{status}{exit_summary}</summary>\n"
            "</task-notification>"
        ),
    }


@pytest.mark.parametrize(
    "terminal_status",
    [
        "Task compile exited with code 0",
        "Process exited with code 0",
        "Command exited with code 0",
        "Exit code: 0",
    ],
)
def test_compile_validation_accepts_tty_and_non_tty_success(
    criteria,
    monkeypatch,
    tmp_path: Path,
    terminal_status: str,
) -> None:
    steps = [
        _step(
            1,
            "arbitrary-shell-tool",
            {"command": "python3 -m py_compile textstats.py analyze.py"},
            terminal_status,
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_compile_validation_follows_async_session(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec_command",
            {"cmd": "python3 -m py_compile textstats.py analyze.py"},
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


def test_compile_validation_accepts_terra_cell_completion(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec",
            {
                "input": (
                    'const r = await tools.exec_command({"cmd":"python3 -m py_compile textstats.py analyze.py"}); '
                    "text(r.output);"
                )
            },
            "Script running with cell ID 4\nWall time 11.0 seconds\nOutput:\n",
        ),
        _step(
            2,
            "wait",
            {"cell_id": "4", "yield_time_ms": 30000},
            "Script completed\nWall time 17.4 seconds\nOutput:\n",
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_compile_validation_accepts_luna_structured_status(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec",
            {
                "input": (
                    'const r = await tools.exec_command({cmd:"python3 -m py_compile textstats.py analyze.py"}); '
                    "text(JSON.stringify(r));"
                )
            },
            (
                "Script completed\nWall time 1.2 seconds\nOutput:\n"
                '\'{"session_id":38781,"output":"gofumpt"}\''
            ),
        ),
        _step(
            2,
            "exec",
            {
                "input": (
                    "const r = await tools.write_stdin({session_id:38781}); "
                    "text(JSON.stringify(r));"
                )
            },
            ('Script completed\nWall time 8.0 seconds\nOutput:\n\'{"exit_code":0,"output":""}\''),
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_compile_validation_accepts_sonnet_synchronous_success(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "Bash",
            {"command": "python3 -m py_compile textstats.py analyze.py", "timeout": 300000},
            "gofumpt\ngoimports\nrevive",
            {
                "tool_result_metadata": {
                    "tool_use_result": {
                        "stdout": "gofumpt\ngoimports\nrevive",
                        "stderr": "",
                        "interrupted": False,
                    },
                    "raw_tool_result": {"is_error": False},
                },
                "tool_result_is_error": False,
            },
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


@pytest.mark.parametrize("status", ["Process exited with code 1", "Exit code: 2"])
def test_compile_validation_rejects_failure(
    criteria,
    monkeypatch,
    tmp_path: Path,
    status: str,
) -> None:
    steps = [_step(1, "shell", {"cmd": "python3 -m py_compile textstats.py analyze.py"}, status)]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_haiku_background_launch(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [_background_compile_step()]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_accepts_correlated_background_completion(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [_background_compile_step(), _task_notification_step()]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


@pytest.mark.parametrize(
    ("task_id", "tool_use_id"),
    [
        ("other-task", "call-1"),
        ("compile-task", "other-call"),
    ],
)
def test_compile_validation_ignores_uncorrelated_background_completion(
    criteria,
    monkeypatch,
    tmp_path: Path,
    task_id: str,
    tool_use_id: str,
) -> None:
    steps = [
        _background_compile_step(),
        _task_notification_step(task_id=task_id, tool_use_id=tool_use_id),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


@pytest.mark.parametrize(
    ("status", "exit_code"),
    [
        ("completed", 1),
        ("completed", None),
        ("failed", 0),
        ("cancelled", 0),
        ("canceled", 0),
    ],
)
def test_compile_validation_rejects_unsuccessful_background_completion(
    criteria,
    monkeypatch,
    tmp_path: Path,
    status: str,
    exit_code: int | None,
) -> None:
    steps = [
        _background_compile_step(),
        _task_notification_step(status=status, exit_code=exit_code),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_clears_correlated_background_failure(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _background_compile_step(),
        _task_notification_step(status="failed", exit_code=1),
        _task_notification_step(step_id=3),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_ignores_agent_background_completion_spoof(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _background_compile_step(),
        _task_notification_step(source="agent"),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_ignores_ordinary_user_completion_claim(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _background_compile_step(),
        {
            "step_id": 2,
            "source": "user",
            "message": "The compile task completed with exit code 0.",
        },
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_clears_background_attempt_on_new_compile_run(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _background_compile_step(),
        _step(
            2,
            "Bash",
            {"command": "python3 -m py_compile textstats.py analyze.py"},
            "Process exited with code 1",
        ),
        _task_notification_step(step_id=3),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


@pytest.mark.parametrize(
    ("is_error", "interrupted"),
    [
        (True, False),
        (False, True),
    ],
)
def test_compile_validation_rejects_failed_or_interrupted_tool_result(
    criteria,
    monkeypatch,
    tmp_path: Path,
    is_error: bool,
    interrupted: bool,
) -> None:
    steps = [
        _step(
            1,
            "Bash",
            {"command": "python3 -m py_compile textstats.py analyze.py"},
            "compile output",
            {
                "tool_result_metadata": {
                    "tool_use_result": {"interrupted": interrupted},
                    "raw_tool_result": {"is_error": is_error},
                },
                "tool_result_is_error": is_error,
            },
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_structured_failure_despite_completion(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec",
            {"input": 'tools.exec_command({cmd:"python3 -m py_compile textstats.py analyze.py"})'},
            (
                "Script completed\nWall time 1.0 seconds\nOutput:\n"
                '\'{"exit_code":1,"output":"compile failed"}\''
            ),
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_completion_while_session_is_running(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "exec",
            {"input": 'tools.exec_command({cmd:"python3 -m py_compile textstats.py analyze.py"})'},
            (
                "Script completed\nWall time 1.0 seconds\nOutput:\n"
                '\'{"session_id":38781,"output":"gofumpt"}\''
            ),
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_incomplete_session(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "shell",
            {"cmd": "python3 -m py_compile textstats.py analyze.py"},
            "Process running with session ID compile-session",
        )
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_unrelated_command_success(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "shell",
            {"cmd": "python3 -m py_compile textstats.py analyze.py"},
            "Process running with session ID compile-session",
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
            {"session_id": "compile-session"},
            "Process exited with code 0",
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_accepts_successful_retry(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        _step(
            1,
            "shell",
            {"cmd": "python3 -m py_compile textstats.py analyze.py"},
            "Process exited with code 1",
        ),
        _step(
            2,
            "shell",
            {"cmd": "python3 -m py_compile textstats.py analyze.py"},
            "Process exited with code 0",
        ),
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 1.0


def test_compile_validation_ignores_agent_claims(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    steps = [
        {
            "step_id": 1,
            "source": "agent",
            "message": "python3 -m py_compile textstats.py analyze.py completed. Process exited with code 0",
        }
    ]

    assert _score(criteria, monkeypatch, tmp_path, steps) == 0.0


def test_compile_validation_rejects_malformed_trajectory(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    trajectory = tmp_path / "trajectory.json"
    trajectory.write_text("[]")
    monkeypatch.setenv("RUNME_AGENT_TRAJECTORY", str(trajectory))

    assert criteria._compile_validation_score(tmp_path) == 0.0


def test_oracle_solution_records_successful_compile_validation(
    criteria,
    monkeypatch,
    tmp_path: Path,
) -> None:
    (tmp_path / "sample.txt").write_text(
        "the quick brown fox jumps over the lazy dog and the quick brown fox "
        "jumps over the lazy dog again"
    )
    trajectory = tmp_path / "trajectory.json"
    env = os.environ.copy()
    env.pop("RUNME_AGENT_TRAJECTORY", None)
    env["RUNME_AGENT_LOG_DIR"] = str(tmp_path)

    subprocess.run(
        ["sh", str(SOLUTION_PATH)],
        cwd=tmp_path,
        env=env,
        check=True,
    )

    monkeypatch.setenv("RUNME_AGENT_TRAJECTORY", str(trajectory))
    assert criteria._compile_validation_score(tmp_path) == 1.0
