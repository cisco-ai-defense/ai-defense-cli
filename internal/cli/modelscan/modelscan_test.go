package modelscan

import (
	"testing"
)

func TestNewDeprecatedScanCmd_Structure(t *testing.T) {
	alias := NewDeprecatedScanCmd(testCLI(t))
	if !alias.Hidden {
		t.Fatal("expected hidden deprecated alias")
	}
	if alias.Use != "scan" {
		t.Fatalf("Use = %q, want scan", alias.Use)
	}
	if len(alias.Commands()) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(alias.Commands()))
	}
}
