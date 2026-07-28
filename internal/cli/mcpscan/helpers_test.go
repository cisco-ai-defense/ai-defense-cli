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
	"testing"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestBuildAuthConfig(t *testing.T) {
	tests := []struct {
		name      string
		authType  string
		header    string
		key       string
		oauthID   string
		oauthSec  string
		oauthURL  string
		wantType  mcpscansdk.AuthType
		wantCode  int
	}{
		{
			name:     "no-auth",
			authType: "no-auth",
			wantType: mcpscansdk.AuthTypeNoAuth,
		},
		{
			name:     "api-key ok",
			authType: "api-key",
			header:   "X-Key",
			key:      "secret",
			wantType: mcpscansdk.AuthTypeAPIKey,
		},
		{
			name:     "api-key missing header",
			authType: "api-key",
			key:      "secret",
			wantCode: clierror.CodeUsage,
		},
		{
			name:     "oauth ok",
			authType: "oauth",
			oauthID:  "client",
			oauthURL: "https://auth.example.com",
			wantType: mcpscansdk.AuthTypeOAuth,
		},
		{
			name:     "oauth missing client id",
			authType: "oauth",
			oauthURL: "https://auth.example.com",
			wantCode: clierror.CodeUsage,
		},
		{
			name:     "unknown auth type",
			authType: "basic",
			wantCode: clierror.CodeUsage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAuthConfig(tt.authType, tt.header, tt.key, tt.oauthID, tt.oauthSec, tt.oauthURL)
			if tt.wantCode != 0 {
				if err == nil {
					t.Fatal("expected error")
				}
				if code := clierror.Classify(err).Code; code != tt.wantCode {
					t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.AuthType != tt.wantType {
				t.Fatalf("AuthType = %q, want %q", got.AuthType, tt.wantType)
			}
		})
	}
}

func TestParseTransport(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    mcpscansdk.TransportType
		wantErr bool
	}{
		{"SSE", "SSE", mcpscansdk.TransportSSE, false},
		{"sse case insensitive", "sse", mcpscansdk.TransportSSE, false},
		{"STREAMABLE", "STREAMABLE", mcpscansdk.TransportStreamable, false},
		{"invalid", "websocket", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTransport(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if clierror.Classify(err).Code != clierror.CodeUsage {
					t.Fatalf("expected usage error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("transport = %q, want %q", got, tt.want)
			}
		})
	}
}
