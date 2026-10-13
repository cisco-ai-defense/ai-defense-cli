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
// The scan keeps running server-side, so StatusCommand (how to check it) and
// BudgetEnvVars (how to wait longer next time) drive the remediation hint.
type PollTimeoutError struct {
	ScanID        string
	BudgetSecs    int
	StatusCommand string
	BudgetEnvVars []string
}

func (e *PollTimeoutError) Error() string {
	return fmt.Sprintf("scan %s did not complete within %ds", e.ScanID, e.BudgetSecs)
}

// UploadError is a failed presigned upload to the object store: one multipart
// part, or the whole file when Part is 0. Its cause (an ApiError or
// TransportError from S3) is not an AI Defense API failure, so commands must
// classify UploadError before looking for those types in the chain.
type UploadError struct {
	Part int
	// Transient reports that the last failure was a network error, throttling,
	// a server error, or an expired presigned URL.
	Transient bool
	Cause     error
}

// Error omits the cause (parity with the Python SDK); Detail describes it.
func (e *UploadError) Error() string {
	if e.Part > 0 {
		return fmt.Sprintf("multipart upload part %d failed", e.Part)
	}
	return "file upload failed"
}

func (e *UploadError) Unwrap() error { return e.Cause }

// Detail describes the object store's response without the presigned URL.
func (e *UploadError) Detail() string {
	var apiErr *ApiError
	if errors.As(e.Cause, &apiErr) {
		if apiErr.Message == "" {
			return fmt.Sprintf("object store returned status %d", apiErr.StatusCode)
		}
		return fmt.Sprintf("object store returned status %d: %s", apiErr.StatusCode, apiErr.Message)
	}
	var transportErr *TransportError
	if errors.As(e.Cause, &transportErr) {
		if isTimeout(transportErr.Cause) {
			return "request to the object store timed out"
		}
		return fmt.Sprintf("request to the object store failed: %v", transportErr.Cause)
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return ""
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
