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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newDeleteCmd(cli *cmdutil.CLI) *cobra.Command {
	var analysisID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a BOM and its associated data",
		Long: `Permanently remove a BOM and all associated component data.

Examples:
  cisco-aid-cli ai-bom delete --analysis-id <uuid>
  cisco-aid-cli ai-bom delete --analysis-id <uuid> -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Delete BOM %s and all its data? This cannot be undone [y/N]: ", analysisID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			fmt.Fprintln(os.Stderr, "Deleting BOM...")
			if err := client.DeleteBom(analysisID); err != nil {
				return err
			}

			output.PrintJSON(map[string]string{"analysis_id": analysisID, "status": "deleted"})
			return nil
		},
	}

	cmd.Flags().StringVar(&analysisID, "analysis-id", "", "Analysis ID of the BOM")
	_ = cmd.MarkFlagRequired("analysis-id")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
