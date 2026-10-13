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

package api

import (
	"context"
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

func TestDoRawPUTWithHeaders_ReturnsHeaders(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		h.Set("ETag", `"abc"`)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: h}, nil
	})
	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.UploadHTTPClient.Transport = transport

	h, err := c.DoRawPUTWithHeaders(context.Background(), "https://uploads.example.com/part/1", strings.NewReader("x"), 1)
	if err != nil {
		t.Fatalf("DoRawPUTWithHeaders() error = %v", err)
	}
	if h.Get("ETag") != `"abc"` {
		t.Fatalf("ETag = %q", h.Get("ETag"))
	}
}

func TestDoRawPUTNoRedirect_ReturnsRedirectAsError(t *testing.T) {
	calls := 0
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		h := make(http.Header)
		h.Set("Location", "https://uploads.example.com/elsewhere")
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Body: io.NopCloser(strings.NewReader("")), Header: h}, nil
	})
	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.UploadHTTPClient.Transport = transport

	_, err := c.DoRawPUTNoRedirect(context.Background(), "https://uploads.example.com/part/1", strings.NewReader("x"), 1)
	var apiErr *ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("error = %v, want 307 ApiError", err)
	}
	if calls != 1 {
		t.Fatalf("transport calls = %d, want 1 (redirect must not be followed)", calls)
	}
	if c.UploadHTTPClient.CheckRedirect == nil {
		t.Fatal("shared upload client must keep its redirect validation")
	}
}

func TestNewUploadHTTPClient_PoolsConnectionsPerHost(t *testing.T) {
	transport, ok := newUploadHTTPClient(5).Transport.(*http.Transport)
	if !ok || transport.MaxIdleConnsPerHost != uploadMaxIdleConnsPerHost {
		t.Fatalf("upload transport = %#v, want MaxIdleConnsPerHost %d", transport, uploadMaxIdleConnsPerHost)
	}
}

func TestDoRawPUTWithHeaders_TransportErrorOmitsSignature(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset by peer")
	})
	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.UploadHTTPClient.Transport = transport

	_, err := c.DoRawPUTWithHeaders(context.Background(), "https://uploads.example.com/part/1?X-Amz-Signature=topsecret", strings.NewReader("x"), 1)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("expected TransportError, got %T: %v", err, err)
	}
	if strings.Contains(err.Error(), "topsecret") {
		t.Fatalf("transport error leaks presigned signature: %v", err)
	}
	if !strings.Contains(err.Error(), "connection reset by peer") {
		t.Fatalf("transport error lost its cause: %v", err)
	}
}

func TestUploadErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"s3 xml", `<?xml version="1.0"?><Error><Code>SignatureDoesNotMatch</Code><Message>bad sig</Message><AWSAccessKeyId>AKIAEXAMPLE</AWSAccessKeyId><StringToSign>PUT...</StringToSign></Error>`, "SignatureDoesNotMatch: bad sig"},
		{"code only", `<Error><Code>RequestExpired</Code></Error>`, "RequestExpired"},
		{"empty body", "", ""},
		{"plain text is not echoed", "  service unavailable  ", "unexpected response from object store (23 bytes, not shown)"},
		{"long message", `<Error><Code>X</Code><Message>` + strings.Repeat("m", maxUploadErrorField+10) + `</Message></Error>`, "X: " + strings.Repeat("m", maxUploadErrorField) + "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uploadErrorMessage([]byte(tt.body)); got != tt.want {
				t.Fatalf("uploadErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUploadErrorMessage_ReflectedPresignedURL(t *testing.T) {
	const reflected = "/bucket/model.bin?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
		"&X-Amz-Credential=test-credential-placeholder%2F20260101%2Fus-west-2%2Fs3%2Faws4_request" +
		"&X-Amz-Security-Token=test-session-token-placeholder" +
		"&X-Amz-Signature=fake-signature-value"
	secrets := []string{"test-credential-placeholder", "test-session-token-placeholder", "fake-signature-value"}
	s3XML := "<Error><Code>AccessDenied</Code><Message>denied " + strings.ReplaceAll(reflected, "&", "&amp;") + "</Message></Error>"

	tests := []struct {
		name string
		body string
	}{
		{"html proxy page", "<html><body><h1>Request blocked</h1><p>URI: " + reflected + "</p></body></html>"},
		{"plain text WAF", "Forbidden: PUT " + reflected},
		{"json gateway", `{"error":"denied","path":"` + reflected + `"}`},
		{"s3-shaped xml", s3XML},
		{"sigv2 query", "<Error><Code>AccessDenied</Code><Message>?AWSAccessKeyId=test-credential-placeholder&amp;Signature=fake-signature-value</Message></Error>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uploadErrorMessage([]byte(tt.body))
			for _, s := range secrets {
				if strings.Contains(got, s) {
					t.Fatalf("uploadErrorMessage() leaks %q: %q", s, got)
				}
			}
		})
	}

	got := uploadErrorMessage([]byte(s3XML))
	if !strings.HasPrefix(got, "AccessDenied: denied") || !strings.Contains(got, "X-Amz-Signature=REDACTED") {
		t.Fatalf("S3 code and message should survive with values redacted, got %q", got)
	}
}

func TestDoRawPUT_ReflectedURLBodyNotEchoed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "<html>Blocked "+r.URL.RequestURI()+"</html>")
	}))
	defer srv.Close()

	base := srv.Client().Transport
	srvHost := strings.TrimPrefix(srv.URL, "https://")
	c := NewBaseClient("k", NewConfig("https://example.cisco.com"))
	c.UploadHTTPClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Host = srvHost
		return base.RoundTrip(req)
	})}

	err := c.DoRawPUT("https://uploads.example.com/part/1?X-Amz-Signature=fake-signature-value", strings.NewReader("x"), 1)
	var apiErr *ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("error = %v, want 403 ApiError", err)
	}
	if strings.Contains(err.Error(), "fake-signature-value") {
		t.Fatalf("upload error echoes the presigned signature: %v", err)
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

func TestPostWithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("expected page=2, got %q", r.URL.Query().Get("page"))
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	resp, err := c.PostWithParams("items", map[string][]string{"page": {"2"}}, map[string]string{"name": "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestPut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Write([]byte(`{"updated":true}`))
	}))
	defer srv.Close()

	c := NewBaseClient("k", NewConfig(srv.URL))
	resp, err := c.Put("items/1", map[string]string{"name": "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestNewBaseClient_CustomHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"via":"custom"}`))
	}))
	defer srv.Close()

	cfg := NewConfig(srv.URL)
	cfg.HTTPClient = srv.Client()
	c := NewBaseClient("k", cfg)
	resp, err := c.Get("ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(resp), "custom") {
		t.Fatalf("expected custom client response, got %s", resp)
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
