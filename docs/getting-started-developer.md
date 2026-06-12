# Getting Started — Cisco AI Defense CLI (Developers)

Build, run, and extend `aid-cisco-cli` locally. For using released binaries, see [getting-started-client.md](getting-started-client.md).

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
| `internal/embedded/python/` | Embedded CPython bundle (build tag `embed_python`) |
| `scripts/` | Python bundle build (`build-python-bundle.sh`) |

See [docs/adr/0001-package-layout.md](adr/0001-package-layout.md).

---

## 1. Go-only build (fastest)

```bash
make build-go
./bin/aid-cisco-cli --help
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
./bin/aid-cisco-cli ai-bom generate --source ./my-app ...
```

---

## 4. Full embed (local host)

Bundles relocatable CPython + `cisco-aibom[all]` for your **current OS/arch**:

```bash
make bundle build-embed
./bin/aid-cisco-cli ai-bom generate ...
```

On macOS this produces a darwin binary with embedded Python. First `make bundle` downloads python-build-standalone and pip packages (network required).

---

## 5. Release binaries (`dist/`)

| Target | Platform | Method |
|--------|----------|--------|
| `make build-release-linux GOARCH=amd64` | Linux amd64 | Docker → `dist/aid-cisco-cli-<version>-linux-amd64` |
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
make test vet
```

---

## 8. Adding commands

- **API command:** `internal/cli/<group>/`, register in `internal/cli/root.go`, client in `api/<domain>/`
- **Python-backed:** pin in `scripts/requirements-bundle.txt`, invoke via `internal/embedded/python`

See [spec 001](specs/001-cli-core/design.md).
