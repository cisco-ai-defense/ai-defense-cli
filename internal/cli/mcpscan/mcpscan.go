// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
