package onboard

import (
	"github.com/spf13/cobra"

	configcmd "github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config"
)

// NewCmd returns the onboard command.
func NewCmd() *cobra.Command {
	var force bool
	var profile string
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Interactive setup for new users",
		Long: `Guided setup for AI Defense CLI: core credentials plus optional
capabilities (private HuggingFace repos, local AIBOM analysis).

If you already started setup, you can continue from your saved config file
instead of starting over. Run this after installing the binary from GitHub Releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configcmd.ApplyProfileFlag(profile)
			ctx, err := configcmd.RunWizard(configcmd.RunOptions{
				Force:   force,
				Profile: profile,
			})
			if err != nil {
				return err
			}
			configcmd.PrintNextSteps(ctx)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Start over and replace an existing config file")
	configcmd.ProfileFlag(cmd, &profile)
	return cmd
}
