package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

// Manager loads and resolves CLI configuration from TOML files and env vars.
type Manager struct {
	data       map[string]interface{}
	configPath string
	loadErr    error
}

// NewManager creates a config manager, loading the TOML file if present.
// Search order: AI_DEFENSE_CONFIG_PATH → ~/.ai-defense/config.toml → legacy paths.
func NewManager() *Manager {
	return NewManagerFromPath(ResolveConfigPath())
}

// NewManagerFromPath loads config from an explicit path (used by config edit/show).
func NewManagerFromPath(configPath string) *Manager {
	m := &Manager{data: make(map[string]interface{})}
	if configPath == "" {
		return m
	}
	m.configPath = configPath
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			m.loadErr = fmt.Errorf("read config file %s: %w", configPath, err)
		}
		return m
	}
	if err := toml.Unmarshal(raw, &m.data); err != nil {
		m.loadErr = fmt.Errorf("invalid config file %s: %w", configPath, err)
	}
	return m
}

// ConfigPath returns the path this manager loaded from (may be empty).
func (m *Manager) ConfigPath() string {
	return m.configPath
}

// Data returns a copy of the loaded config map (for edit/show).
func (m *Manager) Data() map[string]interface{} {
	if m.data == nil {
		return map[string]interface{}{}
	}
	return cloneMap(m.data)
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		if nested, ok := v.(map[string]interface{}); ok {
			out[k] = cloneMap(nested)
		} else {
			out[k] = v
		}
	}
	return out
}

// LoadError returns a TOML read/parse failure for the config file, or nil.
func (m *Manager) LoadError() error {
	return m.loadErr
}

// GetAPIKey returns the API key from env or config file.
func (m *Manager) GetAPIKey() (string, error) {
	if err := m.LoadError(); err != nil {
		return "", err
	}
	if v := os.Getenv("AI_DEFENSE_API_KEY"); v != "" {
		return v, nil
	}
	if v := m.getNestedString("default", "api_key"); v != "" {
		return v, nil
	}
	return "", fmt.Errorf(
		"API key not configured. Set via:\n" +
			"  --api-key flag\n" +
			"  AI_DEFENSE_API_KEY environment variable\n" +
			"  api_key in config file")
}

// GetBaseURL returns the base URL from env or config file.
func (m *Manager) GetBaseURL() (string, error) {
	if err := m.LoadError(); err != nil {
		return "", err
	}
	if v := os.Getenv("AI_DEFENSE_BASE_URL"); v != "" {
		return v, nil
	}
	if v := m.getNestedString("default", "base_url"); v != "" {
		return v, nil
	}
	return "", fmt.Errorf(
		"Base URL not configured. Set via:\n" +
			"  --base-url flag\n" +
			"  AI_DEFENSE_BASE_URL environment variable\n" +
			"  base_url in config file")
}

// GetRepoToken returns the HuggingFace token from config (integration id: huggingface).
func (m *Manager) GetRepoToken() string {
	ig, ok := spec.ByID("huggingface")
	if !ok {
		return ""
	}
	return ig.ResolveField("token", "", m)
}

// WriteConfig serializes data to path with mode 0600, creating parent dirs.
func WriteConfig(path string, data map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	raw, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// RedactSecrets returns a copy of data with known secret fields masked.
func RedactSecrets(data map[string]interface{}) map[string]interface{} {
	out := cloneMap(data)
	for _, sec := range spec.Sections() {
		for _, path := range sec.SecretTomlPaths() {
			redactNested(out, path...)
		}
	}
	return out
}

func redactNested(data map[string]interface{}, keys ...string) {
	if len(keys) == 0 {
		return
	}
	cur := data
	for i, k := range keys[:len(keys)-1] {
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			return
		}
		if i == len(keys)-2 {
			if _, ok := next[keys[len(keys)-1]]; ok {
				next[keys[len(keys)-1]] = "***"
			}
			return
		}
		cur = next
	}
}

// NestedString reads a nested string value from loaded TOML (no env overlay).
func (m *Manager) NestedString(keys ...string) string {
	return m.getNestedString(keys...)
}

func (m *Manager) getNestedString(keys ...string) string {
	var current interface{} = m.data
	for _, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[key]
		default:
			return ""
		}
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

// ResolveAPIKeyAndBaseURL resolves API key and base URL with priority:
// CLI flag > env/config. Validates the base URL domain.
func ResolveAPIKeyAndBaseURL(flagAPIKey, flagBaseURL string, mgr *Manager) (string, string, error) {
	apiKey := flagAPIKey
	if apiKey == "" {
		var err error
		apiKey, err = mgr.GetAPIKey()
		if err != nil {
			return "", "", err
		}
	}

	baseURL := flagBaseURL
	if baseURL == "" {
		var err error
		baseURL, err = mgr.GetBaseURL()
		if err != nil {
			return "", "", err
		}
	}

	baseURL = strings.TrimRight(baseURL, "/")

	if err := spec.ValidateBaseURLHost(baseURL); err != nil {
		return "", "", err
	}

	return apiKey, baseURL, nil
}
