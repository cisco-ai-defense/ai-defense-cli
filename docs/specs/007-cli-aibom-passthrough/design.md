# 007 — `ai-bom` analyzer passthrough — design

See [requirements.md](requirements.md). Story:
[AIFW-27300](https://cisco-sbg.atlassian.net/browse/AIFW-27300).

## Architecture

`ai-bom` is a hybrid command group. Two backends sit behind one parent command:

```
ai-bom (parent — aid_cli renders this help only)
├── analyze     ─┐
├── report      │
├── watch       │ passthrough → embedded cisco-aibom
├── diff        │ (DisableFlagParsing; exec bundled interpreter)
├── benchmark   │
├── kb          │
├── cache       │
├── plugin      ─┘
├── list        ─┐
├── info        │
├── delete      │ AI Defense management API (Go SDK, api/aibom)
├── components  │
└── summary     ─┘
```

## Passthrough mechanism (`internal/cli/aibom/passthrough.go`)

`runAibom(aibomCmd, args, extraEnv)`:

1. Resolve the interpreter via `embeddedpython.Interpreter()` (honors
   `AID_CLI_PYTHON` for local dev).
2. Exec `python -c <bootstrap> <aibomCmd> <args…>` with stdio bound to the real
   terminal.
3. The bootstrap imports `aibom.cli` and runs `app(prog_name="cisco-aid-cli
   ai-bom")`; on a help request it brackets the analyzer help with `aid_cli`
   context panels (`AID_BOM_HELP_PREAMBLE` / `AID_BOM_HELP_EPILOGUE`) rendered as
   Rich panels so the framing matches the analyzer's own width/style.
4. `extraEnv` carries only `aid_cli`-derived values (credentials), appended after
   `os.Environ()`.

`runAibomPassthrough` mirrors the analyzer's exit code (`*exec.ExitError →
os.Exit(code)`) and returns non-exec failures (missing interpreter, spawn error)
to `aid_cli`'s error handler.

`newPassthroughCmd(cli, use, short, aibomCmd, buildEnv)` builds the thin Cobra
command (`DisableFlagParsing`, passthrough annotation). `buildEnv` is `nil` for
pure passthroughs and supplied for `analyze` (LLM env), `report` (upload env),
and `kb` (tenant-key bridge).

## Credential bridging

- **LLM (`analyze`)** — `aibomEnvInjection` reads the embedded `aibom` config
  section (`config.spec.yaml`), resolves env → TOML, and emits `AIBOM_LLM_*` for
  any var not already set in the environment. CLI flags pass through verbatim and
  win at the analyzer level.
- **Upload (`report upload`)** — `reportUploadEnv` fires only when the first
  positional is `upload`. It injects `AIBOM_POST_URL`
  (`<base-url>/api/ai-defense/v1/aibom/analysis`, built from `api.APIPrefix` /
  `api.DefaultAPIVersion` / `aibom.AIBomAnalysis`) and `AI_DEFENSE_API_KEY` only
  when absent from flags and env, deriving them from `cli.ResolveAuth()`.
- **kb build commands (`kb.go`)** — `kbCredentialEnv` fires only for
  `request`/`request-status`/`list-requests` and injects just the tenant key as
  `CISCO_AI_DEFENSE_API_KEY` (from `ConfigManager().GetAPIKey()`) when the user
  didn't pass `--api-key`/set the env. The kb API base and manifest URL are left
  native (the kb host family differs from the management base URL, so aid_cli
  does not guess it); the analyzer enforces anything still missing.

`hasFlag` / `firstNonFlag` are small arg scanners used to honor "user input
always wins" without parsing the analyzer's full flag grammar.

## Help ownership

The root command sets the panel help/usage funcs; the `ai-bom` parent inherits
them and lists the union from each child's `Short`. Passthrough children carry
the `PassthroughAnnotation`; `SmokeCommandPaths` skips annotated commands so the
in-process `--help` smoke test does not require the embedded interpreter.

## Trade-offs

- `DisableFlagParsing` means `aid_cli`'s global flags (`--api-key`, `--base-url`,
  `--profile`) are inert on passthrough subcommands. Credentials therefore come
  from env/config (and, for uploads, analyzer flags like `--post-url`). This is
  the cost of perfect passthrough fidelity and is documented in the user guide.
- The Go-SDK submit path (`CreateAnalysis` / `SubmitReportFile` /
  `createAnalysisRequestFromReport`) is removed from `api/aibom`; uploads go
  through the analyzer's `report upload`, which builds the same envelope (plus
  schema canonicalization and retries). The cloud-management methods
  (`ListBoms`/`GetBom`/`DeleteBom`/`ListBomComponents`/`GetBomSummary`) and the
  `AIBomAnalysis` route constant remain.
