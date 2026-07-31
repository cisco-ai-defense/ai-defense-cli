# Build scripts

## Python bundle (`build-python-bundle.sh`)

Produces embed artifacts for the full CLI build:

```text
internal/embedded/python/bin/aid-py-bundle.tar.gz
internal/embedded/python/bin/aid-py-bundle.id
```

Invoked by `make bundle`. Uses [python-build-standalone](https://github.com/astral-sh/python-build-standalone) and `requirements-bundle.txt`. Post-install cleanup runs via `prune-python-bundle.sh` (also used in the Dockerfile `py-bundler` stage).

**Local cache:** `make bundle` skips the full rebuild when inputs are unchanged (`requirements-bundle.txt`, `prune-python-bundle.sh`, PBS pins). Cached PBS tarballs live under `.build/cache/`; pip wheels under `.build/pip-cache/`. Force rebuild with `make bundle-force` or `BUNDLE_FORCE=1 make bundle`.

The tagged Release workflow runs `make bundle-force` on each target-native runner.
`verify-python-bundle-target.sh` confirms the embedded interpreter architecture
and content id before the secret-bearing GoReleaser compile step.

The legacy/local Linux build target (`make build-release-linux`) uses a root
`.dockerignore` so local bundle artifacts are not sent as Docker build context.

## Post-build smoke tests

| Target | Binary | Use |
|--------|--------|-----|
| `make smoke-test` | `bin/cisco-aid-cli` (after `build-go`) | Local / CI: all `--help` paths |
| `make smoke-test-bin BIN=dist/…` | Release or embed binary | Secret-free workflow step after GoReleaser, or after `build-release-*` |
| `SMOKE_RUNNER=docker` | Linux binary on macOS | `make smoke-test-bin BIN=dist/…-linux-amd64 SMOKE_RUNNER=docker` |

Command paths are defined once in `internal/cli/smoke.go` and shared with `go test` (`TestSmokeHelpAllCommands`).

## Makefile build modes

| Target | Embedded Python | Use case |
|--------|-----------------|----------|
| `make build-go` | No (stub) | Fast dev/CI tests; set `AID_CLI_PYTHON` for local aibom |
| `make build-embed` | Yes | Release binaries; run `make bundle` first |
| `make build BUILD_MODE=full` | Yes | Alias for embed path |

```bash
make build-go test vet          # Go-only
make bundle && make build-embed # Full CLI with cisco-aibom
```

## Release builds

| Target | Output |
|--------|--------|
| `make build-release-linux` | `dist/cisco-aid-cli-linux-$(GOARCH)` via Docker |
| `make build-release-darwin` | `dist/cisco-aid-cli-darwin-$(GOARCH)` native on macOS |
| `make checksums` | `dist/checksums.txt` |

## Release automation

| Script | Role |
|--------|------|
| `autofill-changelog.py` | Add missing `[Unreleased]` bullets from merged PRs since last `v*` tag |
| `prepare-release.sh` | Bump `VERSION` and finalize CHANGELOG section |
| `extract-changelog-section.py` | Extract `## [VERSION]` body for GitHub Release notes |
| `validate-release-tag.sh` | Shared validation before pushing `v*` tags |
| `generate-release-manifest.sh` | Build `manifest.json` for legacy/local release artifacts |
| `release-manifest.json.tmpl` | Render the compatibility manifest from GoReleaser's merged artifacts |
| `verify-python-bundle-target.sh` | Reject an embedded CPython bundle for the wrong target |
| `verify-goreleaser-release.py` | Verify exact checksum and manifest contracts after publication; failures require operator remediation and rerun |

GoReleaser Pro owns tagged release compilation, raw-binary packaging, checksums,
manifest rendering, and GitHub Release publication. Three target-filtered
`release --split` jobs run on Linux amd64, macOS amd64, and macOS arm64; the
publisher restores their immutable partials and runs `continue --merge` without
rebuilding. Release runs require the masked repository secret `GORELEASER_KEY`;
configuration checks do not require a key and remain safe for fork pull requests.

```bash
python3 scripts/autofill-changelog.py --ref main --dry-run
./scripts/prepare-release.sh patch
# Requires the CI-pinned GoReleaser Pro v2.17.1; `check` does not need a key.
make goreleaser-check
```
