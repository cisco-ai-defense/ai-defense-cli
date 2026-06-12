# Changelog

All notable changes to `aid-cisco-cli` are documented here. This project adheres
to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-06-12

### Added
- **Prepare release** workflow — hybrid changelog autofill, VERSION bump, opens `release/v*` PR
- **Release on merge** — auto-tag when a prepare PR merges to `main` or `release/**`.
- **Create release tag** workflow (`workflow_dispatch`) — dry-run or manual tag escape hatch.
- **Semantic versioning** — `VERSION` file, `make version`, versioned release binaries
- **GitHub Release workflow** — push tag `v*` to publish versioned downloads.
- **Public repository** with Vault-inspired layout (`api/`, `internal/cli/`).
- **Multi-platform CI** — Linux and macOS embed builds on every PR; versioned artifacts
- **Install/upgrade** and user/developer getting-started guides.
- **Deterministic exit-code contract** for scan commands so CI/CD pipelines can
- **Unified verdict envelope** emitted on stdout by `model-scan trigger`/`info`
- **Gating flags** `--fail-on` and `--block-on` (`none|low|medium|high|critical`)
- `--raw` flag to print the verbatim API response instead of the envelope

### Changed
- CI: version resolution folded into the `test` job; `checksums` runs only on
- Upload client fallback uses redirect-safe HTTP client when `UploadHTTPClient` is unset.
- **Dropped `linux-arm64` release artifacts** — publish `linux-amd64` only; ARM Linux
- **Backward-incompatible vs. the Python `aidcli` this CLI replaces** (the Go CLI

## [Unreleased]

### Changed

- **Prepare release** — pushes `release/v*` branch only; open the version-bump PR manually
  (workflow summary link). Avoids org restriction on Actions creating pull requests.