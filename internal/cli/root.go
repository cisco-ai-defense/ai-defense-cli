package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/aibom"
	cliconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/install"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/onboard"
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
	flagAPIKey   string
	flagBaseURL  string
	flagProfile  string
	cfgManager   *config.Manager
)

var rootCmd = &cobra.Command{
	Use:     "cisco-aid-cli",
	Short:   "Cisco AI Defense CLI",
	Long:    "Command-line interface for Cisco AI Defense — model scanning, MCP server scanning, and AI Bill of Materials.",
	Version: version + " (" + commit + ")",
	// The top-level handler in Run() owns all error reporting and exit codes, so
	// cobra must not print errors or usage itself (that would double-print and
	// bypass the exit-code contract).
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.SetProfileOverride(flagProfile)
		cfgManager = config.NewManager()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "",
		"API key for AI Defense service (env: AI_DEFENSE_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "",
		"Base URL for AI Defense service (env: AI_DEFENSE_BASE_URL)")
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "",
		"Configuration profile name (env: AI_DEFENSE_PROFILE)")

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
	rootCmd.AddCommand(install.NewCmd(version))
	rootCmd.AddCommand(cliconfig.NewCmd())
	rootCmd.AddCommand(onboard.NewCmd())

	// Enable "Did you mean 'x'?" suggestions for every command. SuggestionsFor
	// only honors the Levenshtein path when this is > 0 (it is 0 by default), so
	// without this only prefix matches would suggest. 2 mirrors cobra's own
	// default; higher over-suggests on short command names (e.g. info/list/diff).
	setSuggestionDistance(rootCmd, 2)
}

// setSuggestionDistance sets SuggestionsMinimumDistance on cmd and all its
// descendants so near-miss subcommand suggestions work at every level.
func setSuggestionDistance(cmd *cobra.Command, d int) {
	cmd.SuggestionsMinimumDistance = d
	for _, c := range cmd.Commands() {
		setSuggestionDistance(c, d)
	}
}

// VersionString returns the CLI version baked in at build time.
func VersionString() string { return version }

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

	// Usage errors get the aibom/Typer-style treatment: a red "Error" panel
	// followed by the command's full usage/help, all rendered straight to stderr
	// so the color gate sees the real stream (cobra's UsageString() captures into
	// a non-TTY buffer and would strip ANSI). Other errors keep the plain
	// "Error:"/"Hint:" lines.
	if cliErr.Code == clierror.CodeUsage && executed != nil {
		fmt.Fprint(os.Stderr, renderError(cleanUsageMessage(cliErr.Message, executed), os.Stderr))
		// renderHelp already starts with a leading newline, so the blank line it
		// emits separates the error panel from the usage block.
		fmt.Fprint(os.Stderr, renderHelp(executed, os.Stderr))
		return cliErr.Code
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n", cliErr.Message)
	if cliErr.RequestID != "" {
		fmt.Fprintf(os.Stderr, "Request ID: %s\n", cliErr.RequestID)
	}
	if cliErr.Hint != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", cliErr.Hint)
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

// unknownCommandRE matches cobra's unknown-command error, capturing the typed
// name (group 1). Cobra formats it as: unknown command "x" for "parent".
var unknownCommandRE = regexp.MustCompile(`^unknown command "([^"]+)" for "[^"]+"`)

// didYouMeanRE captures a suggestion cobra appends after the message
// ("\n\nDid you mean this?\n\t<suggestion>\n"). cobra only adds this for root
// commands; for subcommands we compute the suggestion ourselves (see below).
var didYouMeanRE = regexp.MustCompile(`Did you mean this\?\s*\n\s*([^\n]+)`)

// cleanUsageMessage rewrites cobra's verbose usage-error text into a concise,
// single-line message for the error panel, matching the embedded analyzer's
// phrasing. Unknown-command errors become "No such command 'x'." plus a "Did
// you mean 'y'?" when a close subcommand exists; other usage errors are
// flattened to a single line unchanged. cmd is the command cobra resolved to
// (the parent of the unknown subcommand), used to compute suggestions because
// cobra omits them for non-root commands.
func cleanUsageMessage(msg string, cmd *cobra.Command) string {
	if m := unknownCommandRE.FindStringSubmatch(msg); m != nil {
		out := "No such command '" + m[1] + "'."
		if s := suggestSubcommand(cmd, m[1], msg); s != "" {
			out += " Did you mean '" + s + "'?"
		}
		return out
	}
	// Collapse any embedded newlines/tabs so the panel stays tidy.
	return strings.Join(strings.Fields(msg), " ")
}

// suggestSubcommand returns the best near-match subcommand name for typed, or ""
// if none is close. It first honors any suggestion cobra already embedded in the
// message (root commands), then falls back to the command's own SuggestionsFor
// (which covers subcommands, where cobra does not add suggestions itself).
func suggestSubcommand(cmd *cobra.Command, typed, msg string) string {
	if s := didYouMeanRE.FindStringSubmatch(msg); s != nil {
		return strings.TrimSpace(s[1])
	}
	if cmd == nil {
		return ""
	}
	if sugg := cmd.SuggestionsFor(typed); len(sugg) > 0 {
		return sugg[0]
	}
	return ""
}
