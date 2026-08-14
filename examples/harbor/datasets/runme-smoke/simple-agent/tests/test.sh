#!/usr/bin/env sh
set -eu

workspace="${RUNME_TASK_WORKDIR:-$PWD}"
verifier_dir="${RUNME_VERIFIER_DIR:-/logs/verifier}"
reward_path="${RUNME_REWARD_PATH:-$verifier_dir/reward.txt}"
artifacts_dir="${RUNME_ARTIFACTS_DIR:-/logs/artifacts}"
expected="${RUNME_HARBOR_EXPECTED:-hello from a real agent}"
result_path="$workspace/result.txt"
mkdir -p "$verifier_dir"

if [ ! -f "$result_path" ]; then
  printf 'missing result.txt\n' >&2
  printf '0.0' > "$reward_path"
  exit 1
fi

actual="$(cat "$result_path")"

if [ "$actual" = "$expected" ]; then
  mkdir -p "$artifacts_dir"
  cp "$result_path" "$artifacts_dir/result.txt"
  printf '1.0' > "$reward_path"
else
  printf 'expected %s, got %s\n' "$expected" "$actual" >&2
  printf '0.0' > "$reward_path"
  exit 1
fi
