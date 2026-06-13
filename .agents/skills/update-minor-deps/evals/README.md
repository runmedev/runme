# Harbor Evals

Harbor evals for the `update-minor-deps` skill live here.

Run them from the Runme repository root:

```sh
runme eval .agents/skills/update-minor-deps/evals/regression --agent codex
```

The current task mutates the checkout that launches it, so run it from a disposable branch or worktree.
