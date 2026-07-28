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

package cliexitdoc

import "testing"

func TestValidateRegistry(t *testing.T) {
	if err := ValidateRegistry(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateHandbookNonEmpty(t *testing.T) {
	md := GenerateHandbook()
	if len(md) < 1000 {
		t.Fatalf("handbook too short: %d bytes", len(md))
	}
	if !contains(md, "internal/cliexitdoc/registry.go") {
		t.Fatal("expected generator attribution in handbook")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
