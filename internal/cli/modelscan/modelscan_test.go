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
