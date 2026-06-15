# CLI Specifications

Spec-driven development for ai-defense-cli. Update specs **before** code changes.

| ID | Area | Status |
|----|------|--------|
| [001-cli-core](001-cli-core/) | Commands, exit codes, config, auth | Active |
| 002-cli-sdk | HTTP client (`api/`) | Planned |
| 003-cli-release | Build, versioning, distribution | Active |
| [004-bundle-size](004-bundle-size/) | Embedded Python bundle size | Active |

Each spec directory should contain:

- `design.md` — structure, data flow, contracts
- `requirements.md` — numbered requirements (REQ-XX)

See `CONTEXT.md` and `.cursor/skills/spec-driven-dev/SKILL.md` for the workflow.
