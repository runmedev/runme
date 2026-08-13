#!/usr/bin/env sh
set -eu

workspace=/app/examples/harbor/datasets/runme-llm-judge/poem-judge/workdir
verifier_dir="${RUNME_VERIFIER_DIR:-/logs/verifier}"
mkdir -p "$verifier_dir"

uv run /tests/llm_judge.py "$workspace/poem.txt" > "$verifier_dir/test-stdout.txt"

if [ -f "$workspace/poem.txt" ]; then
  mkdir -p /logs/artifacts
  cp "$workspace/poem.txt" /logs/artifacts/poem.txt
fi
