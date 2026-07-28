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

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSmokeCommandPathsNonEmpty(t *testing.T) {
	paths := SmokeCommandPaths()
	if len(paths) < 10 {
		t.Fatalf("expected many command paths, got %d: %v", len(paths), paths)
	}
}

func TestSmokeHelpAllCommands(t *testing.T) {
	for _, path := range SmokeCommandPaths() {
		path := path
		t.Run(smokeTestName(path), func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)
			args := smokeHelpArgs(path)
			rootCmd.SetArgs(args)
			t.Cleanup(func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
				rootCmd.SetArgs(nil)
			})

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", strings.Join(args, " "), err)
			}
			got := out.String()
			if got == "" {
				t.Fatal("expected non-empty help output")
			}
			if strings.Contains(got, "Error:") {
				t.Fatalf("help output contains error: %q", got)
			}
		})
	}
}

func smokeTestName(path string) string {
	if path == "" {
		return "root"
	}
	return strings.ReplaceAll(path, " ", "_")
}

func smokeHelpArgs(path string) []string {
	if path == "" {
		return []string{"--help"}
	}
	return append(strings.Split(path, " "), "--help")
}
