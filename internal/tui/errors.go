package tui

import (
	"errors"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// NormalizeFormErr maps interactive form errors to CLI contract errors.
// Ctrl+C is intentional user cancellation (exit 1), not a usage error (exit 2).
func NormalizeFormErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return clierror.Unexpected("Cancelled.", err)
	}
	return err
}
