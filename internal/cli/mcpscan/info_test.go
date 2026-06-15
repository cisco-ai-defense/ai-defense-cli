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
