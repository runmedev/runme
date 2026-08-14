"""Custom criteria for the text-stats-reward task."""

import importlib.util
import json
import os
import re
import subprocess
from pathlib import Path

from rewardkit import criterion

_DEFAULT_AGENT_TRAJECTORY = Path("/logs/agent/trajectory.json")
_COMPILE_COMMAND_PATTERN = re.compile(
    r"(?:^|&&|\|\||;|\n|[\"']?cmd[\"']?\s*:\s*[\"'])"
    r"\s*python3\s+-m\s+py_compile\s+textstats\.py\s+analyze\.py"
    r"(?=\s|[\"'`;|&)]|$)"
)
_TERMINAL_STATUS_PATTERNS = (
    re.compile(r"\bTask compile exited with code (-?\d+)\b", re.IGNORECASE),
    re.compile(r"\b(?:Process|Command) exited with code (-?\d+)\b", re.IGNORECASE),
    re.compile(r"\bExit code:\s*(-?\d+)\b", re.IGNORECASE),
    re.compile(r"""["']exit_code["']\s*:\s*(-?\d+)""", re.IGNORECASE),
)
_RUNNING_SESSION_PATTERNS = (
    re.compile(
        r"\b(?:Process|Script) running with (?:session|cell) ID ([\w.-]+)\b",
        re.IGNORECASE,
    ),
    re.compile(
        r"""["']session_id["']\s*:\s*["']?([\w.-]+)["']?""",
        re.IGNORECASE,
    ),
)
_SCRIPT_COMPLETED_PATTERN = re.compile(r"\bScript completed\b", re.IGNORECASE)
_MOVED_TO_BACKGROUND_PATTERN = re.compile(r"\bmoved to the background\b", re.IGNORECASE)
_TASK_NOTIFICATION_PATTERN = re.compile(
    r"<task-notification>(.*?)</task-notification>",
    re.DOTALL,
)
_TASK_NOTIFICATION_EXIT_PATTERN = re.compile(
    r"\bexit code\s+(-?\d+)\b",
    re.IGNORECASE,
)
_TERMINAL_TASK_STATUSES = frozenset({"completed", "failed", "canceled", "cancelled"})
_SHELL_ARGUMENT_KEYS = frozenset({"cmd", "command", "script"})


def _load_module(workspace: Path, name: str):
    spec = importlib.util.spec_from_file_location(name, workspace / f"{name}.py")
    if spec is None or spec.loader is None:
        raise ImportError(f"Cannot find {name}.py in workspace")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _word_count_score(workspace: Path) -> float:
    module = _load_module(workspace, "textstats")

    cases = [
        ("hello world", 2),
        ("one", 1),
        ("", 0),
        ("the quick brown fox", 4),
    ]
    correct = sum(1 for text, expected in cases if module.word_count(text) == expected)
    return correct / len(cases)


def _most_common_score(workspace: Path) -> float:
    module = _load_module(workspace, "textstats")

    cases = [
        ("the cat and the dog", "the"),
        ("hello hello world", "hello"),
        ("", ""),
    ]
    correct = sum(1 for text, expected in cases if module.most_common(text) == expected)
    return correct / len(cases)


def _file_exists_score(workspace: Path) -> float:
    paths = ["textstats.py", "analyze.py", "results.json"]
    return sum(1 for path in paths if (workspace / path).exists()) / len(paths)


def _functions_defined_score(workspace: Path) -> float:
    try:
        text = (workspace / "textstats.py").read_text()
    except OSError:
        return 0.0

    required = ["def word_count", "def most_common"]
    return sum(1 for marker in required if marker in text) / len(required)


def _pipeline_runs_score(workspace: Path) -> float:
    checks = 0

    result = subprocess.run(
        ["python", "analyze.py"],
        cwd=workspace,
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        timeout=30,
    )
    if result.returncode == 0:
        checks += 1

    try:
        data = json.loads((workspace / "results.json").read_text())
    except (OSError, ValueError):
        data = {}

    if isinstance(data, dict) and data.get("word_count") == 20:
        checks += 1
    if isinstance(data, dict) and data.get("most_common") == "the":
        checks += 1

    return checks / 3


def _correctness_score(workspace: Path) -> float:
    try:
        return (_word_count_score(workspace) + _most_common_score(workspace)) / 2
    except Exception:
        return 0.0


def _structure_score(workspace: Path) -> float:
    return (
        _file_exists_score(workspace)
        + _functions_defined_score(workspace)
        + _pipeline_runs_score(workspace)
    ) / 3


def _agent_trajectory_path() -> Path:
    configured = os.environ.get("RUNME_AGENT_TRAJECTORY")
    return Path(configured) if configured else _DEFAULT_AGENT_TRAJECTORY


def _strings(value: object):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for nested in value.values():
            yield from _strings(nested)
    elif isinstance(value, list):
        for nested in value:
            yield from _strings(nested)


def _terminal_status(value: object) -> int | None:
    statuses = [
        int(match.group(1))
        for text in _strings(value)
        for pattern in _TERMINAL_STATUS_PATTERNS
        for match in pattern.finditer(text)
    ]
    return statuses[-1] if statuses else None


def _running_sessions(value: object) -> set[str]:
    return {
        match.group(1)
        for text in _strings(value)
        for pattern in _RUNNING_SESSION_PATTERNS
        for match in pattern.finditer(text)
    }


def _references_session(value: object, sessions: set[str]) -> bool:
    if isinstance(value, (str, int)):
        text = str(value)
        return any(
            re.search(rf"(?<![\w.-]){re.escape(session)}(?![\w.-])", text)
            for session in sessions
        )
    if isinstance(value, dict):
        return any(_references_session(nested, sessions) for nested in value.values())
    if isinstance(value, list):
        return any(_references_session(nested, sessions) for nested in value)
    return False


def _starts_shell_command(value: object) -> bool:
    if not isinstance(value, dict):
        return False
    for key, nested in value.items():
        if key in _SHELL_ARGUMENT_KEYS and isinstance(nested, str):
            return True
        if _starts_shell_command(nested):
            return True
    return False


def _linked_results(
    step: dict[str, object], call_id: object
) -> list[dict[str, object]]:
    observation = step.get("observation")
    if not isinstance(observation, dict):
        return []
    results = observation.get("results")
    if not isinstance(results, list):
        return []
    return [
        result
        for result in results
        if isinstance(result, dict) and result.get("source_call_id") == call_id
    ]


def _synchronous_success(results: list[dict[str, object]]) -> bool:
    for result in results:
        extra = result.get("extra")
        if (
            not isinstance(extra, dict)
            or extra.get("tool_result_is_error") is not False
        ):
            continue
        if any(_MOVED_TO_BACKGROUND_PATTERN.search(text) for text in _strings(result)):
            continue

        metadata = extra.get("tool_result_metadata")
        if not isinstance(metadata, dict):
            continue
        tool_result = metadata.get("tool_use_result")
        if isinstance(tool_result, dict) and (
            tool_result.get("interrupted") is True
            or "backgroundTaskId" in tool_result
            or "timedOutAfterMs" in tool_result
        ):
            continue

        return True

    return False


def _background_task_id(results: list[dict[str, object]]) -> str | None:
    for result in results:
        extra = result.get("extra")
        if (
            not isinstance(extra, dict)
            or extra.get("tool_result_is_error") is not False
        ):
            continue
        metadata = extra.get("tool_result_metadata")
        if not isinstance(metadata, dict):
            continue
        tool_result = metadata.get("tool_use_result")
        if not isinstance(tool_result, dict):
            continue
        task_id = tool_result.get("backgroundTaskId")
        if isinstance(task_id, str) and task_id:
            return task_id
    return None


def _task_notification(message: object) -> dict[str, str] | None:
    if not isinstance(message, str):
        return None
    notification = _TASK_NOTIFICATION_PATTERN.search(message)
    if notification is None:
        return None

    fields: dict[str, str] = {}
    body = notification.group(1)
    for tag in ("task-id", "tool-use-id", "status", "summary"):
        match = re.search(rf"<{tag}>\s*(.*?)\s*</{tag}>", body, re.DOTALL)
        if match is None:
            return None
        fields[tag] = match.group(1)
    return fields


def _compile_validation_score(workspace: Path) -> float:
    del workspace

    try:
        trajectory = json.loads(_agent_trajectory_path().read_text())
    except (OSError, ValueError):
        return 0.0
    if not isinstance(trajectory, dict):
        return 0.0

    active_sessions: set[str] = set()
    pending_background: tuple[str, str] | None = None
    for step in trajectory.get("steps", []):
        if not isinstance(step, dict):
            continue

        if step.get("source") == "user" and pending_background is not None:
            notification = _task_notification(step.get("message"))
            if notification is None:
                continue

            task_id, tool_call_id = pending_background
            if (
                notification["task-id"] != task_id
                or notification["tool-use-id"] != tool_call_id
            ):
                continue

            status = notification["status"].strip().lower()
            exit_match = _TASK_NOTIFICATION_EXIT_PATTERN.search(notification["summary"])
            if (
                status == "completed"
                and exit_match is not None
                and int(exit_match.group(1)) == 0
            ):
                return 1.0
            if status in _TERMINAL_TASK_STATUSES or (
                exit_match is not None and int(exit_match.group(1)) != 0
            ):
                pending_background = None
            continue

        if step.get("source") != "agent":
            continue
        tool_calls = step.get("tool_calls")
        if not isinstance(tool_calls, list):
            continue

        for tool_call in tool_calls:
            if not isinstance(tool_call, dict):
                continue

            arguments = tool_call.get("arguments")
            results = _linked_results(step, tool_call.get("tool_call_id"))
            runs_validation = any(
                _COMPILE_COMMAND_PATTERN.search(text) for text in _strings(arguments)
            )

            if runs_validation:
                active_sessions.clear()
                pending_background = None
                status = _terminal_status(results)
                if status is not None:
                    if status == 0:
                        return 1.0
                    continue
                running_sessions = _running_sessions(results)
                active_sessions.update(running_sessions)
                background_task_id = _background_task_id(results)
                tool_call_id = tool_call.get("tool_call_id")
                if background_task_id is not None and isinstance(tool_call_id, str):
                    pending_background = (background_task_id, tool_call_id)
                if not running_sessions and _synchronous_success(results):
                    return 1.0
                if not running_sessions and any(
                    _SCRIPT_COMPLETED_PATTERN.search(text) for text in _strings(results)
                ):
                    return 1.0
                continue

            if active_sessions and _starts_shell_command(arguments):
                active_sessions.clear()
                continue

            if active_sessions and _references_session(arguments, active_sessions):
                status = _terminal_status(results)
                if status is not None:
                    if status == 0:
                        return 1.0
                    active_sessions.clear()
                    continue
                running_sessions = _running_sessions(results)
                active_sessions.update(running_sessions)
                if not running_sessions and any(
                    _SCRIPT_COMPLETED_PATTERN.search(text) for text in _strings(results)
                ):
                    return 1.0

    return 0.0


def _gated_reward_score(workspace: Path) -> float:
    if _compile_validation_score(workspace) == 0.0:
        return 0.0
    return (_correctness_score(workspace) + _structure_score(workspace)) / 2


@criterion(shared=True)
def word_count_correct(workspace: Path) -> float:
    try:
        return _word_count_score(workspace)
    except Exception:
        return 0.0


@criterion(shared=True)
def most_common_correct(workspace: Path) -> float:
    try:
        return _most_common_score(workspace)
    except Exception:
        return 0.0


@criterion(shared=True)
def correctness(workspace: Path) -> float:
    return _correctness_score(workspace)


@criterion(shared=True)
def structure(workspace: Path) -> float:
    return _structure_score(workspace)


@criterion(shared=True)
def compile_validation(workspace: Path) -> float:
    return _compile_validation_score(workspace)


@criterion(shared=True)
def gated_reward(workspace: Path) -> float:
    return _gated_reward_score(workspace)
