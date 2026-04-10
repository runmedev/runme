# Runme Menu Bar Prototype

This directory contains a prototype macOS menu bar app (`RunmeMenuBar`) that:

- discovers config files matching `config*.yaml` / `config*.yml` in `~/.runme-agent`
- shows one submenu per config file in the menu bar menu
- launches `runme agent serve --config <path>` as a subprocess
- checks server health by polling `GET /metrics`
- exposes `Start/Stop`, `Open UI`, and `Open Log` per config

## Status

Prototype only. This is not yet integrated into release packaging, login items, auto-updates, or code signing/notarization.

## Run

```bash
cd experimental/macos/runme-menu
./scripts/bundle-runme.sh
swift run
```

`bundle-runme.sh` builds `runme` from this repository and places it at:

`Sources/RunmeMenuBar/Resources/runme`

The app prefers that bundled binary by default.

## Menu Behavior

- Each discovered config gets its own submenu.
- Submenu shows status and actions based on process state.
- Prototype currently enforces a single active config at a time to avoid port conflicts.

## Environment Variables

- `RUNME_BIN`: full path to `runme` executable (default: bundle, then Homebrew/common paths)
- `RUNME_CONFIG_DIR`: directory to scan for `config*.yaml` files (default: `~/.runme-agent`)
- `RUNME_CONFIG`: optional explicit config path to include in the discovered list
- `RUNME_ENDPOINT`: optional endpoint override for all configs
- `RUNME_LOG_PATH`: optional single log path override for all configs

## Notes

- Stop currently sends `SIGINT` to the subprocess.
- Endpoint parsing is best-effort from `assistantServer.bindAddress` and `assistantServer.port`.
