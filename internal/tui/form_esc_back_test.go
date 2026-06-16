package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestRunFormEscBackCancelMapsToCLIError(t *testing.T) {
	t.Parallel()
	// Simulate the cancel branch without a TTY.
	err := NormalizeFormErr(huh.ErrUserAborted)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected *clierror.Error, got %T", err)
	}
	if cliErr.Code != clierror.CodeUnexpected {
		t.Fatalf("code = %d, want %d", cliErr.Code, clierror.CodeUnexpected)
	}
	if cliErr.Message != "Cancelled." {
		t.Fatalf("message = %q", cliErr.Message)
	}
}
