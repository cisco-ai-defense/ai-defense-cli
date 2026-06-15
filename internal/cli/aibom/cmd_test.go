package aibom

import (
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
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
