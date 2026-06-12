//go:build !embed_python

package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInterpreter_UsesAIDCliPythonWhenSet(t *testing.T) {
	tmpDir := t.TempDir()
	pyPath := filepath.Join(tmpDir, "python3")
	if err := os.WriteFile(pyPath, []byte("#!/bin/sh\n"), 0700); err != nil {
		t.Fatalf("write test interpreter: %v", err)
	}

	t.Setenv("AID_CLI_PYTHON", pyPath)

	got, err := Interpreter()
	if err != nil {
		t.Fatalf("Interpreter() error = %v", err)
	}
	if got != pyPath {
		t.Fatalf("Interpreter() = %q, want %q", got, pyPath)
	}
}

func TestInterpreter_ErrorsWhenAIDCliPythonMissing(t *testing.T) {
	t.Setenv("AID_CLI_PYTHON", filepath.Join(t.TempDir(), "does-not-exist"))

	if _, err := Interpreter(); err == nil {
		t.Fatal("Interpreter() error = nil, want non-nil")
	}
}

func TestInterpreter_ReturnsStubErrorWhenUnset(t *testing.T) {
	t.Setenv("AID_CLI_PYTHON", "")

	_, err := Interpreter()
	if err == nil {
		t.Fatal("Interpreter() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "embedded Python is not available") {
		t.Fatalf("Interpreter() error = %q, missing stub message", err.Error())
	}
}
