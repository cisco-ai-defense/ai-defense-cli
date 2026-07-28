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

func newListCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		limit, offset int
		search        string
		bomStatus     string
		sourceKind    string
		sortBy        string
		order         string
		fromTime      string
		toTime        string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List BOMs with optional filters and pagination",
		Long: `Retrieve a paginated list of AI Bills of Materials.

Examples:
  cisco-aid-cli ai-bom list
  cisco-aid-cli ai-bom list --status BOM_STATUS_COMPLETED --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			req := &aibomsdk.ListBomsRequest{
				Limit:  limit,
				Offset: offset,
			}
			if search != "" {
				req.Search = search
			}
			if bomStatus != "" {
				req.Status = aibomsdk.BomStatus(strings.ToUpper(bomStatus))
			}
			if sourceKind != "" {
				req.SourceKind = aibomsdk.SourceKind(strings.ToUpper(sourceKind))
			}
			if sortBy != "" {
				req.SortBy = aibomsdk.SortBy(strings.ToUpper(sortBy))
			}
			if order != "" {
				switch strings.ToUpper(order) {
				case "ASC":
					req.Order = aibomsdk.SortOrderAscending
				case "DESC":
					req.Order = aibomsdk.SortOrderDescending
				}
			}
			if fromTime != "" {
				req.FromTime = fromTime
			}
			if toTime != "" {
				req.ToTime = toTime
			}

			resp, err := client.ListBoms(req)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of records")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of records to skip")
	cmd.Flags().StringVar(&search, "search", "", "Search term")
	cmd.Flags().StringVar(&bomStatus, "status", "", "Filter by BOM status")
	cmd.Flags().StringVar(&sourceKind, "source-kind", "", "Filter by source kind")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field")
	cmd.Flags().StringVar(&order, "order", "", "Sort order: ASC, DESC")
	cmd.Flags().StringVar(&fromTime, "from", "", "Start timestamp (RFC3339)")
	cmd.Flags().StringVar(&toTime, "to", "", "End timestamp (RFC3339)")

	return cmd
}
