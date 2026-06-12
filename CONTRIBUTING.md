# How to Contribute

Thanks for your interest in contributing to `ai-defense-cli`! Here are a few
general guidelines on contributing and reporting bugs that we ask you to review.
Following these guidelines helps to communicate that you respect the time of the
contributors managing and developing this open source project. In return, they
should reciprocate that respect in addressing your issue, assessing changes, and
helping you finalize your pull requests. In that spirit of mutual respect, we
endeavor to review incoming issues and pull requests within 10 days, and will
close any lingering issues or pull requests after 60 days of inactivity.

Please note that all of your interactions in the project are subject to our
[Code of Conduct](/CODE_OF_CONDUCT.md). This includes creation of issues or pull
requests, commenting on issues or pull requests, and extends to all interactions
in any real-time space e.g., Slack, Discord, etc.

## Reporting Issues

Before reporting a new issue, please ensure that the issue was not already
reported or fixed by searching through our [issues
list](https://github.com/cisco-ai-defense/ai-defense-cli/issues).

When creating a new issue, please be sure to include a **title and clear
description**, as much relevant information as possible, and, if possible, a
test case. For CLI bugs, include:

- CLI version (`aid-cisco-cli version`)
- OS and architecture
- Command and flags used (redact secrets)
- Exit code and relevant output

**If you discover a security bug, please do not report it through GitHub.
Instead, please see security procedures in [SECURITY.md](/SECURITY.md).**

## Sending Pull Requests

Before sending a new pull request, take a look at existing pull requests and
issues to see if the proposed change or fix has been discussed in the past, or
if the change was already implemented but not yet released.

We expect new pull requests to include tests for any affected behavior, and, as
we follow semantic versioning, we may reserve breaking changes until the next
major version release.

### Merge requirements (`main`)

Pull requests must pass all **required** CI checks before merge:

| Check | What it validates |
|-------|-------------------|
| `test` | `make build-go test vet` |
| `linux-amd64` | Full embed release binary (Docker) + `--version` smoke test |
| `darwin-arm64` / `darwin-amd64` | Full embed release binary (macOS) + `--version` smoke test |

The `checksums` job runs only on **push to `main` or `release/*`** (when artifacts are
published). It does not run on pull requests.

**Branch protection (repo admins):** Required PR checks should be exactly the four jobs
above (`test` + three platform embed builds). Do **not** require `meta` (removed),
`linux-arm64` (not published), or `checksums` (main/release push only).

**CI artifacts** (versioned binaries + `checksums.txt` + `manifest.json`) are uploaded
only from `main` and `release/*` pushes, not from pull requests. PRs still run full
embed builds to catch breaking changes.

### Versioning and releases

- **Source of truth:** [`VERSION`](VERSION) at repo root (semver, e.g. `0.1.0`).
- **Local builds:** `make version` prints `VERSION` and `COMMIT`; embedded via `-ldflags`.
- **Release binaries:** `dist/aid-cisco-cli-<version>-<os>-<arch>`.
- **Day to day:** feature/fix PRs SHOULD add bullets under `[Unreleased]` in [`CHANGELOG.md`](CHANGELOG.md).

#### Standard release from `main`

1. **Prepare:** GitHub → **Actions** → **Prepare release** → Run workflow.
   - **bump:** `patch` / `minor` / `major`, or set **custom_version** (e.g. `1.0.0`)
   - **ref:** branch to prepare from (default `main`; use `release/0.1.x` for patch lines)
   - Autofills missing changelog bullets, bumps `VERSION`, moves `[Unreleased]` → dated section.
   - Opens PR `release/vX.Y.Z` → **`ref`** (same branch you prepared from).
2. **Review + merge** the prepare PR.
3. **Publish:** **Release on merge** pushes `vX.Y.Z` automatically → **Release** workflow publishes binaries and GitHub Release assets.

#### Release line (`release/0.1.x`) and cherry-picks

Use a long-lived `release/0.1.x` branch when stabilizing or patching while `main` continues:

1. Cut `release/0.1.x` from a freeze commit.
2. Cherry-pick PRs target the release branch; **include `[Unreleased]` on that branch** in each CP PR.
3. **Prepare release** with `ref: release/0.1.x` — bot PR merges into the release line.
4. Merge → **Release on merge** auto-tags (same as `main`).
5. **Backport** code and CHANGELOG entries to `main` after shipping so histories stay aligned.

Fixes merged to `main` first are **not** auto-copied to a release line — CP the commit and changelog hunk.

#### Escape hatch

- **Create release tag** — dry-run validation, tag a specific SHA without a prepare PR, or emergency retag.
- **Local:** `./scripts/autofill-changelog.py --ref main`, then
  `./scripts/prepare-release.sh patch|minor|major [custom_version]`, then manual tag.

Repo admins should restrict **Prepare release** and **Create release tag**
(`workflow_dispatch`) to trusted roles under **Settings → Actions → General → Workflow permissions**.

**PR Persona Review** is advisory (optional); it does not block merge.

Branch protection on `main` enforces these checks. Repo admins can adjust them under
**Settings → Branches → Branch protection rules**.

### Development setup

```bash
git clone https://github.com/cisco-ai-defense/ai-defense-cli.git
cd ai-defense-cli
make build-go test vet
```

**Build modes:**

- `make build-go` — Go only; fast iteration and unit tests (embedded Python stub)
- `make bundle && make build-embed` — Go + relocatable CPython + `cisco-aibom` (release binary)

For local AIBOM work without a full embed, use `make build-go` and set `AID_CLI_PYTHON` to a Python with `cisco-aibom[all]` installed.

See [README.md](/README.md), `Makefile`, and `docs/` for architecture and release docs.

## Other Ways to Contribute

We welcome anyone that wants to contribute to `ai-defense-cli` to triage and
reply to open issues to help troubleshoot and fix existing bugs. Here is what
you can do:

- Help ensure that existing issues follow the recommendations from the
  _[Reporting Issues](#reporting-issues)_ section.
- Review and improve documentation under `docs/` (install, upgrade, getting started).
- Review existing pull requests and test patches against real CI/CD workflows.
- Write or extend tests in `*_test.go` for CLI commands, exit codes, and SDK clients.

Thanks again for your interest in contributing to `ai-defense-cli`!
