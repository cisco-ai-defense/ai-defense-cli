# internal/config/spec

Embedded **configuration schema** for the CLI. Like OpenAPI for HTTP APIs, `config.spec.yaml` defines the contract for fields, env vars, flags, onboarding wizard, and validation.

User secrets are **not** stored here — they live in user `config.toml` (see [docs/config.sample.toml](../../../docs/config.sample.toml)).

## Key files

| File | Role |
|------|------|
| `config.spec.yaml` | Schema source (embedded at build time) |
| `load.go` | Parse YAML, build section registry (`Sections`, `ByID`, `OptionalSections`) |
| `spec.go` | `SectionSpec` — resolve, apply, redact paths |
| `wizard.go` | TUI field collection from spec |
| `validators.go` | Named validators for complex rules (e.g. `aibom_llm`) |

## Schema shape

One root section (`default`) whose `fields` mirror TOML `[default]`:

- **Scalars** (`api_key`, `base_url`) — string/secret fields collected on every onboard
- **`kind: group`** (`huggingface`, `aibom`) — nested TOML tables; `optional: true` registers them in the capability hub

At runtime, group fields are expanded into `SectionSpec` entries (`ByID("huggingface")`, etc.) for resolve/apply/wizard — same API as before.

## Add a new optional capability

1. Add a `kind: group` field under `default.fields` in **`config.spec.yaml`** with `optional: true`.
2. For complex validation, register a handler in **`validators.go`** and reference `validator: my_validator` on the group.
3. Rebuild — wizard, capability hub, redaction, and `spec.OptionalSections()` pick it up automatically.

Commands look up groups by key:

```go
sec := spec.MustByID("aibom")
value := sec.ResolveField("llm_model", flagValue, mgr)
```

## Docs

- Design: [docs/specs/006-cli-config/design.md](../../../docs/specs/006-cli-config/design.md)
- Requirements: [docs/specs/006-cli-config/requirements.md](../../../docs/specs/006-cli-config/requirements.md)
