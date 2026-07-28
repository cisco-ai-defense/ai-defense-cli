// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package verdict

import (
	"encoding/json"
	"strings"
	"testing"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestParseSeverity(t *testing.T) {
	cases := map[string]Severity{
		"":                        SevNone,
		"SAFE":                    SevNone,
		"NONE_SEVERITY":           SevNone,
		"SEVERITY_UNKNOWN":        SevNone,
		"THREAT_SEVERITY_UNKNOWN": SevNone,
		"low":                     SevLow,
		"LOW":                     SevLow,
		"medium":                  SevMedium,
		"HIGH":                    SevHigh,
		"THREAT_SEVERITY_HIGH":    SevHigh,
		"critical":                SevCritical,
	}
	for in, want := range cases {
		if got := ParseSeverity(in); got != want {
			t.Errorf("ParseSeverity(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseThreshold(t *testing.T) {
	for _, in := range []string{"none", "off", "never", "", "NONE"} {
		th, err := ParseThreshold(in)
		if err != nil {
			t.Fatalf("ParseThreshold(%q) unexpected err: %v", in, err)
		}
		if !th.Disabled {
			t.Errorf("ParseThreshold(%q) should be disabled", in)
		}
	}
	th, err := ParseThreshold("high")
	if err != nil || th.Disabled || th.Sev != SevHigh {
		t.Fatalf("ParseThreshold(high) = %+v, err %v", th, err)
	}
	if _, err := ParseThreshold("bogus"); err == nil {
		t.Fatalf("ParseThreshold(bogus) should error")
	}
}

// modelInfoWithSeverities builds a ScanStatusInfo carrying one threat per given
// severity, all on one file.
func modelInfoWithSeverities(status modelscansdk.ScanStatus, sevs ...modelscansdk.Severity) modelscansdk.ScanStatusInfo {
	var items []modelscansdk.ThreatInfo
	for _, s := range sevs {
		items = append(items, modelscansdk.ThreatInfo{Severity: s, ThreatType: "T", Details: "d"})
	}
	return modelscansdk.ScanStatusInfo{
		ScanID: "scan-1",
		Status: status,
		AnalysisResults: modelscansdk.AnalysisResult{
			Items: []modelscansdk.FileInfo{{
				Name: "model.pth",
				Threats: modelscansdk.ThreatInfoList{
					Items: []modelscansdk.Technique{{
						Items: []modelscansdk.SubTechnique{{
							SubTechniqueName: "ST",
							Items:            items,
						}},
					}},
				},
			}},
		},
	}
}

func TestDecideExitCodes(t *testing.T) {
	failLow, _ := ParseThreshold("low")
	blockCrit, _ := ParseThreshold("critical")

	tests := []struct {
		name     string
		sevs     []modelscansdk.Severity
		failOn   Threshold
		blockOn  Threshold
		wantCode int
		wantDec  Decision
	}{
		{"clean is pass", nil, failLow, blockCrit, clierror.CodeOK, DecisionPass},
		{"low fails by default", []modelscansdk.Severity{modelscansdk.SeverityLow}, failLow, blockCrit, clierror.CodeFail, DecisionFail},
		{"high fails", []modelscansdk.Severity{modelscansdk.SeverityHigh}, failLow, blockCrit, clierror.CodeFail, DecisionFail},
		{"critical blocks", []modelscansdk.Severity{modelscansdk.SeverityCritical}, failLow, blockCrit, clierror.CodeBlock, DecisionBlock},
		{"below fail-on passes", []modelscansdk.Severity{modelscansdk.SeverityLow}, mustTh(t, "high"), blockCrit, clierror.CodeOK, DecisionPass},
		{"fail-on none never gates", []modelscansdk.Severity{modelscansdk.SeverityCritical}, mustTh(t, "none"), mustTh(t, "none"), clierror.CodeOK, DecisionPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := FromModelScan(modelInfoWithSeverities(modelscansdk.ScanStatusCompleted, tt.sevs...))
			code := r.Decide(tt.failOn, tt.blockOn)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if r.Decision != tt.wantDec {
				t.Errorf("decision = %s, want %s", r.Decision, tt.wantDec)
			}
			if r.ExitCode != tt.wantCode {
				t.Errorf("envelope ExitCode = %d, want %d", r.ExitCode, tt.wantCode)
			}
		})
	}
}

func mustTh(t *testing.T, s string) Threshold {
	t.Helper()
	th, err := ParseThreshold(s)
	if err != nil {
		t.Fatalf("ParseThreshold(%q): %v", s, err)
	}
	return th
}

func TestFromModelScanEnvelope(t *testing.T) {
	info := modelInfoWithSeverities(modelscansdk.ScanStatusCompleted,
		modelscansdk.SeverityLow, modelscansdk.SeverityHigh, modelscansdk.SeverityHigh)
	r := FromModelScan(info)
	failLow, _ := ParseThreshold("low")
	blockCrit, _ := ParseThreshold("critical")
	r.Decide(failLow, blockCrit)

	if r.EnvelopeVersion != EnvelopeVersion || r.ScanType != "model-scan" || r.ScanID != "scan-1" {
		t.Errorf("unexpected envelope identity: %+v", r)
	}
	if r.ThreatCount != 3 {
		t.Errorf("threat count = %d, want 3", r.ThreatCount)
	}
	if r.HighestSeverity != "HIGH" {
		t.Errorf("highest = %s, want HIGH", r.HighestSeverity)
	}
	if r.CountsBySeverity["HIGH"] != 2 || r.CountsBySeverity["LOW"] != 1 {
		t.Errorf("counts = %v", r.CountsBySeverity)
	}

	// JSON envelope must contain the contract keys.
	out := r.Render("json")
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("render json invalid: %v", err)
	}
	for _, key := range []string{"envelope_version", "scan_type", "decision", "exit_code", "highest_severity", "counts_by_severity", "threat_count"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("json envelope missing key %q", key)
		}
	}

	text := r.Render("text")
	if !strings.Contains(text, "Decision:") || !strings.Contains(text, "HIGH") {
		t.Errorf("text render missing fields: %q", text)
	}
}

func TestGateError(t *testing.T) {
	failLow, _ := ParseThreshold("low")
	blockCrit, _ := ParseThreshold("critical")

	clean := FromModelScan(modelInfoWithSeverities(modelscansdk.ScanStatusCompleted))
	clean.Decide(failLow, blockCrit)
	if err := clean.GateError(); err != nil {
		t.Errorf("clean scan should not gate, got %v", err)
	}

	blocked := FromModelScan(modelInfoWithSeverities(modelscansdk.ScanStatusCompleted, modelscansdk.SeverityCritical))
	blocked.Decide(failLow, blockCrit)
	gerr := blocked.GateError()
	if gerr == nil {
		t.Fatal("blocked scan should gate")
	}
	cliErr := clierror.Classify(gerr)
	if cliErr.Code != clierror.CodeBlock {
		t.Errorf("gate exit code = %d, want %d", cliErr.Code, clierror.CodeBlock)
	}
}

func TestFromMCPScanAndExtractSeverities(t *testing.T) {
	raw := json.RawMessage(`{
		"scan_id":"mcp-1",
		"status":"COMPLETED",
		"threats":[{"threat_severity":"HIGH"},{"severity":"low"}],
		"nested":{"maxSeverity":"CRITICAL"}
	}`)
	sevs := ExtractMCPSeverities(raw)
	if len(sevs) != 3 {
		t.Fatalf("expected 3 severities extracted, got %d (%v)", len(sevs), sevs)
	}
	r := FromMCPScan("mcp-1", "COMPLETED", sevs)
	failLow, _ := ParseThreshold("low")
	blockCrit, _ := ParseThreshold("critical")
	if code := r.Decide(failLow, blockCrit); code != clierror.CodeBlock {
		t.Errorf("mcp with critical should block, got %d", code)
	}
	if r.ScanType != "mcp-scan" {
		t.Errorf("scan type = %s, want mcp-scan", r.ScanType)
	}
}

func TestExtractMCPSeveritiesIgnoresNonThreat(t *testing.T) {
	raw := json.RawMessage(`{"status":"COMPLETED","severity":"SAFE","other":"HIGH"}`)
	// "severity":"SAFE" maps to SevNone (skipped); "other" is not a severity key.
	if got := ExtractMCPSeverities(raw); len(got) != 0 {
		t.Errorf("expected no severities, got %v", got)
	}
}
