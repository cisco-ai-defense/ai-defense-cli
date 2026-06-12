package modelscan

import (
	"encoding/json"
	"os"
	"testing"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
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

func infoWith(status modelscansdk.ScanStatus, sevs ...modelscansdk.Severity) modelscansdk.ScanStatusInfo {
	var items []modelscansdk.ThreatInfo
	for _, s := range sevs {
		items = append(items, modelscansdk.ThreatInfo{Severity: s})
	}
	return modelscansdk.ScanStatusInfo{
		ScanID: "scan-1",
		Status: status,
		AnalysisResults: modelscansdk.AnalysisResult{
			Items: []modelscansdk.FileInfo{{
				Name: "m.pth",
				Threats: modelscansdk.ThreatInfoList{
					Items: []modelscansdk.Technique{{
						Items: []modelscansdk.SubTechnique{{Items: items}},
					}},
				},
			}},
		},
	}
}

func defaultOpts() *cmdutil.ScanResultOptions {
	return &cmdutil.ScanResultOptions{FailOn: "low", BlockOn: "critical", Output: "json"}
}

func TestEmitScanResultExitCodes(t *testing.T) {
	silenceStdout(t)
	tests := []struct {
		name     string
		info     modelscansdk.ScanStatusInfo
		wantCode int // -1 means nil error
	}{
		{"clean pass", infoWith(modelscansdk.ScanStatusCompleted), -1},
		{"low fails", infoWith(modelscansdk.ScanStatusCompleted, modelscansdk.SeverityLow), clierror.CodeFail},
		{"critical blocks", infoWith(modelscansdk.ScanStatusCompleted, modelscansdk.SeverityCritical), clierror.CodeBlock},
		{"failed status", infoWith(modelscansdk.ScanStatusFailed), clierror.CodeScanFailed},
		{"canceled status", infoWith(modelscansdk.ScanStatusCanceled), clierror.CodeScanFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, _ := json.Marshal(modelscansdk.GetScanStatusResponse{ScanStatusInfo: tt.info})
			err := emitScanResult(tt.info, raw, defaultOpts())
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
	err := emitScanResult(infoWith(modelscansdk.ScanStatusCompleted), nil, opts)
	if got := clierror.Classify(err).Code; got != clierror.CodeUsage {
		t.Fatalf("want usage error, got %d", got)
	}
}
