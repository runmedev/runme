# chore: update minor and patch dependencies (2026-06-30)

## Summary

- Ran Runme's documented non-breaking Go dependency update command via `.agents/skills/update-minor-deps/scripts/update-go-deps.sh`.
- Updated root module dependency metadata in `go.mod` and `go.sum`.
- Refreshed minor/patch versions for direct dependencies including `connectrpc.com/grpchealth`, `github.com/mark3labs/mcp-go`, `github.com/pelletier/go-toml/v2`, `google.golang.org/api`, and `google.golang.org/grpc`.
- Refreshed related indirect dependencies and checksums, including Google Cloud/genproto packages, go-openapi packages, `golang.org/x/tools`, `github.com/prometheus/procfs`, and `k8s.io/kube-openapi`.

## Compatibility Fixes

- None required. The dependency refresh did not expose code or test compatibility regressions.

## Validation

- `runme run lint test`
  - `lint` passed.
  - `test` passed, including the full `TZ=UTC go test -ldflags=... -run=".*" -tags="test_with_txtar" -timeout=60s -race=false "./..."` suite.

## Changed Files

- `go.mod`
- `go.sum`
