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

package aibom

import (
	"github.com/spf13/cobra"

	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newInfoCmd(cli *cmdutil.CLI) *cobra.Command {
	var analysisID string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get BOM details by analysis ID",
		Long: `Retrieve detailed information about a specific BOM.

Examples:
  cisco-aid-cli ai-bom info --analysis-id <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			resp, err := client.GetBom(analysisID)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&analysisID, "analysis-id", "", "Analysis ID of the BOM")
	_ = cmd.MarkFlagRequired("analysis-id")

	return cmd
}
