package modelscan

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cliexitdoc"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/testutil/apitest"
)

func testCLI(t *testing.T) *cmdutil.CLI {
	t.Helper()
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")
	return &cmdutil.CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
	}
}

func execCmd(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestTriggerCmd_UsageValidation(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://example.com")

	root := NewCmd(testCLI(t))

	t.Run("missing source", func(t *testing.T) {
		err := execCmd(t, root, "trigger")
		if err == nil {
			t.Fatal("expected usage error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})

	t.Run("both file and repo", func(t *testing.T) {
		err := execCmd(t, root, "trigger", "--file-path", "m.pth", "--repo-url", "hf.co/x/y")
		if err == nil {
			t.Fatal("expected usage error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})
}

func TestInfoCmd_HTTP(t *testing.T) {
	const body = `{"scan_status_info":{"scan_id":"scan-1","status":"COMPLETED","analysis_results":{"items":[],"paging":{"total":0,"count":10,"offset":0}}}}`
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "info", "--scan-id", "scan-1"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "scan-1") {
		t.Fatalf("expected scan JSON on stdout, got %q", out)
	}
}

func TestListCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"scans":{"items":[{"scan_id":"s1"}]}}`))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "list"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "s1") {
		t.Fatalf("expected list JSON, got %q", out)
	}
}

func TestTriggerCmd_InvalidThreshold(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "trigger", "--file-path", "model.pth", "--fail-on", "bogus")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRunCmd_DeprecatedUsageError(t *testing.T) {
	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "run")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
	if !strings.Contains(err.Error(), "deprecated") {
		t.Fatalf("expected deprecation message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "trigger") {
		t.Fatalf("expected trigger hint, got %q", err.Error())
	}
}

func TestModelScanCmd_UnknownSubcommand(t *testing.T) {
	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "definitely-not-a-subcommand")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %q", err.Error())
	}
}

func TestModelScanCmd_BareShowsHelp(t *testing.T) {
	want, ok := cliexitdoc.ContractCode(cliexitdoc.ModelScanContract(), "No subcommand")
	if !ok {
		t.Fatal("registry missing bare model-scan row")
	}
	root := NewCmd(testCLI(t))
	if err := execCmd(t, root); err != nil {
		t.Fatalf("bare model-scan should show help: %v", err)
	}
	if want != 0 {
		t.Fatalf("registry documents exit %d for bare parent; Cobra help returns nil (mapped to 0 at root)", want)
	}
}
