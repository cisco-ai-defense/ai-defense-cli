package modelscan

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func newDeprecatedRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Deprecated: use 'trigger' instead",
		Long: `Deprecated: use "model-scan trigger" instead of "model-scan run".

This subcommand existed in the Python aidcli; it was renamed to trigger.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parent := "model-scan"
			if p := cmd.Parent(); p != nil && p.Name() != "" {
				parent = p.Name()
			}
			return clierror.Usage(fmt.Sprintf(
				"'%s run' is deprecated; use '%s trigger' instead",
				parent, parent,
			))
		},
	}
}
