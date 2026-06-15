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
