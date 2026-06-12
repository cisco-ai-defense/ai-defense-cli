package aibom

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newListCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		limit, offset int
		search        string
		bomStatus     string
		sourceKind    string
		sortBy        string
		order         string
		fromTime      string
		toTime        string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List BOMs with optional filters and pagination",
		Long: `Retrieve a paginated list of AI Bills of Materials.

Examples:
  aid-cisco-cli ai-bom list
  aid-cisco-cli ai-bom list --status BOM_STATUS_COMPLETED --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			req := &aibomsdk.ListBomsRequest{
				Limit:  limit,
				Offset: offset,
			}
			if search != "" {
				req.Search = search
			}
			if bomStatus != "" {
				req.Status = aibomsdk.BomStatus(strings.ToUpper(bomStatus))
			}
			if sourceKind != "" {
				req.SourceKind = aibomsdk.SourceKind(strings.ToUpper(sourceKind))
			}
			if sortBy != "" {
				req.SortBy = aibomsdk.SortBy(strings.ToUpper(sortBy))
			}
			if order != "" {
				switch strings.ToUpper(order) {
				case "ASC":
					req.Order = aibomsdk.SortOrderAscending
				case "DESC":
					req.Order = aibomsdk.SortOrderDescending
				}
			}
			if fromTime != "" {
				req.FromTime = fromTime
			}
			if toTime != "" {
				req.ToTime = toTime
			}

			resp, err := client.ListBoms(req)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")
	cmd.Flags().StringVar(&search, "search", "", "Search term")
	cmd.Flags().StringVar(&bomStatus, "status", "", "Filter by BOM status")
	cmd.Flags().StringVar(&sourceKind, "source-kind", "", "Filter by source kind")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field")
	cmd.Flags().StringVar(&order, "order", "", "Sort order: ASC, DESC")
	cmd.Flags().StringVar(&fromTime, "from", "", "Start timestamp (RFC3339)")
	cmd.Flags().StringVar(&toTime, "to", "", "End timestamp (RFC3339)")

	return cmd
}
