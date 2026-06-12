---
cwd: ../..
---

# Harbor Examples

These examples exercise Runme through Harbor's custom environment interface via
`runme eval`.

Run the smoke task with the oracle, which uses the known-good solution:

```sh {"name":"smoke-oracle"}
runme eval examples/harbor/datasets/runme-integration \
  --task simple-agent \
  --agent oracle
```

Run a deterministic weighted-scoring task with the oracle:

```sh {"name":"scoring-oracle"}
runme eval examples/harbor/datasets/runme-integration \
  --task text-stats-reward \
  --agent oracle
```

Run the same smoke task with Codex through Runme's Harbor adapter:

```sh {"name":"smoke-codex"}
runme eval examples/harbor/datasets/runme-integration \
  --task simple-agent \
  --agent codex \
  -y
```

The dataset root is `examples/harbor/datasets/runme-integration`. Each
`runme eval` creates Harbor job and trial metadata under `.runme/harbor/jobs`.
The `--task` flag selects a task directory inside the dataset, such as
`simple-agent` or `text-stats-reward`, even though the public task names in
`task.toml` are namespaced as `runmedev/simple-agent` and
`runmedev/text-stats-reward`.

`runme eval` delegates to `runme-harbor`, so these examples remain compatible
with the underlying `harbor run` workflow. The adapter supports `oracle`,
`codex`, `claude-code`, and `openclaw` agents and runs local agent CLIs through
`runme harbor stdio`.

Set `--runme-bin` to use a specific Runme binary. Set `--runme-arg` one or more
times to pass global Runme flags before `harbor stdio`.
