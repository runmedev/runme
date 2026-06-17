# Runme Harbor

Runme Harbor is the optional Python adapter used by `runme eval`. It connects
Harbor's environment interface to a local `runme harbor stdio` process.

Runme Harbor builds on the upstream Harbor Python package.

## Install

Install the adapter as an isolated Python CLI tool:

```sh {"name":"install-harbor"}
uv tool install runme-harbor
```

The `runme` CLI must be installed separately and available on `PATH`.

## Build

Run the full local validation and build workflow:

```sh {"name":"build-harbor"}
set -euo pipefail

rm -rf dist

buf generate ../../api/proto \
  --template buf.gen.yaml \
  --path ../../api/proto/runme/harbor/v1/harbor.proto

uv sync --locked --all-extras --dev
uv run ruff check .
uv run pytest

(cd ../.. && go test ./cmd ./internal/harbor)

uv build

tmp_venv="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_venv"
}
trap cleanup EXIT

python3 -m venv "$tmp_venv"
"$tmp_venv/bin/pip" install -q dist/*.whl
"$tmp_venv/bin/python" -c "import runme_harbor"
"$tmp_venv/bin/runme-harbor" --help >/dev/null
```

Publishing is intentionally not part of `build-harbor`. Release automation runs
this workflow first, then calls `uv publish`.
