# 004 — Linux bundle size optimization (design)

## Problem

Embedded CLI builds bundle relocatable CPython from [python-build-standalone](https://github.com/astral-sh/python-build-standalone) (PBS). The `install_only` flavor includes debug symbols and extra runtime bulk; on Linux the compressed tarball is roughly **~101 MB** vs **~33 MB** for `install_only_stripped` (same release triple).

Smaller embed artifacts reduce Docker build time, release download size, and cold-start extraction cost.

## Approach

| Layer | Change |
|-------|--------|
| PBS download | Switch asset suffix from `install_only` → `install_only_stripped` |
| Integrity | Update pinned SHA256 per triple (20250723 / 3.12.11) |
| Pruning | Shared `scripts/prune-python-bundle.sh` after `pip install` (drop pip/setuptools/wheel, tests, strip `.so` on Linux) |
| Docker context | Root `.dockerignore` excludes local bundle staging from `COPY . .` |

## Touchpoints

| File | Role |
|------|------|
| [`.dockerignore`](../../../.dockerignore) | Exclude local embed artifacts from Docker build context |
| [`Dockerfile`](../../../Dockerfile) | `py-bundler` stage: stripped PBS + `prune-python-bundle.sh` |
| [`Makefile`](../../../Makefile) | `PBS_SHA256_amd64` / `PBS_SHA256_arm64` for `build-release-linux` |
| [`scripts/build-python-bundle.sh`](../../../scripts/build-python-bundle.sh) | Local/macOS bundle: stripped asset, SHA256 table, prune script |
| [`scripts/prune-python-bundle.sh`](../../../scripts/prune-python-bundle.sh) | Post-`pip install` cleanup shared by Docker and native bundle builds |

## PBS pins (cpython 3.12.11 + 20250723, install_only_stripped)

| Triple | SHA256 |
|--------|--------|
| `x86_64-unknown-linux-gnu` | `7263ed87e9d2d4c52b01f1ad8cdca315cf859c0848b8ffdcb70de205fa8b0336` |
| `aarch64-unknown-linux-gnu` | `32b17b62fd621481bb4eab0ed5549cbad43d88780e29fc2c0a222a1722470c4b` |
| `x86_64-apple-darwin` | `de7786cb936a0ae87ee8b89906ba7a5e02b2880d8665f7ba5d3ecff5f5e14d1f` |
| `aarch64-apple-darwin` | `ba814d8d611898aee3c6daed53a535a93345a6b4984ebe62f9aa38646255a5bf` |

## Verification

- `make build-release-linux` (or CI embed job) succeeds with SHA256 check.
- `scripts/build-python-bundle.sh` on developer host downloads stripped asset and produces `aid-py-bundle.tar.gz`.
- Compare bundle byte size before/after on linux-amd64 release artifact (expect material reduction from PBS layer).
