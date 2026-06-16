package tui

import (
	"errors"
	"os"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var initRenderer sync.Once

// ensureRenderer points lipgloss at stderr (same stream bubbletea uses) so
// adaptive colors match the live TUI. This replaces the process-wide default
// lipgloss renderer; native --help uses raw ANSI and is unaffected.
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
	return Run(NewForm(groups...))
}

// RunFormEscBack runs a form where esc returns ErrFormBack (previous wizard step).
// Ctrl+C cancels the wizard (same as RunForm → NormalizeFormErr).
func RunFormEscBack(groups ...*huh.Group) error {
	var cancelled bool
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "back · ctrl+c cancel"),
	)
	form := NewForm(groups...).
		WithKeyMap(km).
		WithProgramOptions(tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if k, ok := msg.(tea.KeyMsg); ok && k.Type == tea.KeyCtrlC {
				cancelled = true
			}
			return msg
		}))
	err := form.Run()
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) {
		if cancelled {
			return NormalizeFormErr(huh.ErrUserAborted)
		}
		return ErrFormBack
	}
	return err
}

// Run executes a styled form and normalizes cancellation errors.
func Run(form *huh.Form) error {
	return NormalizeFormErr(form.Run())
}

func terminalWidth(w *os.File) int {
	if width, _, err := term.GetSize(int(w.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}
