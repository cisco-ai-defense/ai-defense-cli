# ai-defense-cli

**Cisco AI Defense CLI** (`aid-cisco-cli`)

Command-line tool for AI Defense scanning and inventory: **model-scan**, **mcp-scan**, and **ai-bom**. Deterministic exit codes and machine-readable output for CI/CD.

## Status

| Area | Status |
|------|--------|
| CLI source | In review ([PR #1](https://github.com/cisco-ai-defense/ai-defense-cli/pull/1)) |
| Multi-platform CI | Active on PRs and `main` |
| Install docs | [install-upgrade.md](docs/install-upgrade.md) |

## Features

- **model-scan** — AI/ML model security scans
- **mcp-scan** — MCP server registration and scanning
- **ai-bom** — AI Bill of Materials (embedded `cisco-aibom` analyzer)
- **Exit codes** — operational (1–5) vs policy verdicts (10, 20)
- **CI output** — JSON verdict envelope, `--fail-on` / `--block-on`

## Installation

- **Users:** [docs/install-upgrade.md](docs/install-upgrade.md) — download binaries, verify checksums, upgrade
- **Developers:** [docs/getting-started-developer.md](docs/getting-started-developer.md) — build from source

Quick build from source:

```bash
git clone https://github.com/cisco-ai-defense/ai-defense-cli.git
cd ai-defense-cli
make build-go
./bin/aid-cisco-cli --help
```

## Quick start

```bash
export AI_DEFENSE_API_KEY="your-api-key"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"

./bin/aid-cisco-cli model-scan list --limit 5
```

## Repository layout

```
api/                 # HTTP client (importable)
internal/cli/        # Cobra commands
internal/            # config, exit codes, verdict, embedded Python
scripts/             # Python bundle build
docs/                # specs, architecture, guides
```

## Development

```bash
make build-go test vet           # daily dev
make bundle build-embed          # full embed (local OS)
make build-release-darwin        # macOS release → dist/
make build-release-linux GOARCH=amd64   # Linux release via Docker
```

See [docs/getting-started-developer.md](docs/getting-started-developer.md).

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) · [SECURITY.md](SECURITY.md) · [LICENSE](LICENSE)

Full command reference: [docs/getting-started-client.md](docs/getting-started-client.md) · [docs/service-spec.md](docs/service-spec.md)
