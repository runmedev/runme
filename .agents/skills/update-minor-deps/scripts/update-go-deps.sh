#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: update-go-deps.sh [--no-tidy] [--no-diff]

Run Runme's documented non-breaking Go dependency update for the root module:
  go get -t -u ./...

Options:
  --no-tidy  Skip go mod tidy after go get.
  --no-diff  Skip printing the go.mod/go.sum diff summary.
  -h, --help Show this help.
USAGE
}

run_tidy=1
show_diff=1

for arg in "$@"; do
  case "$arg" in
    --no-tidy)
      run_tidy=0
      ;;
    --no-diff)
      show_diff=0
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! -f go.mod ]]; then
  echo "go.mod not found; run this script from the Runme repository root" >&2
  exit 1
fi

unset RUNME_SESSION_STRATEGY RUNME_TLS_DIR RUNME_SERVER_ADDR
export TZ="${TZ:-UTC}"

go get -t -u ./...

if [[ "$run_tidy" -eq 1 ]]; then
  go mod tidy
fi

if [[ "$show_diff" -eq 1 ]]; then
  git diff --stat -- go.mod go.sum || true
  git diff -- go.mod go.sum || true
fi
