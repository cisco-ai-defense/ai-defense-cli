# Install and Upgrade

`cisco-aid-cli` is distributed as **per-platform binaries** with embedded Python for
local AIBOM analysis. There is no separate container image or Python install required
for release builds.

## Supported platforms

Release binaries are named `cisco-aid-cli-<version>-<os>-<arch>` (releases before
the rename used `aid-cisco-cli-<version>-<os>-<arch>`). For version `0.1.0`:

| OS | Architecture | Artifact name |
|----|--------------|---------------|
| Linux | amd64 | `cisco-aid-cli-0.1.0-linux-amd64` |
| macOS | arm64 (Apple Silicon) | `cisco-aid-cli-0.1.0-darwin-arm64` |
| macOS | amd64 (Intel) | `cisco-aid-cli-0.1.0-darwin-amd64` |

Linux **arm64** (e.g. AWS Graviton) is not published as a release binary. Use the
`linux-amd64` binary in a container (`platform: linux/amd64`), or
[build from source](getting-started-developer.md) on the ARM host.

> **macOS:** Release binaries are **unsigned and not notarized** (no Apple Developer ID).
> Gatekeeper blocks the first run until you clear quarantine or approve the binary — see
> [macOS install](#macos-install) below.

Each release also ships `checksums.txt` and `manifest.json` (version, commit, per-arch SHA-256).

Releases are staged as drafts when a **Prepare release** PR (`release/v*`) merges to
`main` or a `release/*` line — **Release on merge** pushes the final `v*` tag and starts
the Release workflow. Internal QA tests those exact draft assets, then an approved
**Promote draft release** run publishes the existing draft as the latest release.
Prepare release pushes the branch; open the PR manually from the workflow summary link.

## Internal QA of a draft release

Draft releases are visible to authenticated repository members with push access
(including Write access). In this public repository the `v*` tag and Actions logs are
public, but the draft release notes and assets are not listed for the general public.
Never place credentials or confidential information in release assets or workflow logs.

QA can use the signed-in GitHub Releases page or GitHub CLI:

```bash
gh auth status
gh release view v0.2.0 --repo cisco-ai-defense/ai-defense-cli
mkdir qa-v0.2.0
gh release download v0.2.0 \
  --repo cisco-ai-defense/ai-defense-cli \
  --dir qa-v0.2.0
```

From the download directory, verify the appropriate binary and run smoke tests:

```bash
cd qa-v0.2.0
grep cisco-aid-cli-0.2.0-darwin-arm64 checksums.txt | shasum -a 256 -c
chmod +x cisco-aid-cli-0.2.0-darwin-arm64
./cisco-aid-cli-0.2.0-darwin-arm64 --version
./cisco-aid-cli-0.2.0-darwin-arm64 --help
```

Record the tag, commit from `manifest.json`, platform, binary digest, and test result.
Do not replace draft assets during QA; any replacement requires the checks and approval
to be repeated.

After QA approval, a maintainer runs **Promote draft release** with the same stable tag
(for example `v0.2.0`). A read-only job validates the draft and expected assets before
the configured `production-release` environment requests approval. After approval, the
publishing job revalidates the draft and publishes it without a rebuild. Repository
administrators must configure required reviewers for that GitHub environment before
relying on it as the publication gate.

## Install from GitHub Release

1. Open [Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases) and pick a version.
2. Download the binary for your OS/arch and `checksums.txt`.
3. Verify the checksum for your binary only:

   ```bash
   grep cisco-aid-cli-0.1.0-linux-amd64 checksums.txt | shasum -a 256 -c
   ```

   Replace `0.1.0` and the platform suffix with your release version and OS/arch.

4. Install on your `PATH` (Linux example):

   ```bash
   chmod +x cisco-aid-cli-0.1.0-linux-amd64
   sudo mv cisco-aid-cli-0.1.0-linux-amd64 /usr/local/bin/cisco-aid-cli
   ```

   On macOS, follow [macOS install](#macos-install) instead.

5. Confirm:

   ```bash
   cisco-aid-cli --version
   cisco-aid-cli --help
   ```

## macOS install

> **Unsigned binary:** macOS release builds are not code-signed or notarized. This is
> expected until an org Apple Developer ID is available.

1. Download and verify the `darwin-*` binary (steps 1–3 above).
2. Clear Gatekeeper quarantine and install:

   ```bash
   xattr -d com.apple.quarantine ./cisco-aid-cli-0.1.0-darwin-arm64
   chmod +x ./cisco-aid-cli-0.1.0-darwin-arm64
   sudo mv ./cisco-aid-cli-0.1.0-darwin-arm64 /usr/local/bin/cisco-aid-cli
   ```

   Replace the filename with your version and architecture (`darwin-amd64` on Intel Macs).

3. If Gatekeeper still blocks the first run ("cannot verify … is free of malware"), use
   **System Settings → Privacy & Security → Open Anyway**, then re-run the command.

**Alternatives:** build from source on the Mac (no quarantine), or use the Linux binary
inside Docker/WSL for CI-style workflows.

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

After the initial install from GitHub Releases, change versions with the built-in
install command (checksum-verified download from Releases):

```bash
cisco-aid-cli install                  # latest release
cisco-aid-cli install --version 0.1.2  # pin a version
cisco-aid-cli install --check          # show available update without installing
cisco-aid-cli install --prefix /usr/local/bin
```

Supported platforms for self-update match release binaries: `linux-amd64`,
`darwin-amd64`, `darwin-arm64`.

By default, `install` replaces the binary in the same directory as the running
executable (or `--prefix`). On Linux and macOS the file is swapped atomically;
the running process keeps the old build until you start a new invocation. Run
`cisco-aid-cli --version` after upgrade to confirm.

Manual upgrade (download + replace) still works:

1. Download the new release binary for your platform.
2. Verify the checksum line from `checksums.txt`.
3. Replace the existing binary on your `PATH`.
4. Run `cisco-aid-cli --version` to confirm the new build metadata.

To pin a version in CI, download a specific release URL and verify the checksum in your pipeline.

## Configuration

After install, run the interactive setup wizard:

```bash
cisco-aid-cli onboard
```

See [config.md](config.md) for paths, resolution order, and manual TOML setup.

This writes a config file (core API credentials plus optional capabilities such as
private HuggingFace repos and local AIBOM LLM settings). You can also set credentials
manually — see [getting-started-client.md](getting-started-client.md):

```bash
export AI_DEFENSE_API_KEY="<key>"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"
```

## Build from source

See [getting-started-developer.md](getting-started-developer.md).
