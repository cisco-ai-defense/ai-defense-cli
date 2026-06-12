package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewBaseClient_Defaults(t *testing.T) {
	c := NewBaseClient("test-key", NewConfig("https://example.cisco.com"))
	if c.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", c.apiKey)
	}
	if !strings.Contains(c.apiPrefix, "api/ai-defense/v1") {
		t.Errorf("expected apiPrefix to contain 'api/ai-defense/v1', got %q", c.apiPrefix)
	}
	if got := c.HTTPClient.Timeout; got != 30*time.Second {
		t.Errorf("expected API timeout 30s, got %v", got)
	}
	if got := c.UploadHTTPClient.Timeout; got != 30*time.Second {
		t.Errorf("expected default upload timeout 30s, got %v", got)
	}
}

func TestNewBaseClient_NilConfig(t *testing.T) {
	c := NewBaseClient("k", nil)
	if c.apiPrefix != "/api/ai-defense/v1" {
		t.Errorf("expected default prefix, got %q", c.apiPrefix)
	}
}

func TestNewBaseClient_CustomAPIVersion(t *testing.T) {
	cfg := &Config{ManagementBaseURL: "https://test.cisco.com", APIVersion: "v2"}
	c := NewBaseClient("k", cfg)
	if !strings.HasSuffix(c.apiPrefix, "/v2") {
		t.Errorf("expected prefix ending in /v2, got %q", c.apiPrefix)
	}
}

func TestNewBaseClient_CustomUploadTimeout(t *testing.T) {
	cfg := &Config{
		ManagementBaseURL: "https://test.cisco.com",
		Timeout:           5,
		UploadTimeout:     120,
		APIVersion:        "v1",
	}
	c := NewBaseClient("k", cfg)
	if got := c.HTTPClient.Timeout; got != 5*time.Second {
		t.Errorf("expected API timeout 5s, got %v", got)
	}
	if got := c.UploadHTTPClient.Timeout; got != 120*time.Second {
		t.Errorf("expected upload timeout 120s, got %v", got)
	}
}

func TestDoRequest_SetsHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(AuthHeader); got != "my-api-key" {
			t.Errorf("expected auth header 'my-api-key', got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept 'application/json', got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewBaseClient("my-api-key", NewConfig(srv.URL))
	resp, err := c.Get("test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestDoRequest_PostBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]string
		json.Unmarshal(body, &m)
		if m["name"] != "test" {
			t.Errorf("expected body name 'test', got %q", m["name"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	resp, err := c.Post("items", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	json.Unmarshal(resp, &result)
	if result["id"] != "123" {
		t.Errorf("expected id '123', got %q", result["id"])
	}
}

func TestDoRequest_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"forbidden","request_id":"req-123"}`))
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	_, err := c.Get("test", nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	apiErr, ok := err.(*ApiError)
	if !ok {
		t.Fatalf("expected *ApiError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected status 403, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "forbidden" {
		t.Errorf("expected message 'forbidden', got %q", apiErr.Message)
	}
	if apiErr.RequestID != "req-123" {
		t.Errorf("expected request-id 'req-123', got %q", apiErr.RequestID)
	}
}

func TestDoRequest_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	resp, err := c.Delete("items/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for 204, got %s", string(resp))
	}
}

func TestDoRequest_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %q", r.URL.Query().Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	params := map[string][]string{"limit": {"5"}}
	_, err := c.Get("items", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApiError_String(t *testing.T) {
	e := &ApiError{StatusCode: 404, Message: "not found"}
	if !strings.Contains(e.Error(), "404") || !strings.Contains(e.Error(), "not found") {
		t.Errorf("unexpected error string: %s", e.Error())
	}

	e2 := &ApiError{StatusCode: 500, Message: "fail", RequestID: "abc"}
	if !strings.Contains(e2.Error(), "abc") {
		t.Errorf("expected request-id in error string: %s", e2.Error())
	}
}

func TestValidationError_String(t *testing.T) {
	e := &ValidationError{Message: "invalid input"}
	if e.Error() != "invalid input" {
		t.Errorf("expected 'invalid input', got %q", e.Error())
	}
}

func TestDoRawPUT_UsesUploadHTTPClient(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		time.Sleep(20 * time.Millisecond)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.HTTPClient.Timeout = 1 * time.Millisecond
	c.UploadHTTPClient.Timeout = 1 * time.Second
	c.UploadHTTPClient.Transport = transport

	if err := c.DoRawPUT("https://uploads.example.com/presigned", strings.NewReader("x"), 1); err != nil {
		t.Fatalf("DoRawPUT() error = %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateUploadURL(t *testing.T) {
	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))

	if err := c.DoRawPUT("http://example.com/upload", strings.NewReader("x"), 1); err == nil {
		t.Fatal("expected error for non-https upload URL")
	}
	if err := c.DoRawPUT("https://127.0.0.1/upload", strings.NewReader("x"), 1); err == nil {
		t.Fatal("expected error for localhost upload URL")
	}
}

func TestDoRawPUT_NilUploadClientUsesRedirectSafeFallback(t *testing.T) {
	redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://127.0.0.1/evil", http.StatusTemporaryRedirect)
	}))
	defer redirectSrv.Close()

	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.UploadHTTPClient = nil

	err := c.DoRawPUT(redirectSrv.URL, strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("expected error when fallback upload client hits redirect to private host")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDoRawPUT_RejectsRedirectToPrivateHost(t *testing.T) {
	redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://127.0.0.1/evil", http.StatusTemporaryRedirect)
	}))
	defer redirectSrv.Close()

	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	err := c.DoRawPUT(redirectSrv.URL, strings.NewReader("x"), 1)
	if err == nil {
		t.Fatal("expected error when upload redirects to private host")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}
