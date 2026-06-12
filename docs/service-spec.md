# aid-cisco-cli Service Specification

## Overview

`aid-cisco-cli` is a Go-based CLI for Cisco AI Defense. It wraps the AI Defense Management API and optionally embeds a relocatable CPython interpreter (with `cisco-aibom[all]`) for local AIBOM generation.

## Command Tree

```
aid-cisco-cli [global flags] <command> <subcommand> [flags]

Global Flags:
  --api-key string    API key (env: AI_DEFENSE_API_KEY)
  --base-url string   Base URL (env: AI_DEFENSE_BASE_URL)
  --version           Print version

Commands:
  model-scan          Scan AI/ML models for security vulnerabilities
    trigger           Submit a model file or repo URL for cloud scanning
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

  ai-bom              Manage AI Bills of Materials
    generate          Generate AIBOM locally (embedded analyzer)
    submit            Submit an existing AIBOM report file
    run               Generate + submit in one step
    list              List BOMs with filters
    info              Get BOM details by analysis ID
    delete            Delete a BOM
    components        List components in a BOM
    summary           Get aggregate BOM statistics
```

## Configuration Resolution

Priority (highest to lowest):

1. CLI flags (`--api-key`, `--base-url`)
2. Environment variables (`AI_DEFENSE_API_KEY`, `AI_DEFENSE_BASE_URL`)
3. TOML config file

Config file search order:
- `AI_DEFENSE_CONFIG_PATH` environment variable
- `.config.toml` in the current working directory

### Config File Format

```toml
[default]
api_key = "<api-key>"
base_url = "https://us.api.aidefense.security.cisco.com"

[default.huggingface]
token = "hf_xxx"
```

## Base URL Validation

The CLI validates that the base URL hostname ends with `.cisco.com`. Other domains are rejected.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | Error (auth failure, API error, validation error, etc.) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AI_DEFENSE_API_KEY` | API key for authentication |
| `AI_DEFENSE_BASE_URL` | Management API base URL |
| `AI_DEFENSE_CONFIG_PATH` | Path to TOML config file |
| `AI_DEFENSE_MCP_RETRY_COUNT` | Max poll iterations for MCP scan (default: 60) |
| `AI_DEFENSE_MCP_WAIT_TIME_SECS` | Seconds between MCP scan polls (default: 10) |
| `AI_DEFENSE_RETRY_COUNT_FOR_SCANNING` | Max poll iterations for model scan (default: 60) |
| `AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK` | Seconds between model scan polls (default: 10) |
| `AIBOM_LLM_MODEL` | Default for `--llm-model` (required for `ai-bom generate`/`run`) |
| `AIBOM_LLM_PROVIDER` | Default for `--llm-provider` |
| `AIBOM_LLM_API_KEY` | Default for `--llm-api-key`; passed to the analyzer via the environment |
| `AIBOM_LLM_API_BASE` | Default for `--llm-api-base` |
| `AIBOM_LLM_API_VERSION` | Default for `--llm-api-version` |
| `AID_CLI_PYTHON` | Dev override: path to a local Python with `cisco-aibom` installed (bypasses the embedded bundle) |

### Local AIBOM Generation (LLM Requirement)

`ai-bom generate` and `ai-bom run` analyze code locally with the embedded
`cisco-aibom` analyzer, whose `analyze` command requires an LLM to classify
scanner candidates. Therefore:

- `--llm-model` (or `AIBOM_LLM_MODEL`) is **required** — the CLI fails fast with a
  clear error if it is missing (matching the analyzer, which also rejects a run
  without a model).
- A resolvable credential for the model is required, and the CLI fails fast if
  none is found: `--llm-api-key` / `AIBOM_LLM_API_KEY` (OpenAI, Azure, Anthropic,
  Google), the provider's native credential chain (e.g. AWS credentials for
  Bedrock via `--llm-provider bedrock`), a local endpoint (`--llm-api-base`, e.g.
  Ollama), or an exported native key (e.g. `OPENAI_API_KEY`). The API key is
  forwarded to the analyzer through the environment, never via argv.
- The remaining `ai-bom` subcommands (`submit`, `list`, `info`, `delete`,
  `components`, `summary`) call the AI Defense API and require no LLM.

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

`aid-cisco-cli --version` prints the binary version and source commit embedded at
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

Produces a lightweight binary without the embedded AIBOM analyzer. Model-scan, MCP-scan, and AIBOM submit/list/info work; `ai-bom generate` and `ai-bom run` return a helpful error.

```bash
cd ai-defense-cli
go build -o aid-cisco-cli .
```

Optional version metadata at link time:

```bash
go build -o aid-cisco-cli \
  -ldflags="-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=dev \
            -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=$(git rev-parse --short HEAD)" \
  .
```

### 2. Go-only build with a local interpreter (AIBOM generate/run in dev)

Build the default (non-embedded) binary, then point it at a local Python that has cisco-aibom installed:

```bash
cd ai-defense-cli
go build -o aid-cisco-cli .

python3 -m venv .venv && . .venv/bin/activate
pip install 'cisco-aibom[all]'

# Point the CLI at that interpreter (it runs `python -m aibom analyze ...`)
export AID_CLI_PYTHON="$PWD/.venv/bin/python"
./aid-cisco-cli ai-bom generate --source ./my-app --llm-model gpt-5.4
```

See `scripts/README.md` for the bundle layout and the LLM requirement.

### 3. Full embedded build (Linux via Docker)

Exports a release binary to `dist/` (not a container image):

```bash
make build-release-linux GOARCH=amd64
# → dist/aid-cisco-cli-<version>-linux-amd64 (version from VERSION file)
```

Docker stages (`Dockerfile`):

1. **py-bundler** — relocatable CPython + `pip install cisco-aibom[all]`
2. **go-builder** — `go build -tags embed_python`
3. **release** — scratch stage; `docker build --target release --output dist/`

The CLI extracts the embedded bundle on first use and runs `python3 -m aibom analyze ...`.

### 4. Full embedded build (local Go, after building the bundle)

If you already have `internal/embedded/python/bin/aid-py-bundle.tar.gz` (and `.id`) from a bundle build:

```bash
cd ai-defense-cli
go build -tags embed_python -o aid-cisco-cli \
  -ldflags="-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=dev \
            -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=$(git rev-parse --short HEAD)" \
  .
```

### 5. CI / release builds

Pull request and `main` branch CI (`.github/workflows/ci.yml`) builds three release
platform binaries (`linux-amd64`, `darwin-arm64`, `darwin-amd64`) and uploads them
on `main` / `release/**` pushes. **Prepare release** opens a version-bump PR;
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
