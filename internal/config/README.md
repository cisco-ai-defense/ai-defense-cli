# internal/config

Runtime configuration: load user TOML, resolve paths, redact secrets.

## Owns

- Config file **search order**, **profiles**, and path helpers (`paths.go`, `profiles.go`)
- **TOML** read/write with mode `0600` (`config.go`)
- Core API key / base URL resolution (`GetAPIKey`, `GetBaseURL`, `ResolveAPIKeyAndBaseURL`)
- Secret **redaction** for `config show` (driven by `spec` secret paths)

## Does not own

| Concern | Package |
|---------|---------|
| Interactive wizard (TUI) | `internal/cli/config` |
| Schema (fields, env, flags, wizard) | `internal/config/spec` |
| Cobra commands | `internal/cli/onboard`, `internal/cli/config` |

## Read order

1. `paths.go` / `profiles.go` — `ResolveConfigPath`, `ActiveProfile`, `~/.ai-defense/`
2. `config.go` — `Manager`, write/redact, integration lookups via `spec.ByID`
3. `../config/spec/` — embedded `config.spec.yaml`

## User vs schema files

| File | Format | Purpose |
|------|--------|---------|
| User `config.toml` | TOML | Secrets and values on disk |
| `spec/config.spec.yaml` | YAML | Embedded schema (not user-edited) |

## Docs

- Users: [docs/config.md](../../docs/config.md)
- Contract: [docs/specs/006-cli-config/](../../docs/specs/006-cli-config/)
- Reference: [docs/service-spec.md](../../docs/service-spec.md)
