"""Custom criteria for the text-stats-reward task."""

import importlib.util
import json
import subprocess
from pathlib import Path

from rewardkit import criterion


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
