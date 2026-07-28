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

package modelscan

import (
	"encoding/json"

	"github.com/spf13/cobra"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func newInfoCmd(cli *cmdutil.CLI) *cobra.Command {
	var scanID string
	var limit, offset int
	var resultOpts *cmdutil.ScanResultOptions

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get detailed information about a specific scan",
		Long: `Retrieve scan status, scanned files, detected threats, and severity breakdown.

Prints raw API JSON by default. Use --envelope for the normalized verdict and
the same exit-code contract as 'trigger' (0 pass, 10 fail, 20 block; 5 if failed).

Examples:
  cisco-aid-cli model-scan info --scan-id <uuid>
  cisco-aid-cli model-scan info --scan-id <uuid> --limit 50 --output text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := resultOpts.Validate(); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := modelscansdk.NewClient(apiKey, config)

			req := &modelscansdk.GetScanStatusRequest{
				FileLimit:  limit,
				FileOffset: offset,
			}
			raw, err := client.GetScanRaw(scanID, req)
			if err != nil {
				return err
			}
			var resp modelscansdk.GetScanStatusResponse
			if err := json.Unmarshal(raw, &resp); err != nil {
				return clierror.Unexpected("failed to parse scan result", err)
			}
			return emitScanResult(resp.ScanStatusInfo, raw, resultOpts)
		},
	}

	cmd.Flags().StringVar(&scanID, "scan-id", "", "UUID of the scan")
	_ = cmd.MarkFlagRequired("scan-id")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of file results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of file results to skip")
	resultOpts = cmdutil.AddScanResultFlags(cmd)

	return cmd
}
