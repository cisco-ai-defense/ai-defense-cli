package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandVersionFlag(t *testing.T) {
	var out bytes.Buffer

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--version"})
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(out.String())
	want := "aid-cisco-cli version " + version + " (" + commit + ")"
	if got != want {
		t.Fatalf("expected version output %q, got %q", want, got)
	}
}
