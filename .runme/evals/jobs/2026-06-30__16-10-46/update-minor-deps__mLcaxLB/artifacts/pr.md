# chore: update minor and patch dependencies (2026-06-30)

## Summary

- Ran the documented Go dependency update command: `runme run update-go-deps`.
- Tidied the root module with `go mod tidy`.
- Updated root dependency files only: `go.mod` and `go.sum`.
- Bumped Google dependencies:
  - `google.golang.org/api` from `v0.286.0` to `v0.287.0`
  - `google.golang.org/genproto` from `v0.0.0-20260622175928-b703f567277d` to `v0.0.0-20260630182238-925bb5da69e7`
  - `google.golang.org/genproto/googleapis/api` from `v0.0.0-20260622175928-b703f567277d` to `v0.0.0-20260630182238-925bb5da69e7`
  - `google.golang.org/genproto/googleapis/rpc` from `v0.0.0-20260622175928-b703f567277d` to `v0.0.0-20260630182238-925bb5da69e7`
- No code or test compatibility fixes were needed.

## Validation

- `go test ./internal/owl ./pkg/agent/runme/stream ./pkg/agent/server ./runner ./runner/client`
- `runme run lint test`
