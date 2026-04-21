# Consolidation plan for runmedev/vscode-runme Dependabot PRs

Date audited: 2026-04-21

## Passing Dependabot PRs detected

- #2269 `axios` 1.13.6 → 1.15.0
- #2268 `basic-ftp` 5.2.0 → 5.2.2
- #2266 `yaml` 2.8.2 → 2.8.3
- #2264 `picomatch` 2.3.1 → 2.3.2
- #2255 `ajv` 6.12.6 → 6.14.0
- #2247 `examples/k8s lodash` 4.17.21 → 4.17.23
- #2246 `lodash` 4.17.21 → 4.17.23
- #2243 `filenamify` 6.0.0 → 7.0.1
- #2236 `@google-cloud/compute` 5.3.0 → 6.6.0
- #2192 `@connectrpc/connect-node` 1.6.1 → 1.7.0
- #2140 `eslint-import-resolver-typescript` 3.8.5 → 4.4.4

## Suggested single-branch consolidation workflow

```bash
# in a local clone of runmedev/vscode-runme
git checkout main
git pull --ff-only
git checkout -b chore/consolidate-passing-dependabot

for pr in 2269 2268 2266 2264 2255 2247 2246 2243 2236 2192 2140; do
  git fetch origin pull/$pr/head:pr-$pr
  git merge --no-edit pr-$pr || break
done

npm ci
npm test
```

If lockfile conflicts occur, resolve by re-running `npm install` for the requested package versions, then re-run test/CI.
