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

package onboard

import (
	"github.com/spf13/cobra"

	configcmd "github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config"
)

// NewCmd returns the onboard command.
func NewCmd() *cobra.Command {
	var force bool
	var profile string
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Interactive setup for new users",
		Long: `Guided setup for AI Defense CLI: core credentials plus optional
capabilities (private HuggingFace repos, local AIBOM analysis).

If you already started setup, you can continue from your saved config file
instead of starting over. Run this after installing the binary from GitHub Releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configcmd.ApplyProfileFlag(profile)
			ctx, err := configcmd.RunWizard(configcmd.RunOptions{
				Force:   force,
				Profile: profile,
			})
			if err != nil {
				return err
			}
			configcmd.PrintNextSteps(ctx)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Start over and replace an existing config file")
	configcmd.ProfileFlag(cmd, &profile)
	return cmd
}
