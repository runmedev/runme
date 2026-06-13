# Update Runme Minor/Patch Dependencies

Run this task from the repository root.

Perform Runme's normal minor and patch dependency maintenance cycle. Refresh non-breaking dependencies, handle any small regressions caused by the update, and validate the result.

Stay on the starting branch. Do not create a branch, switch branches, stage files, commit, push, or open a real pull request. Leave the workspace changes unstaged so the verifier can inspect them.

Instead of opening a PR, write a Markdown PR draft to:

```text
/logs/artifacts/pr.md
```
