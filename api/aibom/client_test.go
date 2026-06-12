package aibom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("test-key", api.NewConfig(srv.URL))
}

func TestListBoms(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %q", r.URL.Query().Get("limit"))
		}
		w.Write([]byte(`{"items":[],"paging":{"total":0}}`))
	})

	resp, err := c.ListBoms(&ListBomsRequest{Limit: 5, Order: SortOrderAscending})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestListBoms_OrderUsesBackendEnumValues(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("order"); got != "asc" {
			t.Fatalf("expected order=asc, got %q", got)
		}
		w.Write([]byte(`{"items":[]}`))
	})

	if _, err := c.ListBoms(&ListBomsRequest{Order: SortOrderAscending}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBom(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path == "" {
			t.Error("expected non-empty path")
		}
		w.Write([]byte(`{"analysis_id":"abc-123"}`))
	})

	resp, err := c.GetBom("abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resp, &result)
	if result["analysis_id"] != "abc-123" {
		t.Errorf("expected analysis_id 'abc-123', got %q", result["analysis_id"])
	}
}

func TestDeleteBom(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.DeleteBom("abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubmitReportFile(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if strings.TrimSpace(req["run_id"].(string)) == "" {
			t.Fatalf("expected non-empty run_id")
		}
		if req["analyzer_version"] != "test-analyzer" {
			t.Fatalf("expected analyzer_version=test-analyzer, got %#v", req["analyzer_version"])
		}
		if _, ok := req["submitted_at"].(string); !ok || strings.TrimSpace(req["submitted_at"].(string)) == "" {
			t.Fatalf("expected non-empty submitted_at, got %#v", req["submitted_at"])
		}
		if _, ok := req["report"].(map[string]interface{}); !ok {
			t.Fatalf("expected nested report payload")
		}
		w.Write([]byte(`{"analysis_id":"new-123","status":"accepted"}`))
	})

	resp, err := c.SubmitReportFile(map[string]interface{}{
		"aibom_analysis": map[string]interface{}{
			"metadata": map[string]interface{}{
				"run_id":           "run-123",
				"analyzer_version": "test-analyzer",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCreateAnalysisRequestFromReport_UsesMetadataSubmittedAt(t *testing.T) {
	req, err := createAnalysisRequestFromReport(map[string]interface{}{
		"aibom_analysis": map[string]interface{}{
			"metadata": map[string]interface{}{
				"run_id":           "run-123",
				"analyzer_version": "test-analyzer",
				"submitted_at":     "2026-03-19T12:00:00Z",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.SubmittedAt == nil {
		t.Fatal("expected submitted_at to be set")
	}
	expected := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	if !req.SubmittedAt.Equal(expected) {
		t.Fatalf("expected submitted_at %v, got %v", expected, req.SubmittedAt)
	}
}

func TestCreateAnalysisRequestFromReport_DefaultsSubmittedAt(t *testing.T) {
	before := time.Now().UTC()
	req, err := createAnalysisRequestFromReport(map[string]interface{}{
		"aibom_analysis": map[string]interface{}{
			"metadata": map[string]interface{}{
				"run_id":           "run-123",
				"analyzer_version": "test-analyzer",
			},
		},
	})
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.SubmittedAt == nil {
		t.Fatal("expected submitted_at default")
	}
	if req.SubmittedAt.Before(before) || req.SubmittedAt.After(after) {
		t.Fatalf("expected submitted_at between %v and %v, got %v", before, after, req.SubmittedAt)
	}
}

func TestListBomComponents(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("type") != "COMPONENT_CATEGORY_MODEL" {
			t.Errorf("expected type filter, got %q", r.URL.Query().Get("type"))
		}
		w.Write([]byte(`{"items":[]}`))
	})

	resp, err := c.ListBomComponents("id-1", &ListBomComponentsRequest{
		ComponentType: ComponentCategoryModel,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetBomSummary(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"summary":{"total_boms":42}}`))
	})

	resp, err := c.GetBomSummary(&BomsSummaryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCreateAnalysis(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"analysis_id":"new-id","status":"created"}`))
	})

	resp, err := c.CreateAnalysis(&CreateAnalysisRequest{RunID: "run-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AnalysisID != "new-id" {
		t.Errorf("expected 'new-id', got %q", resp.AnalysisID)
	}
}
