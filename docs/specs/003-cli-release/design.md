# 003 — CLI release process (design)

## Workflows

| Workflow | Trigger | Outcome |
|----------|---------|---------|
| **Prepare release** | `workflow_dispatch` | Autofill → bump VERSION → finalize CHANGELOG → push `release/vX.Y.Z` |
| **Release on merge** | `release/v*` PR merged to `main` or `release/**` | Push `vX.Y.Z` tag |
| **Release** | Push tag `v*` | Embed builds, checksums, manifest, GitHub Release (title = tag; body = `CHANGELOG.md` `[VERSION]` section via `extract-changelog-section.py`) |
| **Create release tag** | `workflow_dispatch` | Manual/dry-run tag (escape hatch) |

```mermaid
sequenceDiagram
  participant M as Maintainer
  participant Prep as Prepare_release
  participant PR as release_vX_PR
  participant Tag as Release_on_merge
  participant Rel as Release

  M->>Prep: dispatch ref plus bump
  Prep->>Prep: autofill_changelog.py
  Prep->>Prep: prepare-release.sh
  Prep->>Prep: push release/vX.Y.Z
  M->>PR: open PR manually base equals ref
  PR->>Tag: pull_request closed merged
  Tag->>Rel: push vX.Y.Z
```

## Scripts

| Script | Role |
|--------|------|
| [`scripts/autofill-changelog.py`](../../../scripts/autofill-changelog.py) | Fill missing `[Unreleased]` bullets from merged PRs / git since last tag |
| [`scripts/prepare-release.sh`](../../../scripts/prepare-release.sh) | Semver bump + finalize CHANGELOG section |
| [`scripts/validate-release-tag.sh`](../../../scripts/validate-release-tag.sh) | Shared tag validation for create-release and release-on-merge |
| [`scripts/check-coverage.sh`](../../../scripts/check-coverage.sh) | Enforce `COVER_MIN` against `coverage.out` from `make test-cover` |

## CI test job

The `test` workflow job runs `make build-go test-cover-check vet`. On pull requests it also runs `make test-diff-cover-check` against the merge base. Package scope and exclusions for the coverage gate are defined in `Makefile` (`GO_COVER_PKGS`, `COVER_MIN`, `DIFF_COVER_MIN`); see **REQ-08** and **REQ-09** in `requirements.md`.

## Autofill sources

1. `gh pr list --state merged --base {ref}` when `gh` and `GITHUB_TOKEN` are available.
2. Fallback: `git log {since_tag}..HEAD --merges --first-parent` parsing `#NNN` from merge subjects.

## Branch naming

| Pattern | Meaning |
|---------|---------|
| `release/v0.1.1` | Ephemeral prepare branch (VERSION/CHANGELOG bump); PR opened manually |
| `release/0.1.x` | Long-lived stabilization / patch line (merge base for prepare PRs) |

## Permissions

- **Prepare release:** `contents: write`
- **Release on merge:** `contents: write`
- Concurrency group `release-tag` prevents double tag push
