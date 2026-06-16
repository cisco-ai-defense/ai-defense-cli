# Build scripts

## Python bundle (`build-python-bundle.sh`)

Produces embed artifacts for the full CLI build:

```text
internal/embedded/python/bin/aid-py-bundle.tar.gz
internal/embedded/python/bin/aid-py-bundle.id
```

Invoked by `make bundle`. Uses [python-build-standalone](https://github.com/astral-sh/python-build-standalone) and `requirements-bundle.txt`. Post-install cleanup runs via `prune-python-bundle.sh` (also used in the Dockerfile `py-bundler` stage).

**Local cache:** `make bundle` skips the full rebuild when inputs are unchanged (`requirements-bundle.txt`, `prune-python-bundle.sh`, PBS pins). Cached PBS tarballs live under `.build/cache/`; pip wheels under `.build/pip-cache/`. Force rebuild with `make bundle-force` or `BUNDLE_FORCE=1 make bundle`.

Linux release builds (`make build-release-linux`) use a root `.dockerignore` so local bundle artifacts are not sent as Docker build context.

## Post-build smoke tests

| Target | Binary | Use |
|--------|--------|-----|
| `make smoke-test` | `bin/cisco-aid-cli` (after `build-go`) | Local / CI: all `--help` paths |
| `make smoke-test-bin BIN=dist/…` | Release or embed binary | After `build-release-*` |
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
| `generate-release-manifest.sh` | Build `manifest.json` for release artifacts |

```bash
python3 scripts/autofill-changelog.py --ref main --dry-run
./scripts/prepare-release.sh patch
```
