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
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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

func TestTriggerCmd_InvalidUploadConcurrency(t *testing.T) {
	want, ok := cliexitdoc.ContractCode(cliexitdoc.ModelScanContract(), "Invalid `--upload-concurrency`")
	if !ok {
		t.Fatal("registry missing --upload-concurrency row")
	}
	for _, n := range []string{"0", "33", "-4"} {
		t.Run(n, func(t *testing.T) {
			called := false
			cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) { called = true })
			err := execCmd(t, NewCmd(cli), "trigger", "--file-path", "model.pth", "--upload-concurrency", n)
			if code := clierror.Classify(err).Code; code != want {
				t.Fatalf("exit code = %d, want %d (err %v)", code, want, err)
			}
			if called {
				t.Fatal("no API call expected before flag validation passes")
			}
		})
	}
}

func TestTriggerCmd_UploadConcurrencyIgnoredOutsideMultipart(t *testing.T) {
	t.Setenv("AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK", "0")
	filePath := t.TempDir() + "/model.bin"
	if err := os.WriteFile(filePath, []byte("0123456789abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{"repository scan", []string{"--repo-url", "https://huggingface.co/org/model"}},
		{"single-PUT file scan", []string{"--file-path", filePath, "--no-multipart"}},
	}
	for _, tt := range tests {
		for _, n := range []string{"0", "33", "-4"} {
			t.Run(tt.name+"/"+n, func(t *testing.T) {
				var mu sync.Mutex
				var triggered bool
				cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					p := r.URL.Path
					switch {
					case strings.HasSuffix(p, "/scans/register"):
						w.Write([]byte(`{"scan_id":"scan-1"}`))
					case strings.HasSuffix(p, "/scans/scan-1/validate_url"):
						w.Write([]byte(`{"is_accessible":true}`))
					case strings.HasSuffix(p, "/scans/scan-1/objects"):
						w.Write([]byte(`{"object_id":"obj-1","upload_url":"https://uploads.example.com/single"}`))
					case r.Method == http.MethodPut && p == "/single":
					case strings.HasSuffix(p, "/scans/scan-1/run"):
						triggered = true
					case r.Method == http.MethodGet && strings.HasSuffix(p, "/scans/scan-1"):
						w.Write([]byte(`{"scan_status_info":{"scan_id":"scan-1","status":"COMPLETED"}}`))
					default:
						t.Errorf("unexpected %s %s", r.Method, p)
					}
				})

				args := append([]string{"trigger", "--fail-on", "none", "--upload-concurrency", n}, tt.args...)
				captureStdout(t, func() {
					if err := execCmd(t, NewCmd(cli), args...); err != nil {
						t.Fatalf("trigger: %v (exit %d)", err, clierror.Classify(err).Code)
					}
				})
				if !triggered {
					t.Fatal("expected the scan to run")
				}
			})
		}
	}
}

func TestTriggerCmd_UploadModes(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantMultipart bool
	}{
		{name: "multipart default", wantMultipart: true},
		{name: "no-multipart flag", args: []string{"--no-multipart"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK", "0")
			filePath := t.TempDir() + "/model.bin"
			if err := os.WriteFile(filePath, []byte("0123456789abcdef"), 0o644); err != nil {
				t.Fatal(err)
			}

			var mu sync.Mutex
			var sawMultipartReq, completed, singlePut bool
			cli := apitest.NewTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				p := r.URL.Path
				switch {
				case strings.HasSuffix(p, "/scans/register"):
					w.Write([]byte(`{"scan_id":"scan-1"}`))
				case strings.HasSuffix(p, "/scans/scan-1/objects"):
					body, _ := io.ReadAll(r.Body)
					if strings.Contains(string(body), `"use_multipart_upload":true`) {
						sawMultipartReq = true
						w.Write([]byte(`{"object_id":"obj-1","multipart_upload":{"upload_id":"u-1","part_size_bytes":"8"}}`))
						return
					}
					w.Write([]byte(`{"object_id":"obj-1","upload_url":"https://uploads.example.com/single"}`))
				case strings.HasSuffix(p, "/multipart/part-urls"):
					w.Write([]byte(`{"parts":[{"part_number":1,"upload_url":"https://model-bucket.s3.us-west-2.amazonaws.com/p1"},{"part_number":2,"upload_url":"https://model-bucket.s3.us-west-2.amazonaws.com/p2"}]}`))
				case r.Method == http.MethodPut && (p == "/p1" || p == "/p2"):
					w.Header().Set("ETag", `"`+p+`"`)
				case r.Method == http.MethodPut && p == "/single":
					singlePut = true
				case strings.HasSuffix(p, "/multipart/complete"):
					completed = true
				case strings.HasSuffix(p, "/scans/scan-1/run"):
				case r.Method == http.MethodGet && strings.HasSuffix(p, "/scans/scan-1"):
					w.Write([]byte(`{"scan_status_info":{"scan_id":"scan-1","status":"COMPLETED"}}`))
				default:
					t.Errorf("unexpected %s %s", r.Method, p)
				}
			})

			args := append([]string{"trigger", "--file-path", filePath, "--fail-on", "none"}, tt.args...)
			out := captureStdout(t, func() {
				if err := execCmd(t, NewCmd(cli), args...); err != nil {
					t.Fatalf("trigger: %v", err)
				}
			})
			if !strings.Contains(out, "scan-1") {
				t.Fatalf("expected scan JSON on stdout, got %q", out)
			}
			if strings.Contains(out, "Uploading") {
				t.Fatalf("progress must never go to stdout: %q", out)
			}
			if sawMultipartReq != tt.wantMultipart || completed != tt.wantMultipart || singlePut == tt.wantMultipart {
				t.Fatalf("multipart req=%v complete=%v single=%v, want multipart=%v", sawMultipartReq, completed, singlePut, tt.wantMultipart)
			}
		})
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
