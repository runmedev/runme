#!/usr/bin/env sh
set -u

cd /app
current_date="$(date +%F)"
mkdir -p /logs/artifacts

echo "Using update-minor-deps skill workflow"
echo "+ git status --short"
git status --short
echo "+ sed -n '1,180p' CONTRIBUTING.md"
sed -n '1,180p' CONTRIBUTING.md >/tmp/update-minor-deps-contributing.txt

echo "+ runme run update-go-deps"
runme run update-go-deps
echo "+ go mod tidy"
go mod tidy

if git diff --quiet -- go.mod go.sum; then
  echo "+ runme run lint test"
  final_status=0
  runme run lint test || final_status=$?
  {
    printf '# chore: update minor and patch dependencies (%s)\n\n' "$current_date"
    cat <<'PR'
## Summary
- Ran `runme run update-go-deps` and `go mod tidy`.
- No root `go.mod` or `go.sum` updates were available.
- No compatibility or test fixes were needed.

## Tests
- Focused tests: not needed because no dependency files changed.
PR
    printf -- '- Final validation: `runme run lint test`'
    if [ "$final_status" -eq 0 ]; then
      printf ' passed.\n'
    else
      printf ' failed with exit code %s.\n' "$final_status"
    fi
  } > /logs/artifacts/pr.md
  exit 0
fi

changed_packages="$(git diff --name-only -- '*.go' | xargs -r dirname | sort -u | sed 's#^#./#')"
focused_status=0
if [ -n "$changed_packages" ]; then
  # shellcheck disable=SC2086
  echo "+ go test $changed_packages"
  go test $changed_packages || focused_status=$?
  focused_command="go test $changed_packages"
else
  echo "+ go test ./..."
  go test ./... || focused_status=$?
  focused_command="go test ./..."
fi

echo "+ runme run lint test"
final_status=0
runme run lint test || final_status=$?

{
  printf '# chore: update minor and patch dependencies (%s)\n\n' "$current_date"
  cat <<'PR'
## Summary
- Ran `runme run update-go-deps` and `go mod tidy`.
- Updated root dependency files: `go.mod` and `go.sum`.
PR
  if [ "$final_status" -eq 0 ]; then
    printf -- '- No compatibility or test fixes were needed beyond the dependency refresh.\n'
  else
    printf -- '- Compatibility/test fixes may be needed; validation reported failures after the dependency refresh.\n'
  fi

  printf '\n## Tests\n'
  printf -- '- Focused tests: `%s`' "$focused_command"
  if [ "$focused_status" -eq 0 ]; then
    printf ' passed.\n'
  else
    printf ' failed with exit code %s.\n' "$focused_status"
  fi
  printf -- '- Final validation: `runme run lint test`'
  if [ "$final_status" -eq 0 ]; then
    printf ' passed.\n'
  else
    printf ' failed with exit code %s.\n' "$final_status"
  fi
} > /logs/artifacts/pr.md
