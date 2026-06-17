# 007 — `ai-bom` analyzer passthrough integration

Story: [AIFW-27300](https://cisco-sbg.atlassian.net/browse/AIFW-27300) —
"Integrate aibom CLI into aid-cli" (epic AIFW-24111).

## Goal

Make `cisco-aid-cli ai-bom` a single, unified surface over **both** AIBOM
backends:

- the embedded `cisco-aibom` analyzer (local analysis and report tooling), and
- the AI Defense management API (cloud BOM inventory).

The CLI forwards analyzer subcommands verbatim (passthrough) so it never drifts
from the analyzer's flags or help, while keeping the Go-SDK commands that have no
analyzer equivalent. `aid_cli` owns only the top-level `ai-bom` help.

## Requirements

### REQ-01 — Command set is the union of both backends

- **REQ-01.1** Analyzer passthroughs (forwarded to embedded `cisco-aibom`):
  `analyze`, `report`, `watch`, `diff`, `benchmark`, `kb`, `cache`, `plugin`.
- **REQ-01.2** AI Defense management commands (Go SDK, direct API calls):
  `list`, `info`, `delete`, `components`, `summary`.
- **REQ-01.3** No subcommand name collisions between the two sets.

### REQ-02 — Passthrough fidelity

- **REQ-02.1** Passthrough commands set `DisableFlagParsing` so every flag and
  positional (including `-h`/`--help`) flows to the analyzer unchanged.
- **REQ-02.2** The analyzer runs under the embedded interpreter
  (`internal/embedded/python.Interpreter`); stdin/stdout/stderr are wired to the
  real terminal so Rich rendering and interactive prompts work.
- **REQ-02.3** The analyzer program name is overridden so usage reads
  `cisco-aid-cli ai-bom <cmd> …`, not `cisco-aibom …`.
- **REQ-02.4** The analyzer's exit code is propagated verbatim.
- **REQ-02.5** When the embedded interpreter is unavailable (Go-only build), the
  passthrough returns the interpreter's own actionable error.

### REQ-03 — Command renaming (analyzer terminology)

- **REQ-03.1** `generate` is removed; local analysis is `analyze` (passthrough).
- **REQ-03.2** `submit` and `run` are removed; upload is `report upload`
  (analyzer passthrough).

### REQ-04 — Credential bridging (user input always wins)

- **REQ-04.1** `analyze`: `[default.aibom]` LLM settings resolve (env → TOML) and
  are injected as `AIBOM_LLM_*` env vars only for keys the user has not already
  set, so config-stored credentials reach the analyzer.
- **REQ-04.2** `report upload`: when neither flag nor env supplies them, derive
  `AIBOM_POST_URL = <base-url>/api/ai-defense/v1/aibom/analysis` and
  `AI_DEFENSE_API_KEY` from resolved `aid_cli` auth.
- **REQ-04.3** Secrets are passed via the child environment, never on argv.
- **REQ-04.4** `kb` build commands (`request`/`request-status`/`list-requests`):
  inject the tenant key as `CISCO_AI_DEFENSE_API_KEY` from resolved config only
  when the user didn't pass `--api-key`/set the env. `CISCO_AI_DEFENSE_API_BASE`
  and `CISCO_AIBOM_MANIFEST_URL` stay native kb flags/env (the kb host family
  differs from the management base URL and has no AI Defense config source).

### REQ-05 — Help ownership

- **REQ-05.1** Only the top-level `ai-bom` help is rendered by `aid_cli` (the
  titled-panel renderer), listing the full union with one-line descriptions.
- **REQ-05.2** Each passthrough subcommand delegates its own `--help` to the
  analyzer.
- **REQ-05.3** The in-process smoke test (`--help` for every command) skips
  passthrough commands, since their help is produced by the analyzer (present
  only in embedded builds).

### REQ-06 — Quality gates

- **REQ-06.1** New Go meets the repo diff-coverage gate (75% of changed lines).
- **REQ-06.2** `make build-go`, `make vet`, and `make test` pass.
