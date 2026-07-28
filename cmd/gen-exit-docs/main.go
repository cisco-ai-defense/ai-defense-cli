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

// gen-exit-docs writes docs/exit-codes-handbook.md from internal/cliexitdoc.
//
// Usage:
//
//	go run ./cmd/gen-exit-docs [-o docs/exit-codes-handbook.md]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cliexitdoc"
)

func main() {
	out := flag.String("o", "docs/exit-codes-handbook.md", "output markdown path")
	flag.Parse()

	if err := cliexitdoc.ValidateRegistry(); err != nil {
		fmt.Fprintf(os.Stderr, "registry validation failed: %v\n", err)
		os.Exit(1)
	}

	content := cliexitdoc.NormalizeGenerated(cliexitdoc.GenerateHandbook())
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *out)
}
