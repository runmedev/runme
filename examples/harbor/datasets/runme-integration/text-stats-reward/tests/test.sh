#!/usr/bin/env sh
set -eu

uvx --from harbor-rewardkit==0.1 rewardkit \
  --workspace /app/examples/harbor/datasets/runme-integration/text-stats-reward/workdir \
  --output /logs/verifier/reward.json \
  /tests
