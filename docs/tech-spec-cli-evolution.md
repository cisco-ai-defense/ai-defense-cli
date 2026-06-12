> **Historical document** — describes migration from Python `aidcli` and monorepo layout.
> For the current CLI, see [spec 001](specs/001-cli-core/design.md) and [architecture](architecture/ai-defense-cli.md).

# Tech Spec: CLI Evolution — From Python aidcli to Go aid_cli

| Field | Value |
|---|---|
| **Status** | Draft |
| **Author** | AI Defense Platform Team |
| **Created** | 2026-05-13 |
| **Jira Epic** | AIFW-23470 |
| **Repo** | `ai-firewall-services` |

---

## Table of Contents

- [1. Executive Summary](#1-executive-summary)
- [2. Background and Motivation](#2-background-and-motivation)
- [3. Problems with the Python CLI (aidcli)](#3-problems-with-the-python-cli-aidcli)
  - [3.1 Distribution and Packaging](#31-distribution-and-packaging)
  - [3.2 Runtime Fragility](#32-runtime-fragility)
  - [3.3 Dependency Surface](#33-dependency-surface)
  - [3.4 Build Pipeline Complexity](#34-build-pipeline-complexity)
  - [3.5 SDK Coupling and Version Drift](#35-sdk-coupling-and-version-drift)
  - [3.6 Performance and Startup](#36-performance-and-startup)
- [4. Solution: Go-based CLI (aid_cli)](#4-solution-go-based-cli-aid_cli)
  - [4.1 Architecture Overview](#41-architecture-overview)
  - [4.2 Command Parity](#42-command-parity)
  - [4.3 In-repo Go SDK (sdk/aidefense)](#43-in-repo-go-sdk-sdkaidefense)
  - [4.4 Python Embedding via a relocatable CPython bundle](#44-python-embedding-via-a-relocatable-cpython-bundle)
  - [4.5 Configuration and Auth](#45-configuration-and-auth)
  - [4.6 Build Pipeline Integration](#46-build-pipeline-integration)
  - [4.7 Backward Compatibility](#47-backward-compatibility)
- [5. How the Go CLI Improves the Experience](#5-how-the-go-cli-improves-the-experience)
- [6. Scaling Strategy](#6-scaling-strategy)
  - [6.1 Adding Pure Go Commands](#61-adding-pure-go-commands)
  - [6.2 Adding Python Submodules (Embedded)](#62-adding-python-submodules-embedded)
- [7. Future Work](#7-future-work)
  - [7.1 Go SDK for AI Defense Scans (External Package)](#71-go-sdk-for-ai-defense-scans-external-package)
  - [7.2 Plugin Architecture for Python Tools](#72-plugin-architecture-for-python-tools)
  - [7.3 Additional Enhancements](#73-additional-enhancements)
- [8. Migration and Deprecation Plan](#8-migration-and-deprecation-plan)
- [9. Decision Log](#9-decision-log)

---

## 1. Executive Summary

The Cisco AI Defense CLI (`aid-cisco-cli`) is the primary command-line tool for model scanning, MCP server scanning, and AI Bill of Materials (AIBOM) management. The original CLI was written in Python using the Click framework (`aidcli`). Due to distribution, runtime stability, dependency management, and performance challenges, we built a replacement Go CLI (`aid_cli`) using the Cobra framework.

The Go CLI achieves full command parity with `aidcli`, ships as a single statically-linked binary, eliminates the Python runtime dependency for API-based commands, and selectively embeds Python-based tooling (AIBOM analyzer) as a relocatable CPython interpreter bundle for local analysis workloads that have no Go equivalent. The Go CLI drives the analyzer by invoking its own CLI (`python -m aibom analyze`) as a subprocess, so no Python glue code is shipped.

This document covers the problems with the old approach, the architecture of the new CLI, how it scales, and planned future work including an external Go SDK for AI Defense scans.

---

## 2. Background and Motivation

The AI Defense platform provides three core CLI workflows:

| Workflow | Description |
|---|---|
| **Model Scanning** | Submit AI/ML model files or HuggingFace repo URLs for cloud-based security analysis |
| **MCP Server Scanning** | Register, scan, and monitor MCP (Model Context Protocol) servers for security threats |
| **AI BOM** | Generate, submit, and manage AI Bills of Materials for supply chain transparency |

These workflows were originally implemented in a Python CLI (`aidcli/`) that used the `cisco-aidefense-sdk` Python package. As adoption grew and deployment moved into CI pipelines, air-gapped environments, and customer-facing toolchains, the limitations of the Python approach became blocking.

---

## 3. Problems with the Python CLI (aidcli)

### 3.1 Distribution and Packaging

**Single-binary shipping requires Nuitka compilation of the entire CLI.**

The Python CLI has no `console_scripts` entry point in `pyproject.toml`. Distribution is exclusively via a Docker image or a Nuitka-compiled onefile binary. The Nuitka compilation:

- Takes 10-15 minutes on CI, compiling the entire Python runtime and all transitive dependencies into a single executable.
- Requires explicit `--include-package` flags for every dependency tree (HuggingFace hub, `hf_xet`, truststore, aibom, duckdb, litellm, rich, modelhawk, pandas, h5py, typer, etc.).
- Produces a ~200+ MB binary that self-extracts at runtime to a temp directory.
- Breaks silently if a new transitive dependency is added without updating the Nuitka include list.

**No `pip install` path for end users.** The private git dependencies (`modelhawk`, `datasource-connector` from `ai-common-py`) require internal repo access, making `poetry install` unusable outside the development team.

### 3.2 Runtime Fragility

The Python CLI experiences several runtime stability issues documented in `aidcli/docs/known-issues.md`:

| Issue | Impact | Mitigation |
|---|---|---|
| **`hf_xet` Rust teardown crash** | `Fatal Python error: _PyImport_Init` on interpreter shutdown when the HuggingFace XET extension's Rust runtime cleans up | Defensive `os._exit()` instead of clean shutdown; `HF_HUB_DISABLE_XET=1` env workaround |
| **TLS/truststore in Nuitka binary** | `truststore.inject_into_ssl()` cannot hook into macOS Keychain from the compiled binary | Requires `SSL_CERT_FILE` env pointing to a PEM bundle |
| **Corporate proxy / TLS inspection** | XET downloads fail behind TLS-inspecting proxies | Per-file HTTP retry fallback; manual CA bundle configuration |
| **Config path resolution** | Relative paths in `.config.toml` may not resolve correctly from within the Nuitka binary | Workaround: use absolute `AI_DEFENSE_CONFIG_PATH` |

The `main()` function wraps execution in defensive `try/except` with `os._exit()` specifically to avoid triggering the `hf_xet` crash during Python interpreter finalization — a fragility that would not exist in a compiled language.

### 3.3 Dependency Surface

The Python CLI has a large transitive dependency surface:

```
cisco-aidefense-sdk (pinned to git main — no stable release)
├── requests, aiohttp, pydantic, tenacity
├── cisco-aibom (optional)
│   ├── duckdb, pandas, litellm, typer, rich, h5py
│   └── ... (LLM provider SDKs via litellm)
├── huggingface_hub
│   └── hf_xet (Rust extension)
modelhawk (private, from ai-common-py)
datasource-connector (private, from ai-common-py)
dynaconf, nuitka, truststore, click
```

Key concerns:

- **`cisco-aidefense-sdk` is pinned to `rev = "main"`** — any upstream commit can break the CLI without warning.
- **Private git dependencies** (`modelhawk`, `datasource-connector`) prevent reproducible builds outside the organization's network.
- **Heavyweight ML/data stack** (pandas, duckdb, litellm, h5py) is pulled in by AIBOM features but inflates the binary for users who only need model scanning or MCP scanning.
- **Version metadata drift**: `pyproject.toml` declares `version = "1.0.0"` while `main.py` reports `VERSION = "0.0.1"`.

### 3.4 Build Pipeline Complexity

The Python CLI is excluded from the standard monorepo build pipeline:

- The root `Makefile` explicitly strips `aidcli` from `PY_SERVICES` (via `sed`), so `make all` does not build it.
- There is no root-level `make aidcli` target; the image is built separately from `aidcli/Dockerfile`.
- The Nuitka build stage within `aidcli/Dockerfile` requires meticulous management of include/exclude lists — adding `--nofollow-import-to` for test modules while being careful not to exclude modules like `jinja2.tests` that are actual runtime dependencies.
- Sealights instrumentation config explicitly notes "aidcli removed" from the scanned service list.

### 3.5 SDK Coupling and Version Drift

The Python CLI imports `aidefense` package internals directly:

```python
import aidefense.modelscan.model_scan
aidefense.modelscan.model_scan.RETRY_COUNT_FOR_SCANNING = int(os.environ.get(...))
```

This pattern of monkey-patching SDK module globals creates tight coupling to SDK internals that can break on any upstream change.

### 3.6 Performance and Startup

- **Cold start**: The Nuitka onefile binary self-extracts on first invocation, adding 2-5 seconds to startup.
- **Warm start**: Even cached, the Python interpreter initialization and module import chain is measurably slower than a compiled Go binary for simple API calls.
- **Memory footprint**: The Python runtime plus loaded ML libraries consume significantly more memory than equivalent Go HTTP calls.

---

## 4. Solution: Go-based CLI (aid_cli)

### 4.1 Architecture Overview

```
aid_cli/
├── main.go                      # Entry point → cmd.Execute()
├── cmd/                         # Cobra command tree
│   ├── root.go                  # Root command, global flags, PersistentPreRun
│   ├── modelscan/               # model-scan {trigger, list, info}
│   ├── mcpscan/                 # mcp-scan {register, delete, run, list, info, trigger, capabilities}
│   └── aibom/                   # ai-bom {generate, submit, run, list, info, delete, components, summary}
├── internal/
│   ├── cmdutil/                 # Shared CLI struct, auth resolution helpers
│   ├── config/                  # TOML config + env var resolution (go-toml/v2)
│   ├── output/                  # JSON formatting
│   └── embedded/python/         # Build-tag gated CPython-bundle embedding
│       ├── embed.go             # //go:build embed_python — extracts + locates interpreter
│       └── stub.go              # //go:build !embed_python — AID_CLI_PYTHON or error
├── sdk/aidefense/               # In-repo Go SDK for AI Defense API
│   ├── client.go                # BaseClient (net/http)
│   ├── config.go                # SDK config, auth header, endpoints
│   ├── errors.go                # Typed errors
│   ├── modelscan/               # Model scan API client + models
│   ├── mcpscan/                 # MCP scan API client + models
│   └── aibom/                   # AIBOM API client + models
├── scripts/
│   └── requirements-bundle.txt  # cisco-aibom[all] pin for the embedded bundle
└── Dockerfile                   # Multi-stage: CPython bundle → Go → minimal runner
```

### 4.2 Command Parity

The Go CLI implements full command parity with the Python CLI:

| Command Group | Python CLI (aidcli) | Go CLI (aid_cli) | Notes |
|---|---|---|---|
| `model-scan trigger` | `model-scan trigger` | `model-scan trigger` | File upload + repo URL scanning |
| `model-scan list` | `model-scan list` | `model-scan list` | Pagination via `--limit`/`--offset` |
| `model-scan info` | `model-scan info` | `model-scan info` | Scan details with file pagination |
| `mcp-scan register` | `mcp-scan register` | `mcp-scan register` | SSE/Streamable transport, auth configs |
| `mcp-scan delete` | `mcp-scan delete` | `mcp-scan delete` | Interactive confirmation with `-y` override |
| `mcp-scan run` | `mcp-scan run` | `mcp-scan run` | Remote scan with polling |
| `mcp-scan list` | `mcp-scan list` | `mcp-scan list` | Filter by name/status/transport |
| `mcp-scan info` | `mcp-scan info` | `mcp-scan info` | By server-id or scan-id |
| `mcp-scan trigger` | `mcp-scan trigger` | `mcp-scan trigger` | On-demand scan for registered server |
| `mcp-scan capabilities` | `mcp-scan capabilities` | `mcp-scan capabilities` | Tool/prompt/resource listing |
| `ai-bom generate` | `ai-bom generate` | `ai-bom generate` | Local AIBOM generation (embedded) |
| `ai-bom submit` | `ai-bom submit` | `ai-bom submit` | Upload existing report |
| `ai-bom run` | `ai-bom run` | `ai-bom run` | Generate + submit |
| `ai-bom list` | `ai-bom list` | `ai-bom list` | With filters |
| `ai-bom info` | `ai-bom info` | `ai-bom info` | By analysis ID |
| `ai-bom delete` | `ai-bom delete` | `ai-bom delete` | By analysis ID |
| `ai-bom components` | `ai-bom components` | `ai-bom components` | Component listing |
| `ai-bom summary` | `ai-bom summary` | `ai-bom summary` | Aggregate statistics |
| `scan` (deprecated) | `scan` → `model-scan` | `scan` → `model-scan` | Hidden alias with deprecation warning |

**Not ported**: `model-scan run` (local scanning via modelhawk + ClamAV) — this command has been intentionally removed. Local model scanning is replaced by cloud-only `model-scan trigger`.

### 4.3 In-repo Go SDK (sdk/aidefense)

Rather than depending on an external SDK, the Go CLI ships an in-repo HTTP client under `sdk/aidefense/`:

```
sdk/aidefense/
├── client.go          # BaseClient: net/http, JSON marshal, auth header injection
├── config.go          # Config struct, API version, header constants
├── errors.go          # APIError, ValidationError types
├── modelscan/
│   ├── client.go      # ScanFile, ScanRepo, ListScans, GetScan, poll logic
│   ├── models.go      # Request/response structs
│   ├── routes.go      # Endpoint path constants
│   └── client_test.go # httptest-based tests
├── mcpscan/
│   ├── client.go      # RegisterServer, StartScan, GetScanStatus, poll logic
│   ├── models.go      # TransportType, AuthConfig, request/response structs
│   ├── routes.go      # Endpoint path constants
│   └── client_test.go
└── aibom/
    ├── client.go      # SubmitReportFile, ListReports, GetReport, etc.
    ├── models.go      # Report models
    ├── routes.go      # Endpoint path constants
    └── client_test.go
```

This approach:
- **Eliminates the external SDK dependency** — the CLI and its API client are versioned together.
- **Uses only `net/http`** — no third-party HTTP libraries; total `go.mod` has just 3 direct dependencies (cobra, go-toml, pflag).
- **Is fully testable** with `httptest.NewServer` — each SDK package has unit tests mocking the AI Defense API.

### 4.4 Python Embedding via a relocatable CPython bundle

For AIBOM generation (which depends on `cisco-aibom[all]`, a Python-only library pulling duckdb, pandas, the langchain/cloud/scanner stacks, and native extensions), the Go CLI uses a build-tag-gated embedding strategy.

> **History:** this was originally a Nuitka onefile compilation. Because `cisco-aibom[all]` imports its backends lazily and grows over time, Nuitka required an ever-expanding list of `--include-package` flags (and namespace-package/native-extension workarounds) that broke on every new aibom backend. The analyzer is now shipped as a relocatable CPython interpreter with the dependency tree `pip install`ed into it — no compilation, so adding a backend never touches the build.

**Build flow:**
```
Dockerfile Stage 1 (py-bundler):
  download relocatable CPython (python-build-standalone)
  python -m pip install cisco-aibom[all]
  tar -czf aid-py-bundle.tar.gz  (+ short content id)

Dockerfile Stage 2 (go-builder):
  COPY --from=py-bundler aid-py-bundle.tar.gz → internal/embedded/python/bin/
  go build -tags embed_python → /aid-cisco-cli

Dockerfile Stage 3 (runner):
  COPY --from=go-builder /aid-cisco-cli → /app/aid-cisco-cli
```

**Runtime extraction + invocation:**
```go
// embed.go (//go:build embed_python)
//go:embed bin/aid-py-bundle.tar.gz
var bundleFS embed.FS

func Interpreter() (string, error) {
    // Extract once per content id to a cache dir (sync.Once + atomic rename),
    // then return <cache>/python/bin/python3.
}

// cmd/aibom drives aibom's own CLI with the bundled interpreter:
//   exec <python3> -m aibom analyze <sources...> --output-format json --output-file <tmp>
// and reads the JSON report back from <tmp>.
```

**Graceful degradation:**
```go
// stub.go (//go:build !embed_python)
func Interpreter() (string, error) {
    // AID_CLI_PYTHON if set, else: "embedded Python is not available; use make aid_cli or Docker"
}
```

This means a plain `go build` produces a lightweight Go binary where `ai-bom generate` returns a helpful error (unless `AID_CLI_PYTHON` is set), while Docker builds (`make aid_cli`) produce the full-featured binary.

### 4.5 Configuration and Auth

The Go CLI mirrors the Python CLI's credential resolution order:

| Priority | Source |
|---|---|
| 1 (highest) | CLI flags: `--api-key`, `--base-url` |
| 2 | Environment variables: `AI_DEFENSE_API_KEY`, `AI_DEFENSE_BASE_URL` |
| 3 | TOML config file: `.config.toml` or `AI_DEFENSE_CONFIG_PATH` |

Implementation uses `go-toml/v2` with a `config.Manager` that loads once in the root command's `PersistentPreRun`. Base URL validation enforces `*.cisco.com` hostnames.

### 4.6 Build Pipeline Integration

The Go CLI integrates with the monorepo Docker Bake pipeline:

```hcl
# builder/aid_cli.docker-bake.hcl
target "aid_cli_images" {
  context    = "aid_cli"
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64"]
  args = {
    PY_RUNNER_DEBIAN_IMAGE = PY_RUNNER_DEBIAN_IMAGE
    GO_BUILDER_IMAGE       = GO_BUILDER_IMAGE
    CLI_VERSION            = CLI_VERSION
    CLI_COMMIT             = CLI_COMMIT
  }
  output = [service_output("aid_cli", "images")]
  tags   = split(",", service_tags("aid_cli", "images"))
}
```

This uses the same `service_output()` and `service_tags()` functions as all other services, ensuring consistent tagging, registry push behavior, and CI integration.

### 4.7 Backward Compatibility

The Go CLI maintains backward compatibility with the Python CLI:

- **`scan` → `model-scan` alias**: A hidden command registered via `NewDeprecatedScanCmd()` that prints a deprecation warning to stderr and delegates to the `model-scan` command tree. The alias chains the root command's `PersistentPreRun` to ensure config initialization is not skipped.
- **Same binary name**: `aid-cisco-cli` — drop-in replacement.
- **Same config format**: `.config.toml` with identical `[default]` section structure.
- **Same env vars**: `AI_DEFENSE_API_KEY`, `AI_DEFENSE_BASE_URL`, and scan polling env vars are supported with matching semantics.
- **Same JSON output**: API responses are printed as JSON to stdout, matching the Python CLI's output format for scripting compatibility.

---

## 5. How the Go CLI Improves the Experience

| Dimension | Python CLI (aidcli) | Go CLI (aid_cli) | Improvement |
|---|---|---|---|
| **Binary size** | ~200+ MB (Nuitka onefile with entire Python runtime + ML stack) | ~15-20 MB Go binary (without AIBOM); ~220 MB with embedded AIBOM | API-only binary is 10x smaller |
| **Cold startup** | 2-5s (Nuitka self-extraction) | <50ms | 40-100x faster startup |
| **Runtime crashes** | `hf_xet` Rust teardown, `_PyImport_Init` | None — no Python interpreter for API commands | Eliminates entire crash class |
| **Dependencies** | ~50+ transitive Python packages, poetry.lock, private git deps | 3 direct Go modules (cobra, go-toml, pflag) | 95% reduction in supply chain surface |
| **TLS/proxy issues** | truststore injection failures in compiled binary | Go's native `crypto/tls` with system cert pool | Works out of the box |
| **SDK versioning** | Pinned to `git main` — any commit can break | In-repo `sdk/aidefense/` — versioned with CLI | Zero external SDK drift |
| **Build time** | 10-15 min (Nuitka compilation) | ~30s (Go build) or 5-8 min (with Nuitka for AIBOM) | 3-5x faster for API-only builds |
| **Cross-compilation** | Requires Nuitka on target architecture | `GOOS=linux GOARCH=amd64 go build` | Trivial cross-compilation |
| **CI pipeline** | Excluded from standard build; separate Dockerfile | Integrated via Docker Bake HCL | Standard pipeline |
| **Testing** | CliRunner + mocked SDK | `httptest.NewServer` + Go test | Faster, no Python env needed |
| **Config path bugs** | Relative paths broken in Nuitka binary | `go-toml` reads correctly from any cwd | Eliminated |

---

## 6. Scaling Strategy

### 6.1 Adding Pure Go Commands

For new commands that only require HTTP calls to the AI Defense API (or any REST API):

1. Create `cmd/<group>/` package with Cobra commands.
2. Add SDK client in `sdk/aidefense/<domain>/` with `client.go`, `models.go`, `routes.go`.
3. Register in `cmd/root.go`: `rootCmd.AddCommand(newgroup.NewCmd(cli))`.
4. Write tests using `httptest.NewServer`.

**Cost**: Minimal — adds a few KB to the binary. No new build stages, no Docker changes. This is the preferred path for all future API-based features.

### 6.2 Adding Python Submodules (Embedded)

When a feature requires a Python library with no Go equivalent (like `cisco-aibom`
for AIBOM generation), it rides along in the existing relocatable CPython bundle —
there is no per-tool build step and no Python glue package.

**How a Python tool is shipped today:**

1. **Pin it** in `scripts/requirements-bundle.txt` (e.g. `cisco-aibom[all]==…`).
   The Docker `py-bundler` stage `pip install`s it into the relocatable CPython
   interpreter; `pip` resolves the full (lazily-imported) dependency tree,
   including namespace packages and native extensions, and it ships as real files.
2. **Invoke it** from Go as a subprocess via the tool's own CLI, using the
   interpreter returned by `embeddedpython.Interpreter()`:

```go
py, _ := embeddedpython.Interpreter()       // path to bundled python3
cmd := exec.Command(py, "-m", "aibom", "analyze", /* args... */)
```

Because Go calls the tool's real CLI, the tool's own argument parser fills every
default and validates inputs — the Go side never re-declares the tool's flags.

**Adding a second Python tool** (e.g. a local model analyzer) is therefore cheap
and uniform:

- Add its distribution to `scripts/requirements-bundle.txt` — no second build
  stage, no `//go:embed` of a separate binary, no hand-managed include lists.
- Add a thin Go command that execs `python -m <tool> …` and parses its output.
- The bundle already contains a shared Python stdlib, so the size delta is only
  the new tool's own dependencies.

This is the property that motivated dropping Nuitka (see §4.4 and §9): `pip` owns
dependency resolution, so a new backend or tool never touches the Dockerfile or
the Go embedding code.

---

## 7. Future Work

### 7.1 Go SDK for AI Defense Scans (External Package)

**Current state**: The Go CLI contains an in-repo SDK at `aid_cli/sdk/aidefense/`. This works for the CLI but is not usable by other Go services or customer integrations.

**Proposal**: Extract and publish a standalone Go SDK package.

**Motivation:**
- Other Go services in the monorepo (e.g., `ai_access_manager`, `ai_firewall_manager`) may need to call AI Defense scan APIs programmatically.
- Customer CI/CD pipelines written in Go cannot integrate AI Defense scanning without shelling out to the CLI.
- A Go SDK provides a type-safe, importable alternative to raw HTTP calls against the OpenAPI spec.

**Proposed package structure:**
```
github.com/cisco-ai-defense/aidefense-go-sdk/
├── aidefense.go            # Config, NewClient
├── client.go               # Shared HTTP transport, auth, error handling
├── modelscan/
│   ├── client.go           # ScanFile, ScanRepo, ListScans, GetScan
│   └── models.go           # Request/response types
├── mcpscan/
│   ├── client.go           # RegisterServer, StartScan, ListServers, etc.
│   └── models.go
├── aibom/
│   ├── client.go           # SubmitReport, ListReports, GetReport, etc.
│   └── models.go
└── examples/
    └── scan-model/main.go
```

**API design principles:**
- Configuration via functional options: `aidefense.NewClient(aidefense.WithAPIKey("..."), aidefense.WithBaseURL("..."))`
- Context-aware methods: `client.ModelScan.TriggerFile(ctx, filePath, opts...)`
- Structured error types matching API error codes
- No CLI or I/O dependencies — pure library

**Migration path:**
1. Extract current `aid_cli/sdk/aidefense/` into a new repository.
2. Add functional options, context support, and documentation.
3. Publish as `github.com/cisco-ai-defense/aidefense-go-sdk`.
4. Update `aid_cli` to import the external SDK instead of the in-repo copy.
5. Publish versioned releases aligned with API version changes.

**Timeline**: This is a medium-priority item. The in-repo SDK is functional and tested; extraction becomes valuable when a second Go consumer emerges or when customers request Go SDK support.

### 7.2 Plugin Architecture for Python Tools

Dependency management already scales automatically — `pip` resolves the union of
every pinned tool in `scripts/requirements-bundle.txt`, so there are no manual
include lists to maintain. As the number of embedded Python tools grows, the
remaining opportunity is *tool discovery* rather than dependency wiring:

- A manifest (`scripts/tools.yaml`) declaring each tool's module and CLI entry point.
- The Go CLI generating its subcommands from that manifest instead of hand-writing
  a thin `exec` command per tool.
- Optional validation in CI that every declared tool is importable from the bundle.

### 7.3 Additional Enhancements

| Enhancement | Priority | Description |
|---|---|---|
| **Structured output modes** | Low | Extend output formats (`table`/`yaml`) to all commands. Scan commands already emit a normalized JSON/text verdict envelope with a deterministic exit-code contract — see `internal/verdict` and `internal/clierror` (AIFW-24131). |
| **Shell completions** | Low | Generate bash/zsh/fish completions via Cobra |
| **Profile management** | Medium | `--profile <name>` to switch between TOML config sections (multi-tenant support) |
| **Offline mode / cache** | Low | Cache scan results locally for disconnected review |
| **Telemetry** | Medium | Opt-in usage telemetry for CLI command frequency and error rates |
| **Interactive mode** | Low | TUI for browsing scan results (via bubbletea or similar) |
| **Local model scanning (Go)** | High | Port modelhawk scanning logic to Go, eliminating the last Python dependency for scan workflows |

---

## 8. Migration and Deprecation Plan

| Phase | Action | Timeline |
|---|---|---|
| **Phase 1 (Current)** | Go CLI at feature parity; Python CLI still available; `scan` alias maintained | Complete |
| **Phase 2** | Documentation updated to reference Go CLI as primary; Python CLI docs marked as legacy | In Progress |
| **Phase 3** | Python CLI (`aidcli`) removed from CI builds; Docker image deprecated | TBD |
| **Phase 4** | Python CLI source archived; Go CLI is sole distribution | TBD |

The `scan` → `model-scan` deprecation alias will be maintained in the Go CLI for at least two major release cycles after the Python CLI is fully deprecated.

---

## 9. Decision Log

| Decision | Rationale | Alternatives Considered |
|---|---|---|
| **Go over Rust** | Team expertise; Go is already the primary language for monorepo services; Cobra ecosystem maturity | Rust (higher learning curve), staying with Python |
| **In-repo SDK over external** | Faster iteration; versioned with CLI; no cross-repo release coordination needed at this stage | External Go SDK from day one (premature for single consumer) |
| **Relocatable CPython bundle over Nuitka** | No compilation treadmill — `pip` resolves `cisco-aibom[all]` (lazy imports, namespace packages, native extensions) and it ships as-is; the bundled interpreter means no Python runtime required on target. Superseded the original Nuitka onefile, whose `--include-package` list broke on every new aibom backend | Nuitka onefile (fragile include lists, long compiles); rewrite AIBOM in Go (high effort, cisco-aibom is complex) |
| **Build-tag gating over always embedding** | Lightweight `go build` for development; full Docker build for release | Always embed (slower dev builds); separate binaries (confusing distribution) |
| **Docker Bake integration over standalone Docker build** | Consistent with monorepo CI; shared registry/tagging logic; remote BuildKit support | Standalone `docker build` (breaks in CI without local daemon) |
| **Cobra over stdlib flags** | Subcommand hierarchy; built-in help generation; completion support; ecosystem familiarity | Click-like Go libs (less mature); stdlib flag package (no subcommand support) |
