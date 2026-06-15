# CLI Specifications

Spec-driven development for ai-defense-cli. Update specs **before** code changes.

| ID | Area | Status |
|----|------|--------|
| [001-cli-core](001-cli-core/) | Commands, exit codes, config, auth | Active |
| 002-cli-sdk | HTTP client (`api/`) | Deferred — covered by `api/` and 001 for now |
| [003-cli-release](003-cli-release/) | Build, versioning, distribution | Active |
| [004-bundle-size](004-bundle-size/) | Embedded Python bundle size | Active |
| [005-cli-install](005-cli-install/) | Install command, self-update | Active |
| [006-cli-config](006-cli-config/) | Config TUI, onboard, capabilities | Active |

Each spec directory should contain:

- `design.md` — structure, data flow, contracts
- `requirements.md` — numbered requirements (REQ-XX)

See `CONTEXT.md` and `.cursor/skills/spec-driven-dev/SKILL.md` for the workflow.
