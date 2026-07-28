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

package aibom

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

// fakeSource implements spec.ValueSource for TOML-only resolution tests.
type fakeSource map[string]string

func (f fakeSource) NestedString(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	return f[keys[len(keys)-1]]
}

// cliWithConfig returns a CLI whose config manager loads a temp config.toml.
func cliWithConfig(t *testing.T, data map[string]interface{}) *cmdutil.CLI {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := config.WriteConfig(path, data); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_DEFENSE_CONFIG_PATH", path)
	return &cmdutil.CLI{
		ConfigManager: func() *config.Manager { return config.NewManagerFromPath(path) },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
	}
}

// fakePython points AID_CLI_PYTHON at a stub interpreter that exits with code.
func fakePython(t *testing.T, code int) {
	t.Helper()
	py := filepath.Join(t.TempDir(), "fake-python")
	if err := os.WriteFile(py, []byte(fmt.Sprintf("#!/bin/sh\nexit %d\n", code)), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AID_CLI_PYTHON", py)
}

func clearAIBOMEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"AIBOM_LLM_MODEL", "AIBOM_LLM_PROVIDER", "AIBOM_LLM_API_KEY",
		"AIBOM_LLM_API_BASE", "AIBOM_LLM_API_VERSION",
		"AIBOM_POST_URL", "AI_DEFENSE_API_KEY", "AI_DEFENSE_BASE_URL",
	} {
		t.Setenv(k, "")
	}
}

func TestHasFlag(t *testing.T) {
	args := []string{"upload", "report.json", "--post-url", "https://x", "--format=json"}
	if !hasFlag(args, "--post-url") {
		t.Error("expected --post-url (space form) detected")
	}
	if !hasFlag(args, "--format") {
		t.Error("expected --format (= form) detected")
	}
	if hasFlag(args, "--ai-defense-api-key") {
		t.Error("did not expect --ai-defense-api-key")
	}
}

func TestFirstNonFlag(t *testing.T) {
	if got := firstNonFlag([]string{"--raw-json", "upload", "f.json"}); got != "upload" {
		t.Fatalf("got %q", got)
	}
	if got := firstNonFlag([]string{"--raw-json"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestNormalizeHelpFlags(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"--h"}, []string{"--help"}},
		{[]string{"--he"}, []string{"--help"}},
		{[]string{"--hel"}, []string{"--help"}},
		{[]string{"--help"}, []string{"--help"}},
		{[]string{"-h"}, []string{"-h"}},
		{[]string{".", "--h"}, []string{".", "--help"}},
		// non-help flags that merely start with "--h" are untouched.
		{[]string{"--host", "x"}, []string{"--host", "x"}},
		{nil, []string{}},
	}
	for _, tc := range cases {
		got := normalizeHelpFlags(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("normalizeHelpFlags(%v) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("normalizeHelpFlags(%v) = %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestNormalizeHelpFlags_DoesNotMutateInput(t *testing.T) {
	in := []string{"--h"}
	_ = normalizeHelpFlags(in)
	if in[0] != "--h" {
		t.Fatalf("input mutated: %v", in)
	}
}

func TestIsReportUpload(t *testing.T) {
	cases := map[string]bool{
		"upload f.json": true,
		"UPLOAD f.json": true,
		"show f.json":   false,
		"f.json":        false,
		"--raw-json":    false,
	}
	for in, want := range cases {
		if got := isReportUpload(strings.Fields(in)); got != want {
			t.Errorf("isReportUpload(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAibomAnalysisURL(t *testing.T) {
	want := "https://us.api.aidefense.security.cisco.com/api/ai-defense/v1/aibom/analysis"
	if got := aibomAnalysisURL("https://us.api.aidefense.security.cisco.com/"); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestAibomEnvInjection(t *testing.T) {
	clearAIBOMEnv(t)
	src := fakeSource{"llm_model": "gpt-x", "llm_api_key": "sk-toml"}

	env := aibomEnvInjection(aibomSpec(), src)
	if !containsEnv(env, "AIBOM_LLM_MODEL=gpt-x") || !containsEnv(env, "AIBOM_LLM_API_KEY=sk-toml") {
		t.Fatalf("expected model+key injected, got %v", env)
	}
	if containsEnv(env, "AIBOM_LLM_PROVIDER=") {
		t.Fatalf("did not expect empty provider, got %v", env)
	}

	// An exported env var wins: that key is not re-injected from TOML.
	t.Setenv("AIBOM_LLM_MODEL", "env-model")
	env = aibomEnvInjection(aibomSpec(), src)
	for _, e := range env {
		if strings.HasPrefix(e, "AIBOM_LLM_MODEL=") {
			t.Fatalf("env-set var should not be re-injected, got %v", env)
		}
	}
	if !containsEnv(env, "AIBOM_LLM_API_KEY=sk-toml") {
		t.Fatalf("expected api key still injected, got %v", env)
	}
}

func TestAibomLLMEnv_FromConfig(t *testing.T) {
	clearAIBOMEnv(t)
	cli := cliWithConfig(t, map[string]interface{}{
		"default": map[string]interface{}{
			"aibom": map[string]interface{}{
				"llm_model":   "gpt-5.4",
				"llm_api_key": "from-toml",
			},
		},
	})
	env, err := aibomLLMEnv(cli, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsEnv(env, "AIBOM_LLM_MODEL=gpt-5.4") || !containsEnv(env, "AIBOM_LLM_API_KEY=from-toml") {
		t.Fatalf("expected TOML LLM creds injected, got %v", env)
	}
}

func TestConfigSource(t *testing.T) {
	if configSource(nil) != nil {
		t.Fatal("nil cli should yield nil source")
	}
	if configSource(&cmdutil.CLI{}) != nil {
		t.Fatal("cli without manager should yield nil source")
	}
	nilMgr := &cmdutil.CLI{ConfigManager: func() *config.Manager { return nil }}
	if configSource(nilMgr) != nil {
		t.Fatal("cli whose manager is nil should yield nil source")
	}
	cli := cliWithConfig(t, map[string]interface{}{"default": map[string]interface{}{}})
	if configSource(cli) == nil {
		t.Fatal("cli with manager should yield a source")
	}
}

func TestReportUploadEnv_NotUpload(t *testing.T) {
	env, err := reportUploadEnv(testCLI(t), []string{"show", "report.json"})
	if err != nil || env != nil {
		t.Fatalf("show should be a no-op, got env=%v err=%v", env, err)
	}
}

func TestReportUploadEnv_UserProvidedWins(t *testing.T) {
	clearAIBOMEnv(t)
	args := []string{"upload", "f.json", "--post-url", "https://x.cisco.com", "--ai-defense-api-key", "k"}
	env, err := reportUploadEnv(testCLI(t), args)
	if err != nil || env != nil {
		t.Fatalf("fully user-provided upload should not inject, got env=%v err=%v", env, err)
	}
}

func TestReportUploadEnv_DerivesFromConfig(t *testing.T) {
	clearAIBOMEnv(t)
	cli := cliWithConfig(t, map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  "tenant-key",
			"base_url": "https://us.api.aidefense.security.cisco.com",
		},
	})
	env, err := reportUploadEnv(cli, []string{"upload", "f.json"})
	if err != nil {
		t.Fatal(err)
	}
	wantURL := "AIBOM_POST_URL=https://us.api.aidefense.security.cisco.com/api/ai-defense/v1/aibom/analysis"
	if !containsEnv(env, wantURL) || !containsEnv(env, "AI_DEFENSE_API_KEY=tenant-key") {
		t.Fatalf("expected derived post URL + key, got %v", env)
	}
}

func TestReportUploadEnv_OnlyURLWhenKeyInEnv(t *testing.T) {
	clearAIBOMEnv(t)
	t.Setenv("AI_DEFENSE_API_KEY", "env-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://eu.api.aidefense.security.cisco.com")
	env, err := reportUploadEnv(testCLI(t), []string{"upload", "f.json"})
	if err != nil {
		t.Fatal(err)
	}
	if containsEnvPrefix(env, "AI_DEFENSE_API_KEY=") {
		t.Fatalf("api key already in env should not be re-injected, got %v", env)
	}
	if !containsEnv(env, "AIBOM_POST_URL=https://eu.api.aidefense.security.cisco.com/api/ai-defense/v1/aibom/analysis") {
		t.Fatalf("expected derived post URL, got %v", env)
	}
}

func TestReportUploadEnv_AuthError(t *testing.T) {
	clearAIBOMEnv(t)
	_, err := reportUploadEnv(testCLI(t), []string{"upload", "f.json"})
	if err == nil {
		t.Fatal("expected auth error when no credentials are configured")
	}
}

func TestNewPassthroughCmd_Shape(t *testing.T) {
	cmd := newPassthroughCmd(testCLI(t), "analyze [SOURCES...]", "short help", "analyze", nil)
	if cmd.Name() != "analyze" {
		t.Fatalf("name = %q", cmd.Name())
	}
	if cmd.Short != "short help" {
		t.Fatalf("short = %q", cmd.Short)
	}
	if !cmd.DisableFlagParsing {
		t.Fatal("expected DisableFlagParsing")
	}
	if cmd.Annotations[PassthroughAnnotation] != "true" {
		t.Fatal("expected passthrough annotation")
	}
}

func TestPassthroughRunE_BuildEnvError(t *testing.T) {
	boom := errors.New("boom")
	cmd := newPassthroughCmd(testCLI(t), "analyze", "s", "analyze",
		func(*cmdutil.CLI, []string) ([]string, error) { return nil, boom })
	if err := cmd.RunE(cmd, []string{"."}); !errors.Is(err, boom) {
		t.Fatalf("expected buildEnv error to propagate, got %v", err)
	}
}

func TestPassthroughRunE_Success(t *testing.T) {
	fakePython(t, 0)
	cmd := newPassthroughCmd(testCLI(t), "analyze", "s", "analyze", nil)
	if err := cmd.RunE(cmd, []string{"--help"}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRunAibom_ExitCodePropagated(t *testing.T) {
	fakePython(t, 7)
	err := runAibom("analyze", []string{"."}, nil)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T (%v)", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.ExitCode())
	}
}

func TestRunAibom_InterpreterMissing(t *testing.T) {
	t.Setenv("AID_CLI_PYTHON", filepath.Join(t.TempDir(), "does-not-exist"))
	if err := runAibom("analyze", nil, nil); err == nil {
		t.Fatal("expected interpreter resolution error")
	}
}

func TestRunAibomPassthrough_Success(t *testing.T) {
	fakePython(t, 0)
	if err := runAibomPassthrough("analyze", []string{"--help"}, nil); err != nil {
		t.Fatalf("expected nil on success, got %v", err)
	}
}

func TestNewCmd_RegistersUnion(t *testing.T) {
	root := NewCmd(testCLI(t))
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}
	for _, name := range []string{
		"analyze", "report", "watch", "diff", "benchmark", "kb", "cache", "plugin",
		"list", "info", "delete", "components", "summary",
	} {
		if !got[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestNewCmd_PassthroughAnnotations(t *testing.T) {
	root := NewCmd(testCLI(t))
	annotated := map[string]bool{
		"analyze": true, "report": true, "watch": true, "diff": true,
		"benchmark": true, "kb": true, "cache": true, "plugin": true,
	}
	for _, c := range root.Commands() {
		isAnnotated := c.Annotations[PassthroughAnnotation] == "true"
		if annotated[c.Name()] != isAnnotated {
			t.Errorf("%s: passthrough annotation = %v, want %v", c.Name(), isAnnotated, annotated[c.Name()])
		}
	}
}

func containsEnv(env []string, kv string) bool {
	for _, e := range env {
		if e == kv {
			return true
		}
	}
	return false
}

func containsEnvPrefix(env []string, prefix string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}
