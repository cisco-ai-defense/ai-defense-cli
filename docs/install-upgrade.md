# Install and Upgrade

`aid-cisco-cli` is distributed as **per-platform binaries** with embedded Python for
local AIBOM analysis. There is no separate container image or Python install required
for release builds.

## Supported platforms

| OS | Architecture | Artifact name |
|----|--------------|---------------|
| Linux | amd64 | `aid-cisco-cli-linux-amd64` |
| Linux | arm64 | `aid-cisco-cli-linux-arm64` |
| macOS | arm64 (Apple Silicon) | `aid-cisco-cli-darwin-arm64` |
| macOS | amd64 (Intel) | `aid-cisco-cli-darwin-amd64` |

## Install from GitHub Release

1. Open [Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases) and pick a version.
2. Download the binary for your OS/arch and `checksums.txt`.
3. Verify the checksum:

   ```bash
   shasum -a 256 -c checksums.txt
   ```

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

## Install from PR CI artifacts (testing)

While a release is pending, contributors can download binaries from a pull request:

1. Open the PR → **Checks** → **CI** workflow run.
2. Download artifacts: `aid-cisco-cli-linux-amd64`, `aid-cisco-cli-darwin-arm64`, etc.
3. Artifacts expire after **7 days**; they are not permanent distribution channels.

## Upgrade

1. Download the new release binary for your platform.
2. Verify `checksums.txt`.
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
