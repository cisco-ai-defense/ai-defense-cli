package cmdutil

import (
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestResolveAuth_FromEnv(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "env-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	cli := &CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
	}

	apiKey, baseURL, err := cli.ResolveAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiKey != "env-key" {
		t.Errorf("expected 'env-key', got %q", apiKey)
	}
	if baseURL != "https://test.cisco.com" {
		t.Errorf("expected 'https://test.cisco.com', got %q", baseURL)
	}
}

func TestResolveAuth_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "env-key")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://env.cisco.com")
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	cli := &CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "flag-key" },
		FlagBaseURL:   func() string { return "https://flag.cisco.com" },
	}

	apiKey, baseURL, err := cli.ResolveAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiKey != "flag-key" {
		t.Errorf("expected 'flag-key', got %q", apiKey)
	}
	if baseURL != "https://flag.cisco.com" {
		t.Errorf("expected 'https://flag.cisco.com', got %q", baseURL)
	}
}

func TestResolveAuth_MissingKeyFails(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	cli := &CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
	}

	_, _, err := cli.ResolveAuth()
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if code := clierror.Classify(err).Code; code != clierror.CodeAuth {
		t.Errorf("missing credential should be an auth error (exit %d), got %d", clierror.CodeAuth, code)
	}
}

func TestAPIConfig_DefaultFactory(t *testing.T) {
	cli := &CLI{}
	cfg := cli.APIConfig("https://test.cisco.com")
	if cfg == nil || cfg.ManagementBaseURL != "https://test.cisco.com" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestAPIConfig_CustomFactory(t *testing.T) {
	cli := &CLI{
		APIConfigFactory: func(baseURL string) *api.Config {
			cfg := api.NewConfig(baseURL)
			cfg.Timeout = 99
			return cfg
		},
	}
	cfg := cli.APIConfig("https://test.cisco.com")
	if cfg.Timeout != 99 {
		t.Fatalf("expected custom timeout, got %d", cfg.Timeout)
	}
}

func TestGetRepoToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"
	t.Setenv("AI_DEFENSE_CONFIG_PATH", cfgPath)
	t.Setenv("AI_DEFENSE_API_KEY", "k")
	t.Setenv("AI_DEFENSE_BASE_URL", "https://test.cisco.com")

	cli := &CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
	}
	if cli.GetRepoToken() != "" {
		t.Fatalf("expected empty repo token by default, got %q", cli.GetRepoToken())
	}
}

func TestEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		def      int
		expected int
	}{
		{"empty uses default", "", 42, 42},
		{"valid int", "10", 42, 10},
		{"invalid falls back", "abc", 42, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_VAR", tt.envVal)
			got := EnvIntOrDefault("TEST_INT_VAR", tt.def)
			if got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}
