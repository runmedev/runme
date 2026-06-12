---
name: update-minor-deps
description: Run this repository's recurring minor and patch dependency maintenance. Use when asked to refresh non-breaking dependencies, reproduce a PR like runmedev/runme#1150, keep dependencies from falling behind, or prepare the associated branch, tests, commit, and pull request according to CONTRIBUTING.md.
---

# Update Minor Deps

## Overview

Perform the maintenance task that updates minor and patch dependencies, then fix the small regressions or stale test assumptions exposed by the new dependency graph.

## Workflow

1. Start from the repository root and inspect the working tree with `git status --short`. Do not overwrite unrelated user changes.
2. Read `CONTRIBUTING.md` before running project commands. Prefer its named Runme commands over ad hoc equivalents.
3. Create or use a branch named like `chore/minor-patch-dependencies`; add a date suffix if the branch already exists.
4. Run `scripts/update-go-deps.sh` from this skill to execute the documented update command:

```sh
.agents/skills/update-minor-deps/scripts/update-go-deps.sh
```

5. Review `go.mod` and `go.sum`. The expected baseline is dependency and checksum churn in the root module. Do not intentionally upgrade major versions or edit module paths unless the user asks.
6. Run focused tests first for any touched or failing packages. Use failures to find real compatibility regressions, not to mask problems.
7. Make narrowly scoped code or test fixes when the updated dependencies changed behavior. PR 1150 is the model: module updates plus small fixes for go-git/go-billy path-walk behavior and a stale Ctrl-C test input assumption.
8. Run the required final validation after changes:

```sh
runme run lint test
```

If integration dependencies are unavailable locally, try `runme run test-docker` and report exactly what could not be validated.

## Investigation Rules

- Treat root `go.mod` and `go.sum` as the normal output. Treat `.dagger/go.mod` as out of scope unless the user explicitly asks to update Dagger dependencies too.
- Keep fixes tied to dependency-update fallout. Avoid opportunistic refactors, broad formatting churn, and unrelated cleanup.
- When tests fail, isolate with package-level or test-level commands before editing. Capture the focused commands in the final PR summary.
- If a failure appears flaky, repeat the narrow test with `-count=3` before changing code.
- Preserve generated files unless a project command updates them as part of the normal workflow.

## PR Shape

Use a concise commit and PR title:

```text
chore: update minor and patch dependencies
```

Commit with DCO signoff as required by this repo:

```sh
git commit -s -m "chore: update minor and patch dependencies"
```

Draft the PR summary with:

- The documented update command that was run.
- The dependency files updated.
- Any compatibility or test fixes made.
- The focused tests and `runme run lint test` result.

Open as a draft PR first, following `CONTRIBUTING.md`, unless the user asks otherwise.
