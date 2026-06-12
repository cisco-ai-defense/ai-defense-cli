package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

// ApiError represents an error returned by the AI Defense API.
type ApiError struct {
	StatusCode int
	Message    string
	RequestID  string
}

func (e *ApiError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("API error (status %d, request-id %s): %s", e.StatusCode, e.RequestID, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// ValidationError represents a client-side validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// PollTimeoutError indicates a scan did not reach a terminal status within the
// polling budget. Commands map this to exit code 4 (connectivity/timeout).
type PollTimeoutError struct {
	ScanID     string
	BudgetSecs int
}

func (e *PollTimeoutError) Error() string {
	return fmt.Sprintf("scan %s did not complete within %ds", e.ScanID, e.BudgetSecs)
}

// TransportError wraps HTTP client failures (timeout, DNS, TLS, connection refused).
type TransportError struct {
	Method string
	URL    string
	Cause  error
}

func (e *TransportError) Error() string {
	host := e.URL
	if u, err := url.Parse(e.URL); err == nil && u.Host != "" {
		host = u.Host
	}
	if isTimeout(e.Cause) {
		return fmt.Sprintf("request timed out contacting AI Defense API (%s %s)", e.Method, host)
	}
	return fmt.Sprintf("request failed contacting AI Defense API (%s %s): %s", e.Method, host, e.Cause)
}

func (e *TransportError) Unwrap() error { return e.Cause }

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
