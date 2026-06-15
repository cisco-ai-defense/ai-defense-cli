# 004 — Linux bundle size optimization

## Requirements

### REQ-01 — Stripped python-build-standalone (PBS) assets

- **REQ-01.1** Linux and macOS embed builds MUST download PBS `install_only_stripped` tarballs (not `install_only`) for release 20250723 / cpython 3.12.11.
- **REQ-01.2** Stripped assets MUST be verified with pinned SHA256 before extraction (Dockerfile build args, Makefile linux release pins, `scripts/build-python-bundle.sh` defaults).
- **REQ-01.3** Asset URL pattern: `cpython-${PYTHON_VERSION}+${PBS_RELEASE}-${PBS_TRIPLE}-install_only_stripped.tar.gz`.

### REQ-02 — Post-install bundle pruning parity

- **REQ-02.1** After `pip install`, both the Dockerfile py-bundler stage and `scripts/build-python-bundle.sh` MUST invoke `scripts/prune-python-bundle.sh` on the staged Python tree.
- **REQ-02.2** The prune script MUST remove `__pycache__`, `*.pyc`/`*.pyo`, install-time packages (`pip`, `setuptools`, `wheel`), and `tests`/`test`/`testing` directories under `site-packages`.
- **REQ-02.3** On Linux, the prune script MUST run `strip --strip-unneeded` on `*.so` native extensions (best-effort).

### REQ-03 — Docker build context

- **REQ-03.1** A root `.dockerignore` MUST exclude local embed artifacts (`internal/embedded/python/bin/`), `.build/`, `dist/`, and `bin/` so `build-release-linux` does not upload unused bundle staging to the daemon.

### REQ-04 — Documentation and traceability

- **REQ-04.1** Spec `004-bundle-size` documents rationale (stripped PBS size reduction on Linux) and pin sources (PBS GitHub release SHA256SUMS or computed checksums).
- **REQ-04.2** `[Unreleased]` CHANGELOG entry notes smaller embedded Python bundle for release builds.
