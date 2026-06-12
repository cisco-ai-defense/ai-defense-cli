# AI Defense CLI — Project Context

Last updated: 2026-06-12

## Purpose

Public OSS home for `aid-cisco-cli`: model-scan, mcp-scan, ai-bom with deterministic exit codes and release binaries.

## Status

- [x] Repo bootstrap
- [x] CLI source migrated on `feature/migrate-cli-5789`
- [x] PR/main CI (`.github/workflows/ci.yml`)
- [x] Advisory persona review on PRs (`.github/workflows/pr-persona-review.yml`, read-only) — test + 4-platform embed builds
- [x] Version file + versioned artifacts (`VERSION`, `manifest.json`)
- [x] GitHub Release workflow (`.github/workflows/release.yml` on `v*` tags)
- [x] Install/upgrade docs (`docs/install-upgrade.md`)
- [ ] First public GitHub release (`v0.1.0` tag after merge)
- [x] PR review fixes: transport errors (exit 4), TOML parse errors, macOS Gatekeeper docs
- [x] Upload redirect SSRF hardening, verdict `envelope_version`, mcp-scan `PollTimeoutError`
- [x] CI: fold `meta` into `test`; `checksums` job only on main/release push

## Deferred (post-merge)

- macOS Developer ID signing + notarization in release CI
- GitHub Actions SHA pinning (org policy)

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
| `make build-release-linux` | `dist/aid-cisco-cli-<version>-linux-*` |
| `make build-release-darwin` | `dist/aid-cisco-cli-<version>-darwin-*` |

## Conventions

- Spec-first: `.cursor/skills/spec-driven-dev`
- Review: `.cursor/skills/pr-review` before merge

## Merge gate

PRs require: `test`, `linux-amd64`, `darwin-amd64`, `darwin-arm64` only.

Do not require `meta` (folded into `test`) or `checksums` (main/release push only).
