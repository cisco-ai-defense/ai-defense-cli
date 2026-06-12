# Changelog

All notable changes to `aid-cisco-cli` are documented here. This project adheres
to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Scan commands (`model-scan trigger`/`info`, `mcp-scan run`/`info --scan-id`) now
  print **raw API JSON by default** for Python CLI migration parity; use
  `--envelope` for the normalized verdict JSON/text. **Migration:** pipelines on
  `0.1.1` that parsed `envelope_version` on stdout without `--raw` must add
  `--envelope` (or `--raw=false`) after upgrade.
- `model-scan list` and `model-scan info` print verbatim API JSON
  (pass-through), matching Python CLI output shape; paging serializes as
  `count`/`offset`/`total` instead of `limit`.

## [0.1.1] - 2026-06-12

### Fixed

- **Release on merge** / **Create release tag** — explicitly dispatch the **Release** workflow
  after pushing a tag (`GITHUB_TOKEN` tag pushes do not trigger `on: push: tags` workflows).

### Added

- **Prepare release** workflow — hybrid changelog autofill, VERSION bump, pushes `release/v*`
  branch (open PR manually from workflow summary; base = chosen `ref`, including `release/0.1.x`).
- **Release on merge** — auto-tag when a prepare PR merges to `main` or `release/**`.
- **Create release tag** workflow (`workflow_dispatch`) — dry-run or manual tag escape hatch.
- **Semantic versioning** — `VERSION` file, `make version`, versioned release binaries
  (`aid-cisco-cli-<version>-<os>-<arch>`), `checksums.txt`, and `manifest.json`.
- **GitHub Release workflow** — push tag `v*` to publish versioned downloads.
- **Public repository** with Vault-inspired layout (`api/`, `internal/cli/`).
- **Multi-platform CI** — Linux and macOS embed builds on every PR; versioned artifacts
  published on `main` / `release/**` pushes only (7-day retention).
- **Install/upgrade** and user/developer getting-started guides.
- **Deterministic exit-code contract** for scan commands so CI/CD pipelines can
  branch on `$?` instead of parsing output. Operational failures (`1`-`9`) are
  partitioned from policy verdicts (`>= 10`):
  `0` pass, `2` usage, `3` auth/config, `4` connectivity/timeout, `5` scan
  failed/cancelled, `1` unexpected, `10` fail, `20` block.
- **Unified verdict envelope** emitted on stdout by `model-scan trigger`/`info`
  and `mcp-scan run`/`info --scan-id`: `envelope_version`, normalized decision,
  highest severity, threat counts, and per-severity breakdown (`--output json|text`).
- **Gating flags** `--fail-on` and `--block-on` (`none|low|medium|high|critical`)
  to configure which severities fail (`10`) or hard-block (`20`) the command;
  fail-closed by default.
- `--raw` flag to print the verbatim API response instead of the envelope
  (gating still applies).

### Changed

- CI: version resolution folded into the `test` job; `checksums` runs only on
  `main` / `release/**` pushes (not on pull requests).
- Upload client fallback uses redirect-safe HTTP client when `UploadHTTPClient` is unset.
- **Dropped `linux-arm64` release artifacts** — publish `linux-amd64` only; ARM Linux
  users can use Docker (`platform: linux/amd64`) or build from source.
- **Backward-incompatible vs. the Python `aidcli` this CLI replaces** (the Go CLI
  itself has not previously shipped):
  - Scan commands print the verdict envelope by default instead of the raw
    API JSON. Use `--raw` to restore the verbatim payload.
  - A completed scan with findings now exits `10`/`20` instead of `0`. Use
    `--fail-on none --block-on none` to report without gating.
  - Operational errors are now classified into `3`/`4`/`5` instead of the
    catch-all `1`. Usage errors remain `2`.
  - Command and flag names are unchanged, so existing invocations still run.
  - See [Migrating from the Python `aidcli`](docs/getting-started-client.md#7-migrating-from-the-python-aidcli).
