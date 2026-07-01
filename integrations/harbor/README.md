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

## Runtime Environment

Runme Harbor exposes these runtime paths to task and verifier commands:

- `RUNME_TASK_WORKDIR`: resolved task workspace.
- `RUNME_TASK_DIR`: task definition directory.
- `RUNME_TASK_NAME`: task name.
- `RUNME_TESTS_DIR`: uploaded verifier tests directory.
- `RUNME_LOGS_DIR`: base trial logs directory.
- `RUNME_AGENT_LOG_DIR`: agent logs directory.
- `RUNME_ARTIFACTS_DIR`: artifact output directory.
- `RUNME_VERIFIER_DIR`: verifier output directory.
- `RUNME_REWARD_PATH`: canonical reward JSON path.
- `RUNME_REWARD_DETAILS_PATH`: optional detailed reward JSON path.

## Build

Build the distribution artifacts:

```sh {"name":"build-harbor"}
set -euo pipefail

rm -rf dist

proto_src="../../api/gen/proto/python/runme/harbor/v1"
proto_dst="src/runme_harbor/_proto/runme/harbor/v1"
mkdir -p "$proto_dst"
cp "$proto_src/harbor_pb2.py" "$proto_src/harbor_pb2.pyi" "$proto_dst/"

uv build
```

Publishing is intentionally not part of `build-harbor`. Release automation runs
this workflow first, then calls `uv publish`.
