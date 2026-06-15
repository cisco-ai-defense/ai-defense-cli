# 006 — CLI config TUI and onboarding

## Requirements

### REQ-01 — Config file location

- **REQ-01.1** Wizard prompts for save location: XDG `~/.config/aid-cisco-cli/config.toml` (recommended), project `./.config.toml`, or custom absolute path.
- **REQ-01.2** Extend read order (lowest priority last): flags → env → `AI_DEFENSE_CONFIG_PATH` → cwd `.config.toml` → XDG config if exists.

### REQ-02 — `aid-cisco-cli config` command group

| Subcommand | Behavior |
|------------|----------|
| `config init` | Core wizard + capability hub; create file (fail if exists unless `--force`) |
| `config edit` | Load existing; core + hub with configured capabilities pre-checked |
| `config show` | Resolved config with secrets redacted |
| `config path` | Print active or last-written config path |
| `config validate` | Required core fields + `.cisco.com` base URL rule |

- Non-TTY: exit `2` with manual TOML/env instructions
- Config write mode `0600`

### REQ-03 — `aid-cisco-cli onboard`

- Wraps config init with welcome copy and personalized next steps
- Idempotent when config already complete

### REQ-04 — Modular capability hub

- **REQ-04.1** Core (always): location, region → `base_url`, API key
- **REQ-04.2** Optional multi-select capabilities (v1):
  - `huggingface` — private HF repos → `[default.huggingface]`
  - `aibom` — local `ai-bom generate`/`run` → `[default.aibom]`
- **REQ-04.3** Unselected capabilities omitted from TOML (not empty sections)
- **REQ-04.4** Extensible `Capability` interface for future modules

### REQ-05 — Runtime wiring

- AIBOM LLM: flags → env → TOML `[default.aibom]`
- HuggingFace: existing `GetRepoToken()` for model-scan trigger
- Contextual hints on missing config (onboard / config edit)

### REQ-06 — Dependencies and schema

- Charm `huh` for TUI (pinned in go.mod)
- Integration schema in embedded `internal/config/spec/config.spec.yaml` (user config remains TOML)
