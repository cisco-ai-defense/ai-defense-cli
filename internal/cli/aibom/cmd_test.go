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

package aibom

import (
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func testCLI(t *testing.T) *cmdutil.CLI {
	t.Helper()
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")
	return &cmdutil.CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
	}
}
