# internal/config/spec

Embedded **configuration schema** for the CLI. Like OpenAPI for HTTP APIs, `config.spec.yaml` defines the contract for fields, env vars, flags, onboarding wizard, and validation.

User secrets are **not** stored here — they live in user `config.toml` (see [docs/config.sample.toml](../../../docs/config.sample.toml)).

## Key files

| File | Role |
|------|------|
| `config.spec.yaml` | Schema source (embedded at build time) |
| `load.go` | Parse YAML, build integration registry (`ByID`, `Integrations`) |
| `spec.go` | `IntegrationSpec` — resolve, apply, redact paths |
| `wizard.go` | TUI field collection from spec |
| `validators.go` | Named validators for complex rules (e.g. `aibom_llm`) |

**`core.fields`** — persisted TOML keys; optional `options` + `allow_custom` drive the wizard (e.g. `base_url` regional presets).

## Add a new integration

1. Add an entry under `integrations:` in **`config.spec.yaml`** (fields, env, flags, optional presets/validate).
2. For complex validation, register a handler in **`validators.go`** and reference it with `validator: my_validator` in YAML.
3. Rebuild — wizard, capability hub, redaction, and `spec.Integrations()` pick it up automatically.

If a **command** must read the integration at runtime, look it up by id in that command package:

```go
ig := spec.MustByID("my_integration")
value := ig.ResolveField("some_key", flagValue, mgr)
```

No changes to `load.go` or new package-level variables are required.

## Docs

- Design: [docs/specs/006-cli-config/design.md](../../../docs/specs/006-cli-config/design.md)
- Requirements: [docs/specs/006-cli-config/requirements.md](../../../docs/specs/006-cli-config/requirements.md)
