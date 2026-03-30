# Runme Menu Bar Prototype

This directory contains a prototype macOS menu bar app (`RunmeMenuBar`) that:

- launches `runme agent serve` as a subprocess
- checks server health by polling `GET /metrics`
- exposes `Start`, `Stop`, `Open Runme UI`, and `Open Log File` actions

## Status

Prototype only. This is not yet integrated into release packaging, login items, auto-updates, or code signing/notarization.

## Run

```bash
cd experimental/macos/runme-menu
swift run
```

## Environment Variables

- `RUNME_BIN`: full path to `runme` executable (default: bundle, then Homebrew/common paths)
- `RUNME_CONFIG`: config file path passed to `runme agent serve --config` (default: `~/.runme-agent/config.yaml`)
- `RUNME_ENDPOINT`: UI/server endpoint (default: `http://127.0.0.1:8080`)
- `RUNME_LOG_PATH`: log file path (default: `~/Library/Logs/Runme/agent.log`)

## Notes

- Stop currently sends `SIGINT` to the subprocess.
- The app currently does not bootstrap or validate `RUNME_CONFIG`; it expects an existing working config.
