#!/usr/bin/env sh
set -eu

workspace="${RUNME_TASK_WORKDIR:-$PWD}"
tests_dir="${RUNME_TESTS_DIR:-/tests}"
verifier_dir="${RUNME_VERIFIER_DIR:-/logs/verifier}"
reward_path="${RUNME_REWARD_PATH:-$verifier_dir/reward.json}"
artifacts_dir="${RUNME_ARTIFACTS_DIR:-/logs/artifacts}"
mkdir -p "$verifier_dir"

cd "$workspace"
uvx --from 'harbor-rewardkit~=0.1.0' rewardkit \
  --workspace "$workspace" \
  --output "$reward_path" \
  "$tests_dir" \
  > "$verifier_dir/test-stdout.txt"

if [ -f "$workspace/poem.txt" ]; then
  mkdir -p "$artifacts_dir"
  cp "$workspace/poem.txt" "$artifacts_dir/poem.txt"
fi
