package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTransportError_TimeoutMessage(t *testing.T) {
	err := &TransportError{
		Method: "GET",
		URL:    "https://example.cisco.com/api/ai-defense/v1/scans",
		Cause:  context.DeadlineExceeded,
	}
	got := err.Error()
	for _, want := range []string{"timed out", "GET", "example.cisco.com"} {
		if !strings.Contains(got, want) {
			t.Fatalf("message %q missing %q", got, want)
		}
	}
}

func TestTransportError_ConnectionMessage(t *testing.T) {
	err := &TransportError{
		Method: "GET",
		URL:    "https://example.cisco.com/api/ai-defense/v1/scans",
		Cause:  errors.New("connection refused"),
	}
	got := err.Error()
	for _, want := range []string{"request failed", "connection refused"} {
		if !strings.Contains(got, want) {
			t.Fatalf("message %q missing %q", got, want)
		}
	}
}

func TestTransportError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("wrapped: %w", context.DeadlineExceeded)
	err := &TransportError{Method: "GET", URL: "https://x.cisco.com", Cause: cause}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected unwrap to deadline exceeded")
	}
}
