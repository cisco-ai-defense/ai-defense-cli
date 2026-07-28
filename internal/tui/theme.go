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

package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Colors aligned with internal/cli/help.go (Typer/Rich aibom help): bold-cyan
// options, bold-green switches, bold-yellow accents, faint borders.
var (
	colorOption = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"} // cyan
	colorSwitch = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"} // green
	colorAccent = lipgloss.AdaptiveColor{Light: "#CA8A04", Dark: "#FACC15"} // yellow
	colorText   = lipgloss.AdaptiveColor{Light: "#374151", Dark: "#E5E7EB"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#4B5563"}
	colorError  = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
)

var typerTheme *huh.Theme

// ThemeTyper returns a huh theme matching CLI help panels.
func ThemeTyper() *huh.Theme {
	if typerTheme != nil {
		return typerTheme
	}
	ensureRenderer()

	t := huh.ThemeBase()

	// Charm-style foreground for typed input (256-color ANSI; works on light/dark).
	inputFg := lipgloss.AdaptiveColor{Light: "235", Dark: "252"}

	// Left accent border (huh default) — full rounded boxes break width/ANSI nesting.
	t.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(colorBorder)
	t.Focused.Title = lipgloss.NewStyle().Foreground(colorOption).Bold(true)
	t.Focused.NoteTitle = t.Focused.Title.Copy().MarginBottom(1)
	t.Focused.Description = lipgloss.NewStyle().Foreground(colorMuted)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(colorError)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(colorError)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(colorSwitch).Bold(true).SetString("> ")
	t.Focused.NextIndicator = lipgloss.NewStyle().Foreground(colorAccent).MarginLeft(1).SetString("→")
	t.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(colorAccent).MarginRight(1).SetString("←")
	t.Focused.Option = lipgloss.NewStyle().Foreground(colorText)
	t.Focused.MultiSelectSelector = lipgloss.NewStyle().Foreground(colorSwitch).Bold(true).SetString("> ")
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(colorOption).Bold(true)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(colorSwitch).Bold(true).SetString("✓ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(colorMuted).SetString("  ")
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(colorText)
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#111827"}).
		Background(colorOption).
		Bold(true).
		Padding(0, 2)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(colorText).
		Background(lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}).
		Padding(0, 2)
	t.Focused.Next = t.Focused.FocusedButton

	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(colorSwitch).Bold(true)
	t.Focused.TextInput.CursorText = lipgloss.NewStyle().Foreground(inputFg)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(colorMuted)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(colorSwitch).Bold(true)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(inputFg)

	t.Blurred = t.Focused
	t.Blurred.Base = lipgloss.NewStyle().PaddingLeft(1).BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Title = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	t.Blurred.NoteTitle = t.Blurred.Title
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Help = help.New().Styles
	t.Help.ShortKey = t.Help.ShortKey.Foreground(colorMuted)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(colorText)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(colorBorder)
	t.Help.FullKey = t.Help.FullKey.Foreground(colorMuted)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(colorText)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(colorBorder)
	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(colorMuted)

	typerTheme = t
	return t
}
