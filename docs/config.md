# Configuration

How `aid-cisco-cli` stores and resolves settings.

## Quick start

```bash
aid-cisco-cli onboard          # interactive setup (recommended)
aid-cisco-cli config show      # inspect config (secrets redacted)
aid-cisco-cli config validate  # check required fields
aid-cisco-cli config edit      # add capabilities later
```

Non-interactive (CI): set environment variables or write a TOML file manually — see below.

## Where settings live

The wizard can write to:

| Location | Path |
|----------|------|
| User config (recommended) | `~/.config/aid-cisco-cli/config.toml` |
| Project local | `./.config.toml` in the current directory |
| Custom | Any absolute path you choose |

Override the search path with `AI_DEFENSE_CONFIG_PATH`.

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

[default.aibom]         # optional — local ai-bom generate/run
llm_model = "gpt-5.4"
llm_api_key = "..."
```

Optional sections are omitted when not configured. Files are written with mode `0600`.

## Commands

| Command | Purpose |
|---------|---------|
| `onboard` | First-time guided setup |
| `config init` | Create config (`--force` to overwrite) |
| `config edit` | Change existing config |
| `config show` | Print resolved config (redacted) |
| `config path` | Print active config path |
| `config validate` | Verify required core fields and base URL |

Non-TTY environments: wizard commands exit with code `2`; use env vars or TOML instead.

## For contributors

| Topic | Location |
|-------|----------|
| Behavior spec (REQ-XX) | [specs/005-cli-config/](specs/005-cli-config/) |
| Code layout | [../internal/config/README.md](../internal/config/README.md) |
| Integration schema | [../internal/config/spec/README.md](../internal/config/spec/README.md) |

To add a new optional capability: edit `config.spec.yaml` and rebuild — see the spec README.
