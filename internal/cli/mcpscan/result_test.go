package mcpscan

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func silenceStdout(t *testing.T) {
	t.Helper()
	orig := os.Stdout
	devnull, _ := os.Open(os.DevNull)
	os.Stdout = devnull
	t.Cleanup(func() {
		os.Stdout = orig
		_ = devnull.Close()
	})
}

func defaultOpts() *cmdutil.ScanResultOptions {
	return &cmdutil.ScanResultOptions{
		FailOn: "low", BlockOn: "critical", Output: "json", Raw: true,
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestEmitScanResultMCP(t *testing.T) {
	silenceStdout(t)
	tests := []struct {
		name     string
		status   string
		raw      string
		wantCode int // -1 means nil
	}{
		{"completed no findings passes", "COMPLETED", `{"scan_id":"s","status":"COMPLETED"}`, -1},
		{"completed with high fails", "COMPLETED", `{"status":"COMPLETED","threats":[{"severity":"HIGH"}]}`, clierror.CodeFail},
		{"completed with critical blocks", "COMPLETED", `{"status":"COMPLETED","threats":[{"threat_severity":"CRITICAL"}]}`, clierror.CodeBlock},
		{"failed status", "FAILED", `{"status":"FAILED"}`, clierror.CodeScanFailed},
		{"cancelled status", "CANCELLED", `{"status":"CANCELLED"}`, clierror.CodeScanFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := emitScanResult("s", tt.status, json.RawMessage(tt.raw), defaultOpts())
			if tt.wantCode == -1 {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if got := clierror.Classify(err).Code; got != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestEmitScanResultInvalidThreshold(t *testing.T) {
	silenceStdout(t)
	opts := &cmdutil.ScanResultOptions{FailOn: "bogus", BlockOn: "critical", Output: "json"}
	err := emitScanResult("s", "COMPLETED", json.RawMessage(`{"status":"COMPLETED"}`), opts)
	if got := clierror.Classify(err).Code; got != clierror.CodeUsage {
		t.Fatalf("want usage error, got %d", got)
	}
}

func TestEmitScanResultStdoutMode(t *testing.T) {
	const scanID = "mcp-scan-1"
	raw := json.RawMessage(`{"scan_id":"mcp-scan-1","status":"COMPLETED"}`)
	base := &cmdutil.ScanResultOptions{FailOn: "none", BlockOn: "none", Output: "json", Raw: true}

	t.Run("raw API default", func(t *testing.T) {
		out := captureStdout(t, func() {
			if err := emitScanResult(scanID, "COMPLETED", raw, base); err != nil {
				t.Fatal(err)
			}
		})
		if strings.Contains(out, "envelope_version") {
			t.Fatalf("expected raw API JSON, got envelope: %s", out)
		}
		if !strings.Contains(out, scanID) {
			t.Fatalf("expected scan id in raw output: %s", out)
		}
	})

	t.Run("envelope opt in", func(t *testing.T) {
		opts := *base
		opts.Envelope = true
		out := captureStdout(t, func() {
			if err := emitScanResult(scanID, "COMPLETED", raw, &opts); err != nil {
				t.Fatal(err)
			}
		})
		if !strings.Contains(out, "envelope_version") {
			t.Fatalf("expected verdict envelope: %s", out)
		}
	})
}
