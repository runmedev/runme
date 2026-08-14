#!/usr/bin/env sh
set -eu

workspace="${RUNME_TASK_WORKDIR:-$PWD}"
tests_dir="${RUNME_TESTS_DIR:-/tests}"
verifier_dir="${RUNME_VERIFIER_DIR:-/logs/verifier}"
reward_path="${RUNME_REWARD_PATH:-$verifier_dir/reward.json}"
artifacts_dir="${RUNME_ARTIFACTS_DIR:-/logs/artifacts}"
mkdir -p "$verifier_dir"

uvx --from 'harbor-rewardkit~=0.1.0' rewardkit \
  --workspace "$workspace" \
  --output "$reward_path" \
  "$tests_dir"

if [ -f "$workspace/results.json" ]; then
  mkdir -p "$artifacts_dir"
  cp "$workspace/results.json" "$artifacts_dir/results.json"
fi
