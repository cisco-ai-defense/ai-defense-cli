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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	want := filepath.Join(home, homeConfigDirName, configFileName)
	got := DefaultConfigPath()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if strings.Contains(got, "~") {
		t.Fatalf("path must not contain tilde literal: %q", got)
	}
}

func TestExpandHomeInConfigPathOverride(t *testing.T) {
	home := filepath.Join(t.TempDir(), "ubuntu-user")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "~/.ai-defense/override.toml")

	want := filepath.Join(home, ".ai-defense", "override.toml")
	if got := ResolveConfigPath(); got != want {
		t.Fatalf("ResolveConfigPath: got %q want %q", got, want)
	}
	if got := ConfigPathForWrite(); got != want {
		t.Fatalf("ConfigPathForWrite: got %q want %q", got, want)
	}
}

func TestHomeConfigDirUnixLike(t *testing.T) {
	// Simulates macOS/Linux: HOME drives os.UserHomeDir().
	cases := []struct {
		name string
		home string
	}{
		{"macos", filepath.Join(t.TempDir(), "psg07")},
		{"ubuntu", filepath.Join(t.TempDir(), "ubuntu")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.MkdirAll(tc.home, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", tc.home)
			t.Setenv("AI_DEFENSE_CONFIG_PATH", "")

			dir := HomeConfigDir()
			if dir != filepath.Join(tc.home, homeConfigDirName) {
				t.Fatalf("HomeConfigDir: got %q", dir)
			}
			if !filepath.IsAbs(dir) {
				t.Fatal("expected absolute HomeConfigDir")
			}

			path := DefaultConfigPath()
			if err := WriteConfig(path, map[string]interface{}{"default": map[string]interface{}{}}); err != nil {
				t.Fatal(err)
			}
			if !fileExists(path) {
				t.Fatalf("expected file at %q", path)
			}
		})
	}
}

func TestResolveConfigPath(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "")

	if got := ResolveConfigPath(); got != DefaultConfigPath() {
		t.Fatalf("default: got %q want %q", got, DefaultConfigPath())
	}

	homePath := DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(homePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homePath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolveConfigPath(); got != homePath {
		t.Fatalf("home: got %q want %q", got, homePath)
	}

	override := filepath.Join(dir, "custom.toml")
	t.Setenv("AI_DEFENSE_CONFIG_PATH", override)
	if got := ResolveConfigPath(); got != override {
		t.Fatalf("override: got %q want %q", got, override)
	}
}

func TestResolveConfigPathLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))

	xdgPath := legacyXDGConfigPath()
	if err := os.MkdirAll(filepath.Dir(xdgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xdgPath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolveConfigPath(); got != xdgPath {
		t.Fatalf("legacy xdg: got %q want %q", got, xdgPath)
	}
}

func TestExistingConfigPath(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "")

	if _, ok := ExistingConfigPath(); ok {
		t.Fatal("expected none")
	}

	homePath := DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(homePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(homePath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, ok := ExistingConfigPath()
	if !ok || got != homePath {
		t.Fatalf("got %q ok=%v want %q", got, ok, homePath)
	}
}

func TestConfigLocationLabel(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	def := DefaultConfigPath()
	if !strings.Contains(ConfigLocationLabel(def), "Profile default") {
		t.Fatalf("home label: %q", ConfigLocationLabel(def))
	}
	if !strings.Contains(shortenHome(filepath.Join(home, homeConfigDirName, configFileName)), "~") {
		t.Fatal("expected home shortened")
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.toml")
	data := map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  "secret",
			"base_url": "https://us.api.aidefense.security.cisco.com",
		},
	}
	if err := WriteConfig(path, data); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
	dirSt, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirSt.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o", dirSt.Mode().Perm())
	}
	m := NewManagerFromPath(path)
	if got := m.NestedString("default", "api_key"); got != "secret" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactSecrets(t *testing.T) {
	data := map[string]interface{}{
		"default": map[string]interface{}{
			"api_key": "k",
			"aibom": map[string]interface{}{
				"llm_api_key": "sk",
			},
		},
	}
	red := RedactSecrets(data)
	def := red["default"].(map[string]interface{})
	if def["api_key"] != "***" {
		t.Fatalf("api_key not redacted")
	}
	aibom := def["aibom"].(map[string]interface{})
	if aibom["llm_api_key"] != "***" {
		t.Fatalf("llm_api_key not redacted")
	}
	if data["default"].(map[string]interface{})["api_key"] != "k" {
		t.Fatal("original data mutated")
	}
}
