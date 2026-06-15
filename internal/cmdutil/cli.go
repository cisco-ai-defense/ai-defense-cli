package cmdutil

import (
	"fmt"
	"os"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

// CLI provides shared state and helpers to all command sub-packages.
// Getter functions are used because flag values are only available
// after cobra parses them at RunE time, not at command construction time.
type CLI struct {
	ConfigManager func() *config.Manager
	FlagAPIKey    func() string
	FlagBaseURL   func() string
	// APIConfigFactory overrides api.NewConfig (tests inject httptest HTTP clients).
	APIConfigFactory func(baseURL string) *api.Config
}

// APIConfig returns SDK config for the resolved management base URL.
func (c *CLI) APIConfig(baseURL string) *api.Config {
	if c != nil && c.APIConfigFactory != nil {
		return c.APIConfigFactory(baseURL)
	}
	return api.NewConfig(baseURL)
}

// ResolveAuth resolves the API key and base URL from flags, env vars, and config
// file. Failures are classified as auth/config errors (exit 3) so credential and
// base-URL problems are distinct from network or scan failures.
func (c *CLI) ResolveAuth() (string, string, error) {
	apiKey, baseURL, err := config.ResolveAPIKeyAndBaseURL(c.FlagAPIKey(), c.FlagBaseURL(), c.ConfigManager())
	if err != nil {
		return "", "", clierror.Auth(err.Error(), "Run 'aid-cisco-cli onboard' to configure credentials.", err)
	}
	return apiKey, baseURL, nil
}

// GetRepoToken returns the HuggingFace repo token from config.
func (c *CLI) GetRepoToken() string {
	return c.ConfigManager().GetRepoToken()
}

// ExitOnError prints the error message to stderr and exits with code 1.
func ExitOnError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// EnvIntOrDefault reads an integer from an environment variable, falling back to defaultVal.
func EnvIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return defaultVal
}
