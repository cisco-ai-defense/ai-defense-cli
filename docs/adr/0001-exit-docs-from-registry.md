# ADR 0001: Exit-code handbook from a Go registry

## Status

Accepted (2026-06-16)

## Context

`docs/exit-codes-handbook.md` documents per-command exit codes and error
scenarios for clients and operators. Hand-maintaining it drifts from
`internal/clierror` and command handlers.

Go does not have a standard, tool-supported way to attach machine-readable
“response codes” to functions (unlike OpenAPI or some Java/Kotlin annotations).
`go doc` renders godoc comments but nothing in the ecosystem reliably extracts
structured error catalogs from `RunE` bodies or `return clierror.Usage(...)`.

## Decision

1. **Source of truth:** `internal/cliexitdoc/registry.go` — typed `Scenario`
   rows grouped by command, referencing `clierror` code constants where
   possible.
2. **Generator:** `cmd/gen-exit-docs` writes `docs/exit-codes-handbook.md`.
3. **CI / local check:** `make check-exit-docs` fails if the handbook is stale
   (same pattern as formatting or coverage gates).
4. **Developer workflow:** when changing exit behavior, update the registry
   (and handler/tests), then `make gen-exit-docs`.

Command handlers stay the runtime source of truth for behavior; the registry is
the documentation contract. Tests in `internal/cliexitdoc` validate that
documented codes are allowed contract values.

## Alternatives considered

| Approach | Pros | Cons |
|----------|------|------|
| Godoc-only (`// Errors: …` on `RunE`) | Colocated with code | No standard format; fragile parsers; easy to skip |
| AST linter extracting `clierror.*` calls | Always in sync with returns | Misses `Classify()` / API paths; high false negatives |
| YAML/JSON sidecar | Easy to edit | Duplicated paths; no compile-time code constant checks |
| **Go registry + codegen (chosen)** | Type-checked codes; reviewable diffs; CI gate | Manual registry updates (explicit, like OpenAPI) |

## Consequences

- New commands or exit paths: add registry rows + run `make gen-exit-docs`.
- Phase 2 (optional): AST check that explicit `clierror` calls in a file are
  documented. Table-driven contract tests import registry rows for
  `model-scan` and `config validate` (`internal/cli/exit_contract_test.go`).

## References

- `internal/clierror` — runtime contract
- `docs/specs/001-cli-core/requirements.md` — REQ-D04, REQ-D05
- `make gen-exit-docs` / `make check-exit-docs`
