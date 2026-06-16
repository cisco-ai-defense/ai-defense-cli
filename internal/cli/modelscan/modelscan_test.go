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
	if len(alias.Commands()) != 4 {
		t.Fatalf("expected 4 subcommands, got %d", len(alias.Commands()))
	}
}
