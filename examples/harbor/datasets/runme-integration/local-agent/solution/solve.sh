#!/usr/bin/env sh
set -eu

printf '%s\n' "${RUNME_HARBOR_EXPECTED:-hello from a real agent}" > result.txt
