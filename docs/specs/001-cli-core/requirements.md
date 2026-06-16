# Spec 001 — Requirements

## CLI core

| ID | Requirement |
|----|-------------|
| REQ-01 | Binary name is `cisco-aid-cli` |
| REQ-02 | Command groups: `model-scan`, `mcp-scan`, `ai-bom` |
| REQ-03 | Exit codes 0, 1–5 (operational), 10/20 (policy) per `internal/clierror` |
| REQ-04 | Scan commands emit raw API JSON by default; `--envelope` for verdict (`envelope_version`) |
| REQ-05 | `--fail-on` / `--block-on` severity gating on scan commands |
| REQ-06 | Config precedence: flags > env (`AI_DEFENSE_*`) > TOML |
| REQ-07 | Release builds embed CPython + `cisco-aibom[all]` (`embed_python` tag) |
| REQ-08 | `api/` is importable; must not depend on `internal/cli` |
| REQ-09 | `model-scan run` is removed; invoking it exits **2** with a message to use `model-scan trigger`. Unknown `model-scan` subcommands exit **2**. |

## Documentation

| ID | Requirement |
|----|-------------|
| REQ-D01 | User install path documented in `install-upgrade.md` |
| REQ-D02 | Developer build path in `getting-started-developer.md` |
| REQ-D03 | Command tree in `service-spec.md` |
| REQ-D04 | Per-command exit scenarios in `docs/exit-codes-handbook.md` (client handbook); summary table in spec 001 `design.md` |
| REQ-D05 | Handbook generated from `internal/cliexitdoc/registry.go` via `make gen-exit-docs`; CI `make check-exit-docs` |
