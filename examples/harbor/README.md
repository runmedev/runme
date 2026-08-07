---
cwd: ../..
---

# Harbor Examples

These examples exercise Runme through Harbor's custom environment interface via
`runme eval`.

Create a new Runme-shaped eval task scaffold:

```sh {"name":"new-eval-task"}
runme eval task new runmedev/my-task
```

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

## Attribution

The `runme-rewardkit/text-stats-reward` task is adapted from Harbor's
[`examples/tasks/reward-kit-example`](https://github.com/harbor-framework/harbor/tree/54b478cd2a2627eb07a06d3528a4365a4f997a00/examples/tasks/reward-kit-example),
licensed under Apache-2.0. The Runme copy modifies paths, metadata,
validation behavior, and scoring details for Runme's Harbor integration
examples.

The `runme-smoke/simple-agent` task is original to Runme's Harbor integration
example set.

`runme eval` delegates to `runme-harbor`, so these examples remain compatible
with the underlying `harbor run` workflow. The adapter supports `oracle`,
`nop`, `antigravity-cli`, `codex`, `claude-code`, `cursor-cli`, and `openclaw`
agents and runs local agent CLIs through `runme harbor stdio` by default. The
default environment is `runme`; pass `--env runme` to select it explicitly.
Passing a non-Runme Harbor environment, such as `--env docker`, delegates the
selected environment and agent to Harbor without the Runme-specific agent
wrappers.
Runme environments default to one concurrent trial because they share the host
workspace. Other environments use Harbor's concurrency default. Override either
behavior with passthrough arguments such as `-- --n-concurrent 4`.

Set `--runme-bin` to use a specific Runme binary. Set `--runme-arg` one or more
times to pass global Runme flags before `harbor stdio`.
