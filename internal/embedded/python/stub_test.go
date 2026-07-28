// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
