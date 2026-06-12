package api

import "fmt"

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
