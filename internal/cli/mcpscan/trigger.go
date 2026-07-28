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
	"fmt"
	"os"

	"github.com/spf13/cobra"

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
  cisco-aid-cli mcp-scan trigger --server-id <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
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
