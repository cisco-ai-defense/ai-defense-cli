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
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func buildAuthConfig(authType, authHeader, authKey, oauthClientID, oauthClientSecret, oauthServerURL string) (mcpscansdk.AuthConfig, error) {
	switch strings.ToLower(authType) {
	case "no-auth":
		return mcpscansdk.AuthConfig{AuthType: mcpscansdk.AuthTypeNoAuth}, nil
	case "api-key":
		if authHeader == "" || authKey == "" {
			return mcpscansdk.AuthConfig{}, clierror.Usage("--auth-header and --auth-key are required when --auth-type is api-key")
		}
		return mcpscansdk.AuthConfig{
			AuthType: mcpscansdk.AuthTypeAPIKey,
			APIKey:   &mcpscansdk.ApiKeyConfig{HeaderName: authHeader, APIKey: authKey},
		}, nil
	case "oauth":
		if oauthClientID == "" || oauthServerURL == "" {
			return mcpscansdk.AuthConfig{}, clierror.Usage("--oauth-client-id and --oauth-server-url are required when --auth-type is oauth")
		}
		return mcpscansdk.AuthConfig{
			AuthType: mcpscansdk.AuthTypeOAuth,
			OAuth: &mcpscansdk.OAuthConfig{
				ClientID:      oauthClientID,
				ClientSecret:  oauthClientSecret,
				AuthServerURL: oauthServerURL,
			},
		}, nil
	default:
		return mcpscansdk.AuthConfig{}, clierror.Usage(fmt.Sprintf("unknown auth type: %s", authType))
	}
}

func addAuthFlags(cmd *cobra.Command, authType, authHeader, authKey, oauthClientID, oauthClientSecret, oauthServerURL *string) {
	cmd.Flags().StringVar(authType, "auth-type", "no-auth", "Authentication type: no-auth, api-key, oauth")
	cmd.Flags().StringVar(authHeader, "auth-header", "", "Header name for API key auth")
	cmd.Flags().StringVar(authKey, "auth-key", "", "API key value for the MCP server")
	cmd.Flags().StringVar(oauthClientID, "oauth-client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(oauthClientSecret, "oauth-client-secret", "", "OAuth client secret")
	cmd.Flags().StringVar(oauthServerURL, "oauth-server-url", "", "OAuth authorization server URL")
}

func parseTransport(transport string) (mcpscansdk.TransportType, error) {
	switch {
	case strings.EqualFold(transport, "SSE"):
		return mcpscansdk.TransportSSE, nil
	case strings.EqualFold(transport, "STREAMABLE"):
		return mcpscansdk.TransportStreamable, nil
	default:
		return "", clierror.Usage(fmt.Sprintf("invalid --transport %q (must be SSE or STREAMABLE)", transport))
	}
}
