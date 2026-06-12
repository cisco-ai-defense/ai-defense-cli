# ADR 0001: Package layout (`api/` + `internal/cli/`)

## Status

Accepted

## Context

The CLI was developed in the monorepo as `aid_cli/` with `cmd/`, `sdk/aidefense/`, and `internal/`.
This repository is public OSS and may later publish a Go client library separately from the binary.

## Decision

Adopt a Vault/Terraform-inspired layout:

| Path | Package | Importable |
|------|---------|------------|
| `main.go` | `main` | no |
| `internal/cli/` | `cli` (+ subpackages) | no |
| `internal/cmdutil/` | `cmdutil` | no |
| `api/` | `api` (+ service subpackages) | **yes** |
| `internal/*` | private helpers | no |

`internal/cmdutil` remains a separate package (not merged into `internal/cli`) to avoid an import cycle: `internal/cli` registers subcommands in `internal/cli/modelscan`, which needs shared CLI context types.

## Consequences

- HTTP client surface is `github.com/cisco-ai-defense/ai-defense-cli/api`
- Command tree is not importable; thin `main.go` calls `internal/cli.Run()`
- Future extraction of `api/` to its own module is straightforward
