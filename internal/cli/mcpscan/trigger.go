package mcpscan

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newTriggerCmd(cli *cmdutil.CLI) *cobra.Command {
	var serverID string

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger an on-demand scan for a registered server",
		Long: `Trigger an on-demand scan for a registered MCP server.

Examples:
  aid-cisco-cli mcp-scan trigger --server-id <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			fmt.Fprintln(os.Stderr, "Triggering MCP server scan...")
			if err := client.TriggerServerScan(serverID); err != nil {
				return err
			}

			output.PrintJSON(map[string]string{"server_id": serverID, "status": "scan_triggered"})
			return nil
		},
	}

	cmd.Flags().StringVar(&serverID, "server-id", "", "UUID of the registered MCP server")
	_ = cmd.MarkFlagRequired("server-id")

	return cmd
}
