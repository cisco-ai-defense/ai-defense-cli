# 006 — CLI config design

## UX flow

```mermaid
flowchart TD
  entry[onboard or config init/edit]
  required[Required section: location api_key base_url]
  hub[Capability multi-select]
  modules[Selected module forms]
  summary[Redacted summary]
  write[Write TOML 0600]

  entry --> required --> hub --> modules --> summary --> write
```

## TOML shape

```toml
[default]
api_key = "..."
base_url = "https://us.api.aidefense.security.cisco.com"

[default.huggingface]  # optional
token = "..."

[default.aibom]  # optional
llm_model = "..."
llm_provider = ""
llm_api_key = "..."
llm_api_base = ""
llm_api_version = ""
```

## Spec-driven config

The CLI config **schema** is declared in [`config.spec.yaml`](../../../internal/config/spec/config.spec.yaml).
The `default` section's `fields` mirror TOML: scalars (`api_key`, `base_url`) and `kind: group`
entries (`huggingface`, `aibom`) for nested tables. User secrets are written to **`config.toml`**.
The YAML spec is embedded at build time and drives:

- Onboarding wizard (`CollectWizard`, `CollectRequiredSection`)
- Runtime resolution (`Resolve`: flags → env → TOML)
- Secret redaction (`SecretTomlPaths`)
- Capability hub registration (`optional: true` on group fields)

Example **user config** output shape: [`docs/config.sample.toml`](../../../docs/config.sample.toml).

### Adding an optional capability

1. Add a `kind: group` field under `default.fields` in `config.spec.yaml` with `optional: true`
2. For complex validation, register a named validator in `validators.go`
3. Rebuild — wizard, redaction, resolution, and capability hub pick it up via `spec.OptionalSections()` / `spec.ByID(id)`

Commands that consume an integration at runtime look it up by id (e.g. `spec.MustByID("aibom")` in `internal/cli/aibom/`). No changes to `load.go` or per-integration package vars.

Simple validation uses declarative rules:

```yaml
validate:
  - type: non_empty
    field: token
    message: huggingface token is required
```

## Package layout

```
internal/config/spec/       declarative section specs (config.spec.yaml)
internal/config/            paths, read/write, redact via spec
internal/cli/config/
  wizard.go
  capabilities/
    registry.go
    spec_capability.go
internal/cli/onboard/
```

## Capability interface

```go
type Capability interface {
    ID() string
    Label() string
    Description() string
    IsConfigured(m *config.Manager) bool
    Collect(ctx *WizardContext) error
    Validate(ctx *WizardContext) error
    Apply(ctx *WizardContext, data map[string]any)
    NextSteps() []string
    MissingHint() string
}
```

## Base URL field

`base_url` is a field on the required `default` section with wizard `options` (regional endpoints) and
`allow_custom: true` so users can pick a preset or enter a custom `*.cisco.com` URL.
Only `base_url` is stored in TOML — see `config.spec.yaml` under `default.fields`.
