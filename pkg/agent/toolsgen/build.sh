#!/bin/bash
set -ex

# Resolve the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$SCRIPT_DIR"

# Run buf in-place using the workspace and tools module.
pushd ./
cd "${SCRIPT_DIR}/../../.."
buf generate --template "${SCRIPT_DIR}/buf.gen.yaml" --path api/proto-tools

NOTEBOOKS_FILE="api/gen/proto-tools/agent/v1/agentv1mcp/notebooks.pb.mcp.go"
if [[ ! -f "${NOTEBOOKS_FILE}" ]]; then
  echo "Expected generated file ${NOTEBOOKS_FILE} not found" >&2
  exit 1
fi

# Keep only the tool descriptors; downstream code consumes just the var block.
python3 - "${NOTEBOOKS_FILE}" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
lines = path.read_text().splitlines(keepends=True)
keep = []
inside_var_block = False
found_var_block = False
trimmed = False

for line in lines:
    keep.append(line)
    if not inside_var_block:
        if line.startswith("var ("):
            inside_var_block = True
            found_var_block = True
    else:
        if line.strip() == ")":
            trimmed = True
            break

if not found_var_block:
    raise SystemExit("Failed to locate top-level var block in {}".format(path))

if not trimmed:
    raise SystemExit("Failed to find end of top-level var block in {}".format(path))

path.write_text("".join(keep))
PY

popd

# Rewrite imports so generated code references our vendored fork.
#OLD_IMPORT="github.com/mark3labs/mcp-go/mcp"
#NEW_IMPORT="go.openai.org/project/aisre/toolsgen/mark3labs/mcp"
#if ls aisremcp/*.go >/dev/null 2>&1; then
#  # macOS sed requires an empty string argument for -i backups.
#  find aisremcp -type f -name '*.go' -exec sed -i '' -e "s|${OLD_IMPORT}|${NEW_IMPORT}|g" {} +
#fi

# Strip unused imports
# You can install with go install golang.org/x/tools/cmd/goimports@latest
goimports -w "${NOTEBOOKS_FILE}"
