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
  cisco-aid-cli mcp-scan list
  cisco-aid-cli mcp-scan list --limit 20 --status COMPLETED`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
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
