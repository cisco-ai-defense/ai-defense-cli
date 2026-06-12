package aibom

import (
	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// NewCmd creates the ai-bom parent command with all subcommands registered.
func NewCmd(cli *cmdutil.CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-bom",
		Short: "Manage AI Bills of Materials",
		Long: `Generate, submit, and manage AI Bills of Materials (AIBOM).

Available commands:
  generate    Generate an AIBOM locally (embedded analyzer)
  submit      Submit an existing AIBOM report file
  run         Generate and submit an AIBOM in one step (embedded analyzer)
  list        List BOMs with filters and pagination
  info        Get BOM details by analysis ID
  delete      Delete a BOM
  components  List components in a BOM
  summary     Get aggregate BOM statistics`,
	}

	cmd.AddCommand(newGenerateCmd(cli))
	cmd.AddCommand(newSubmitCmd(cli))
	cmd.AddCommand(newRunCmd(cli))
	cmd.AddCommand(newListCmd(cli))
	cmd.AddCommand(newInfoCmd(cli))
	cmd.AddCommand(newDeleteCmd(cli))
	cmd.AddCommand(newComponentsCmd(cli))
	cmd.AddCommand(newSummaryCmd(cli))

	return cmd
}
