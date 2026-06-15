package mcpscan

import (
	"encoding/json"
	"testing"
)

func TestStartMCPServerScanResponse_UnmarshalJSON(t *testing.T) {
	t.Run("snake_case", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{"scan_id":"s1"}`), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.ScanID != "s1" {
			t.Fatalf("got %q", resp.ScanID)
		}
	})

	t.Run("camelCase", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{"scanId":"s2"}`), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.ScanID != "s2" {
			t.Fatalf("got %q", resp.ScanID)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{`), &resp); err == nil {
			t.Fatal("expected error")
		}
	})
}
