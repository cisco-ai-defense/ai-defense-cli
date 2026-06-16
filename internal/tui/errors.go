package tui

import (
	"errors"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// ErrFormBack means the user pressed esc to return to the previous wizard step.
var ErrFormBack = errors.New("wizard step back")

// NormalizeFormErr maps interactive form errors to CLI contract errors.
// Ctrl+C is intentional user cancellation (exit 1), not a usage error (exit 2).
func NormalizeFormErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrFormBack) {
		return ErrFormBack
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return clierror.Unexpected("Cancelled.", err)
	}
	return err
}
