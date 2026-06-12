package mcpscan

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("test-key", api.NewConfig(srv.URL))
}

func TestRegisterServer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["name"] != "Test Server" {
			t.Errorf("expected name 'Test Server', got %v", req["name"])
		}
		w.Write([]byte(`{"server_id":"srv-1"}`))
	})

	resp, err := c.RegisterServer(&RegisterMCPServerRequest{Name: "Test Server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestDeleteServer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.DeleteServer("srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"scan_id":"scan-1"}`))
	})

	resp, err := c.StartScan(&StartMCPServerScanRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanID != "scan-1" {
		t.Errorf("expected 'scan-1', got %q", resp.ScanID)
	}
}

func TestStartScan_LowerCamelScanID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"scanId":"scan-camel-1"}`))
	})

	resp, err := c.StartScan(&StartMCPServerScanRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanID != "scan-camel-1" {
		t.Errorf("expected 'scan-camel-1', got %q", resp.ScanID)
	}
}

func TestGetScanStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scan_id":"scan-1","status":"COMPLETED"}`))
	})

	resp, err := c.GetScanStatus("scan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != MCPScanStatusCompleted {
		t.Errorf("expected COMPLETED, got %q", resp.Status)
	}
}

func TestListServers_WithParams(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("limit") != "20" {
			t.Errorf("expected limit=20, got %q", q.Get("limit"))
		}
		if q.Get("server_name_substr") != "test" {
			t.Errorf("expected server_name_substr=test, got %q", q.Get("server_name_substr"))
		}
		w.Write([]byte(`{"items":[]}`))
	})

	resp, err := c.ListServers(ListServersParams{
		Limit:      20,
		NameSubstr: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetServerCapabilities(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("capability_type") != "TOOL" {
			t.Errorf("expected capability_type=TOOL, got %q", q.Get("capability_type"))
		}
		w.Write([]byte(`{"items":[]}`))
	})

	resp, err := c.GetServerCapabilities("srv-1", CapabilityTool, "", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestTriggerServerScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	err := c.TriggerServerScan("srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateServers(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"valid_urls":["https://mcp.example.com"]}`))
	})

	resp, err := c.ValidateServers(&ValidateMCPServersRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
