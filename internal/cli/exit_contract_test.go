package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cliexitdoc"
)

func TestRunModelScanBareShowsHelpExitOK(t *testing.T) {
	want, ok := cliexitdoc.ContractCode(cliexitdoc.ModelScanContract(), "No subcommand")
	if !ok {
		t.Fatal("registry missing bare model-scan row")
	}
	if code := runWithArgs(t, "model-scan"); code != want {
		t.Fatalf("bare model-scan exit = %d, want %d (registry)", code, want)
	}
}

func TestExitContract_ModelScanTrigger(t *testing.T) {
	rows := cliexitdoc.ModelScanContract()

	t.Run("deprecated run", func(t *testing.T) {
		want, ok := cliexitdoc.ContractCode(rows, "Deprecated")
		if !ok {
			t.Fatal("registry missing deprecated run row")
		}
		if code := runWithArgs(t, "model-scan", "run"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})

	t.Run("trigger missing source", func(t *testing.T) {
		t.Setenv("AI_DEFENSE_API_KEY", "test-key")
		t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")
		want, ok := cliexitdoc.ContractCode(rows, "Missing")
		if !ok {
			t.Fatal("registry missing trigger missing-source row")
		}
		if code := runWithArgs(t, "model-scan", "trigger"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})

	t.Run("trigger invalid fail-on", func(t *testing.T) {
		t.Setenv("AI_DEFENSE_API_KEY", "test-key")
		t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")
		want, ok := cliexitdoc.ContractCode(rows, "Invalid")
		if !ok {
			t.Fatal("registry missing trigger invalid-flag row")
		}
		if code := runWithArgs(t, "model-scan", "trigger", "--file-path", "model.pth", "--fail-on", "bogus"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})
}

func TestExitContract_ConfigValidate(t *testing.T) {
	rows := cliexitdoc.ConfigValidateContract()

	t.Run("valid", func(t *testing.T) {
		path := writeValidateConfig(t, `[default]
api_key = "k"
base_url = "https://us.api.cisco.com"`)
		t.Setenv("AI_DEFENSE_CONFIG_PATH", path)
		t.Setenv("AI_DEFENSE_API_KEY", "")
		t.Setenv("AI_DEFENSE_BASE_URL", "")
		want, ok := cliexitdoc.ContractCode(rows, "Valid")
		if !ok {
			t.Fatal("registry missing valid row")
		}
		if code := runWithArgs(t, "config", "validate"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		path := writeValidateConfig(t, "not valid toml [[[")
		t.Setenv("AI_DEFENSE_CONFIG_PATH", path)
		t.Setenv("AI_DEFENSE_API_KEY", "")
		t.Setenv("AI_DEFENSE_BASE_URL", "")
		want, ok := cliexitdoc.ContractCode(rows, "read/parse")
		if !ok {
			t.Fatal("registry missing parse error row")
		}
		if code := runWithArgs(t, "config", "validate"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		path := writeValidateConfig(t, `[default]
base_url = "https://us.api.cisco.com"`)
		t.Setenv("AI_DEFENSE_CONFIG_PATH", path)
		t.Setenv("AI_DEFENSE_API_KEY", "")
		t.Setenv("AI_DEFENSE_BASE_URL", "")
		want, ok := cliexitdoc.ContractCode(rows, "Missing API key")
		if !ok {
			t.Fatal("registry missing auth row")
		}
		if code := runWithArgs(t, "config", "validate"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})

	t.Run("invalid base url host", func(t *testing.T) {
		path := writeValidateConfig(t, `[default]
api_key = "k"
base_url = "https://evil.example.com"`)
		t.Setenv("AI_DEFENSE_CONFIG_PATH", path)
		t.Setenv("AI_DEFENSE_API_KEY", "")
		t.Setenv("AI_DEFENSE_BASE_URL", "")
		want, ok := cliexitdoc.ContractCode(rows, "Invalid base URL")
		if !ok {
			t.Fatal("registry missing invalid host row")
		}
		if code := runWithArgs(t, "config", "validate"); code != want {
			t.Fatalf("exit = %d, want %d", code, want)
		}
	})
}

func writeValidateConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExitContract_ModelScanUnknownSubcommand(t *testing.T) {
	// Bare parent is exit 0; unknown subcommand remains usage (2) per global table.
	if code := runWithArgs(t, "model-scan", "definitely-not-a-subcommand"); code != clierror.CodeUsage {
		t.Fatalf("unknown subcommand exit = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestExitContract_McpScanRegister(t *testing.T) {
	rows := cliexitdoc.McpScanContract()
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")
	want, ok := cliexitdoc.ContractCode(rows, "Missing URL")
	if !ok {
		t.Fatal("registry missing mcp-scan register usage row")
	}
	if code := runWithArgs(t, "mcp-scan", "register", "--name", "srv"); code != want {
		t.Fatalf("exit = %d, want %d", code, want)
	}
}

func TestExitContract_AibomInfoMissingID(t *testing.T) {
	rows := cliexitdoc.AibomContract()
	want, ok := cliexitdoc.ContractCode(rows, "Missing required id flag")
	if !ok {
		t.Fatal("registry missing ai-bom info usage row")
	}
	if code := runWithArgs(t, "ai-bom", "info"); code != want {
		t.Fatalf("exit = %d, want %d", code, want)
	}
}

func TestRunAibomBareShowsHelpExitOK(t *testing.T) {
	// Bare `ai-bom` shows help and exits 0 (NoArgs + RunE: Help).
	if code := runWithArgs(t, "ai-bom"); code != clierror.CodeOK {
		t.Fatalf("bare ai-bom exit = %d, want %d", code, clierror.CodeOK)
	}
}

func TestExitContract_AibomUnknownSubcommand(t *testing.T) {
	// An unknown subcommand and the removed generate/submit/run commands must be
	// usage errors (2), not a silent help/exit-0, so typos and stale invocations
	// do not look successful.
	for _, sub := range []string{"definitely-not-a-subcommand", "generate", "submit", "run"} {
		t.Run(sub, func(t *testing.T) {
			if code := runWithArgs(t, "ai-bom", sub); code != clierror.CodeUsage {
				t.Fatalf("ai-bom %s exit = %d, want %d", sub, code, clierror.CodeUsage)
			}
		})
	}
}
