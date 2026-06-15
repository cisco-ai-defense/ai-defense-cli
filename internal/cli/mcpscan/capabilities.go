package mcpscan

import (
	"strings"

	"github.com/spf13/cobra"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newCapabilitiesCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		serverID       string
		capabilityType string
		capabilityName string
		limit, offset  int
	)

	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "List capabilities on an MCP server",
		Long: `List tools, prompts, or resources discovered on an MCP server.

Examples:
  aid-cisco-cli mcp-scan capabilities --server-id <uuid> --type TOOL
  aid-cisco-cli mcp-scan capabilities --server-id <uuid> --type PROMPT --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			capType := mcpscansdk.CapabilityType(strings.ToUpper(capabilityType))

			resp, err := client.GetServerCapabilities(serverID, capType, capabilityName, limit, offset)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverID, "server-id", "", "UUID of the registered MCP server")
	_ = cmd.MarkFlagRequired("server-id")
	cmd.Flags().StringVar(&capabilityType, "type", "", "Capability type: TOOL, PROMPT, RESOURCE")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVar(&capabilityName, "name", "", "Filter by capability name")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")

	return cmd
}
