---
cwd: ../../..
---

# update-minor-deps

Development notes for the `update-minor-deps` skill.

This file is human-facing documentation for maintainers of the skill. It is not part of the skill's runtime instruction contract. Keep agent-facing execution guidance in `SKILL.md`; keep reusable execution helpers in `scripts/`; keep eval fixtures under `evals/tasks/`.

## Purpose

This skill captures the recurring dependency maintenance workflow that produced PRs like runmedev/runme#1150:

- Refresh minor and patch dependencies.
- Prefer the repository's documented contribution and validation flow.
- Keep compatibility fixes tightly scoped to dependency-update fallout.
- Preserve DCO signoff and PR conventions.

The first implementation is Go-module focused because the current workflow uses `go get -t -u ./...`, `go mod tidy`, and `runme run lint test`. The skill name intentionally avoids Go so the workflow can grow to cover other dependency ecosystems later.

## Boundaries

- `SKILL.md` defines what an agent should do when the skill is active.
- `scripts/update-go-deps.sh` automates the current Go dependency refresh step.
- `evals/tasks/` is reserved for Harbor task fixtures that validate whether agents use the skill as intended.
- This README can document rationale, development notes, and maintenance decisions without changing skill behavior.

## Validation

Use the skill validator after editing skill metadata or instructions:

```sh
python3 ~/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/update-minor-deps
```

Use Harbor eval tasks under `evals/tasks/` to test behavior over time.
