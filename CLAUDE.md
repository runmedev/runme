# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Runme is a tool that makes markdown files executable by running code blocks directly from READMEs, runbooks, and documentation. It supports Shell/Bash, Python, Ruby, JavaScript/TypeScript, and other runtimes via shebang. Environment variables persist across execution like a terminal session.

## Prerequisites

- **Runme CLI installed** (`brew install runme` or `npm install -g runme`) - we use runme to build runme
- Go 1.25.2

## Common Commands

Use `runme list` to see all available named commands. Key development commands:

```bash
runme run build         # Build CLI binary to ./runme
runme run setup         # Install dev tools (gofumpt, revive, etc.)
runme run lint          # Run full linting suite
runme run test          # Run all tests (unsets env vars, clears cache, builds first)
runme run test-docker   # Run tests in Docker (for integration tests with Python/Node)
runme run server-dev    # Run server in dev mode on 127.0.0.1:9999
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for additional make targets and detailed development instructions.

## Git Commit Requirements

**IMPORTANT**: This project requires all contributors to sign off on the Developer Certificate of Origin (DCO) as required by CNCF/LF.

You MUST use the `-s` flag with every git commit:

```bash
git commit -s -m "Your commit message"
```

This adds a "Signed-off-by" line to your commit message, certifying that you have the right to submit the code under the project's license.

**Never commit without the `-s` flag.** Commits without DCO signoff cannot be merged.

## Architecture

**Core Flow**: Markdown parsing → Code block extraction → Command execution → Output streaming

Key packages:

- **`document/`** - Markdown parsing, AST, block identification, frontmatter extraction
- **`runner/`** - Execution engine, PTY/terminal support, session management, environment persistence
- **`command/`** - Command abstractions (shell, inline, Docker, file-based)
- **`project/`** - File system navigation, git integration, markdown file discovery
- **`internal/owl/`** - Experimental environment variable management with dependency tracking and secret manager integration
- **`runnerv2service/`** - ConnectRPC API implementation for v2 runner
- **`api/proto/`** - Protocol buffer definitions; generated code in `api/gen/`
- **`cmd/`** - CLI commands (run, list, print, fmt, env, server)
- **`internal/tui/`** - Terminal UI with Bubble Tea framework
- **`pkg/agent/`** - Experimental AI agent implementation

**Note**: The project includes a v2 beta implementation of the CLI and a separate v2 runner implementation (actively used in production).

## Development Notes

**IMPORTANT: You MUST always run `runme run lint test` after making changes.**

- Tests require `TZ=UTC` environment variable
- Protocol buffers managed with buf CLI; TypeScript definitions published to buf.build registry
- Pre-commit hooks require `python3 -m pip install pre-commit`

## Runme Code Block Attributes

Runme extends standard markdown fenced code blocks with attributes: ` ```sh { name=block-name, interactive=false } `

Key attributes: `name`, `interactive`, `background`, `terminalRows`, `id`

See [Runme configuration docs](https://docs.runme.dev/configuration/) for details.
