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

// Package apitest provides helpers for command-layer tests against httptest servers
// while satisfying *.cisco.com base URL validation.
package apitest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

// TestBaseURL passes config.ResolveAPIKeyAndBaseURL hostname checks in tests.
const TestBaseURL = "https://test.cisco.com"

type rewriteTransport struct {
	srv *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.srv.Scheme
	req.URL.Host = t.srv.Host
	return http.DefaultTransport.RoundTrip(req)
}

// NewTestCLI returns a cmdutil.CLI wired to handler via an httptest server.
func NewTestCLI(t *testing.T, handler http.HandlerFunc) *cmdutil.CLI {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	srvURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse httptest URL: %v", err)
	}
	client := &http.Client{Transport: &rewriteTransport{srv: srvURL}}

	t.Setenv("AI_DEFENSE_API_KEY", "test-key")
	t.Setenv("AI_DEFENSE_BASE_URL", TestBaseURL)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "/nonexistent/config.toml")

	return &cmdutil.CLI{
		ConfigManager: func() *config.Manager { return config.NewManager() },
		FlagAPIKey:    func() string { return "" },
		FlagBaseURL:   func() string { return "" },
		APIConfigFactory: func(baseURL string) *api.Config {
			cfg := api.NewConfig(baseURL)
			cfg.HTTPClient = client
			return cfg
		},
	}
}
