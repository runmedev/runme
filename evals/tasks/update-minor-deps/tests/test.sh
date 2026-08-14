#!/usr/bin/env sh
set -eu

tests_dir="${RUNME_TESTS_DIR:-/tests}"
verifier_dir="${RUNME_VERIFIER_DIR:-/logs/verifier}"
mkdir -p "$verifier_dir"

go run "$tests_dir/score.go" > "$verifier_dir/test-stdout.txt"
