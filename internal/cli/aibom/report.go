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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// newReportCmd is the `report` passthrough (show or upload a generated AIBOM).
// For `report upload`, aid_cli derives the AI Defense upload target from config
// when the user hasn't supplied it; everything else forwards verbatim.
func newReportCmd(cli *cmdutil.CLI) *cobra.Command {
	return newPassthroughCmd(cli,
		"report [show|upload] [REPORT]",
		"Show or upload a generated AIBOM report",
		"report",
		reportUploadEnv,
	)
}

// reportUploadEnv injects the AI Defense upload target for `ai-bom report
// upload`, deriving AIBOM_POST_URL and AI_DEFENSE_API_KEY from resolved aid_cli
// auth — but only for values the user has not already supplied via flag or env
// (user input always wins). It is a no-op for `report show` / `report <file>`.
func reportUploadEnv(cli *cmdutil.CLI, args []string) ([]string, error) {
	if !isReportUpload(args) {
		return nil, nil
	}

	needURL := !hasFlag(args, "--post-url") && os.Getenv("AIBOM_POST_URL") == ""
	needKey := !hasFlag(args, "--ai-defense-api-key") && os.Getenv("AI_DEFENSE_API_KEY") == ""
	if !needURL && !needKey {
		return nil, nil
	}

	apiKey, baseURL, err := cli.ResolveAuth()
	if err != nil {
		return nil, err
	}

	var env []string
	if needURL {
		env = append(env, "AIBOM_POST_URL="+aibomAnalysisURL(baseURL))
	}
	if needKey {
		env = append(env, "AI_DEFENSE_API_KEY="+apiKey)
	}
	return env, nil
}

// isReportUpload reports whether the first positional selects the upload action.
func isReportUpload(args []string) bool {
	return strings.EqualFold(firstNonFlag(args), "upload")
}

// aibomAnalysisURL builds the AIBOM analysis ingest endpoint for a base URL,
// matching the SDK's path layout (api/ai-defense/v1/aibom/analysis).
func aibomAnalysisURL(baseURL string) string {
	return strings.Join([]string{
		strings.TrimRight(baseURL, "/"),
		api.APIPrefix,
		api.DefaultAPIVersion,
		aibomsdk.AIBomAnalysis,
	}, "/")
}
