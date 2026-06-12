package modelscan

import (
	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newListCmd(cli *cmdutil.CLI) *cobra.Command {
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List security scans with pagination",
		Long: `Retrieve a paginated list of all security scans.

Examples:
  aid-cisco-cli model-scan list
  aid-cisco-cli model-scan list --limit 20 --offset 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := modelscansdk.NewClient(apiKey, config)

			raw, err := client.ListScansRaw(&modelscansdk.ListScansRequest{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return err
			}
			output.PrintRawJSON(raw)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")

	return cmd
}
