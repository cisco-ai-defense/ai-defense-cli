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
