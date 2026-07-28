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
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

const (
	showActionOverview = "overview"
	showActionCore     = "core"
	showActionPlain    = "plain"
	showActionExit     = "exit"
)

// RunShowInteractive browses redacted config in the terminal.
func RunShowInteractive(m *appconfig.Manager) error {
	if m.ConfigPath() == "" {
		return clierror.Usage("no config file found; run cisco-aid-cli onboard first")
	}
	if err := m.LoadError(); err != nil {
		return clierror.Usage(err.Error())
	}

	tui.PrintBanner(os.Stdout, "Configuration", "Browse saved settings (secrets redacted).\n↑/↓ move · Enter select · Ctrl+C cancel")

	choice := showActionOverview
	for choice != showActionExit {
		menuOpts := []huh.Option[string]{
			huh.NewOption("Overview", showActionOverview),
			huh.NewOption("Core API settings", showActionCore),
		}
		for _, cap := range capabilities.All() {
			id := cap.ID()
			label := cap.Label()
			if cap.IsConfigured(m) {
				label += " (configured)"
			}
			menuOpts = append(menuOpts, huh.NewOption(label, id))
		}
		menuOpts = append(menuOpts,
			huh.NewOption("Print plain TOML", showActionPlain),
			huh.NewOption("Exit", showActionExit),
		)

		form := tui.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Configuration").
					Description(fmt.Sprintf("File: %s\n↑/↓ move · Enter to view", appconfig.ShortenHome(m.ConfigPath()))).
					Options(menuOpts...).
					Value(&choice),
			),
		)
		if err := tui.Run(form); err != nil {
			return err
		}

		switch choice {
		case showActionExit:
			return nil
		case showActionPlain:
			if err := printPlainConfig(m); err != nil {
				return err
			}
		default:
			desc := sectionShowText(m, choice)
			view := tui.NewForm(
				huh.NewGroup(
					huh.NewNote().
						Title(showSectionTitle(choice)).
						Description(desc),
				),
			)
			if err := tui.Run(view); err != nil {
				return err
			}
		}
	}
	return nil
}

func showSectionTitle(section string) string {
	switch section {
	case showActionOverview:
		return "Overview"
	case showActionCore:
		return "Core API settings"
	default:
		if c := capabilityByID(section); c != nil {
			return c.Label()
		}
		return section
	}
}

func sectionShowText(m *appconfig.Manager, section string) string {
	redacted := appconfig.RedactSecrets(m.Data())
	switch section {
	case showActionOverview:
		return formatConfigOverview(m, redacted)
	case showActionCore:
		return formatCoreSection(redacted)
	default:
		return formatCapabilitySection(m, section, redacted)
	}
}

func formatConfigOverview(m *appconfig.Manager, redacted map[string]interface{}) string {
	var b strings.Builder
	fmt.Fprintf(&b, "File: %s\n", appconfig.ShortenHome(m.ConfigPath()))
	fmt.Fprintln(&b, "\nEnvironment variables and CLI flags override file values at runtime.")
	fmt.Fprintln(&b, "\n── Core API ──")
	fmt.Fprintln(&b, formatCoreSection(redacted))
	for _, cap := range capabilities.All() {
		fmt.Fprintf(&b, "\n── %s ──\n", cap.Label())
		if cap.IsConfigured(m) {
			fmt.Fprintln(&b, formatCapabilitySection(m, cap.ID(), redacted))
		} else {
			fmt.Fprintln(&b, "  Not configured")
		}
	}
	step := m.NestedString("onboard", "step")
	if step != "" {
		fmt.Fprintf(&b, "\nOnboarding step: %s\n", step)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatCoreSection(redacted map[string]interface{}) string {
	def, _ := redacted["default"].(map[string]interface{})
	if def == nil {
		return "  (no core settings in file)"
	}
	var lines []string
	sec := spec.RequiredSection()
	for _, key := range sec.RequiredFieldKeys() {
		f := sec.FieldByKey(key)
		if f == nil {
			continue
		}
		v, _ := def[f.Key].(string)
		if v == "" {
			lines = append(lines, fmt.Sprintf("  %s: (not set)", f.Title))
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", f.Title, v))
	}
	if len(lines) == 0 {
		return "  (no core settings in file)"
	}
	return strings.Join(lines, "\n")
}

func formatCapabilitySection(m *appconfig.Manager, id string, redacted map[string]interface{}) string {
	cap := capabilityByID(id)
	if cap == nil {
		return "  Unknown section"
	}
	sc, ok := cap.(capabilities.SpecCapability)
	if !ok {
		return "  (unavailable)"
	}
	def, _ := redacted["default"].(map[string]interface{})
	if def == nil {
		return "  Not configured"
	}
	nested, _ := def[sc.Spec.ID].(map[string]interface{})
	if nested == nil {
		return "  Not configured"
	}
	var lines []string
	for _, f := range sc.Spec.Fields {
		v, _ := nested[f.Key].(string)
		if v == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", f.Title, v))
	}
	if len(lines) == 0 {
		return "  Not configured"
	}
	return strings.Join(lines, "\n")
}

func capabilityByID(id string) capabilities.Capability {
	for _, c := range capabilities.All() {
		if c.ID() == id {
			return c
		}
	}
	return nil
}

func printPlainConfig(m *appconfig.Manager) error {
	data := appconfig.RedactSecrets(m.Data())
	raw, err := marshalTOML(data)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(raw))
	return nil
}

func marshalTOML(data map[string]interface{}) ([]byte, error) {
	raw, err := toml.Marshal(data)
	if err != nil {
		return nil, clierror.Unexpected("encode config: "+err.Error(), err)
	}
	return raw, nil
}
