# 005 — CLI install design

## Command tree

```
aid-cisco-cli install [--version VERSION] [--check] [--prefix DIR]
```

## Data flow

```mermaid
flowchart LR
  installCmd[install command]
  selfupdate[internal/selfupdate]
  ghReleases[GitHub Releases API]
  checksums[checksums.txt]
  binary[aid-cisco-cli binary]

  installCmd --> selfupdate
  selfupdate --> ghReleases
  selfupdate --> checksums
  selfupdate --> binary
```

## Platform matrix

| GOOS/GOARCH | Artifact suffix |
|-------------|-----------------|
| linux/amd64 | linux-amd64 |
| darwin/arm64 | darwin-arm64 |
| darwin/amd64 | darwin-amd64 |

Unsupported combinations return usage error (exit 2).

## Release resolution

1. `latest` → `GET /repos/cisco-ai-defense/ai-defense-cli/releases/latest`
2. Pinned → `GET /repos/.../releases/tags/vX.Y.Z`
3. Download `checksums.txt` and platform binary from release assets
4. Verify checksum line for artifact filename
5. Write to `<prefix>/aid-cisco-cli` (or `--prefix` + basename)

## Security

- HTTPS only to `github.com` / `api.github.com`
- Checksum verification mandatory; abort on mismatch
- No credentials in install path
