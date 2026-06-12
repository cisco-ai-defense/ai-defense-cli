package aibom

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newDeleteCmd(cli *cmdutil.CLI) *cobra.Command {
	var analysisID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a BOM and its associated data",
		Long: `Permanently remove a BOM and all associated component data.

Examples:
  aid-cisco-cli ai-bom delete --analysis-id <uuid>
  aid-cisco-cli ai-bom delete --analysis-id <uuid> -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Delete BOM %s and all its data? This cannot be undone [y/N]: ", analysisID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			fmt.Fprintln(os.Stderr, "Deleting BOM...")
			if err := client.DeleteBom(analysisID); err != nil {
				return err
			}

			output.PrintJSON(map[string]string{"analysis_id": analysisID, "status": "deleted"})
			return nil
		},
	}

	cmd.Flags().StringVar(&analysisID, "analysis-id", "", "Analysis ID of the BOM")
	_ = cmd.MarkFlagRequired("analysis-id")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
