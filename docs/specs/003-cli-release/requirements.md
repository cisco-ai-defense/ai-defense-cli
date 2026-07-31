# 003 — CLI release process

## Requirements

### REQ-01 — Hybrid changelog autofill

- **REQ-01.1** During **Prepare release**, before VERSION bump, autofill missing `[Unreleased]` bullets from merged PRs/commits since the latest `v*` tag on `origin`.
- **REQ-01.2** Scope autofill to merges into the workflow `ref` branch (e.g. `main` or `release/0.1.x`), not all repository activity. When a prior `v*` tag exists, use `git log ${tag}..HEAD` merges only (not unbounded `gh pr list` history).
- **REQ-01.3** Map entries to Keep a Changelog sections (`Added`, `Changed`, `Fixed`) using PR title prefixes and labels; skip `chore(release):`, Dependabot-only, and `[skip changelog]`.
- **REQ-01.4** Do not duplicate bullets already present under `[Unreleased]` (normalized dedup).
- **REQ-01.5** If `[Unreleased]` is empty after autofill, **Prepare release** MUST fail with a clear error.

### REQ-02 — Prepare release workflow

- **REQ-02.1** `workflow_dispatch` inputs: `bump` (patch|minor|major), optional `custom_version`, `ref` (default `main`).
- **REQ-02.2** Bump `VERSION`, move `[Unreleased]` → `[X.Y.Z] - date`, leave empty `[Unreleased]`.
- **REQ-02.3** Push branch `release/vX.Y.Z` with VERSION/CHANGELOG commit; print compare URL in workflow summary. Maintainer opens PR with **base = `ref`** manually.

### REQ-03 — Auto-tag on merge

- **REQ-03.1** When a `release/v*` PR merges into `main` **or** a long-lived `release/*` line (e.g. `release/0.1.x`), push annotated tag `v{VERSION}` at the merge commit.
- **REQ-03.2** `VERSION` file on merge commit MUST match `release/v{VERSION}` head branch name.
- **REQ-03.3** Tag MUST NOT already exist on `origin`. Shared validation with **Create release tag**.
- **REQ-03.4** After tag push, explicitly dispatch **Release** (`workflow_dispatch`) — tags pushed by `GITHUB_TOKEN` do not trigger `on: push: tags` workflows.
- **REQ-03.5** Auto-tag MUST NOT run for pull requests from repository forks.

### REQ-04 — Create release tag (escape hatch)

- **REQ-04.1** Manual `workflow_dispatch` for dry-run validation, tagging a specific SHA, or emergency retag without a prepare PR.
- **REQ-04.2** Target commit MUST be on `origin/main` or `origin/release/*` history.

### REQ-05 — Release-line maintenance

- **REQ-05.1** Long-lived `release/0.1.x` branches use the same prepare → merge → auto-tag flow as `main`.
- **REQ-05.2** Cherry-pick PRs to a release line MUST include `[Unreleased]` on that branch; autofill is a safety net only.
- **REQ-05.3** After shipping from a release line, maintainers backport code and CHANGELOG entries to `main` manually.

### REQ-06 — Contributor contract

- **REQ-06.1** Feature/fix PRs SHOULD update `[Unreleased]` under Keep a Changelog headings.
- **REQ-06.2** Repo admins restrict **Prepare release** and **Create release tag** to trusted roles.

### REQ-07 — Post-build smoke tests

- **REQ-07.1** After `build-go` and each embed release build, CI MUST run smoke tests: `--version` and `--help` for every command path in `internal/cli.SmokeCommandPaths()`.
- **REQ-07.2** `scripts/smoke-test-cli.sh` is the release-binary entry point; command paths MUST come from `scripts/list-smoke-commands` (same list as unit tests).
- **REQ-07.3** `make smoke-test` (stub build) and `make smoke-test-bin BIN=…` MUST be available for local verification; `SMOKE_RUNNER=docker` supports Linux binaries built on non-Linux hosts.

### REQ-08 — Go test coverage gate

- **REQ-08.1** CI `test` job MUST run `make test-cover-check` after `build-go` (with `vet`).
- **REQ-08.2** Default minimum statement coverage (`COVER_MIN`) is **75%** over packages in `GO_COVER_PKGS` (see `Makefile`). `make test` still runs `./...`; exclusions apply only to coverage measurement.
- **REQ-08.3** Excluded from the gate until they have unit tests: `scripts/`, root `main`, `cmd/`, `internal/testutil/`, and TUI/onboarding packages (`internal/tui`, `internal/selfupdate`, `internal/cli/config`, `internal/cli/install`, `internal/cli/onboard`, `internal/config/spec`).
- **REQ-08.4** `scripts/check-coverage.sh` parses `coverage.out` from `make test-cover` and fails when total is below `COVER_MIN`.

### REQ-09 — PR diff coverage gate

- **REQ-09.1** On `pull_request` events, CI MUST run `make test-diff-cover-check` after `test-cover-check`, comparing `HEAD` to `merge-base(HEAD, base branch)`.
- **REQ-09.2** Default minimum diff coverage (`DIFF_COVER_MIN`) is **75%** of statements on changed production Go lines in scoped packages (same exclusions as REQ-08.3, plus `*_test.go`).
- **REQ-09.3** `cmd/check-diff-coverage` reads `coverage.out` and the git diff; when no coverable production lines changed, the check MUST pass with a skip message.
- **REQ-09.4** Checkout in CI MUST use `fetch-depth: 0` so merge-base is available on PRs.

### REQ-10 — GoReleaser release pipeline

- **REQ-10.1** CI MUST validate `.goreleaser.yaml` with the repository-pinned
  GoReleaser Pro version before release publication can be merged. Configuration
  validation MUST NOT require or receive `GORELEASER_KEY`, so it remains safe on
  fork pull requests.
- **REQ-10.2** GoReleaser MUST build and publish exactly:
  `cisco-aid-cli-<version>-linux-amd64`,
  `cisco-aid-cli-<version>-darwin-amd64`,
  `cisco-aid-cli-<version>-darwin-arm64`, `checksums.txt`, and `manifest.json`.
  Binaries MUST use GoReleaser's raw `binary` archive format with the exact name
  template; source archives, default tar/zip archives, and GoReleaser metadata
  assets MUST NOT be published.
- **REQ-10.3** GoReleaser MUST use the existing curated
  `CHANGELOG.md` version section as release notes; it MUST NOT silently replace it
  with a commit-derived changelog.
- **REQ-10.4** Each binary MUST be compiled by GoReleaser on its target operating
  system with the target-native embedded CPython and `cisco-aibom` bundle. The
  bundle architecture MUST be verified before compilation, and the post-build
  smoke test MUST execute an embedded analyzer help path. Bundle preparation and
  smoke tests MUST run outside the step that receives `GORELEASER_KEY`.
- **REQ-10.5** The workflow MUST pin the GoReleaser binary to an immutable exact
  version and grant `contents: write` only to the merge/publish job. Split build
  and validation jobs retain `contents: read`.
- **REQ-10.6** The release workflow MUST use GoReleaser Pro split/merge with
  `partial.by: target`; the only additional credential is the repository Actions
  secret `GORELEASER_KEY`. Every GoReleaser Action invocation MUST specify
  `distribution: goreleaser-pro`; release invocations MUST receive the secret,
  while pull-request configuration validation MUST NOT. A trusted preflight job
  MUST validate the pinned version, configuration, and license before the
  platform matrix starts.
- **REQ-10.7** Split outputs MUST be transferred to the merge job as immutable
  workflow artifacts. The merge job MUST publish those exact tested outputs
  without rebuilding them. Every split MUST set an exact `GGOOS` and `GGOARCH`,
  and partial artifact identity MUST bind the tag, commit, and target. Split and
  merge checkouts MUST use full Git history (`fetch-depth: 0`).
- **REQ-10.8** GoReleaser MUST generate the SHA-256 `checksums.txt` and render
  `manifest.json` from its merged artifact graph. An automated post-publication
  parity check MUST verify the exact five-asset inventory, one checksum entry per
  final binary basename, and the unchanged manifest fields (`name`, `version`,
  `commit`, `built_at`, and per-artifact `file`, `os`, `arch`, `sha256`).
- **REQ-10.9** A failed merge/publish job MUST be recoverable with GitHub's
  same-run **Re-run failed jobs**, reusing successful immutable partials. A new
  `workflow_dispatch` MUST rebuild every partial from the requested immutable tag
  and MUST NOT import partials from another workflow run.
- **REQ-10.10** GoReleaser Pro license failure MUST stop in preflight before
  expensive platform builds and MUST NOT print the key. Release Engineering owns
  entitlement renewal and replacement of the masked `GORELEASER_KEY` repository
  secret.
