package mcpscan

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
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

func TestRegisterCmd_UsageValidation(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://example.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "register", "--name", "srv")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRegisterCmd_ApiKeyAuthMissing(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "register", "--url", "https://mcp.example.com/sse", "--name", "srv", "--auth-type", "api-key")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRegisterCmd_InvalidTransport(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "register", "--url", "https://mcp.example.com/sse", "--name", "srv", "--transport", "grpc")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRegisterCmd_ConflictingScanFlags(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "register",
		"--url", "https://mcp.example.com/sse",
		"--name", "srv",
		"--scan-enabled",
		"--no-scan-enabled",
	)
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRegisterCmd_OAuthMissingFields(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "register", "--url", "https://mcp.example.com/sse", "--name", "srv", "--auth-type", "oauth")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestInfoCmd_UsageValidation(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://example.com")

	root := NewCmd(testCLI(t))

	t.Run("missing id", func(t *testing.T) {
		err := execCmd(t, root, "info")
		if err == nil {
			t.Fatal("expected usage error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})

	t.Run("both ids", func(t *testing.T) {
		err := execCmd(t, root, "info", "--server-id", "a", "--scan-id", "b")
		if err == nil {
			t.Fatal("expected usage error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})
}

func TestInfoCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"server_id":"srv-1"}`))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "info", "--server-id", "srv-1"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "srv-1") {
		t.Fatalf("expected server JSON on stdout, got %q", out)
	}
}

func TestInfoCmd_ScanID_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"scan_id":"scan-1","status":"COMPLETED"}`))
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
		w.Write([]byte(`{"items":[{"server_id":"s1"}]}`))
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

func TestDeleteCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "delete", "--server-id", "srv-1", "-y"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected delete JSON, got %q", out)
	}
}

func TestCapabilitiesCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("capability_type") != "TOOL" {
			t.Errorf("expected TOOL filter, got %q", r.URL.Query().Get("capability_type"))
		}
		w.Write([]byte(`{"items":[{"name":"search"}]}`))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "capabilities", "--server-id", "srv-1", "--type", "TOOL"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "search") {
		t.Fatalf("expected capabilities JSON, got %q", out)
	}
}

func TestRegisterCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"server_id":"new-srv"}`))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "register",
			"--url", "https://mcp.example.com/sse",
			"--name", "Test",
		); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "new-srv") {
		t.Fatalf("expected register response, got %q", out)
	}
}

func TestTriggerCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "srv-1") {
			t.Errorf("expected server id in path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "trigger", "--server-id", "srv-1"); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "scan_triggered") {
		t.Fatalf("expected trigger response, got %q", out)
	}
}

func setFastMCPPollEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AI_DEFENSE_MCP_WAIT_TIME_SECS", "0")
	t.Setenv("AI_DEFENSE_MCP_RETRY_COUNT", "5")
}

func TestRunCmd_HTTP(t *testing.T) {
	setFastMCPPollEnv(t)
	var statusPolls int
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/mcp/servers/scan"):
			w.Write([]byte(`{"scan_id":"run-scan-1"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "run-scan-1"):
			statusPolls++
			if statusPolls < 2 {
				w.Write([]byte(`{"scan_id":"run-scan-1","status":"PENDING"}`))
			} else {
				w.Write([]byte(`{"scan_id":"run-scan-1","status":"COMPLETED"}`))
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	})
	root := NewCmd(cli)
	out := captureStdout(t, func() {
		if err := execCmd(t, root, "run",
			"--url", "https://mcp.example.com/sse",
			"--name", "Test Server",
		); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "run-scan-1") {
		t.Fatalf("expected scan JSON on stdout, got %q", out)
	}
}

func TestRunCmd_PollTimeout(t *testing.T) {
	t.Setenv("AI_DEFENSE_MCP_WAIT_TIME_SECS", "0")
	t.Setenv("AI_DEFENSE_MCP_RETRY_COUNT", "2")
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Write([]byte(`{"scan_id":"run-scan-1"}`))
			return
		}
		w.Write([]byte(`{"scan_id":"run-scan-1","status":"PENDING"}`))
	})
	root := NewCmd(cli)
	err := execCmd(t, root, "run",
		"--url", "https://mcp.example.com/sse",
		"--name", "Test Server",
	)
	if err == nil {
		t.Fatal("expected poll timeout error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeNetwork {
		t.Fatalf("exit code = %d, want %d (poll timeout)", code, clierror.CodeNetwork)
	}
}

func TestRunCmd_InvalidThreshold(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "run",
		"--url", "https://mcp.example.com/sse",
		"--name", "Test Server",
		"--fail-on", "bogus",
	)
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestInfoCmd_InvalidThreshold(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	root := NewCmd(testCLI(t))
	err := execCmd(t, root, "info", "--scan-id", "scan-1", "--fail-on", "bogus")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}
