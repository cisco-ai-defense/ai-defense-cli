# 006 — CLI config TUI and onboarding

## Requirements

### REQ-01 — Config file location

- **REQ-01.1** Default config path: `~/.ai-defense/config.toml` (directory mode `0700`, file `0600`).
- **REQ-01.2** Value precedence (lowest last): flags → env → TOML file at resolved path.
- **REQ-01.3** File resolution: `AI_DEFENSE_CONFIG_PATH` (bypasses profiles) → active profile path → legacy cwd `.config.toml` → legacy XDG (legacy only when profile is `default`). See REQ-08.
- **REQ-01.4** Wizard always writes to `ConfigPathForWrite()` (`AI_DEFENSE_CONFIG_PATH` or default home path). Legacy files are read for resume/migration only.
- **REQ-01.5** If a config file exists, `onboard` / `config init` prompts continue vs start over (no multi-file picker).

### REQ-02 — `cisco-aid-cli config` command group

| Subcommand | Behavior |
|------------|----------|
| `config init` | Onboarding wizard; create file (prompt if exists unless `--force`) |
| `config edit` | TTY: section menu (core API, each capability configure/edit/remove, Done). Non-TTY: exit `2` with manual instructions |
| `config show` | TTY: interactive browser (overview, per-section, plain TOML option). Non-TTY or `--plain`: redacted TOML on stdout |
| `config path` | Print active config file path |
| `config validate` | Required default section fields + `.cisco.com` base URL rule |
| `config profile list` | List profiles; mark active with `*` |
| `config profile current` | Print active profile name and config path |
| `config profile use <name>` | Set active profile |
| `config profile create <name>` | Run onboarding wizard for a profile (`--force` to replace) |
| `config profile delete <name>` | Remove a named profile file (not `default`) |

- Non-TTY wizard commands: exit `2` with manual TOML/env instructions
- Interactive wizard Ctrl+C / cancel: exit `1` with message `Cancelled.` (not exit `2`; usage is reserved for invalid flags/args)
- Config write mode `0600`
- `config show --plain` forces redacted TOML even on a terminal

### REQ-03 — `cisco-aid-cli onboard`

- Wraps `config init` with welcome copy and personalized next steps (public docs URL, not repo-relative paths)
- Saves **draft progress after each wizard step** (each core field, each capability field)
- Persists progress in `[onboard].step` (see REQ-07)
- On re-run: prompt continue vs start over; resume skips completed steps
- **REQ-03.1** Unless `--profile` is set, wizard opens with a profile picker (default pre-selected to stored active profile, usually `default`). Overview lists existing profiles and marks the active one (`*`).
- **REQ-03.2** User may select an existing profile, create a new named profile, or pass `--profile` to skip the picker.

### REQ-04 — Modular capability hub

- **REQ-04.1** Core (always): `api_key`, `base_url` (preset or custom)
- **REQ-04.2** Optional capabilities presented **one at a time** (v1):
  - `huggingface` — private HF repos → `[default.huggingface]`
  - `aibom` — local `ai-bom analyze` → `[default.aibom]`
  - Each step: Configure or Skip for now
- **REQ-04.3** Unselected capabilities omitted from TOML (not empty sections)
- **REQ-04.4** Extensible `Capability` interface for future modules

### REQ-05 — Runtime wiring

- AIBOM LLM: flags → env → TOML `[default.aibom]`
- HuggingFace: existing `GetRepoToken()` for model-scan trigger
- Contextual hints on missing config (onboard / config edit)

### REQ-06 — Dependencies and schema

- Charm `huh` for TUI (pinned in go.mod)
- Integration schema in embedded `internal/config/spec/config.spec.yaml` (user config remains TOML)

### REQ-07 — Onboarding progress metadata

- `[onboard].step` records last completed step: `api_key`, `base_url`, capability ids (or `capability.field` mid-collection), `complete`
- Legacy values `core` and `location` treated as completed core / not started respectively
- Enables resume without re-prompting completed steps

### REQ-08 — Config profiles

- **REQ-08.1** Named profiles stored under `~/.ai-defense/profiles/<name>.toml`; the `default` profile uses `~/.ai-defense/config.toml` (backward compatible).
- **REQ-08.2** Active profile persisted in `~/.ai-defense/profile` (single line, mode `0600`); missing file means `default`.
- **REQ-08.3** Profile selection (highest first): `AI_DEFENSE_CONFIG_PATH` (bypasses profiles) → `--profile` flag → `AI_DEFENSE_PROFILE` → active profile file → `default`.
- **REQ-08.4** Global `--profile` flag on all commands; `config profile` subcommands: `list`, `current`, `use`, `create`, `delete`.
- **REQ-08.5** `onboard`, `config init`, and `config edit` accept `--profile` to target a named profile; `create` runs the wizard and sets the profile active.
- **REQ-08.6** Profile names: `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`; reserved name `default` maps to the root config file.
- **REQ-08.7** Legacy path fallbacks apply only when the active profile is `default`.
