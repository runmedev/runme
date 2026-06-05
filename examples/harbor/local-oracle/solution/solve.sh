#!/usr/bin/env sh
set -eu

printf '%s\n' "${RUNME_HARBOR_EXPECTED:-hello from runme harbor}" > result.txt
