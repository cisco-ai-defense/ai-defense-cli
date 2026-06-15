package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

// Set at build time via -ldflags:
//
//	-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=...
//	-X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=...
var (
	version = "dev"
	commit  = "unknown"
)

var (
	flagAPIKey  string
	flagBaseURL string
	cfgManager  *config.Manager
)

var rootCmd = &cobra.Command{
	Use:     "aid-cisco-cli",
	Short:   "Cisco AI Defense CLI",
	Long:    "Command-line interface for Cisco AI Defense — model scanning, MCP server scanning, and AI Bill of Materials.",
	Version: version + " (" + commit + ")",
	// The top-level handler in Run() owns all error reporting and exit codes, so
	// cobra must not print errors or usage itself (that would double-print and
	// bypass the exit-code contract).
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfgManager = config.NewManager()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "",
		"API key for AI Defense service (env: AI_DEFENSE_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "",
		"Base URL for AI Defense service (env: AI_DEFENSE_BASE_URL)")

	// Flag-parse failures are usage errors (exit 2). Wrapping them here lets the
	// handler treat them uniformly and still show usage.
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return clierror.Usage(err.Error())
	})

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Unified help styling: render native (Cobra) command help as titled panels
	// so it matches the embedded aibom (Typer/Rich) help. aibom passthrough
	// commands forward --help to aibom directly (DisableFlagParsing).
	rootCmd.SetHelpFunc(panelHelpFunc)
	rootCmd.SetUsageFunc(panelUsageFunc)

	cli := &cmdutil.CLI{
		ConfigManager: func() *config.Manager { return cfgManager },
		FlagAPIKey:    func() string { return flagAPIKey },
		FlagBaseURL:   func() string { return flagBaseURL },
	}

	rootCmd.AddCommand(modelscan.NewCmd(cli))
	rootCmd.AddCommand(modelscan.NewDeprecatedScanCmd(cli))
	rootCmd.AddCommand(mcpscan.NewCmd(cli))
	rootCmd.AddCommand(aibom.NewCmd(cli))
}

// Execute runs the root command and returns the raw, unclassified error. It is
// kept for tests and embedders; the binary entry point uses Run, which applies
// the exit-code contract.
func Execute() error {
	return rootCmd.Execute()
}

// Run executes the CLI and returns the process exit code per the documented
// contract (see internal/clierror): 0 pass, 1-9 operational errors, >=10 policy
// verdicts. It classifies the error, prints "Error:"/"Hint:"/request-id to
// stderr, and shows command usage for usage errors.
func Run() int {
	executed, err := rootCmd.ExecuteC()
	if err == nil {
		return clierror.CodeOK
	}

	cliErr := clierror.Classify(err)
	// Cobra reports unknown commands, missing required flags, and bad positional
	// args as plain errors from Execute (not via FlagErrorFunc); recognize them
	// so they map to the usage exit code instead of the generic one.
	if cliErr.Code == clierror.CodeUnexpected && isUsageError(err) {
		cliErr = clierror.Usage(err.Error())
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n", cliErr.Message)
	if cliErr.RequestID != "" {
		fmt.Fprintf(os.Stderr, "Request ID: %s\n", cliErr.RequestID)
	}
	if cliErr.Hint != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", cliErr.Hint)
	}
	if cliErr.Code == clierror.CodeUsage && executed != nil {
		fmt.Fprint(os.Stderr, "\n", executed.UsageString())
	}
	return cliErr.Code
}

// usageErrorPrefixes are the leading strings cobra/pflag use for argument and
// command errors that should be treated as usage failures (exit 2).
var usageErrorPrefixes = []string{
	"unknown command",
	"unknown flag",
	"unknown shorthand flag",
	"required flag",
	"flag needs an argument",
	"invalid argument",
	"accepts ",
	"requires ",
	"unknown command ",
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, p := range usageErrorPrefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	return false
}
