package aibom

import (
	"github.com/spf13/cobra"

	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newInfoCmd(cli *cmdutil.CLI) *cobra.Command {
	var analysisID string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get BOM details by analysis ID",
		Long: `Retrieve detailed information about a specific BOM.

Examples:
  cisco-aid-cli ai-bom info --analysis-id <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			resp, err := client.GetBom(analysisID)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&analysisID, "analysis-id", "", "Analysis ID of the BOM")
	_ = cmd.MarkFlagRequired("analysis-id")

	return cmd
}
