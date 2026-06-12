package mcpscan

import (
	"encoding/json"
	"os"
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
	return &cmdutil.ScanResultOptions{FailOn: "low", BlockOn: "critical", Output: "json"}
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
