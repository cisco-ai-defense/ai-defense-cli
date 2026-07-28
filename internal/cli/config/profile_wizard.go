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
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

const profileCreateNew = "__new__"

// resolveWizardProfile picks the target profile (flag skips the picker).
func resolveWizardProfile(opts RunOptions) (string, error) {
	if opts.Profile != "" {
		if err := appconfig.ValidateProfileName(opts.Profile); err != nil {
			return "", clierror.Usage(err.Error())
		}
		return opts.Profile, nil
	}
	return promptProfileSelection()
}

func promptProfileSelection() (string, error) {
	profiles, err := appconfig.ListProfilesForOnboarding()
	if err != nil {
		return "", clierror.Unexpected("list profiles: "+err.Error(), err)
	}

	storedActive := appconfig.ReadStoredActiveProfile()
	choice := storedActive
	if choice == "" {
		choice = appconfig.DefaultProfileName
	}

	options := make([]huh.Option[string], 0, len(profiles)+1)
	for _, p := range profiles {
		options = append(options, huh.NewOption(profileSelectLabel(p), p.Name))
	}
	options = append(options, huh.NewOption("Create new profile…", profileCreateNew))

	desc := formatProfileOverview(profiles, storedActive)

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Configuration profile").
				Description(desc).
				Options(options...).
				Value(&choice),
		),
	)
	if err := tui.Run(form); err != nil {
		return "", err
	}

	if choice == profileCreateNew {
		return promptNewProfileName(profiles)
	}
	return choice, nil
}

func profileSelectLabel(p appconfig.ProfileInfo) string {
	var b strings.Builder
	b.WriteString(p.Name)
	if p.Active {
		b.WriteString(" (active)")
	}
	if p.Exists {
		b.WriteString(" · ready")
	} else {
		b.WriteString(" · not configured")
	}
	return b.String()
}

func formatProfileOverview(profiles []appconfig.ProfileInfo, storedActive string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Stored active profile: %s\n\n", storedActive))
	if len(profiles) == 0 {
		b.WriteString("No profiles on disk yet. Choose default or create a new one.")
		return b.String()
	}
	b.WriteString("Existing profiles:\n")
	for _, p := range profiles {
		marker := " "
		if p.Active {
			marker = "*"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", marker, profileSelectLabel(p)))
	}
	b.WriteString("\nSelect a profile to configure, or create a new one.")
	return b.String()
}

func promptNewProfileName(existing []appconfig.ProfileInfo) (string, error) {
	name := ""
	known := map[string]bool{appconfig.DefaultProfileName: true}
	for _, p := range existing {
		known[p.Name] = true
	}

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New profile name").
				Description("Letters, digits, hyphen, underscore (not \"default\").").
				Placeholder("work").
				Value(&name).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("profile name is required")
					}
					if err := appconfig.ValidateProfileName(s); err != nil {
						return err
					}
					if s == appconfig.DefaultProfileName {
						return fmt.Errorf("use the default profile option instead of creating %q", s)
					}
					if known[s] {
						return fmt.Errorf("profile %q already exists; select it from the list", s)
					}
					return nil
				}),
		),
	)
	if err := tui.Run(form); err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}
