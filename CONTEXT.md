# AI Defense CLI — Project Context

Last updated: 2026-06-12

## Purpose

Public OSS home for `aid-cisco-cli`: model-scan, mcp-scan, ai-bom with deterministic exit codes and release binaries.

## Status

- [x] Repo bootstrap
- [x] CLI source migrated (PR #5789) on `feature/migrate-cli-5789`
- [x] PR/main CI (`.github/workflows/ci.yml`)
- [x] Advisory persona review on PRs (`.github/workflows/pr-persona-review.yml`, read-only) — test + 4-platform embed builds
- [ ] Release workflow from PR #5790
- [ ] Install/upgrade docs from PR #5810
- [ ] First public GitHub release

## Layout

```
main.go
api/                      # public HTTP client
internal/cli/             # Cobra commands
internal/cmdutil/         # shared CLI context (avoids import cycle)
internal/clierror/        # exit codes
internal/verdict/         # scan envelope
internal/config/          # flags > env > TOML
internal/embedded/python/ # CPython + cisco-aibom
scripts/                  # bundle + docker helpers
docs/specs/001-cli-core/
Dockerfile                # linux release → dist/ binary
Makefile                  # build-go, build-embed, build-release-*
```

Module: `github.com/cisco-ai-defense/ai-defense-cli`

## Build

| Target | Output |
|--------|--------|
| `make build-go` | `bin/aid-cisco-cli` (stub Python) |
| `make build-embed` | `bin/aid-cisco-cli` (full embed) |
| `make build-release-linux` | `dist/aid-cisco-cli-linux-*` |
| `make build-release-darwin` | `dist/aid-cisco-cli-darwin-*` |

## Conventions

- Spec-first: `.cursor/skills/spec-driven-dev`
- Review: `.cursor/skills/pr-review` before merge

## Merge gate

`main` requires CI checks: test, linux-amd64/arm64, darwin-amd64/arm64, checksums.
