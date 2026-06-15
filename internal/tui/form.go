package tui

import (
	"os"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var initRenderer sync.Once

// ensureRenderer points lipgloss at stderr (same stream bubbletea uses) so
// adaptive colors match the live TUI.
func ensureRenderer() {
	initRenderer.Do(func() {
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr))
	})
}

// NewForm builds a huh form with Typer/Rich styling and terminal width.
func NewForm(groups ...*huh.Group) *huh.Form {
	ensureRenderer()
	width := terminalWidth(os.Stderr)
	if width > 4 {
		width -= 4 // match help panel inner padding
	}
	return huh.NewForm(groups...).
		WithTheme(ThemeTyper()).
		WithWidth(width).
		WithShowHelp(true)
}

// RunForm executes a styled form.
func RunForm(groups ...*huh.Group) error {
	return NewForm(groups...).Run()
}

func terminalWidth(w *os.File) int {
	if width, _, err := term.GetSize(int(w.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}
