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

package mcpscan

import (
	"encoding/json"
	"testing"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
)

func TestExtractStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"status snake_case", `{"status":"FAILED"}`, "FAILED"},
		{"scanStatus camelCase", `{"scanStatus":"COMPLETED"}`, "COMPLETED"},
		{"status takes precedence", `{"status":"PENDING","scanStatus":"COMPLETED"}`, "PENDING"},
		{"empty defaults to completed", `{}`, string(mcpscansdk.MCPScanStatusCompleted)},
		{"nil raw defaults to completed", "", string(mcpscansdk.MCPScanStatusCompleted)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if tt.raw != "" {
				raw = json.RawMessage(tt.raw)
			}
			if got := extractStatus(raw); got != tt.want {
				t.Fatalf("extractStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
