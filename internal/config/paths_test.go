package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXDGConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	got := XDGConfigPath()
	want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "aid-cisco-cli", "config.toml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPathForLocation(t *testing.T) {
	p, err := PathForLocation(LocationXDG, "")
	if err != nil {
		t.Fatal(err)
	}
	if p != XDGConfigPath() {
		t.Fatalf("got %q", p)
	}

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	p, err = PathForLocation(LocationProject, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, ".config.toml")
	if p != want {
		evalP, _ := filepath.EvalSymlinks(p)
		evalWant, _ := filepath.EvalSymlinks(want)
		if evalP != evalWant {
			t.Fatalf("got %q want %q", p, want)
		}
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
