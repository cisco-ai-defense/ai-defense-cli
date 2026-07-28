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

func TestFormatConfigOverview(t *testing.T) {
	t.Parallel()
	m := appconfig.NewManagerFromPath("")
	mData := map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  "secret",
			"base_url": "https://us.api.aidefense.security.cisco.com",
		},
		"onboard": map[string]interface{}{
			"step": "complete",
		},
	}
	// inject via path write
	dir := t.TempDir()
	path := dir + "/config.toml"
	if err := appconfig.WriteConfig(path, mData); err != nil {
		t.Fatal(err)
	}
	m = appconfig.NewManagerFromPath(path)

	text := formatConfigOverview(m, appconfig.RedactSecrets(m.Data()))
	if !strings.Contains(text, "API key: ***") {
		t.Fatalf("expected redacted api key, got:\n%s", text)
	}
	if !strings.Contains(text, "us.api.aidefense") {
		t.Fatalf("expected base url in overview, got:\n%s", text)
	}
	if !strings.Contains(text, "Onboarding step: complete") {
		t.Fatalf("expected onboard step, got:\n%s", text)
	}
}
