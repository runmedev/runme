#!/usr/bin/env python3
from __future__ import annotations

import json
import re
import subprocess
from pathlib import Path


ROOT = Path("/app")
AGENT_LOG_DIR = Path("/logs/agent")
ARTIFACTS_DIR = Path("/logs/artifacts")
VERIFIER_DIR = Path("/logs/verifier")
PR_DRAFT = ARTIFACTS_DIR / "pr.md"
REWARD_PATH = VERIFIER_DIR / "reward.json"
PR_TITLE_WITH_DATE_RE = re.compile(
    r"chore:\s*update minor and patch dependencies.*\b\d{4}-\d{2}-\d{2}\b"
)


def run_git(*args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=ROOT,
        check=False,
        text=True,
        capture_output=True,
    )
    return result.stdout


def changed_files() -> list[str]:
    files = set()
    for mode_args in (
        ("diff", "--name-only"),
        ("diff", "--cached", "--name-only"),
    ):
        files.update(line.strip() for line in run_git(*mode_args).splitlines() if line.strip())
    return sorted(files)


def read_text(path: Path) -> str:
    try:
        return path.read_text(errors="replace")
    except OSError:
        return ""


def collect_agent_text() -> str:
    chunks: list[str] = []
    if AGENT_LOG_DIR.exists():
        for path in sorted(AGENT_LOG_DIR.rglob("*")):
            if path.is_file() and path.stat().st_size <= 5_000_000:
                chunks.append(read_text(path))
    chunks.append(read_text(PR_DRAFT))
    return "\n".join(chunks).lower()


def score_dependency_update(files: list[str], text: str) -> float:
    root_dep_changed = "go.mod" in files or "go.sum" in files
    dagger_untouched = ".dagger/go.mod" not in files and ".dagger/go.sum" not in files
    update_attempted = (
        ".agents/skills/update-minor-deps/scripts/update-go-deps.sh" in text
        or "go get -t -u ./..." in text
        or "update-go-deps.sh" in text
    )
    if root_dep_changed and dagger_untouched:
        return 1.0
    if dagger_untouched and update_attempted:
        return 0.5
    if root_dep_changed:
        return 0.5
    return 0.0


def score_scoped_changes(files: list[str]) -> float:
    if not files:
        return 1.0

    forbidden_prefixes = (
        "api/gen/",
        "app/",
        "docs/",
        "examples/",
        "integrations/",
        "web/",
        ".github/",
        ".dagger/",
    )
    forbidden_names = {
        "README.md",
        "CONTRIBUTING.md",
        "Makefile",
        "package.json",
        "pnpm-lock.yaml",
        "yarn.lock",
        "bun.lock",
    }

    unrelated = []
    for file in files:
        if file in {"go.mod", "go.sum"}:
            continue
        if file.endswith(".go"):
            continue
        if file.startswith(".agents/skills/update-minor-deps/evals/tasks/"):
            continue
        if file in forbidden_names or file.startswith(forbidden_prefixes):
            unrelated.append(file)
            continue
        unrelated.append(file)

    if not unrelated:
        return 1.0
    if len(unrelated) <= 2:
        return 0.5
    return 0.0


def score_workflow_evidence(text: str) -> float:
    checks = [
        "git status --short" in text or "git status -s" in text,
        "contributing.md" in text,
        ".agents/skills/update-minor-deps/scripts/update-go-deps.sh" in text
        or "go get -t -u ./..." in text,
        "go mod tidy" in text or "update-go-deps.sh" in text,
    ]
    return sum(1 for check in checks if check) / len(checks)


def score_skill_activation_evidence(text: str) -> float:
    checks = [
        "update-minor-deps" in text,
        "skill" in text and ("dependency" in text or "dependencies" in text),
        ".agents/skills/update-minor-deps/scripts/update-go-deps.sh" in text,
    ]
    return 1.0 if any(checks) else 0.0


def score_validation_evidence(text: str) -> float:
    focused = bool(re.search(r"go test\s+(\./|\S*/)", text)) or "focused test" in text
    final = "runme run lint test" in text
    if focused and final:
        return 1.0
    if final:
        return 0.6
    if focused:
        return 0.4
    return 0.0


def score_pr_draft() -> float:
    text = read_text(PR_DRAFT).lower()
    if not text:
        return 0.0

    checks = [
        "chore: update minor and patch dependencies" in text,
        bool(PR_TITLE_WITH_DATE_RE.search(text)),
        "update-go-deps.sh" in text or "go get -t -u ./..." in text,
        "go.mod" in text,
        "go.sum" in text,
        "compat" in text or "fix" in text or "none" in text,
        "go test" in text or "focused" in text,
        "runme run lint test" in text,
    ]
    return sum(1 for check in checks if check) / len(checks)


def score_no_real_pr_or_commit(text: str) -> float:
    forbidden = [
        r"\bgit\s+commit\b",
        r"\bgit\s+push\b",
        r"\bgh\s+pr\s+create\b",
        r"\bhub\s+pull-request\b",
    ]
    return 0.0 if any(re.search(pattern, text) for pattern in forbidden) else 1.0


def main() -> int:
    VERIFIER_DIR.mkdir(parents=True, exist_ok=True)
    ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)

    files = changed_files()
    text = collect_agent_text()
    scores = {
        "dependency_update": score_dependency_update(files, text),
        "scoped_changes": score_scoped_changes(files),
        "skill_activation_evidence": score_skill_activation_evidence(text),
        "workflow_evidence": score_workflow_evidence(text),
        "validation_evidence": score_validation_evidence(text),
        "pr_draft_quality": score_pr_draft(),
        "no_real_pr_or_commit": score_no_real_pr_or_commit(text),
    }

    diagnostics = {
        "changed_files": files,
        "pr_draft": str(PR_DRAFT),
        "agent_log_dir": str(AGENT_LOG_DIR),
    }
    (VERIFIER_DIR / "diagnostics.json").write_text(json.dumps(diagnostics, indent=2))
    REWARD_PATH.write_text(json.dumps(scores, indent=2) + "\n")
    print(json.dumps(scores, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
