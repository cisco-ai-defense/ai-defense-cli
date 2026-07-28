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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func newDeprecatedRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Deprecated: use 'trigger' instead",
		Long: `Deprecated: use "model-scan trigger" instead of "model-scan run".

This subcommand existed in the Python aidcli; it was renamed to trigger.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parent := "model-scan"
			if p := cmd.Parent(); p != nil && p.Name() != "" {
				parent = p.Name()
			}
			return clierror.Usage(fmt.Sprintf(
				"'%s run' is deprecated; use '%s trigger' instead",
				parent, parent,
			))
		},
	}
}
