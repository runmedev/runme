#!/usr/bin/env sh
set -eu

expected="${RUNME_HARBOR_EXPECTED:-hello from a real agent}"

if [ ! -f result.txt ]; then
  printf 'missing result.txt\n' >&2
  printf '0.0' > /logs/verifier/reward.txt
  exit 1
fi

actual="$(cat result.txt)"

if [ "$actual" = "$expected" ]; then
  mkdir -p /logs/artifacts
  cp result.txt /logs/artifacts/result.txt
  printf '1.0' > /logs/verifier/reward.txt
else
  printf 'expected %s, got %s\n' "$expected" "$actual" >&2
  printf '0.0' > /logs/verifier/reward.txt
  exit 1
fi
