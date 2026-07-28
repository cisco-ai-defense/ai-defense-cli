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

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// NewCmd creates the ai-bom parent command. Its subcommands are the union of two
// backends:
//
//   - embedded analyzer passthroughs (report, analyze, watch, benchmark, kb,
//     cache, plugin, diff) forwarded verbatim to the bundled cisco-aibom CLI; and
//   - AI Defense management commands (list, info, components, summary, delete)
//     served directly by the Go SDK against the AI Defense API.
//
// aid_cli renders only this top-level help (titled panels listing the union);
// each passthrough subcommand delegates its own --help to the analyzer. The
// commands are listed in a fixed order (analyzer commands first, mirroring
// cisco-aibom's order; cloud-management commands grouped at the end) via the
// HelpOrderAnnotation.
func NewCmd(cli *cmdutil.CLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-bom",
		Short: "Generate, manage, and explore AI Bills of Materials",
		// Bare `ai-bom` shows help and exits 0; an unknown subcommand (a typo or
		// a stale `generate`/`submit`/`run` invocation) must be a usage error (2)
		// so it never looks successful. NoArgs makes cobra reject stray positional
		// args, and the RunE shows help for the bare case (mirrors model-scan).
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Display order: embedded-analyzer passthroughs first (mirroring the
	// cisco-aibom command order), then aid_cli's AI Defense cloud-management
	// commands grouped at the end. analyze bridges LLM config; report bridges the
	// upload target; kb bridges the tenant key; the rest forward verbatim.
	ordered := []*cobra.Command{
		newReportCmd(cli),
		newPassthroughCmd(cli, "analyze [SOURCES...]",
			"Analyze source paths or images and generate an AIBOM", "analyze", aibomLLMEnv),
		newPassthroughCmd(cli, "watch [SOURCES...]",
			"Watch source paths and re-scan on change", "watch", nil),
		newPassthroughCmd(cli, "benchmark",
			"Compare scan output against ground-truth YAML", "benchmark", nil),
		newKBCmd(cli),
		newPassthroughCmd(cli, "cache",
			"Manage the analyzer result cache", "cache", nil),
		newPassthroughCmd(cli, "plugin",
			"Discover and manage analyzer plugins", "plugin", nil),
		newPassthroughCmd(cli, "diff",
			"Compare two AIBOM JSON reports", "diff", nil),

		// AI Defense management API (Go SDK; no analyzer equivalent).
		newListCmd(cli),
		newInfoCmd(cli),
		newComponentsCmd(cli),
		newSummaryCmd(cli),
		newDeleteCmd(cli),
	}

	for i, c := range ordered {
		if c.Annotations == nil {
			c.Annotations = map[string]string{}
		}
		c.Annotations[HelpOrderAnnotation] = fmt.Sprintf("%02d", i)
		cmd.AddCommand(c)
	}

	return cmd
}
