package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager_NoFile(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/path/config.toml")
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("AI_DEFENSE_BASE_URL", "")

	m := NewManager()
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(m.data) != 0 {
		t.Errorf("expected empty data, got %v", m.data)
	}
}

func TestNewManager_LoadsTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.toml")
	content := `[default]
api_key = "test-key-from-toml"
base_url = "https://test.cisco.com"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_DEFENSE_CONFIG_PATH", cfgPath)
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("AI_DEFENSE_BASE_URL", "")

	m := NewManager()
	key, err := m.GetAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "test-key-from-toml" {
		t.Errorf("expected 'test-key-from-toml', got %q", key)
	}
}

func TestGetAPIKey_EnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.toml")
	content := `[default]
api_key = "toml-key"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_DEFENSE_CONFIG_PATH", cfgPath)
	t.Setenv("AI_DEFENSE_API_KEY", "env-key")

	m := NewManager()
	key, err := m.GetAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "env-key" {
		t.Errorf("expected env key 'env-key', got %q", key)
	}
}

func TestNewManager_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(cfgPath, []byte("not valid toml [[["), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_DEFENSE_CONFIG_PATH", cfgPath)
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("AI_DEFENSE_BASE_URL", "")

	m := NewManager()
	_, err := m.GetAPIKey()
	if err == nil {
		t.Fatal("expected error for invalid TOML config")
	}
	if !strings.Contains(err.Error(), "invalid config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAPIKey_MissingReturnsError(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")
	t.Setenv("AI_DEFENSE_API_KEY", "")

	m := NewManager()
	_, err := m.GetAPIKey()
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestGetBaseURL_EnvOverridesConfig(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://env.cisco.com")

	m := NewManager()
	url, err := m.GetBaseURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://env.cisco.com" {
		t.Errorf("expected env URL, got %q", url)
	}
}

func TestGetBaseURL_MissingReturnsError(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")
	t.Setenv("AI_DEFENSE_BASE_URL", "")

	m := NewManager()
	_, err := m.GetBaseURL()
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func TestGetRepoToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.toml")
	content := `[default.huggingface]
token = "hf-test-token"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_DEFENSE_CONFIG_PATH", cfgPath)

	m := NewManager()
	if got := m.GetRepoToken(); got != "hf-test-token" {
		t.Errorf("expected 'hf-test-token', got %q", got)
	}
}

func TestGetRepoToken_Missing(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	m := NewManager()
	if got := m.GetRepoToken(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestResolveAPIKeyAndBaseURL_FlagsOverride(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "env-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://env.cisco.com")
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	m := NewManager()
	key, url, err := ResolveAPIKeyAndBaseURL("flag-key", "https://flag.cisco.com", m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "flag-key" {
		t.Errorf("expected flag key, got %q", key)
	}
	if url != "https://flag.cisco.com" {
		t.Errorf("expected flag URL, got %q", url)
	}
}

func TestResolveAPIKeyAndBaseURL_TrailingSlash(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	m := NewManager()
	_, url, err := ResolveAPIKeyAndBaseURL("k", "https://test.cisco.com///", m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://test.cisco.com" {
		t.Errorf("expected trailing slashes trimmed, got %q", url)
	}
}

func TestResolveAPIKeyAndBaseURL_InvalidDomain(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	m := NewManager()
	_, _, err := ResolveAPIKeyAndBaseURL("k", "https://evil.example.com", m)
	if err == nil {
		t.Fatal("expected error for non-cisco domain")
	}
}

func TestResolveAPIKeyAndBaseURL_InvalidURL(t *testing.T) {
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	m := NewManager()
	_, _, err := ResolveAPIKeyAndBaseURL("k", "://bad", m)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestGetNestedString_NonExistentKeys(t *testing.T) {
	m := &Manager{data: map[string]interface{}{}}
	if got := m.getNestedString("a", "b", "c"); got != "" {
		t.Errorf("expected empty string for non-existent path, got %q", got)
	}
}

func TestGetNestedString_NonStringValue(t *testing.T) {
	m := &Manager{data: map[string]interface{}{
		"default": map[string]interface{}{
			"api_key": 12345,
		},
	}}
	if got := m.getNestedString("default", "api_key"); got != "" {
		t.Errorf("expected empty string for non-string value, got %q", got)
	}
}
