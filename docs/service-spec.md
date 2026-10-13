# cisco-aid-cli Service Specification

## Overview

`cisco-aid-cli` is a Go-based CLI for Cisco AI Defense. It wraps the AI Defense Management API and optionally embeds a relocatable CPython interpreter (with `cisco-aibom[all]`) for local AIBOM generation.

## Command Tree

```
cisco-aid-cli [global flags] <command> <subcommand> [flags]

Global Flags:
  --api-key string    API key (env: AI_DEFENSE_API_KEY)
  --base-url string   Base URL (env: AI_DEFENSE_BASE_URL)
  --profile string    Config profile (env: AI_DEFENSE_PROFILE)
  --version           Print version

Commands:
  install             Download and install a release build (self-update)
  onboard             Interactive setup for new users
  config              Manage CLI configuration
    init              Create config interactively
    edit              Edit existing config (TTY)
    show              Show config (TTY browser or redacted TOML)
    path              Print active config file path
    validate          Validate required fields
    profile           List, switch, create, and delete named profiles
  model-scan          Scan AI/ML models for security vulnerabilities
    trigger           Submit a model file or repo URL for cloud scanning
    run               Deprecated (usage error — use trigger)
    list              List previous scans
    info              Get scan details by ID

  mcp-scan            Manage MCP server security scans
    register          Register an MCP server for monitoring
    delete            Delete a registered MCP server
    run               Scan a remote MCP server (poll to completion)
    list              List registered MCP servers
    info              Get server or scan details by ID
    trigger           Trigger on-demand scan for registered server
    capabilities      List tools/prompts/resources on a server

  ai-bom              Generate, manage, and explore AI Bills of Materials
    analyze           Analyze sources and generate an AIBOM (analyzer passthrough)
    report            Show or upload a generated AIBOM report (analyzer passthrough)
    watch             Watch sources and re-scan on change (analyzer passthrough)
    diff              Compare two AIBOM reports (analyzer passthrough)
    benchmark         Compare scan output against ground truth (analyzer passthrough)
    kb                Manage the analyzer knowledge base (analyzer passthrough)
    cache             Manage the analyzer result cache (analyzer passthrough)
    plugin            Discover and manage analyzer plugins (analyzer passthrough)
    list              List BOMs with filters (AI Defense API)
    info              Get BOM details by analysis ID (AI Defense API)
    delete            Delete a BOM (AI Defense API)
    components        List components in a BOM (AI Defense API)
    summary           Get aggregate BOM statistics (AI Defense API)
```

The `analyze`/`report`/`watch`/`diff`/`benchmark`/`kb`/`cache`/`plugin`
subcommands are forwarded verbatim to the embedded `cisco-aibom` analyzer
(`DisableFlagParsing`), so their flags and `--help` are the analyzer's own. The
global `--api-key` / `--base-url` / `--profile` flags do not apply to them;
supply configuration via environment variables or the config file instead.

## Configuration Resolution

Priority (highest to lowest):

1. CLI flags (`--api-key`, `--base-url`)
2. Environment variables (`AI_DEFENSE_API_KEY`, `AI_DEFENSE_BASE_URL`)
3. TOML config file

Config file search order:
- `AI_DEFENSE_CONFIG_PATH` environment variable (if set; bypasses profiles)
- Active profile config (`~/.ai-defense/config.toml` for `default`, or `~/.ai-defense/profiles/<name>.toml`)
- Legacy (only when active profile is `default`): cwd `.config.toml`, then XDG path

Active profile: `~/.ai-defense/profile`, or `--profile` / `AI_DEFENSE_PROFILE`.

### Config File Format

```toml
[default]
api_key = "<api-key>"
base_url = "https://us.api.aidefense.security.cisco.com"

[default.huggingface]
token = "hf_xxx"

[default.aibom]
llm_model = "gpt-5.4"
llm_provider = ""
llm_api_key = "..."
llm_api_base = ""
llm_api_version = ""
```

Optional sections (`huggingface`, `aibom`) are omitted when not configured via
`cisco-aid-cli onboard` / `config init`. Wizard progress is stored under `[onboard]` (not read at runtime).

### Config commands (TTY)

| Command | TTY behavior |
|---------|----------------|
| `config show` | Section browser; `--plain` prints redacted TOML |
| `config edit` | Menu: core API, each capability, Done |
| `onboard` / `config init` | Step-by-step wizard with draft save per field (`--profile`) |
| `config profile` | `list`, `current`, `use`, `create`, `delete` |

## Base URL Validation

The CLI validates that the base URL hostname ends with `.cisco.com`. Other domains are rejected.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unexpected error |
| 2 | Usage error (bad flags/args; non-TTY wizard; unsupported install platform) |
| 3 | Auth/config error (missing API key, invalid base URL) |
| 4 | Connectivity/timeout (install download, API network) |
| 5 | Scan failed |
| 10 | Policy fail (`--fail-on`) |
| 20 | Policy block (`--block-on`) |

`install` and `config validate` follow the same mapping (see specs 005 and 006).
See [architecture/ai-defense-cli.md](architecture/ai-defense-cli.md) and [spec 001](specs/001-cli-core/design.md).

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AI_DEFENSE_API_KEY` | API key for authentication |
| `AI_DEFENSE_BASE_URL` | Management API base URL |
| `AI_DEFENSE_CONFIG_PATH` | Path to TOML config file |
| `AI_DEFENSE_MCP_RETRY_COUNT` | Max poll iterations for MCP scan (default: 60) |
| `AI_DEFENSE_MCP_WAIT_TIME_SECS` | Seconds between MCP scan polls (default: 10) |
| `AI_DEFENSE_RETRY_COUNT_FOR_SCANNING` | Max poll iterations for model scan. Unset (with the wait variable also unset): 1,500 polls at 10 s, a 15,000 s budget. Set: this count (default 60 when only the wait variable is set) |
| `AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK` | Seconds between model scan polls (default: 10). Setting either model-scan variable replaces the 15,000 s default with retry count × wait time |
| `AIBOM_LLM_MODEL` | Default for `--llm-model` (required for `ai-bom analyze`) |
| `AIBOM_LLM_PROVIDER` | Default for `--llm-provider` |
| `AIBOM_LLM_API_KEY` | Default for `--llm-api-key`; passed to the analyzer via the environment |
| `AIBOM_LLM_API_BASE` | Default for `--llm-api-base` |
| `AIBOM_LLM_API_VERSION` | Default for `--llm-api-version` |
| `AID_CLI_PYTHON` | Dev override: path to a local Python with `cisco-aibom` installed (bypasses the embedded bundle) |

### Local AIBOM Analysis (LLM Requirement)

`ai-bom analyze` runs the embedded `cisco-aibom` analyzer, which requires an LLM
to classify scanner candidates. Therefore:

- `--llm-model` (or `AIBOM_LLM_MODEL`, or `[default.aibom].llm_model` in config) is **required**
- A resolvable credential for the model is required: `--llm-api-key` /
  `AIBOM_LLM_API_KEY` (OpenAI, Azure, Anthropic, Google), the provider's native
  credential chain (e.g. AWS credentials for Bedrock via `--llm-provider
  bedrock`), a local endpoint (`--llm-api-base`, e.g. Ollama), or an exported
  native key (e.g. `OPENAI_API_KEY`). `[default.aibom]` settings are injected as
  `AIBOM_LLM_*` only when not already set in the environment; CLI flags pass
  through verbatim and always win. The analyzer enforces the requirement and
  reports a clear error when no model/credential is found.
- `ai-bom report upload <report.json>` uploads a generated report. The AI Defense
  endpoint (`AIBOM_POST_URL`) and API key (`AI_DEFENSE_API_KEY`) are derived from
  the resolved config when not supplied via flag or environment.
- `ai-bom kb request` / `request-status` / `list-requests` call the AI Defense API
  with the tenant key. aid_cli injects `CISCO_AI_DEFENSE_API_KEY` from config when
  not user-supplied; the kb API base (`CISCO_AI_DEFENSE_API_BASE`) and manifest URL
  (`CISCO_AIBOM_MANIFEST_URL`) stay native kb flags/env.
- The cloud inventory subcommands (`list`, `info`, `delete`, `components`,
  `summary`) call the AI Defense API and require no LLM.

## API Endpoints

The CLI communicates with the AI Defense Management API at:
`{base_url}/api/ai-defense/v1/`

### Supported Regions

| Region | Endpoint |
|--------|----------|
| us-west-2 | `https://us.api.aidefense.security.cisco.com` |
| eu-central-1 | `https://eu.api.aidefense.security.cisco.com` |
| ap-northeast-1 | `https://ap.api.aidefense.security.cisco.com` |
| me-central-1 | `https://uae.api.aidefense.security.cisco.com` |

## Version Metadata

`cisco-aid-cli --version` prints the binary version and source commit embedded at
build time. Release builds set these values through Go linker flags:

```bash
-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=<version>
-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=<commit>
```

Local or development builds that do not pass these linker flags report
`dev (unknown)`. The version output is local binary metadata only; it does not
query GitHub, S3, or the AI Defense API.

## Building the CLI

All commands below assume the working directory is the repository root unless noted.

### 1. Go-only build (API commands only)

Produces a lightweight binary without the embedded AIBOM analyzer. Model-scan, MCP-scan, and the AIBOM cloud commands (`list`/`info`/`components`/`summary`/`delete`) work; the analyzer passthroughs (`ai-bom analyze`, `report`, `watch`, etc.) return a helpful error.

```bash
cd ai-defense-cli
go build -o cisco-aid-cli .
```

Optional version metadata at link time:

```bash
go build -o cisco-aid-cli \
  -ldflags="-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=dev \
            -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=$(git rev-parse --short HEAD)" \
  .
```

### 2. Go-only build with a local interpreter (AIBOM analyzer passthroughs in dev)

Build the default (non-embedded) binary, then point it at a local Python that has cisco-aibom installed:

```bash
cd ai-defense-cli
go build -o cisco-aid-cli .

python3 -m venv .venv && . .venv/bin/activate
pip install 'cisco-aibom[all]'

# Point the CLI at that interpreter (it runs the embedded aibom analyzer)
export AID_CLI_PYTHON="$PWD/.venv/bin/python"
./cisco-aid-cli ai-bom analyze ./my-app --llm-model gpt-5.4 --output-format json --output-file aibom.json
```

See `scripts/README.md` for the bundle layout and the LLM requirement.

### 3. Full embedded build (Linux via Docker)

Exports a release binary to `dist/` (not a container image):

```bash
make build-release-linux GOARCH=amd64
# → dist/cisco-aid-cli-<version>-linux-amd64 (version from VERSION file)
```

Docker stages (`Dockerfile`):

1. **py-bundler** — relocatable CPython + `pip install cisco-aibom[all]`
2. **go-builder** — `go build -tags embed_python`
3. **release** — scratch stage; `docker build --target release --output dist/`

The CLI extracts the embedded bundle on first use and runs the selected analyzer
subcommand (`analyze`, `report`, `watch`, `diff`, `benchmark`, `kb`, `cache`,
`plugin`) through the embedded interpreter.

### 4. Full embedded build (local Go, after building the bundle)

If you already have `internal/embedded/python/bin/aid-py-bundle.tar.gz` (and `.id`) from a bundle build:

```bash
cd ai-defense-cli
go build -tags embed_python -o cisco-aid-cli \
  -ldflags="-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=dev \
            -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=$(git rev-parse --short HEAD)" \
  .
```

### 5. CI / release builds

Pull request and `main` branch CI (`.github/workflows/ci.yml`) builds three release
platform binaries (`linux-amd64`, `darwin-arm64`, `darwin-amd64`) and uploads them
on `main` / `release/**` pushes. **Prepare release** pushes a `release/v*` branch;
**Release on merge** auto-tags when that PR merges; **Release** (`release.yml`)
publishes GitHub Release assets on `v*` tag push.

```bash
make build-release-linux GOARCH=amd64
make build-release-darwin GOARCH=arm64   # macOS host only
make checksums
```

## Authentication Header

All API requests include:
```
X-Cisco-AI-Defense-Tenant-API-Key: <api-key>
Content-Type: application/json
Accept: application/json
```
