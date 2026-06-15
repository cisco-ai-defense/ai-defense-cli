# 005 — CLI install and version changes

## Requirements

### REQ-01 — Bootstrap from GitHub Releases

- **REQ-01.1** First install remains manual download from [GitHub Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases): pick OS/arch binary, verify `checksums.txt`, install to `PATH`.
- **REQ-01.2** No installer artifact (bash script or separate installer binary) is published.
- **REQ-01.3** Document macOS quarantine clearing for unsigned release binaries.

### REQ-02 — `aid-cisco-cli install` command

- **REQ-02.1** New top-level command `install` replaces ad-hoc manual upgrade/downgrade after bootstrap.
- **REQ-02.2** Flags:
  - `--version` — target version (default `latest`; accepts `X.Y.Z` without `v` prefix)
  - `--check` — report available update and exit without modifying the binary
  - `--prefix` — install directory (default: directory containing the running binary)
- **REQ-02.3** No API key required; command uses GitHub Releases only.
- **REQ-02.4** Exit codes: `0` success, `1` unexpected, `2` usage (unsupported platform, bad version), `4` network/download failure.

### REQ-03 — Self-update implementation

- **REQ-03.1** Package `internal/selfupdate` detects platform (`linux-amd64`, `darwin-amd64`, `darwin-arm64`) and maps to release artifact names `aid-cisco-cli-<version>-<os>-<arch>`.
- **REQ-03.2** Resolve release assets from `https://github.com/cisco-ai-defense/ai-defense-cli/releases` (latest or pinned tag).
- **REQ-03.3** Mandatory SHA-256 verification using `checksums.txt` from the same release before replacing the binary.
- **REQ-03.4** Atomic replace: download to temp file in target directory, chmod executable, rename over `aid-cisco-cli`.
- **REQ-03.5** On macOS (`darwin`), run `xattr -d com.apple.quarantine` on the new binary when possible (best-effort; ignore failure).

### REQ-04 — Documentation

- **REQ-04.1** Update `docs/install-upgrade.md` with `aid-cisco-cli install` for upgrades/downgrades/pinning.
- **REQ-04.2** Register command in `docs/service-spec.md` command tree.
