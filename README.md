# ai-defense-cli

**Cisco AI Defense CLI** (`aid-cisco-cli`) — command-line tool for AI Defense
scanning and inventory: **model-scan**, **mcp-scan**, and **ai-bom**. Deterministic
exit codes and machine-readable output for CI/CD.

## Features

- **model-scan** — AI/ML model security scans
- **mcp-scan** — MCP server registration and scanning
- **ai-bom** — AI Bill of Materials (embedded `cisco-aibom` analyzer)
- **onboard / config** — interactive setup; config stored under `~/.ai-defense/` with named profiles
- **install** — checksum-verified self-update from GitHub Releases
- **Exit codes** — operational (1–5) vs policy verdicts (10, 20)
- **CI output** — raw API JSON by default; `--envelope` + `--fail-on` / `--block-on`

## Installation

- **Users:** [GitHub Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases) · [install-upgrade.md](docs/install-upgrade.md) — download binaries, verify checksums, upgrade
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
# After install from Releases:
aid-cisco-cli onboard

# Or set credentials manually:
export AI_DEFENSE_API_KEY="your-api-key"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"

aid-cisco-cli model-scan list --limit 5
```

See [docs/getting-started-client.md](docs/getting-started-client.md) for the full command
reference, output modes, and exit-code gating. Documentation index: [docs/README.md](docs/README.md).

## Development

```bash
make build-go test-cover-check vet   # daily dev (matches CI test job)
make bundle build-embed          # full embed (local OS)
make build-release-darwin        # macOS release → dist/
make build-release-linux GOARCH=amd64   # Linux release via Docker
```

Details: [docs/getting-started-developer.md](docs/getting-started-developer.md) ·
[CONTRIBUTING.md](CONTRIBUTING.md) · [SECURITY.md](SECURITY.md)
