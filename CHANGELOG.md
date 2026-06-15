# Changelog

All notable changes to `aid-cisco-cli` are documented here. This project adheres
to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`aid-cisco-cli install`** — download and verify release binaries from GitHub
  Releases (`--version`, `--check`, `--prefix`).
- **`aid-cisco-cli onboard`** and **`aid-cisco-cli config`** — interactive TUI wizard
  (Charm `huh`) for core credentials plus optional capability modules (private
  HuggingFace repos, local AIBOM LLM settings).
- **Onboarding draft saves** — config written after each field; `[onboard].step`
  tracks progress for resume.
- **`config show`** — TUI section browser on a terminal; `--plain` for redacted TOML.
- **`config edit`** — section menu to change core API or capabilities (configure / remove).
- **Home config directory** — `~/.ai-defense/config.toml` (like `~/.aws`); wizard always writes here unless `AI_DEFENSE_CONFIG_PATH` is set. Legacy XDG and project `.config.toml` paths still read for migration.
- **Config profiles** — named profiles in `~/.ai-defense/profiles/<name>.toml`; `config profile` commands and global `--profile` / `AI_DEFENSE_PROFILE`; onboard opens a profile picker (active profile pre-selected) unless `--profile` is set.
- **AIBOM LLM from TOML** — `[default.aibom]` fields resolve after flags and env vars.
- **Spec-driven integrations** — `internal/config/spec/config.spec.yaml` (embedded schema;
  user config remains `config.toml`) declares env/TOML/wizard fields; capabilities, CLI
  flags, redaction, and resolution share the same schema.
- CI enforces minimum Go test coverage (`make test-cover-check`, default floor 75%).
  `scripts/check-coverage.sh` and `make test-cover` for local runs. Tooling under
  `scripts/`, `main.go`, and TUI/onboarding packages (`install`, `onboard`, `config`,
  `internal/tui`, `internal/selfupdate`, `internal/config/spec`) are excluded from
  the gate until they have unit tests. Command-layer HTTP tests use
  `internal/testutil/apitest` (httptest + `*.cisco.com` base URL rewrite).

### Changed

- CLI `--help`/usage now renders as titled panels with a color scheme matching
  the embedded AI BOM (`cisco-aibom`) CLI — faint borders/titles, bold-yellow
  `Usage:`, bold-cyan option names, bold-green short flags, bold-yellow type
  metavars — so native and aibom-backed help look consistent. Color is gated to
  the destination stream being a TTY and honors `NO_COLOR` / `FORCE_COLOR`.
- **`make bundle`** — skips rebuild when Python bundle inputs are unchanged; use
  `make bundle-force` or `BUNDLE_FORCE=1` to rebuild. Caches PBS tarball and pip
  wheels under `.build/`.
- **Onboard/config TUI** — interactive wizard and config browse/edit use the same
  Typer/Rich-style colors as `--help` (honors `NO_COLOR` / `FORCE_COLOR`).

## [0.1.2] - 2026-06-12

### Changed

- Scan commands (`model-scan trigger`/`info`, `mcp-scan run`/`info --scan-id`) now
  print **raw API JSON by default** for Python CLI migration parity; use
  `--envelope` for the normalized verdict JSON/text. **Migration:** pipelines on
  `0.1.1` that parsed `envelope_version` on stdout without `--raw` must add
  `--envelope` (or `--raw=false`) after upgrade.
- `model-scan list` and `model-scan info` print verbatim API JSON
  (pass-through), matching Python CLI output shape; paging serializes as
  `count`/`offset`/`total` instead of `limit`.

### Changed

- Embedded Python bundle uses python-build-standalone `install_only_stripped` assets, shared post-install pruning (`scripts/prune-python-bundle.sh`), and a root `.dockerignore` (smaller release artifacts and faster Linux Docker builds).
- Post-build smoke tests (`make smoke-test`, `scripts/smoke-test-cli.sh`) verify `--version` and `--help` for every CLI command after embed and stub builds.

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
