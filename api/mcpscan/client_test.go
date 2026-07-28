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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetServer(t *testing.T) {
	const body = `{"server_id":"srv-1","name":"Test"}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "srv-1") {
			t.Errorf("expected server id in path, got %s", r.URL.Path)
		}
		w.Write([]byte(body))
	})

	raw, err := c.GetServer("srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("expected %s, got %s", body, raw)
	}
}

func TestGetScanStatusRaw(t *testing.T) {
	const body = `{"scan_id":"scan-1","status":"COMPLETED"}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	raw, err := c.GetScanStatusRaw("scan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("expected verbatim body, got %s", raw)
	}
}

func TestGetServerScanReport_NilRequest(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req GetMCPServerScanReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.ServerID != "srv-1" {
			t.Errorf("expected server_id srv-1, got %q", req.ServerID)
		}
		w.Write([]byte(`{"report":{}}`))
	})

	raw, err := c.GetServerScanReport("srv-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetServerScanReport_WithRequest(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req GetMCPServerScanReportRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ServerID != "srv-2" {
			t.Errorf("expected server_id injected, got %q", req.ServerID)
		}
		w.Write([]byte(`{"report":{"status":"ok"}}`))
	})

	raw, err := c.GetServerScanReport("srv-2", &GetMCPServerScanReportRequest{ServerID: "srv-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(raw), "ok") {
		t.Fatalf("expected report body, got %s", raw)
	}
}

func TestGetServerScanSummary(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"total_scans":5}`))
	})

	raw, err := c.GetServerScanSummary("srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(raw), "total_scans") {
		t.Fatalf("expected summary JSON, got %s", raw)
	}
}

func TestGetServerThreats(t *testing.T) {
	capType := CapabilityTool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("capability_type") != "TOOL" {
			t.Errorf("expected capability_type=TOOL, got %q", q.Get("capability_type"))
		}
		if q.Get("limit") != "25" {
			t.Errorf("expected limit=25, got %q", q.Get("limit"))
		}
		if q.Get("threat_severity") != "HIGH" {
			t.Errorf("expected threat_severity=HIGH, got %q", q.Get("threat_severity"))
		}
		w.Write([]byte(`{"items":[]}`))
	})

	raw, err := c.GetServerThreats("srv-1", &capType, []string{"HIGH"}, 25, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestUpdateAuthConfig(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	err := c.UpdateAuthConfig("srv-1", &AuthConfig{AuthType: AuthTypeNoAuth})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListServers_AllFilters(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		checks := map[string]string{
			"limit": "10", "offset": "5", "server_name_substr": "acme",
			"registry_id": "reg-1", "sort_by": "SCAN_DATE", "sort_order": "asc",
		}
		for k, want := range checks {
			if q.Get(k) != want {
				t.Errorf("expected %s=%q, got %q", k, want, q.Get(k))
			}
		}
		if len(q["onboarding_status"]) != 1 || q["onboarding_status"][0] != "COMPLETED" {
			t.Errorf("unexpected onboarding_status: %v", q["onboarding_status"])
		}
		w.Write([]byte(`{"items":[]}`))
	})

	_, err := c.ListServers(ListServersParams{
		Limit:            10,
		Offset:           5,
		NameSubstr:       "acme",
		OnboardingStatus: []OnboardingStatus{OnboardingCompleted},
		TransportType:    []TransportType{TransportSSE},
		Severity:         []SeverityLevel{SeverityHigh},
		RegistryID:       "reg-1",
		SortBy:           SortScanDate,
		SortOrder:        SortOrderAsc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartScan_InvalidJSON(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})

	_, err := c.StartScan(&StartMCPServerScanRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGetScanStatus_InvalidJSON(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{`))
	})

	_, err := c.GetScanStatus("scan-1")
	if err == nil {
		t.Fatal("expected parse error")
	}
}
