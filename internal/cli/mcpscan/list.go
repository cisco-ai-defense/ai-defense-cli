package mcpscan

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newListCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		limit, offset    int
		serverName       string
		onboardingStatus string
		transportType    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered MCP servers",
		Long: `List registered MCP servers with optional filters and pagination.

Examples:
  aid-cisco-cli mcp-scan list
  aid-cisco-cli mcp-scan list --limit 20 --status COMPLETED`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			var obStatus []mcpscansdk.OnboardingStatus
			if onboardingStatus != "" {
				obStatus = []mcpscansdk.OnboardingStatus{mcpscansdk.OnboardingStatus(strings.ToUpper(onboardingStatus))}
			}
			var tpType []mcpscansdk.TransportType
			if transportType != "" {
				tpType = []mcpscansdk.TransportType{mcpscansdk.TransportType(strings.ToUpper(transportType))}
			}

			resp, err := client.ListServers(mcpscansdk.ListServersParams{
				Limit:            limit,
				Offset:           offset,
				NameSubstr:       serverName,
				OnboardingStatus: obStatus,
				TransportType:    tpType,
			})
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")
	cmd.Flags().StringVar(&serverName, "name", "", "Filter by server name")
	cmd.Flags().StringVar(&onboardingStatus, "status", "", "Filter by onboarding status: COMPLETED, INPROGRESS, FAILED")
	cmd.Flags().StringVar(&transportType, "transport", "", "Filter by transport type: SSE, STREAMABLE")

	return cmd
}
