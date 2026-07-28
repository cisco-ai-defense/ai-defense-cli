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

package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

const clierrorHintAuth = "Run cisco-aid-cli onboard or set AI_DEFENSE_API_KEY and AI_DEFENSE_BASE_URL."

// NewCmd returns the config command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Create, edit, and inspect the AI Defense CLI configuration file.",
	}
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newPathCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newProfileCmd())
	return cmd
}

func newInitCmd() *cobra.Command {
	var force bool
	var profile string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new configuration file interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			ApplyProfileFlag(profile)
			ctx, err := RunWizard(RunOptions{Force: force, Profile: profile})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Wrote config to %s\n", ctx.ConfigPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing config file")
	ProfileFlag(cmd, &profile)
	return cmd
}

func newEditCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit an existing configuration file interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			ApplyProfileFlag(profile)
			ctx, err := RunEditHub()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Updated config at %s\n", ctx.ConfigPath)
			return nil
		},
	}
	ProfileFlag(cmd, &profile)
	return cmd
}

func newShowCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show resolved configuration (secrets redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := appconfig.NewManager()
			if err := m.LoadError(); err != nil {
				return clierror.Usage(err.Error())
			}
			if plain || !isTTY() {
				return printPlainConfig(m)
			}
			return RunShowInteractive(m)
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "Print redacted TOML instead of the interactive browser")
	return cmd
}

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path in use",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(appconfig.ResolveConfigPath())
			return nil
		},
	}
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate required configuration fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := appconfig.NewManager()
			if err := m.LoadError(); err != nil {
				return clierror.Usage(err.Error())
			}
			if _, err := m.GetAPIKey(); err != nil {
				return clierror.Auth(err.Error(), clierrorHintAuth, err)
			}
			if _, err := m.GetBaseURL(); err != nil {
				return clierror.Auth(err.Error(), clierrorHintAuth, err)
			}
			_, _, err := appconfig.ResolveAPIKeyAndBaseURL("", "", m)
			if err != nil {
				return clierror.Auth(err.Error(), clierrorHintAuth, err)
			}
			fmt.Fprintln(os.Stdout, "Configuration is valid.")
			return nil
		},
	}
	return cmd
}
