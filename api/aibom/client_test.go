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

package aibom

import (
	"encoding/json"
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
