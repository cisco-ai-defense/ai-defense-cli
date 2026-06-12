# Spec 001: CLI core

## Binary

- Name: `aid-cisco-cli`
- Module: `github.com/cisco-ai-defense/ai-defense-cli`

## Commands

| Group | Purpose |
|-------|---------|
| `model-scan` | Trigger and retrieve AI/ML model security scans |
| `mcp-scan` | Register and scan MCP servers |
| `ai-bom` | Generate and manage AI Bill of Materials (local analyzer via embedded Python) |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success / policy pass |
| 1 | Unexpected error |
| 2 | Usage error |
| 3 | Auth or config error |
| 4 | Connectivity / timeout |
| 5 | Scan operation failed |
| 10 | Policy fail (`--fail-on`) |
| 20 | Policy block (`--block-on`) |

Scan commands emit raw API JSON on stdout by default (Python CLI parity).
`--envelope` prints the normalized verdict (`--output json|text`); gating still applies.

## Configuration precedence

Flags > environment variables > TOML config file (`internal/config`).

## Build modes

| Mode | Tag | Embedded Python |
|------|-----|-----------------|
| Dev / unit tests | `!embed_python` (default) | Stub — set `AID_CLI_PYTHON` for ai-bom dev |
| Release | `embed_python` | Full CPython + `cisco-aibom[all]` bundle |

## Release artifacts

| Target | Output |
|--------|--------|
| `make build-go` | `bin/aid-cisco-cli` |
| `make build-embed` | `bin/aid-cisco-cli` (after `make bundle`) |
| `make build-release-linux` | `dist/aid-cisco-cli-<version>-linux-amd64` via Docker |
| `make build-release-darwin` | `dist/aid-cisco-cli-darwin-{amd64,arm64}` native on macOS |

Linux release uses Debian bookworm Docker stages and exports a binary (`docker build --target release --output`). macOS release runs `make bundle` + `go build -tags embed_python` on a Darwin host (embedded Python is OS-specific).

## Package boundaries

- `api/` must not import `internal/cli` or other private packages
- Commands map errors to `internal/clierror` at the boundary
