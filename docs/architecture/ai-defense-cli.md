# AI Defense CLI — Architecture

## Overview

`aid-cisco-cli` is a standalone Go binary for Cisco AI Defense APIs and local AIBOM analysis.

## Components

| Package | Role |
|---------|------|
| `internal/cli/` | Cobra command tree (`model-scan`, `mcp-scan`, `ai-bom`, `install`, `onboard`, `config`) |
| `api/` | HTTP client (publishable import path) |
| `internal/config` | Auth and endpoint resolution; TOML read/write |
| `internal/config/spec` | Embedded `config.spec.yaml` — wizard, env/flag bindings, validation |
| `internal/selfupdate` | GitHub Releases download, checksum verify, atomic binary replace |
| `internal/clierror` | Exit-code contract |
| `internal/verdict` | Scan result envelope |
| `internal/cmdutil` | `--fail-on`, `--block-on`, `--output`, `--raw` (default true), `--envelope` |
| `internal/embedded/python` | Relocatable CPython + `cisco-aibom` |
| `scripts/` | Bundle build helpers |

Further reading: [docs/README.md](../README.md) · [internal/config/README.md](../../internal/config/README.md)

## Embedded Python

`ai-bom` runs `python -m aibom analyze` using embedded CPython in release builds.

## Build

| Target | Output |
|--------|--------|
| `make build-go` | `bin/aid-cisco-cli` (stub) |
| `make build-embed` | `bin/aid-cisco-cli` (full embed) |
| `make build-release-linux` | `dist/aid-cisco-cli-linux-*` (Docker) |
| `make build-release-darwin` | `dist/aid-cisco-cli-darwin-*` (native) |

## Exit codes

0 success; 2 usage; 3 auth; 4 connectivity; 5 scan failed; 10 fail; 20 block.

See [spec 001](../specs/001-cli-core/design.md).
