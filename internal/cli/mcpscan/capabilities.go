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
  cisco-aid-cli mcp-scan capabilities --server-id <uuid> --type TOOL
  cisco-aid-cli mcp-scan capabilities --server-id <uuid> --type PROMPT --limit 50`,
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
