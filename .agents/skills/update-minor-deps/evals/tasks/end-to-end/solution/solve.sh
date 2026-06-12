#!/usr/bin/env sh
set -eu

cd /app

git status --short
sed -n '1,180p' CONTRIBUTING.md >/tmp/update-minor-deps-contributing.txt

.agents/skills/update-minor-deps/scripts/update-go-deps.sh

if git diff --quiet -- go.mod go.sum; then
  runme run lint test
  mkdir -p /logs/artifacts
  cat > /logs/artifacts/pr.md <<'PR'
# chore: update minor and patch dependencies

## Summary
- Ran `.agents/skills/update-minor-deps/scripts/update-go-deps.sh`.
- No root `go.mod` or `go.sum` updates were available.
- No compatibility or test fixes were needed.

## Tests
- Focused tests: not needed because no dependency files changed.
- Final validation: `runme run lint test`
PR
  exit 0
fi

changed_packages="$(git diff --name-only -- '*.go' | xargs -r dirname | sort -u | sed 's#^#./#')"
if [ -n "$changed_packages" ]; then
  # shellcheck disable=SC2086
  go test $changed_packages
else
  go test ./...
fi

runme run lint test

mkdir -p /logs/artifacts
cat > /logs/artifacts/pr.md <<'PR'
# chore: update minor and patch dependencies

## Summary
- Ran `.agents/skills/update-minor-deps/scripts/update-go-deps.sh`.
- Updated root dependency files: `go.mod` and `go.sum`.
- No compatibility or test fixes were needed beyond the dependency refresh.

## Tests
- Focused tests: `go test ./...`
- Final validation: `runme run lint test`
PR
