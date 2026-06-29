#!/usr/bin/env sh
set -eu

uvx --from harbor-rewardkit==0.1 rewardkit \
  --workspace /app/examples/harbor/datasets/runme-rewardkit/text-stats-reward/workdir \
  --output /logs/verifier/reward.json \
  /tests

if [ -f /app/examples/harbor/datasets/runme-rewardkit/text-stats-reward/workdir/results.json ]; then
  mkdir -p /logs/artifacts
  cp /app/examples/harbor/datasets/runme-rewardkit/text-stats-reward/workdir/results.json \
    /logs/artifacts/results.json
fi
