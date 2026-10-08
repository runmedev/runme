#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_DIR}/../../.." && pwd)"
RESOURCE_DIR="${APP_DIR}/Sources/RunmeMenuBar/Resources"
OUT_BIN="${RESOURCE_DIR}/runme"

mkdir -p "${RESOURCE_DIR}"

echo "Building runme from ${REPO_ROOT}"
(
  cd "${REPO_ROOT}"
  go build -o "${OUT_BIN}" ./main.go
)

chmod +x "${OUT_BIN}"
echo "Bundled runme at ${OUT_BIN}"
