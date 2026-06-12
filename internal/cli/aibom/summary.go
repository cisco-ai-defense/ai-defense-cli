package aibom

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newSummaryCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		search     string
		bomStatus  string
		sourceKind string
		fromTime   string
		toTime     string
	)

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Get aggregate BOM statistics",
		Long: `Get summary counts (total BOMs, completed, failed, total assets) across all BOMs.

Examples:
  aid-cisco-cli ai-bom summary
  aid-cisco-cli ai-bom summary --status BOM_STATUS_COMPLETED`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			req := &aibomsdk.BomsSummaryRequest{}
			if search != "" {
				req.Search = search
			}
			if bomStatus != "" {
				req.Status = aibomsdk.BomStatus(strings.ToUpper(bomStatus))
			}
			if sourceKind != "" {
				req.SourceKind = aibomsdk.SourceKind(strings.ToUpper(sourceKind))
			}
			if fromTime != "" {
				req.FromTime = fromTime
			}
			if toTime != "" {
				req.ToTime = toTime
			}

			resp, err := client.GetBomSummary(req)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Search term")
	cmd.Flags().StringVar(&bomStatus, "status", "", "Filter by BOM status")
	cmd.Flags().StringVar(&sourceKind, "source-kind", "", "Filter by source kind")
	cmd.Flags().StringVar(&fromTime, "from", "", "Start timestamp (RFC3339)")
	cmd.Flags().StringVar(&toTime, "to", "", "End timestamp (RFC3339)")

	return cmd
}
