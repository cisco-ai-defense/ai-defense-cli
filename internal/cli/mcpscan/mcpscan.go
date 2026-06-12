package mcpscan

import (
	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// NewCmd creates the mcp-scan parent command with all subcommands registered.
func NewCmd(cli *cmdutil.CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp-scan",
		Short: "Manage MCP server security scans",
		Long: `Manage MCP (Model Context Protocol) server security scans.

Available commands:
  register        Register an MCP server for ongoing monitoring
  delete          Delete a registered MCP server
  run             Scan a remote MCP server for security threats
  list            List registered MCP servers
  info            Get details of a server or scan by ID
  trigger         Trigger an on-demand scan for a registered server
  capabilities    List tools, prompts, or resources on a server`,
	}

	cmd.AddCommand(newRegisterCmd(cli))
	cmd.AddCommand(newDeleteCmd(cli))
	cmd.AddCommand(newRunCmd(cli))
	cmd.AddCommand(newListCmd(cli))
	cmd.AddCommand(newInfoCmd(cli))
	cmd.AddCommand(newTriggerCmd(cli))
	cmd.AddCommand(newCapabilitiesCmd(cli))

	return cmd
}
