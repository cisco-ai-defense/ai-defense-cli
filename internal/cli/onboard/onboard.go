package onboard

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	configcmd "github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

// NewCmd returns the onboard command.
func NewCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Interactive setup for new users",
		Long: `Guided setup for AI Defense CLI: core credentials plus optional
capabilities (private HuggingFace repos, local AIBOM analysis).

Run this after installing the binary from GitHub Releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := appconfig.NewManager()
			if !force && configComplete(m) {
				fmt.Fprintln(os.Stdout, "Configuration already looks complete.")
				fmt.Fprintln(os.Stdout, "  aid-cisco-cli config show")
				fmt.Fprintln(os.Stdout, "  aid-cisco-cli config edit")
				return nil
			}

			ctx, err := configcmd.RunWizard(configcmd.RunOptions{
				Mode:    "init",
				Force:   force,
				Onboard: true,
			})
			if err != nil {
				return err
			}
			configcmd.PrintNextSteps(ctx)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Re-run setup even if config exists")
	return cmd
}

func configComplete(m *appconfig.Manager) bool {
	if _, err := m.GetAPIKey(); err != nil {
		return false
	}
	if _, err := m.GetBaseURL(); err != nil {
		return false
	}
	return true
}
