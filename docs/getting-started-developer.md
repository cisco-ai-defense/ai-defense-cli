# Getting Started — Cisco AI Defense CLI (Developers)

Build, run, and extend `cisco-aid-cli` locally. For using released binaries, see [getting-started-client.md](getting-started-client.md).

## Prerequisites

- **Go 1.23+**
- **Docker** — Linux release builds (`make build-release-linux`)
- **curl, tar, gzip** — `make bundle` on macOS/Linux
- AI Defense API key + base URL for integration testing

## Repository layout

| Path | Role |
|------|------|
| `internal/cli/` | Cobra command tree (`modelscan`, `mcpscan`, `aibom`) |
| `api/` | Public HTTP client (`github.com/cisco-ai-defense/ai-defense-cli/api`) |
| `internal/cmdutil/` | Shared CLI context and scan flags |
| `internal/cliexitdoc/` | Exit-code scenario registry; generates client handbook |
| `internal/embedded/python/` | Embedded CPython bundle (build tag `embed_python`) |
| `scripts/` | Python bundle build (`build-python-bundle.sh`) |

See [docs/adr/0001-package-layout.md](adr/0001-package-layout.md).

---

## 1. Go-only build (fastest)

```bash
make build-go
./bin/cisco-aid-cli --help
```

API commands work. `ai-bom generate`/`run` need `AID_CLI_PYTHON` or a full embed build (§3–4).

---

## 2. Run and iterate

```bash
export AI_DEFENSE_API_KEY="<key>"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"

go run . model-scan list --limit 5
```

---

## 3. Local AIBOM dev (`AID_CLI_PYTHON`)

```bash
python3 -m venv .venv && . .venv/bin/activate
pip install 'cisco-aibom[all]'
export AID_CLI_PYTHON="$PWD/.venv/bin/python"

make build-go
./bin/cisco-aid-cli ai-bom generate --source ./my-app ...
```

---

## 4. Full embed (local host)

Bundles relocatable CPython + `cisco-aibom[all]` for your **current OS/arch**:

```bash
make bundle build-embed
./bin/cisco-aid-cli ai-bom generate ...
```

On macOS this produces a darwin binary with embedded Python. First `make bundle` downloads python-build-standalone and pip packages (network required). Subsequent runs skip when inputs are unchanged; use `make bundle-force` to rebuild.

---

## 5. Release binaries (`dist/`)

| Target | Platform | Method |
|--------|----------|--------|
| `make build-release-linux GOARCH=amd64` | Linux amd64 | Docker → `dist/cisco-aid-cli-<version>-linux-amd64` |
| `make build-release-darwin` | macOS | Native `bundle` + `go build -tags embed_python` |
| `make build-release` | Host OS | Dispatches linux→Docker, darwin→native |
| `make checksums` | — | `dist/checksums.txt` |

Linux release (exports binary, not a container image):

```bash
make build-release-linux GOARCH=amd64
```

macOS release (on your Mac):

```bash
make build-release-darwin GOARCH=arm64
```

---

## 6. Build tags

| Tag | Behavior |
|-----|----------|
| `embed_python` | Embeds `internal/embedded/python/bin/aid-py-bundle.tar.gz` |
| (default) | Stub Python; `AID_CLI_PYTHON` optional for ai-bom dev |

---

## 7. Tests

```bash
make test vet              # unit tests only
make test-cover            # coverage profile (coverage.out)
make test-cover-check      # enforce COVER_MIN (default 75%; same as CI)
make test-diff-cover-check DIFF_BASE=origin/main  # changed-line coverage (PR CI)
```

CI runs `make build-go test-cover-check vet` on every push; pull requests also run
`make test-diff-cover-check` against the merge base. Coverage gate scope excludes `scripts/`, `main`, `cmd/`, test helpers, and TUI/onboarding packages until they have unit tests — see [CONTRIBUTING.md](../CONTRIBUTING.md). `make test` runs `./...`.

Command-layer HTTP tests use `internal/testutil/apitest` (httptest with a `*.cisco.com` base URL rewrite to satisfy API host validation). Inject a custom client via `api.Config.HTTPClient` and `cmdutil.APIConfigFactory` when testing commands.

---

## 8. Adding commands

- **API command:** `internal/cli/<group>/`, register in `internal/cli/root.go`, client in `api/<domain>/`
- **Python-backed:** pin in `scripts/requirements-bundle.txt`, invoke via `internal/embedded/python`

See [spec 001](specs/001-cli-core/design.md).

---

## 9. Exit-code documentation

Per-command exit scenarios are **not** parsed from godoc comments. Go has no
standard machine-readable “errors this function returns” annotation (unlike
OpenAPI response codes). Instead:

1. Add or update rows in **`internal/cliexitdoc/registry.go`** (use
   `clierror.Code*` constants).
2. Regenerate the client handbook:

```bash
make gen-exit-docs
```

3. CI can run `make check-exit-docs` to fail on a stale handbook.

Runtime behavior lives in command handlers and `internal/clierror.Classify`;
the registry is the documentation contract. See
[ADR 0001](adr/0001-exit-docs-from-registry.md) and
[exit-codes-handbook.md](exit-codes-handbook.md).
