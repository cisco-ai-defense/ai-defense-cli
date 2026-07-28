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
	"encoding/json"

	"github.com/spf13/cobra"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newInfoCmd(cli *cmdutil.CLI) *cobra.Command {
	var serverID, scanID string
	var resultOpts *cmdutil.ScanResultOptions

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get details of an MCP server or scan",
		Long: `Get details of an MCP server or scan by ID. Provide exactly one of --server-id or --scan-id.

With --scan-id, prints raw API JSON by default (--envelope for the verdict) and
applies scan exit codes (gating applies to --scan-id only). With --server-id,
prints the raw server details.

Examples:
  cisco-aid-cli mcp-scan info --server-id <uuid>
  cisco-aid-cli mcp-scan info --scan-id <uuid> --fail-on high`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverID == "" && scanID == "" {
				return clierror.Usage("provide either --server-id or --scan-id")
			}
			if serverID != "" && scanID != "" {
				return clierror.Usage("--server-id and --scan-id are mutually exclusive")
			}
			if scanID != "" {
				if err := resultOpts.Validate(); err != nil {
					return err
				}
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			if serverID != "" {
				resp, err := client.GetServer(serverID)
				if err != nil {
					return err
				}
				output.PrintRawJSON(resp)
				return nil
			}

			raw, err := client.GetScanStatusRaw(scanID)
			if err != nil {
				return err
			}
			status := extractStatus(raw)
			return emitScanResult(scanID, status, raw, resultOpts)
		},
	}

	cmd.Flags().StringVar(&serverID, "server-id", "", "UUID of a registered MCP server")
	cmd.Flags().StringVar(&scanID, "scan-id", "", "UUID of an MCP scan")
	resultOpts = cmdutil.AddScanResultFlags(cmd)

	return cmd
}

// extractStatus parses the MCP scan status field from a raw status payload,
// tolerating snake_case and lowerCamel. An empty/unknown status defaults to
// COMPLETED so any returned report is evaluated as a verdict rather than skipped.
func extractStatus(raw []byte) string {
	var s struct {
		Status      string `json:"status"`
		StatusCamel string `json:"scanStatus"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	if s.Status != "" {
		return s.Status
	}
	if s.StatusCamel != "" {
		return s.StatusCamel
	}
	return string(mcpscansdk.MCPScanStatusCompleted)
}
