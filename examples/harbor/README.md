# Harbor Examples

These examples exercise Runme through Harbor's custom environment interface.

Run the oracle smoke, which uses the known-good solution:

```sh
RUNME_BIN="$PWD/runme" \
  uv run --project integrations/harbor \
    harbor run \
      --path examples/harbor/datasets/runme-integration \
      --include-task-name local-oracle \
      --agent oracle \
      --jobs-dir .runme/harbor/jobs \
      --environment-import-path runme_harbor.environment:RunmeEnvironment
```

Run a local Codex smoke through Runme's Harbor adapter:

```sh
RUNME_BIN="$PWD/runme" \
  uv run --project integrations/harbor \
    harbor run \
      --path examples/harbor/datasets/runme-integration \
      --include-task-name local-agent \
      --agent-import-path runme_harbor.local_codex:LocalCodex \
      --jobs-dir .runme/harbor/jobs \
      --environment-import-path runme_harbor.environment:RunmeEnvironment \
      -y
```

The dataset root is `examples/harbor/datasets/runme-integration`, which keeps
Harbor job metadata readable while `--include-task-name` selects an individual
task directory inside that dataset. Harbor filters by task directory name, so the
example uses `--include-task-name local-agent` even though the task is named
`runme/local-agent` in `task.toml`.

`LocalCodex` reuses Harbor's Codex execution and ATIF conversion logic, but
skips Harbor's container bootstrap phase. It expects the local `codex` CLI and
Codex auth to already be available, then runs Codex through `environment.exec`
so command execution flows through `runme harbor stdio`.

Set `RUNME_BIN` to use a specific Runme binary. Set `RUNME_ARGS` or the Harbor
environment kwarg `runme_args` to pass global Runme flags before `harbor stdio`.
