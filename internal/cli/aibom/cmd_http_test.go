package aibom

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/testutil/apitest"
)

func execSubcmd(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func captureSubcmdStdout(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err = cmd.Execute()
	_ = w.Close()
	os.Stdout = orig
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestListCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"items":[{"analysis_id":"a1"}]}`))
	})
	root := NewCmd(cli)
	out := captureSubcmdStdout(t, root, "list", "--limit", "5")
	if !strings.Contains(out, "a1") {
		t.Fatalf("expected list JSON on stdout, got %q", out)
	}
}

func TestInfoCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"analysis_id":"bom-1"}`))
	})
	root := NewCmd(cli)
	out := captureSubcmdStdout(t, root, "info", "--analysis-id", "bom-1")
	if !strings.Contains(out, "bom-1") {
		t.Fatalf("expected BOM JSON on stdout, got %q", out)
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
	out := captureSubcmdStdout(t, root, "delete", "--analysis-id", "bom-1", "-y")
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected delete confirmation JSON, got %q", out)
	}
}

func TestComponentsCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"components":[{"name":"model"}]}`))
	})
	root := NewCmd(cli)
	out := captureSubcmdStdout(t, root, "components", "--analysis-id", "bom-1")
	if !strings.Contains(out, "model") {
		t.Fatalf("expected components JSON, got %q", out)
	}
}

func TestSummaryCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"total_boms":3}`))
	})
	root := NewCmd(cli)
	out := captureSubcmdStdout(t, root, "summary")
	if !strings.Contains(out, "total_boms") {
		t.Fatalf("expected summary JSON, got %q", out)
	}
}

func TestSubmitCmd_ReadErrors(t *testing.T) {
	root := NewCmd(testCLI(t))

	t.Run("missing file", func(t *testing.T) {
		t.Setenv("AI_DEFENSE_API_KEY", "k")
		t.Setenv("AI_DEFENSE_BASE_URL", apitest.TestBaseURL)
		err := execSubcmd(t, root, "submit", "--file", filepath.Join(t.TempDir(), "missing.json"))
		if err == nil {
			t.Fatal("expected read error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Setenv("AI_DEFENSE_API_KEY", "k")
		t.Setenv("AI_DEFENSE_BASE_URL", apitest.TestBaseURL)
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := execSubcmd(t, root, "submit", "--file", path)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestSubmitCmd_HTTP(t *testing.T) {
	cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"analysis_id":"new-1"}`))
	})
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, []byte(`{"components":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewCmd(cli)
	out := captureSubcmdStdout(t, root, "submit", "--file", path)
	if !strings.Contains(out, "new-1") {
		t.Fatalf("expected submit response, got %q", out)
	}
}

func TestGenerateCmd_WithFakePython(t *testing.T) {
	py := filepath.Join(t.TempDir(), "fake-python")
	script := `#!/bin/sh
while [ $# -gt 0 ]; do
  if [ "$1" = "--output-file" ]; then
    echo '{"components":[]}' > "$2"
    exit 0
  fi
  shift
done
exit 1
`
	if err := os.WriteFile(py, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AID_CLI_PYTHON", py)

	root := NewCmd(testCLI(t))
	out := captureSubcmdStdout(t, root, "generate",
		"--source", ".",
		"--llm-model", "gpt-4",
		"--llm-api-key", "sk-test",
	)
	if !strings.Contains(out, "components") {
		t.Fatalf("expected generated JSON on stdout, got %q", out)
	}
}

func TestRunCmd_MissingLLMModel(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", apitest.TestBaseURL)
	t.Setenv("AIBOM_LLM_MODEL", "")
	t.Setenv("AIBOM_LLM_API_KEY", "")

	root := NewCmd(testCLI(t))
	err := execSubcmd(t, root, "run", "--source", ".", "--llm-api-key", "sk-test")
	if err == nil {
		t.Fatal("expected usage error for missing llm model")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestGenerateCmd_MissingModel(t *testing.T) {
	t.Setenv("AIBOM_LLM_MODEL", "")
	t.Setenv("AIBOM_LLM_API_KEY", "sk-test")

	root := NewCmd(testCLI(t))
	err := execSubcmd(t, root, "generate", "--source", ".")
	if err == nil {
		t.Fatal("expected usage error for missing llm model")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestGenerateCmd_MissingCredential(t *testing.T) {
	t.Setenv("AIBOM_LLM_MODEL", "gpt-4")
	t.Setenv("AIBOM_LLM_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	root := NewCmd(testCLI(t))
	err := execSubcmd(t, root, "generate", "--source", ".")
	if err == nil {
		t.Fatal("expected usage error for missing LLM credential")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
		t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
	}
}
