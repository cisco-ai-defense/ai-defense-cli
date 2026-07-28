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

package modelscan

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
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

func TestEmitScanResultStdoutMode(t *testing.T) {
	info := infoWith(modelscansdk.ScanStatusCompleted)
	raw, err := json.Marshal(modelscansdk.GetScanStatusResponse{ScanStatusInfo: info})
	if err != nil {
		t.Fatal(err)
	}
	base := &cmdutil.ScanResultOptions{FailOn: "none", BlockOn: "none", Output: "json", Raw: true}

	t.Run("raw API default", func(t *testing.T) {
		out := captureStdout(t, func() {
			if err := emitScanResult(info, raw, base); err != nil {
				t.Fatal(err)
			}
		})
		if strings.Contains(out, "envelope_version") {
			t.Fatalf("expected raw API JSON, got envelope: %s", out)
		}
		if !strings.Contains(out, "scan-1") {
			t.Fatalf("expected scan id in raw output: %s", out)
		}
	})

	t.Run("envelope opt in", func(t *testing.T) {
		opts := *base
		opts.Envelope = true
		out := captureStdout(t, func() {
			if err := emitScanResult(info, raw, &opts); err != nil {
				t.Fatal(err)
			}
		})
		if !strings.Contains(out, "envelope_version") {
			t.Fatalf("expected verdict envelope: %s", out)
		}
	})
}

func TestEmitScanResultInvalidThreshold(t *testing.T) {
	silenceStdout(t)
	opts := &cmdutil.ScanResultOptions{FailOn: "bogus", BlockOn: "critical", Output: "json"}
	err := emitScanResult(infoWith(modelscansdk.ScanStatusCompleted), nil, opts)
	if got := clierror.Classify(err).Code; got != clierror.CodeUsage {
		t.Fatalf("want usage error, got %d", got)
	}
}
