package aibom

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/testutil/apitest"
)

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
