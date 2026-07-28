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

package configcmd

import (
	"strings"
	"testing"

	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestProfileSelectLabel(t *testing.T) {
	t.Parallel()
	got := profileSelectLabel(appconfig.ProfileInfo{
		Name:   "work",
		Active: true,
		Exists: true,
	})
	if !strings.Contains(got, "active") || !strings.Contains(got, "ready") {
		t.Fatalf("got %q", got)
	}
}

func TestFormatProfileOverview(t *testing.T) {
	t.Parallel()
	profiles := []appconfig.ProfileInfo{
		{Name: "default", Active: false, Exists: true, Path: "/home/u/.ai-defense/config.toml"},
		{Name: "work", Active: true, Exists: true, Path: "/home/u/.ai-defense/profiles/work.toml"},
	}
	out := formatProfileOverview(profiles, "work")
	if !strings.Contains(out, "Stored active profile: work") {
		t.Fatalf("missing active line: %q", out)
	}
	if !strings.Contains(out, "* work") {
		t.Fatalf("missing active marker: %q", out)
	}
	if strings.Contains(out, "/home/") {
		t.Fatal("overview should not include file paths")
	}
}
