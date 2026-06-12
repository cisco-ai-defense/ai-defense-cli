package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Manager loads and resolves CLI configuration from TOML files and env vars.
type Manager struct {
	data       map[string]interface{}
	configPath string
	loadErr    error
}

// NewManager creates a config manager, loading the TOML file if present.
// Search order: AI_DEFENSE_CONFIG_PATH env var, then .config.toml in cwd.
func NewManager() *Manager {
	m := &Manager{data: make(map[string]interface{})}

	configPath := os.Getenv("AI_DEFENSE_CONFIG_PATH")
	if configPath == "" {
		cwd, err := os.Getwd()
		if err == nil {
			configPath = filepath.Join(cwd, ".config.toml")
		}
	} else if !filepath.IsAbs(configPath) {
		cwd, err := os.Getwd()
		if err == nil {
			configPath = filepath.Join(cwd, configPath)
		}
	}

	if configPath != "" {
		m.configPath = configPath
		raw, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				m.loadErr = fmt.Errorf("read config file %s: %w", configPath, err)
			}
		} else if err := toml.Unmarshal(raw, &m.data); err != nil {
			m.loadErr = fmt.Errorf("invalid config file %s: %w", configPath, err)
		}
	}

	return m
}

func (m *Manager) configLoadError() error {
	return m.loadErr
}

// GetAPIKey returns the API key from env or config file.
func (m *Manager) GetAPIKey() (string, error) {
	if err := m.configLoadError(); err != nil {
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
	if err := m.configLoadError(); err != nil {
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

// GetRepoToken returns the HuggingFace token from config.
func (m *Manager) GetRepoToken() string {
	return m.getNestedString("default", "huggingface", "token")
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

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Hostname() == "" {
		return "", "", fmt.Errorf("invalid base URL: %s", baseURL)
	}
	host := parsed.Hostname()
	if host != "cisco.com" && !strings.HasSuffix(host, ".cisco.com") {
		return "", "", fmt.Errorf("invalid base url for AI DEFENSE")
	}

	return apiKey, baseURL, nil
}
