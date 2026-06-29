---
cwd: ../..
---

# Harbor Examples

These examples exercise Runme through Harbor's custom environment interface via
`runme eval`.

Run the smoke task with the oracle, which uses the known-good solution:

```sh {"name":"smoke-oracle"}
runme eval examples/harbor/datasets/runme-smoke \
  --task-dir simple-agent \
  --agent oracle
```

Run a deterministic weighted-scoring task with the oracle:

```sh {"name":"scoring-oracle"}
runme eval examples/harbor/datasets/runme-rewardkit \
  --task-dir text-stats-reward \
  --agent oracle
```

Run the same smoke task with Codex through Runme's Harbor adapter:

```sh {"name":"smoke-codex"}
runme eval examples/harbor/datasets/runme-smoke \
  --task-dir simple-agent \
  --agent codex
```

Run the smoke task with Harbor's Docker environment for a baseline comparison:

```sh {"name":"smoke-docker"}
runme eval examples/harbor/datasets/runme-smoke \
  --task-dir simple-agent \
  --env docker \
  --agent oracle
```

Run the rewards scoring example with a real agent:

```sh {"name":"scoring-codex"}
runme eval examples/harbor/datasets/runme-rewardkit \
  --task-dir text-stats-reward \
  --agent codex
```

The dataset roots are `examples/harbor/datasets/runme-smoke` and
`examples/harbor/datasets/runme-rewardkit`. Each `runme eval` creates Harbor
job and trial metadata under `.runme/evals/jobs`. The `--task-dir` flag selects
a task directory inside the dataset, such as `simple-agent` or
`text-stats-reward`.

`runme eval` delegates to `runme-harbor`, so these examples remain compatible
with the underlying `harbor run` workflow. The adapter supports `oracle`,
`codex`, `claude-code`, and `openclaw` agents and runs local agent CLIs through
`runme harbor stdio` by default. The default environment is `runme`; pass
`--env runme` to select it explicitly. Passing a non-Runme Harbor environment,
such as `--env docker`, delegates the selected environment and agent to Harbor
without the Runme-specific agent wrappers.

Set `--runme-bin` to use a specific Runme binary. Set `--runme-arg` one or more
times to pass global Runme flags before `harbor stdio`.
