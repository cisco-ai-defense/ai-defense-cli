# AI Defense CLI — Demo hands-on guide

**Audience:** New to `aid-cisco-cli` (may know legacy `aidcli`)  
**Focus:** Distribution + config management (not scan commands)  
**Suggested time:** ~12–15 minutes  
**Repo:** [github.com/cisco-ai-defense/ai-defense-cli](https://github.com/cisco-ai-defense/ai-defense-cli)  
**Version on screen:** adjust release tag / `--version` to match your binary (e.g. `0.1.3`)

This guide walks through two adoption paths: **getting the CLI onto a machine** and **managing configuration** (credentials, endpoints, profiles). Scan commands (`model-scan`, `mcp-scan`, `ai-bom`) are out of scope here.

---

## Prerequisites

- [ ] Terminal full-screen; font size readable
- [ ] Browser tab open on [GitHub Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases)
- [ ] API key ready (paste from password manager — do not leave visible on screen)
- [ ] For config steps: backup existing config or use a throwaway profile name (e.g. `demo-staging`)
- [ ] If already installed: confirm binary is on PATH (`aid-cisco-cli --version`)

---

## Part 1 — Distribution

### What you are demonstrating

- Versioned release binaries from Cisco's public GitHub repo
- Checksum verification before install
- Self-update via `aid-cisco-cli install` after first bootstrap

- [ ] **Step 1 — Pick the right artifact**
  Open the [Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases) page. Each release ships three platform binaries:

  | OS    | Architecture          | Artifact suffix |
  | ----- | --------------------- | --------------- |
  | Linux | amd64                 | `linux-amd64`   |
  | macOS | arm64 (Apple Silicon) | `darwin-arm64`  |
  | macOS | amd64 (Intel)         | `darwin-amd64`  |

  **Important:** Match OS **and** CPU. An arm64 binary will not run on Intel Macs, and vice versa.
  Linux **arm64** (e.g. AWS Graviton) is not published yet. Use the amd64 binary in a container or build from source.
  Each release also includes:
  - `checksums.txt` — SHA-256 for every artifact
  - `manifest.json` — version, commit, per-arch checksums (for automation/audit)
- [ ] **Step 2 — First install from Releases**
  - [ ] Download the binary for your platform and `checksums.txt` from the release
  - [ ] Verify the checksum (replace version and platform suffix):
    ```bash
    grep aid-cisco-cli-0.1.3-darwin-arm64 checksums.txt | shasum -a 256 -c
    ```
    Expected output: `OK`
  - [ ] Install on PATH:
    ```bash
    chmod +x aid-cisco-cli-0.1.3-darwin-arm64
    sudo mv aid-cisco-cli-0.1.3-darwin-arm64 /usr/local/bin/aid-cisco-cli
    ```
  - [ ] Confirm the install:
    ```bash
    aid-cisco-cli --version
    aid-cisco-cli --help
    ```
  **macOS only:** Release builds are not Apple-notarized yet. If Gatekeeper blocks the first run:
  ```bash
  xattr -d com.apple.quarantine ./aid-cisco-cli-0.1.3-darwin-arm64
  ```
  If still blocked: **System Settings → Privacy & Security → Open Anyway**, then re-run.
  See [install-upgrade.md](../install-upgrade.md#macos-install) for full macOS steps.
- [ ] **Step 3 — Check for updates**
  After bootstrap, upgrades stay in the CLI:
  ```bash
  aid-cisco-cli install --check
  ```
  **Expected behavior:** Compares the running version to the latest release on GitHub. Does **not** print a full version catalog — use the Releases page or `--version` to pin older builds.
- [ ] **Step 4 — Upgrade in place**
  - [ ] Run upgrade (latest):
    ```bash
    aid-cisco-cli install
    ```
  - [ ] Or pin a specific version:
    ```bash
    aid-cisco-cli install --version 0.1.2
    ```
  - [ ] Confirm after upgrade:
    ```bash
    aid-cisco-cli --version
    ```
  Optional — install to a custom location:
  ```bash
  aid-cisco-cli install --prefix /usr/local/bin
  ```
  **Note:** The running process keeps the old build until you start a new invocation. Default behavior installs **latest**.
- [ ] **Step 5 (optional) — Release pipeline**
  Releases are cut via the **Prepare release** workflow: version bump, changelog, tag on merge, CI publishes artifacts. PRs **build** all platforms in CI but do **not** publish PR artifacts as official downloads.

---

## Part 2 — Config management

### What you are demonstrating

- Centralized config under `~/.ai-defense/` (same idea as `~/.aws`)
- Interactive first-run wizard (`onboard`)
- Day-2 view / edit / validate commands
- Named profiles for multiple environments
- Consistent precedence: **flags → env → TOML**

- [ ] **Step 1 — Run first-time setup**
  ```bash
  aid-cisco-cli onboard
  ```
  Walk through the wizard:
  - [ ] **Profile** — pick default or create a new one (or pass `--profile <name>` to skip the picker)
  - [ ] **Core** — API key and base URL (regional presets available)
  - [ ] **Optional capabilities** — HuggingFace token, AIBOM LLM settings, etc. (Configure or Skip each)
  **While running, point out:**
  - Progress saves after each step — safe to resume if interrupted
  - Legacy `aidcli` paths (project `.config.toml`, old XDG location) are read for migration when saving to the new layout
  - Config file is written with mode `0600`
  - Secrets are not printed back in plain text in normal views
- [ ] **Step 2 — Inspect config location and validity**
  - [ ] Show active config path:
    ```bash
    aid-cisco-cli config path
    ```
    Expected: path to active config file (e.g. `~/.ai-defense/config.toml`).
  - [ ] Validate config:
    ```bash
    aid-cisco-cli config validate
    ```
    Expected: success if required fields and base URL rules are met.
  - [ ] Browse config (TUI):
    ```bash
    aid-cisco-cli config show
    ```
    Expected: interactive browser — values are **redacted**.
  - [ ] Show redacted TOML (scripts / CI):
    ```bash
    aid-cisco-cli config show --plain
    ```
    Expected: redacted TOML printed to stdout.
- [ ] **Step 3 — Edit existing settings**
  ```bash
  aid-cisco-cli config edit
  ```
  Expected: section menu — core API, each capability, option to remove optional modules. Requires a real terminal (TTY).
- [ ] **Step 4 — Work with profiles**
  - [ ] List profiles:
    ```bash
    aid-cisco-cli config profile list
    ```
    Expected: profile names; `*` marks the active profile.
  - [ ] Create a staging profile (runs onboarding wizard):
    ```bash
    aid-cisco-cli config profile create staging
    ```
  - [ ] Switch active profile:
    ```bash
    aid-cisco-cli config profile use staging
    ```
  - [ ] Run against a specific profile without switching:
    ```bash
    aid-cisco-cli --profile default config show --plain
    ```
  - [ ] Delete a demo profile (not `default`):
    ```bash
    aid-cisco-cli config profile delete demo-staging
    ```
  Profiles live as separate files under `~/.ai-defense/profiles/`. Select via `--profile`, `AI_DEFENSE_PROFILE`, or `config profile use`.
- [ ] **Step 5 — Config precedence and CI**
  Resolution order for settings like `api_key` and `base_url`:
  1. CLI flags (`--api-key`, `--base-url`, …)
  2. Environment variables (`AI_DEFENSE_API_KEY`, `AI_DEFENSE_BASE_URL`, …)
  3. TOML config file
  **CI setup** (no wizard):
  ```bash
  export AI_DEFENSE_API_KEY="<key>"
  export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"
  ```
  Or mount a TOML file and set:
  ```bash
  export AI_DEFENSE_CONFIG_PATH="/path/to/config.toml"
  ```
  Interactive commands (`onboard`, `config edit`, `config show` without `--plain`) require a TTY — that is intentional.

---

## Wrap-up

- [ ] Recap distribution: three platform binaries, checksums, `manifest.json`, self-update via `install`
- [ ] Recap config: `~/.ai-defense/`, `onboard` wizard, profiles, validate/show/edit commands
- [ ] Mention known gaps: macOS not notarized; `install --check` has no full version catalog; no `linux-arm64` release binary
- [ ] Point to docs: [install-upgrade.md](../install-upgrade.md), [config.md](../config.md), [getting-started-client.md](../getting-started-client.md)

---

## Troubleshooting


| Issue                             | Fallback                                                                                         |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `install` network error           | Show Releases page; download and verify manually                                                 |
| Gatekeeper blocks binary (macOS)  | Use `xattr -d com.apple.quarantine` or **Open Anyway**; or demo with an already-installed binary |
| `onboard` — config already exists | Use `config edit` or `config profile create demo2`                                               |
| No TTY / demo terminal odd        | Use `config show --plain` + env vars instead of wizard                                           |
| Wrong arch binary                 | Download the correct artifact from Releases                                                      |


---

## Command cheat sheet

```bash
# Distribution
aid-cisco-cli --version
aid-cisco-cli install --check
aid-cisco-cli install
aid-cisco-cli install --version <semver>
aid-cisco-cli install --prefix /usr/local/bin

# Config
aid-cisco-cli onboard
aid-cisco-cli config path
aid-cisco-cli config validate
aid-cisco-cli config show
aid-cisco-cli config show --plain
aid-cisco-cli config edit
aid-cisco-cli config profile list
aid-cisco-cli config profile create <name>
aid-cisco-cli config profile use <name>
aid-cisco-cli config profile delete <name>
aid-cisco-cli --profile <name> <command>
```

