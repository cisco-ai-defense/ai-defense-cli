package clierror

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

func TestClassifyAPIError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		wantCode int
		wantHint bool
	}{
		{"bad request", 400, CodeUsage, true},
		{"unauthorized", 401, CodeAuth, true},
		{"forbidden", 403, CodeAuth, true},
		{"not found", 404, CodeScanFailed, true},
		{"rate limited", 429, CodeNetwork, true},
		{"server error", 500, CodeNetwork, true},
		{"bad gateway", 503, CodeNetwork, true},
		{"teapot/unknown", 418, CodeUnexpected, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &api.ApiError{StatusCode: tt.status, Message: "boom", RequestID: "req-1"}
			got := Classify(err)
			if got.Code != tt.wantCode {
				t.Fatalf("status %d: want code %d, got %d", tt.status, tt.wantCode, got.Code)
			}
			if got.RequestID != "req-1" {
				t.Errorf("status %d: request id not propagated, got %q", tt.status, got.RequestID)
			}
			if (got.Hint != "") != tt.wantHint {
				t.Errorf("status %d: want hint=%v, got hint=%q", tt.status, tt.wantHint, got.Hint)
			}
		})
	}
}

func TestClassifyTransportError(t *testing.T) {
	err := &api.TransportError{
		Method: "GET",
		URL:    "https://tenant.aidefense.aiteam.cisco.com/api/ai-defense/v1/scans",
		Cause:  context.DeadlineExceeded,
	}
	got := Classify(err)
	if got.Code != CodeNetwork {
		t.Fatalf("want CodeNetwork, got %d", got.Code)
	}
	if got.Hint == "" {
		t.Fatal("expected network hint")
	}
	if !strings.Contains(got.Message, "timed out") {
		t.Fatalf("expected timeout message, got %q", got.Message)
	}
}

func TestClassifyPollTimeoutError(t *testing.T) {
	err := &api.PollTimeoutError{ScanID: "scan-1", BudgetSecs: 300}
	got := Classify(err)
	if got.Code != CodeNetwork {
		t.Fatalf("want CodeNetwork, got %d", got.Code)
	}
	if got.Hint == "" {
		t.Fatal("expected remediation hint")
	}
}

func TestClassifyValidationError(t *testing.T) {
	err := &api.ValidationError{Message: "bad input"}
	got := Classify(err)
	if got.Code != CodeUsage {
		t.Fatalf("want CodeUsage, got %d", got.Code)
	}
	if got.Message != "bad input" {
		t.Errorf("want message preserved, got %q", got.Message)
	}
}

func TestClassifyPassThroughTypedError(t *testing.T) {
	in := Gate(CodeBlock, "blocked")
	got := Classify(in)
	if got != in {
		t.Fatalf("expected the same *Error to be returned unchanged")
	}
	if got.Code != CodeBlock {
		t.Errorf("want CodeBlock, got %d", got.Code)
	}
}

func TestClassifyWrappedTypedError(t *testing.T) {
	in := Auth("no creds", "set --api-key", nil)
	wrapped := fmt.Errorf("context: %w", in)
	got := Classify(wrapped)
	if got.Code != CodeAuth {
		t.Fatalf("want CodeAuth through wrapping, got %d", got.Code)
	}
}

func TestClassifyUnknownError(t *testing.T) {
	got := Classify(errors.New("something odd"))
	if got.Code != CodeUnexpected {
		t.Fatalf("want CodeUnexpected, got %d", got.Code)
	}
}

func TestClassifyNil(t *testing.T) {
	if got := Classify(nil); got != nil {
		t.Fatalf("want nil for nil error, got %#v", got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("root")
	e := Network("net down", "check vpn", cause)
	if !errors.Is(e, cause) {
		t.Fatalf("expected errors.Is to find the cause")
	}
}
