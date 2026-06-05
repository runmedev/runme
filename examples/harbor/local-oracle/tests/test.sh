#!/usr/bin/env sh
set -eu

expected="${RUNME_HARBOR_EXPECTED:-hello from runme harbor}"
actual="$(cat result.txt)"

if [ "$actual" = "$expected" ]; then
  printf '1.0' > /logs/verifier/reward.txt
else
  printf 'expected %s, got %s\n' "$expected" "$actual" >&2
  printf '0.0' > /logs/verifier/reward.txt
  exit 1
fi
