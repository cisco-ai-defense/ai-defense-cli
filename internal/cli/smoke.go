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
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/aibom"
)

// SmokeCommandPaths returns CLI command paths exercised by post-build smoke tests
// (each path is run with --help). Root is the empty string.
func SmokeCommandPaths() []string {
	paths := []string{""}
	var walk func(*cobra.Command, []string)
	walk = func(c *cobra.Command, prefix []string) {
		for _, child := range c.Commands() {
			if !smokeIncludeCommand(child) {
				continue
			}
			parts := append(append([]string{}, prefix...), child.Name())
			paths = append(paths, strings.Join(parts, " "))
			walk(child, parts)
		}
	}
	walk(rootCmd, nil)
	return paths
}

func smokeIncludeCommand(c *cobra.Command) bool {
	if c == nil || c.Name() == "" {
		return false
	}
	// Hidden deprecated alias (scan) remains user-facing; skip cobra help topics.
	if c.IsAdditionalHelpTopicCommand() {
		return false
	}
	// aibom passthrough commands delegate --help to the embedded analyzer, which
	// is only present in embedded builds; exclude them from the in-process and
	// go-only smoke runs.
	if c.Annotations[aibom.PassthroughAnnotation] == "true" {
		return false
	}
	return c.IsAvailableCommand() || c.Hidden
}
