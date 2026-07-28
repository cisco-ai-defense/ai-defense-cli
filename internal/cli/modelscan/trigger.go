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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func newTriggerCmd(cli *cmdutil.CLI) *cobra.Command {
	var filePath string
	var repoURL string
	var repoToken string
	var resultOpts *cmdutil.ScanResultOptions

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger a cloud-based scan of an AI/ML model",
		Long: `Submit a model file or repository URL to the cloud for server-side scanning.

Prints raw API JSON by default (Python CLI parity). Use --envelope for the
normalized verdict JSON/text. Exit codes follow the scan contract: 0 pass,
10 fail (>= --fail-on), 20 block (>= --block-on); operational failures use 1-5.

Examples:
  cisco-aid-cli model-scan trigger -f model.pth
  cisco-aid-cli model-scan trigger --repo-url huggingface.co/username/model
  cisco-aid-cli model-scan trigger -f model.pth --fail-on high --output text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" && repoURL == "" {
				return clierror.Usage("either --file-path or --repo-url must be provided")
			}
			if filePath != "" && repoURL != "" {
				return clierror.Usage("only one of --file-path or --repo-url can be provided")
			}
			if err := resultOpts.Validate(); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := modelscansdk.NewClient(apiKey, config)

			var raw json.RawMessage
			if filePath != "" {
				fmt.Fprintln(os.Stderr, "Scanning file (uploading and processing)...")
				raw, err = client.ScanFile(filePath)
			} else {
				fmt.Fprintln(os.Stderr, "Scanning repository (processing)...")
				repoConfig := &modelscansdk.ModelRepoConfig{
					URL:  repoURL,
					Type: modelscansdk.URLTypeHuggingFace,
				}
				if repoToken != "" {
					repoConfig.Auth = &modelscansdk.Auth{
						HuggingFace: &modelscansdk.HuggingFaceAuth{AccessToken: repoToken},
					}
				} else if token := cli.GetRepoToken(); token != "" {
					repoConfig.Auth = &modelscansdk.Auth{
						HuggingFace: &modelscansdk.HuggingFaceAuth{AccessToken: token},
					}
				}
				raw, err = client.ScanRepo(repoConfig)
			}
			if err != nil {
				return err
			}

			var statusResp modelscansdk.GetScanStatusResponse
			if err := json.Unmarshal(raw, &statusResp); err != nil {
				return clierror.Unexpected("failed to parse scan result", err)
			}
			return emitScanResult(statusResp.ScanStatusInfo, raw, resultOpts)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file-path", "f", "", "Local path to the model file")
	cmd.Flags().StringVar(&repoURL, "repo-url", "", "URL of the model repository")
	cmd.Flags().StringVar(&repoToken, "repo-token", "", "HuggingFace token for private repos")
	resultOpts = cmdutil.AddScanResultFlags(cmd)

	return cmd
}
