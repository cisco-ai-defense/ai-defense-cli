# Install and Upgrade

`aid-cisco-cli` is distributed as **per-platform binaries** with embedded Python for
local AIBOM analysis. There is no separate container image or Python install required
for release builds.

## Supported platforms

Release binaries are named `aid-cisco-cli-<version>-<os>-<arch>`. For version `0.1.0`:

| OS | Architecture | Artifact name |
|----|--------------|---------------|
| Linux | amd64 | `aid-cisco-cli-0.1.0-linux-amd64` |
| Linux | arm64 | `aid-cisco-cli-0.1.0-linux-arm64` |
| macOS | arm64 (Apple Silicon) | `aid-cisco-cli-0.1.0-darwin-arm64` |
| macOS | amd64 (Intel) | `aid-cisco-cli-0.1.0-darwin-amd64` |

Each release also ships `checksums.txt` and `manifest.json` (version, commit, per-arch SHA-256).

## Install from GitHub Release

1. Open [Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases) and pick a version.
2. Download the binary for your OS/arch and `checksums.txt`.
3. Verify the checksum for your binary only:

   ```bash
   grep aid-cisco-cli-0.1.0-linux-amd64 checksums.txt | shasum -a 256 -c
   ```

   Replace `0.1.0` and the platform suffix with your release version and OS/arch.

4. Install on your `PATH`:

   ```bash
   chmod +x aid-cisco-cli-linux-amd64
   sudo mv aid-cisco-cli-linux-amd64 /usr/local/bin/aid-cisco-cli
   ```

   On macOS, you may need to allow the binary in **System Settings → Privacy & Security**
   the first time you run an unsigned build.

5. Confirm:

   ```bash
   aid-cisco-cli --version
   aid-cisco-cli --help
   ```

## Install from CI artifacts (main branch)

After a merge to `main`, the **CI** workflow publishes embed binaries and
`checksums.txt` as workflow artifacts (7-day retention):

1. Open **Actions** → latest **CI** run on `main`.
2. Download the binary for your platform and `checksums`.
3. Verify with the same `grep … | shasum -a 256 -c` pattern above.

Pull requests run the same embed builds to verify they compile, but **do not**
publish downloadable artifacts (security: untrusted PR code must not be distributed
as official-looking binaries).

## Upgrade

1. Download the new release binary for your platform.
2. Verify the checksum line from `checksums.txt`.
3. Replace the existing binary on your `PATH`.
4. Run `aid-cisco-cli --version` to confirm the new build metadata.

To pin a version in CI, download a specific release URL and verify the checksum in your pipeline.

## Configuration

After install, set credentials (see [getting-started-client.md](getting-started-client.md)):

```bash
export AI_DEFENSE_API_KEY="<key>"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"
```

## Build from source

See [getting-started-developer.md](getting-started-developer.md).
