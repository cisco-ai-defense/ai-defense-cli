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
	"strings"

	"github.com/spf13/cobra"

	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newComponentsCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		analysisID    string
		limit, offset int
		search        string
		componentType string
		framework     string
	)

	cmd := &cobra.Command{
		Use:   "components",
		Short: "List components discovered in a BOM",
		Long: `List AI/ML components (models, agents, tools, etc.) found in a BOM.

Examples:
  cisco-aid-cli ai-bom components --analysis-id <uuid>
  cisco-aid-cli ai-bom components --analysis-id <uuid> --type COMPONENT_CATEGORY_MODEL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			req := &aibomsdk.ListBomComponentsRequest{
				Limit:  limit,
				Offset: offset,
			}
			if search != "" {
				req.Search = search
			}
			if componentType != "" {
				req.ComponentType = aibomsdk.ComponentCategory(strings.ToUpper(componentType))
			}
			if framework != "" {
				req.Framework = framework
			}

			resp, err := client.ListBomComponents(analysisID, req)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&analysisID, "analysis-id", "", "Analysis ID of the BOM")
	_ = cmd.MarkFlagRequired("analysis-id")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")
	cmd.Flags().StringVar(&search, "search", "", "Search term")
	cmd.Flags().StringVar(&componentType, "type", "", "Filter by component category")
	cmd.Flags().StringVar(&framework, "framework", "", "Filter by framework name")

	return cmd
}
