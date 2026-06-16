package mcpscan

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newDeleteCmd(cli *cmdutil.CLI) *cobra.Command {
	var serverID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a registered MCP server",
		Long: `Delete a registered MCP server and all its scan data.

Examples:
  cisco-aid-cli mcp-scan delete --server-id <uuid>
  cisco-aid-cli mcp-scan delete --server-id <uuid> -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Delete server %s and all its scan data? This cannot be undone [y/N]: ", serverID)
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

			config := cli.APIConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			fmt.Fprintln(os.Stderr, "Deleting MCP server...")
			if err := client.DeleteServer(serverID); err != nil {
				return err
			}

			output.PrintJSON(map[string]string{"server_id": serverID, "status": "deleted"})
			return nil
		},
	}

	cmd.Flags().StringVar(&serverID, "server-id", "", "UUID of the server to delete")
	_ = cmd.MarkFlagRequired("server-id")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
