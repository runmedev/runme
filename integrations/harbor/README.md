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

proto_src="../../api/gen/proto/python/runme/harbor/v1"
proto_dst="src/runme_harbor/_proto/runme/harbor/v1"
mkdir -p "$proto_dst"
cp "$proto_src/harbor_pb2.py" "$proto_src/harbor_pb2.pyi" "$proto_dst/"

uv sync --locked --all-extras --dev
uv run ruff check .
uv run pytest

(cd ../.. && go test ./cmd ./internal/harbor)

uv build

uv run --with dist/*.whl python -c "import runme_harbor"
uv tool run --from dist/*.whl runme-harbor --help >/dev/null
```

Publishing is intentionally not part of `build-harbor`. Release automation runs
this workflow first, then calls `uv publish`.
