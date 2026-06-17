# Configuration

How `cisco-aid-cli` stores and resolves settings.

## Quick start

```bash
cisco-aid-cli onboard          # interactive setup (recommended)
cisco-aid-cli config show      # TUI browser on a terminal (secrets redacted)
cisco-aid-cli config show --plain   # redacted TOML (scripts / CI)
cisco-aid-cli config validate  # check required fields
cisco-aid-cli config edit      # section menu to change settings
```

Non-interactive (CI): set environment variables or write a TOML file manually — see below.

## Where settings live

All settings are stored under **`~/.ai-defense/`** (like `~/.aws` or `~/.ssh`):

| Item | Path |
|------|------|
| Config file | `~/.ai-defense/config.toml` (default profile) |
| Named profiles | `~/.ai-defense/profiles/<name>.toml` |
| Active profile | `~/.ai-defense/profile` (which profile commands use) |

Override with `AI_DEFENSE_CONFIG_PATH` (absolute path recommended; a leading `~/` is expanded on macOS and Linux). That bypasses profiles entirely.

Select a profile with `--profile <name>`, `AI_DEFENSE_PROFILE`, or `cisco-aid-cli config profile use <name>`.

```bash
cisco-aid-cli config profile list
cisco-aid-cli config profile create work    # guided setup for profile "work"
cisco-aid-cli config profile use work
cisco-aid-cli --profile staging model-scan list
```

The CLI resolves the home directory with Go's `os.UserHomeDir()` (typically `$HOME` on macOS and Ubuntu). Tilde (`~`) in docs is shorthand only — file I/O always uses the expanded absolute path.

**Legacy paths** (`./.config.toml` in the working directory, or `~/.config/cisco-aid-cli/config.toml`) are still read for the **default** profile when `~/.ai-defense/config.toml` does not exist. Running `onboard` writes to the active profile under `~/.ai-defense/` and migrates settings on save.

## Resolution order

For core API settings (`api_key`, `base_url`) and integration fields (e.g. AIBOM LLM):

1. CLI flags (`--api-key`, `--base-url`, `--llm-model`, …)
2. Environment variables (`AI_DEFENSE_API_KEY`, `AIBOM_LLM_MODEL`, …)
3. TOML config file

Full search order for **which file** is loaded: [service-spec.md](service-spec.md#configuration-resolution).

## TOML format

Example file: [config.sample.toml](config.sample.toml).

```toml
[default]
api_key = "..."
base_url = "https://us.api.aidefense.security.cisco.com"

[default.huggingface]   # optional — private model repos
token = "hf_..."

[default.aibom]         # optional — local ai-bom analyze
llm_model = "gpt-5.4"
llm_api_key = "..."
```

Optional sections are omitted when not configured. Files are written with mode `0600`.

## Commands

| Command | Purpose |
|---------|---------|
| `onboard` | First-time guided setup; profile picker first (or `--profile` to skip) |
| `config init` | Create config (`--force`, `--profile`) |
| `config edit` | Interactive section menu (TTY); change core or capabilities (`--profile`) |
| `config show` | TTY: browse sections; else redacted TOML (`--plain` to force TOML) |
| `config path` | Print active config file path |
| `config validate` | Verify required section fields and base URL |
| `config profile list` | List profiles (`*` = stored active) |
| `config profile current` | Show active profile name and config path |
| `config profile use <name>` | Set the stored active profile |
| `config profile create <name>` | Run onboarding wizard for a profile |
| `config profile delete <name>` | Remove a named profile (not `default`) |

Global flag: `--profile <name>` on any command (env: `AI_DEFENSE_PROFILE`).

Non-TTY environments: wizard commands exit with code `2`; use env vars, TOML, or `config show --plain`.

## For contributors

| Topic | Location |
|-------|----------|
| Behavior spec (REQ-XX) | [specs/006-cli-config/](specs/006-cli-config/) |
| Code layout | [../internal/config/README.md](../internal/config/README.md) |
| Integration schema | [../internal/config/spec/README.md](../internal/config/spec/README.md) |

To add a new optional capability: edit `config.spec.yaml` and rebuild — see the spec README.
