package aibom

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// newKBCmd is the `kb` passthrough (manage the analyzer knowledge base). For the
// build commands that call the AI Defense API (request / request-status /
// list-requests), aid_cli bridges the tenant API key from config when the user
// hasn't supplied it. The kb host (CISCO_AI_DEFENSE_API_BASE) and manifest URL
// (CISCO_AIBOM_MANIFEST_URL) stay native kb flags/env, because the kb host
// family differs from the management base URL and has no AI Defense config
// source.
func newKBCmd(cli *cmdutil.CLI) *cobra.Command {
	return newPassthroughCmd(cli,
		"kb",
		"Manage the analyzer knowledge base",
		"kb",
		kbCredentialEnv,
	)
}

// kbCredentialEnv injects CISCO_AI_DEFENSE_API_KEY (the tenant key) for kb build
// commands, derived from resolved config, only when the user hasn't passed
// --api-key or set the env var (user input always wins). It is a no-op for the
// local kb commands (download/check/info/verify) that don't call the API.
func kbCredentialEnv(cli *cmdutil.CLI, args []string) ([]string, error) {
	if !kbNeedsCredentials(args) {
		return nil, nil
	}
	if hasFlag(args, "--api-key") || os.Getenv("CISCO_AI_DEFENSE_API_KEY") != "" {
		return nil, nil
	}
	apiKey, err := cli.ConfigManager().GetAPIKey()
	if err != nil {
		return nil, clierror.Auth(err.Error(), "Run 'cisco-aid-cli onboard' to configure credentials.", err)
	}
	return []string{"CISCO_AI_DEFENSE_API_KEY=" + apiKey}, nil
}

// kbNeedsCredentials reports whether the kb subcommand calls the AI Defense API
// (and therefore needs a tenant key). download/check/info/verify are local or
// manifest-only operations that don't.
func kbNeedsCredentials(args []string) bool {
	switch firstNonFlag(args) {
	case "request", "request-status", "list-requests":
		return true
	default:
		return false
	}
}
