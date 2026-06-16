# Documentation index

Start here if you are not sure which doc to open.

## I want to…

| Goal | Document |
|------|----------|
| Install or upgrade the CLI | [install-upgrade.md](install-upgrade.md) |
| Set up credentials and config | [config.md](config.md) · [getting-started-client.md](getting-started-client.md) §0 |
| Run my first scan | [getting-started-client.md](getting-started-client.md) |
| Look up commands, flags, env vars | [service-spec.md](service-spec.md) |
| Understand exit codes and error scenarios | [exit-codes-handbook.md](exit-codes-handbook.md) |
| Build or hack on the CLI | [getting-started-developer.md](getting-started-developer.md) |
| Understand package layout | [architecture/ai-defense-cli.md](architecture/ai-defense-cli.md) |
| Change CLI behavior (spec-first) | [specs/README.md](specs/README.md) |
| See project / release status | [../CONTEXT.md](../CONTEXT.md) |

## Layout

```
docs/
├── README.md                 ← this file
├── config.md                 ← config overview (users + links for devs)
├── getting-started-client.md ← end-user guide
├── getting-started-developer.md
├── install-upgrade.md
├── service-spec.md           ← canonical command & config reference
├── exit-codes-handbook.md    ← per-command exit codes & error scenarios (client)
├── config.sample.toml        ← example user config (TOML)
├── architecture/             ← component map
├── specs/                    ← numbered behavior specs (REQ-XX)
└── adr/                      ← architectural decision records
```

## Canonical sources (avoid duplicating)

| Topic | Canonical doc |
|-------|----------------|
| Config resolution order | [service-spec.md](service-spec.md#configuration-resolution) |
| Exit codes | [exit-codes-handbook.md](exit-codes-handbook.md) (generated from `internal/cliexitdoc`); summary in [specs/001-cli-core/design.md](specs/001-cli-core/design.md) |
| User TOML shape | [config.sample.toml](config.sample.toml) |
| Config schema (YAML) | [../internal/config/spec/config.spec.yaml](../internal/config/spec/config.spec.yaml) |
| Release / install artifacts | [specs/005-cli-install/](specs/005-cli-install/) |

When updating docs, change the canonical file and add links elsewhere — do not copy tables into multiple places.
