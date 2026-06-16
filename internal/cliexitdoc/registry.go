package cliexitdoc

import "github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"

// GlobalScenarios returns handbook rows that apply across commands.
func GlobalScenarios() []CommandGroup {
	return []CommandGroup{{
		Title:   "Global scenarios (all commands)",
		Intro:   "These apply to any command that calls the AI Defense API or shares auth resolution.",
		Columns: []string{"Scenario", "Code", "Hint / notes", "Retry?"},
		Rows: []Scenario{
			{When: "Success", Code: clierror.CodeOK, Notes: "—", Retry: "—"},
			{When: "Invalid flag or Cobra parse error", Code: clierror.CodeUsage, Notes: "Usage printed", Retry: "No"},
			{When: "Unknown subcommand / missing required flag", Code: clierror.CodeUsage, Notes: "Usage printed", Retry: "No"},
			{When: "Client-side validation (`ValidationError`)", Code: clierror.CodeUsage, Notes: "Fix arguments", Retry: "No"},
			{When: "Missing API key or base URL", Code: clierror.CodeAuth, Notes: "`Run 'cisco-aid-cli onboard'…`", Retry: "No"},
			{When: "Base URL host not `*.cisco.com`", Code: clierror.CodeAuth, Notes: "Use a regional endpoint", Retry: "No"},
			{When: "HTTP 400 from API", Code: clierror.CodeUsage, Notes: "Check command arguments", Retry: "No"},
			{When: "HTTP 401 / 403", Code: clierror.CodeAuth, Notes: "Check API key and tenant region", Retry: "No"},
			{When: "HTTP 404 (bad scan/server/BOM id)", Code: clierror.CodeScanFailed, Notes: "Verify id belongs to your tenant", Retry: "No"},
			{When: "HTTP 429 or 5xx", Code: clierror.CodeNetwork, Notes: "Rate limit or transient outage", Retry: "**Yes**"},
			{When: "DNS / TLS / connection refused / request timeout", Code: clierror.CodeNetwork, Notes: "VPN, proxy, `SSL_CERT_FILE`", Retry: "**Yes**"},
			{When: "Scan poll budget exhausted", Code: clierror.CodeNetwork, Notes: "Increase retry env vars (see below)", Retry: "**Yes**"},
			{When: "Unclassified error", Code: clierror.CodeUnexpected, Notes: "—", Retry: "Maybe"},
		},
	}}
}

// CommandScenarios returns per-command handbook sections.
func CommandScenarios() []CommandGroup {
	return []CommandGroup{
		installScenarios(),
		onboardScenarios(),
		configEditScenarios(),
		configShowScenarios(),
		configPathScenarios(),
		configValidateScenarios(),
		modelScanScenarios(),
		mcpScanScenarios(),
		aibomScenarios(),
	}
}

// MultiCommandScenarios returns tables with heterogeneous columns.
func MultiCommandScenarios() []MultiCommandGroup {
	return []MultiCommandGroup{configProfileScenarios()}
}

func installScenarios() CommandGroup {
	return CommandGroup{
		Title:   "`install`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Install or `--check` succeeded", Code: clierror.CodeOK},
			{When: "Positional arguments", Code: clierror.CodeUsage, Notes: "`install` accepts no args"},
			{When: "Cannot resolve install prefix", Code: clierror.CodeUsage, Notes: "Use `--prefix`"},
			{When: "Unsupported OS/arch, bad checksum, release not found", Code: clierror.CodeUsage},
			{When: "Download / network failure", Code: clierror.CodeNetwork, Notes: "Retry"},
			{When: "Other install error", Code: clierror.CodeUnexpected},
		},
	}
}

func onboardScenarios() CommandGroup {
	return CommandGroup{
		Title: "`onboard` / `config init` / `config profile create`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Wizard completed", Code: clierror.CodeOK},
			{When: "Non-TTY (no interactive terminal)", Code: clierror.CodeUsage, Notes: "Set env vars or edit TOML manually"},
			{When: "Invalid profile name", Code: clierror.CodeUsage},
			{When: "Wizard field validation (e.g. base URL)", Code: clierror.CodeUsage},
			{When: "User cancelled (Ctrl+C)", Code: clierror.CodeUnexpected, Notes: "`Cancelled.`"},
			{When: "Save / list profiles failure", Code: clierror.CodeUnexpected},
		},
	}
}

func configEditScenarios() CommandGroup {
	return CommandGroup{
		Title:   "`config edit`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Edit completed", Code: clierror.CodeOK},
			{When: "Non-TTY", Code: clierror.CodeUsage, Notes: "Edit TOML directly"},
			{When: "No config file", Code: clierror.CodeUsage, Notes: "Run `onboard` first"},
			{When: "Invalid / unreadable config", Code: clierror.CodeUsage},
			{When: "User cancelled", Code: clierror.CodeUnexpected},
		},
	}
}

func configShowScenarios() CommandGroup {
	return CommandGroup{
		Title:   "`config show`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Success (TTY browser or `--plain`)", Code: clierror.CodeOK, Notes: "Non-TTY auto-uses plain mode"},
			{When: "Config load error", Code: clierror.CodeUsage},
			{When: "User cancelled (interactive)", Code: clierror.CodeUnexpected},
		},
	}
}

func configPathScenarios() CommandGroup {
	return CommandGroup{
		Title:   "`config path`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Always prints path", Code: clierror.CodeOK},
		},
	}
}

func configValidateScenarios() CommandGroup {
	return CommandGroup{
		Title:   "`config validate`",
		Columns: []string{"Scenario", "Code", "Notes"},
		Rows: []Scenario{
			{When: "Valid", Code: clierror.CodeOK, Notes: "Prints `Configuration is valid.`"},
			{When: "Config read/parse error", Code: clierror.CodeUsage},
			{When: "Missing API key or base URL", Code: clierror.CodeAuth},
			{When: "Invalid base URL host", Code: clierror.CodeAuth},
		},
	}
}

func configProfileScenarios() MultiCommandGroup {
	return MultiCommandGroup{
		Title:   "`config profile`",
		Columns: []string{"Subcommand", "Scenario", "Code"},
		Rows: []MultiRow{
			{Cols: []string{"`list` / `current`", "Success", "0"}},
			{Cols: []string{"`list`", "List error", "1"}},
			{Cols: []string{"`use <name>`", "Success", "0"}},
			{Cols: []string{"`use <name>`", "Invalid / missing name", "2"}},
			{Cols: []string{"`use <name>`", "Write failure", "1"}},
			{Cols: []string{"`create <name>`", "Same as wizard", "*see onboard*"}},
			{Cols: []string{"`create <name>`", "Profile exists (no `--force`)", "2"}},
			{Cols: []string{"`delete <name>`", "Success", "0"}},
			{Cols: []string{"`delete <name>`", "Cannot delete default / not found", "2"}},
		},
	}
}

func modelScanScenarios() CommandGroup {
	return CommandGroup{Title: "`model-scan`"}
}

func mcpScanScenarios() CommandGroup {
	return CommandGroup{Title: "`mcp-scan`"}
}
func ModelScanRows() []MultiRow {
	return []MultiRow{
		{Cols: []string{"`(parent)`", "No subcommand — show help", "0", "Intentional; unknown subcommand exits 2"}},
		{Cols: []string{"`run`", "Deprecated", "2", "Use `trigger`"}},
		{Cols: []string{"`trigger`", "Success, policy pass", "0", "Raw API JSON on stdout"}},
		{Cols: []string{"`trigger`", "Missing `file-path` / `repo-url` / both set", "2", ""}},
		{Cols: []string{"`trigger`", "Invalid `--fail-on`, `--block-on`, `--output`", "2", ""}},
		{Cols: []string{"`trigger`", "Auth failure", "3", ""}},
		{Cols: []string{"`trigger`", "File missing or unsupported type", "1", ""}},
		{Cols: []string{"`trigger`", "Repo not accessible (API validation)", "2", ""}},
		{Cols: []string{"`trigger`", "Poll timeout", "4", ""}},
		{Cols: []string{"`trigger`", "Scan FAILED / CANCELED", "5", ""}},
		{Cols: []string{"`trigger`", "Policy fail / block", "10 / 20", ""}},
		{Cols: []string{"`info`", "Same gating as `trigger`", "0 / 5 / 10 / 20", "Requires `--scan-id`"}},
		{Cols: []string{"`list`", "Success", "0", "No gating"}},
		{Cols: []string{"`list`", "Auth / API errors", "3 / 4 / 5 / …", "Global table"}},
	}
}

func McpScanRows() []MultiRow {
	return []MultiRow{
		{Cols: []string{"`register`", "Success", "0", ""}},
		{Cols: []string{"`register`", "Missing URL, bad transport/auth flags", "2", ""}},
		{Cols: []string{"`run`", "Success, policy pass", "0", "Polls to completion"}},
		{Cols: []string{"`run`", "Poll timeout", "4", "MCP retry env vars"}},
		{Cols: []string{"`run`", "Scan FAILED / CANCELLED", "5", ""}},
		{Cols: []string{"`run`", "Policy fail / block", "10 / 20", ""}},
		{Cols: []string{"`info --scan-id`", "Same gating as `run`", "0 / 5 / 10 / 20", ""}},
		{Cols: []string{"`info --server-id`", "Success", "0", "No gating"}},
		{Cols: []string{"`info`", "Neither or both id flags", "2", ""}},
		{Cols: []string{"`trigger` / `list` / `capabilities`", "Success", "0", "No gating"}},
		{Cols: []string{"`delete`", "User declined confirmation", "0", "Prints `Aborted.`"}},
		{Cols: []string{"`delete`", "Deleted", "0", ""}},
	}
}

func aibomScenarios() CommandGroup {
	return CommandGroup{
		Title: "`ai-bom`",
		Intro: "No policy gating on any `ai-bom` subcommand.",
	}
}

func AibomRows() []MultiRow {
	return []MultiRow{
		{Cols: []string{"`generate`", "Success", "0", "AIBOM JSON on stdout"}},
		{Cols: []string{"`generate`", "Missing `--source` or `--llm-model`", "2", ""}},
		{Cols: []string{"`generate`", "No LLM credential", "2", "`config edit` → Local AIBOM"}},
		{Cols: []string{"`generate`", "Embedded Python unavailable (dev build)", "1", "Use release binary or `AID_CLI_PYTHON`"}},
		{Cols: []string{"`generate`", "`aibom analyze` subprocess failed", "5", "Check LLM settings"}},
		{Cols: []string{"`run`", "Generate + submit success", "0", ""}},
		{Cols: []string{"`run`", "Analyzer failure", "5", ""}},
		{Cols: []string{"`run`", "Auth failure on submit", "3", ""}},
		{Cols: []string{"`submit`", "Missing `--file` or bad JSON", "2 / 1", ""}},
		{Cols: []string{"`info` / `delete` / `components`", "Missing required id flag", "2", ""}},
		{Cols: []string{"`delete`", "User declined confirmation", "0", ""}},
		{Cols: []string{"`list` / `info` / `components` / `summary`", "API success", "0", ""}},
	}
}
