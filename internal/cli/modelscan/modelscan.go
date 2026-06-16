package modelscan

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// NewCmd creates the model-scan parent command with all subcommands registered.
func NewCmd(cli *cmdutil.CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model-scan",
		Short: "Scan AI/ML models for security vulnerabilities",
		Long: `Manage security scans of AI/ML models.

Available commands:
  trigger   Submit a model to the cloud for server-side scanning
  list      List previous scans with pagination
  info      Get details of a specific scan

Deprecated:
  run       Use trigger instead (exits with usage error)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newTriggerCmd(cli))
	cmd.AddCommand(newDeprecatedRunCmd())
	cmd.AddCommand(newListCmd(cli))
	cmd.AddCommand(newInfoCmd(cli))

	return cmd
}

// NewDeprecatedScanCmd returns a hidden "scan" alias that delegates to
// model-scan for backward compatibility with the Python CLI.
func NewDeprecatedScanCmd(cli *cmdutil.CLI) *cobra.Command {
	alias := &cobra.Command{
		Use:   "scan",
		Short: "Deprecated: use 'model-scan' instead",
		Long: `Manage security scans of AI/ML models.

Deprecated: use "model-scan" instead.`,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	alias.AddCommand(newTriggerCmd(cli))
	alias.AddCommand(newDeprecatedRunCmd())
	alias.AddCommand(newListCmd(cli))
	alias.AddCommand(newInfoCmd(cli))
	alias.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "Warning: 'scan' is deprecated, use 'model-scan' instead.")
		if parent := cmd.Root(); parent != nil && parent.PersistentPreRun != nil {
			parent.PersistentPreRun(parent, args)
		}
	}

	return alias
}
