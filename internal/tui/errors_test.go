package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestNormalizeFormErrUserAborted(t *testing.T) {
	t.Parallel()
	err := NormalizeFormErr(huh.ErrUserAborted)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected *clierror.Error, got %T", err)
	}
	if cliErr.Code != clierror.CodeUnexpected {
		t.Fatalf("code = %d, want %d (cancel is not usage exit 2)", cliErr.Code, clierror.CodeUnexpected)
	}
	if cliErr.Message != "Cancelled." {
		t.Fatalf("message = %q", cliErr.Message)
	}
}

func TestNormalizeFormErrPassthrough(t *testing.T) {
	t.Parallel()
	orig := errors.New("other")
	if got := NormalizeFormErr(orig); !errors.Is(got, orig) {
		t.Fatalf("expected passthrough")
	}
}

func TestNormalizeFormErrBack(t *testing.T) {
	t.Parallel()
	err := NormalizeFormErr(ErrFormBack)
	if !errors.Is(err, ErrFormBack) {
		t.Fatalf("expected ErrFormBack passthrough, got %v", err)
	}
}
