# Getting Started — Cisco AI Defense CLI (Users)

This guide gets you from zero to your first scan with `cisco-aid-cli`. It is for
**people running the CLI**. If you want to build or modify the CLI, see the
[developer guide](getting-started-developer.md).

`cisco-aid-cli` does three things:

| Command group | What it does | Needs |
|---|---|---|
| `model-scan` | Scan AI/ML model files or repos for vulnerabilities | AI Defense API key |
| `mcp-scan` | Register and scan MCP servers | AI Defense API key |
| `ai-bom` | Generate, manage, and explore AI Bills of Materials | AI Defense API key; **local analysis also needs an LLM** |

---

## 0. Onboarding (recommended)

After installing the binary, run the interactive wizard:

```bash
cisco-aid-cli onboard
```

You'll configure:

1. **Core** — choose profile (or use `--profile`); then base URL and API key
2. **Optional capabilities** — HuggingFace and AIBOM, one at a time (Configure or Skip). Progress is saved after each field; run `onboard` again to continue or start over. Re-run shows existing profiles and which is active (`*`).

Settings are saved to **`~/.ai-defense/`** — default profile in `config.toml`, or named profiles under `profiles/` (see [config.md](config.md#where-settings-live)).

Add or change settings later with `cisco-aid-cli config edit` (section menu). Inspect with
`cisco-aid-cli config show` on a terminal, or `config show --plain` for redacted TOML.

Non-interactive environments: set `AI_DEFENSE_API_KEY` / `AI_DEFENSE_BASE_URL` or
create `~/.ai-defense/config.toml` manually (see §2 below).

---

## 1. Get the binary

Release builds ship a **single self-contained binary** per OS/architecture. The
AIBOM analyzer (embedded CPython + `cisco-aibom`) is included — no separate
Python install needed for `ai-bom analyze` and the other analyzer commands.

### GitHub Release (recommended when available)

Download the binary for your platform from the [Releases](https://github.com/cisco-ai-defense/ai-defense-cli/releases)
page. Artifact names:

| Platform | Binary (example v0.1.0) |
|----------|-------------------------|
| Linux x86_64 | `cisco-aid-cli-0.1.0-linux-amd64` |
| macOS Apple Silicon | `cisco-aid-cli-0.1.0-darwin-arm64` |
| macOS Intel | `cisco-aid-cli-0.1.0-darwin-amd64` |

Verify with `checksums.txt` from the same release. See [install-upgrade.md](install-upgrade.md).

### CI artifacts (main branch)

After a merge to `main`, download embed binaries from **Actions** → latest **CI**
run on `main` → **Artifacts** (7-day retention). See [install-upgrade.md](install-upgrade.md).

Pull requests verify embed builds pass but do not publish artifacts.

### Build from source

See the [developer guide](getting-started-developer.md). For a full embed binary locally:

```bash
make bundle build-embed
./bin/cisco-aid-cli --help
```

Verify any binary:

```bash
chmod +x ./cisco-aid-cli   # if needed
./cisco-aid-cli --version
./cisco-aid-cli --help
```

---

## 2. Authenticate

You need an **AI Defense API key** and the **base URL** for your region. The base
URL hostname must end in `.cisco.com`.

| Region | Base URL |
|---|---|
| US | `https://us.api.aidefense.security.cisco.com` |
| EU | `https://eu.api.aidefense.security.cisco.com` |
| AP | `https://ap.api.aidefense.security.cisco.com` |
| UAE | `https://uae.api.aidefense.security.cisco.com` |

Provide credentials in any of these ways (highest priority first):

**1. Flags:**

```bash
./cisco-aid-cli --api-key "<key>" --base-url "https://us.api.aidefense.security.cisco.com" model-scan list
```

**2. Environment variables:**

```bash
export AI_DEFENSE_API_KEY="<key>"
export AI_DEFENSE_BASE_URL="https://us.api.aidefense.security.cisco.com"
./cisco-aid-cli model-scan list
```

**3. TOML config file** (`~/.ai-defense/config.toml`, or set `AI_DEFENSE_CONFIG_PATH`):

```toml
[default]
api_key = "<key>"
base_url = "https://us.api.aidefense.security.cisco.com"

# Optional: token for scanning private HuggingFace repos
[default.huggingface]
token = "hf_xxx"
```

---

## 3. Scan a model

```bash
# Scan a local model file (multipart upload, 10 parallel parts by default)
./cisco-aid-cli model-scan trigger -f ./model.pth

# Tune or disable multipart upload for constrained networks
./cisco-aid-cli model-scan trigger -f ./model.safetensors --upload-concurrency 4
./cisco-aid-cli model-scan trigger -f ./model.pth --no-multipart

# Scan a model repository (private repos need a token)
./cisco-aid-cli model-scan trigger --repo-url https://huggingface.co/org/model --repo-token hf_xxx

# Review results
./cisco-aid-cli model-scan list --limit 10
./cisco-aid-cli model-scan info --scan-id <uuid>
```

Local files are uploaded in parallel parts with per-part retries, so large
models upload reliably. An upload progress bar and then a "waiting for scan
status" spinner are shown on stderr; stdout always carries only the scan JSON. Empty files are rejected, and if
your AI Defense deployment does not support multipart upload yet, use
`--no-multipart`.

---

## 4. Scan an MCP server

```bash
# One-shot scan (polls to completion)
./cisco-aid-cli mcp-scan run --url https://mcp.example.com/sse --name "My Server" --transport SSE

# Or register for ongoing monitoring, then trigger scans
./cisco-aid-cli mcp-scan register --url https://mcp.example.com/sse --name "My Server"
./cisco-aid-cli mcp-scan list
./cisco-aid-cli mcp-scan trigger --server-id <uuid>
./cisco-aid-cli mcp-scan info --server-id <uuid>
./cisco-aid-cli mcp-scan capabilities --url https://mcp.example.com/sse
```

---

## 5. Generate, manage, and explore AI Bills of Materials (AIBOM)

`ai-bom` is a single surface over two backends:

- **analyzer passthroughs** (`analyze`, `report`, `watch`, `diff`, `benchmark`,
  `kb`, `cache`, `plugin`) run the embedded `cisco-aibom` analyzer. Their flags
  and `--help` come straight from the analyzer.
- **cloud inventory** (`list`, `info`, `components`, `summary`, `delete`) call the
  AI Defense API directly.

Local analysis uses an **LLM** to classify what it finds, so you must supply a
model and credentials for it.

```bash
# Analyze locally and write the AIBOM (NOT uploaded)
./cisco-aid-cli ai-bom analyze ./my-app \
  --llm-model gpt-5.4 \
  --llm-api-key "$OPENAI_API_KEY" \
  --output-format json --output-file aibom.json

# Upload a generated report to AI Defense (URL + key derived from your config)
./cisco-aid-cli ai-bom report upload aibom.json
```

You can set the LLM options once via environment variables (or `[default.aibom]`
in your config) instead of flags:

```bash
export AIBOM_LLM_MODEL=gpt-5.4
export AIBOM_LLM_API_KEY="$OPENAI_API_KEY"
./cisco-aid-cli ai-bom analyze ./my-app --output-format json --output-file aibom.json
```

**Provider credentials:**

| Provider | How to supply credentials |
|---|---|
| OpenAI / Azure / Anthropic / Google | `--llm-api-key` (or `AIBOM_LLM_API_KEY`); Azure also needs `--llm-api-base` and `--llm-api-version` |
| AWS Bedrock | Use the standard AWS credential chain (env vars / profile / role); no `--llm-api-key` |
| Ollama / local | No key; set `--llm-api-base` to your endpoint |

Use `--llm-provider` to force a provider when the model name is ambiguous.

> Passthrough subcommands forward every flag to the analyzer, so the global
> `--api-key` / `--base-url` / `--profile` flags do **not** apply to them. Supply
> credentials via environment variables or your config file; `report upload`
> derives the AI Defense URL and key from there automatically (an explicit
> `--post-url` / `--ai-defense-api-key` still wins).

**Manage uploaded BOMs (API only — no LLM needed):**

```bash
./cisco-aid-cli ai-bom list --limit 10 --status COMPLETED
./cisco-aid-cli ai-bom info --analysis-id <id>
./cisco-aid-cli ai-bom components --analysis-id <id>
./cisco-aid-cli ai-bom summary
./cisco-aid-cli ai-bom delete --analysis-id <id> --yes
```

---

## 6. Use exit codes in CI/CD

Full per-command reference: **[exit-codes-handbook.md](exit-codes-handbook.md)**.

Scan commands (`model-scan trigger`/`info`, `mcp-scan run`/`info --scan-id`) print
**raw API JSON on stdout by default** (same shape as the Python `aidcli`). They
also set a deterministic exit code so your pipeline can branch on `$?`. Use
`--envelope` when you want the normalized verdict JSON/text instead. Policy
verdicts (`>= 10`) are always distinct from operational failures (`1-9`):

| Code | Meaning |
|---|---|
| `0` | Pass — nothing at/above `--fail-on` |
| `2` | Usage error (bad flags/args) |
| `3` | Auth/config error (bad key or base URL) |
| `4` | Connectivity/timeout — retryable |
| `5` | Scan failed/cancelled (could not produce a verdict) |
| `1` | Unexpected error |
| `10` | Fail — findings at/above `--fail-on` |
| `20` | Block — findings at/above `--block-on` |

Gating is fail-closed by default and configurable:

```bash
# Fail the build only on HIGH+; hard-block on CRITICAL; print human-readable text.
./cisco-aid-cli model-scan trigger -f ./model.pth \
  --fail-on high --block-on critical --output text

# Report findings without ever failing the command (exit 0 unless the tool errors):
./cisco-aid-cli model-scan trigger -f ./model.pth --fail-on none
```

Branch on the result:

```bash
./cisco-aid-cli model-scan trigger -f ./model.pth --fail-on high
case $? in
  0)  echo "clean" ;;
  10) echo "threats found";        exit 1 ;;
  20) echo "blocked (critical)";   exit 1 ;;
  4)  echo "transient — retrying"; exit 75 ;;
  *)  echo "scan could not run";   exit 1 ;;
esac
```

Raw API JSON is the default. Use `--envelope` (or `--raw=false`) for the
normalized verdict envelope (gating still applies). On any non-zero exit the CLI
prints `Error:` and, where helpful, a `Hint:` line to stderr.

> MCP note: the unregistered `mcp-scan run`/`info --scan-id` status payload may
> report status only. When it carries no severity signal, a completed scan
> reports `PASS` (exit `0`); failed/cancelled is `5` and a poll timeout is `4`.

---

## 7. Migrating from the Python `aidcli`

`cisco-aid-cli` (Go) replaces the earlier Python `aidcli`. **Your existing
commands keep working** — the subcommands and flags are unchanged
(`model-scan trigger -f`, `--repo-url`, `--scan-id`, `mcp-scan run --url/--name`,
the deprecated `scan` alias, `--version`, etc.). Exit-code behavior changed on
purpose, so review any pipeline that branched on `$?`:

| Behavior | Python `aidcli` | Go `cisco-aid-cli` (default) |
|---|---|---|
| Scan stdout | Raw API response JSON | Raw API response JSON (same default) |
| Verdict envelope | N/A | Opt in with `--envelope` |
| Exit on a completed scan **with threats** | `0` (you parsed findings yourself) | `10` (fail) / `20` (block) |
| Exit on usage error | `2` | `2` (unchanged) |
| Exit on operational error | `1` | `1` / `3` / `4` / `5` by class |

What this means in practice:

- Scan stdout matches Python (raw API JSON by default). If you adopted `v0.1.1`
  (verdict envelope on stdout without `--raw`), add `--envelope` (or `--raw=false`).
- A pipeline that relied on "completed scan ⇒ exit `0`" will now **fail the step**
  when threats meet the gate. This is usually what you want, but it is a change.
- A check for *exactly* exit `1` may now see `3`/`4`/`5`. Prefer `!= 0`, or branch
  on the [exit-code table](#6-use-exit-codes-in-cicd) above.

**Match Python exit behavior** (never fail the step on findings) with:

```bash
./cisco-aid-cli model-scan trigger -f ./model.pth --fail-on none --block-on none
```

**Use the verdict envelope** (unified CI JSON across model-scan and mcp-scan):

```bash
./cisco-aid-cli model-scan trigger -f ./model.pth --envelope --fail-on high
```

Operational errors are classified into `3`/`4`/`5` (the old catch-all `1` is no
longer used for those). Prefer the new exit-code contract for new pipelines.

### 7.1 Renamed from `aid-cisco-cli`

From the release that includes this rename, the installed command is
**`cisco-aid-cli`** (not `aid-cisco-cli`). Release artifacts use the same prefix:
`cisco-aid-cli-<version>-<os>-<arch>`.

Update CI scripts, Makefiles, and documentation that invoke the old name. Config
paths (`~/.ai-defense/`), env vars (`AI_DEFENSE_*`), and subcommands are unchanged.

```bash
# was: aid-cisco-cli model-scan list
cisco-aid-cli model-scan list
```

If you installed to `/usr/local/bin/aid-cisco-cli`, remove the old binary and
install the new artifact as `cisco-aid-cli` (see [install-upgrade.md](install-upgrade.md)).

---

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `--llm-model is required` | `ai-bom analyze` needs a model. Pass `--llm-model` or set `AIBOM_LLM_MODEL`. |
| `no LLM credential found` | You passed a model without credentials. Add `--llm-api-key` / `AIBOM_LLM_API_KEY`, use `--llm-api-base` (local), `--llm-provider bedrock` (AWS chain), or export the provider's native key. |
| `The api_key client option must be set ... OPENAI_API_KEY` (in analyzer output) | No LLM credential reached the analyzer. Pass `--llm-api-key` / set `AIBOM_LLM_API_KEY`, or set the provider's native env var. |
| Base URL rejected | The base URL host must end with `.cisco.com`. Use a regional endpoint from the table above. |
| `embedded Python is not available in this build` | You're running a Go-only build. Use the full release binary/image, or (for dev) set `AID_CLI_PYTHON` — see the [developer guide](getting-started-developer.md). |
| Auth/`401` errors | Check `AI_DEFENSE_API_KEY` and that the base URL matches your tenant's region. |
| Command exits `10`/`20` | Not an error — the scan found threats at/above `--fail-on`/`--block-on`. Inspect the verdict envelope; raise the thresholds (e.g. `--fail-on high`) or use `--fail-on none` to report without gating. |
| Command exits `4` on a slow scan | Polling timed out. Increase `AI_DEFENSE_MCP_RETRY_COUNT` / `AI_DEFENSE_MCP_WAIT_TIME_SECS` (MCP) or `AI_DEFENSE_RETRY_COUNT_FOR_SCANNING` / `AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK` (model; default wait is 15,000 s, and setting either variable makes the wait retry count × wait time), then re-run. For `model-scan trigger`, the scan is still running: check it with `model-scan info --scan-id <id>` instead of re-running (which starts a new scan and re-uploads the file). |
| `invalid --fail-on`/`--output` | Use a severity (`none\|low\|medium\|high\|critical`) and `--output json\|text`. |

For the full command and configuration reference, see the
[service specification](service-spec.md).
