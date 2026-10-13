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

// Package clierror defines the CLI's exit-code contract and a typed error that
// carries an exit code plus an actionable remediation hint. A single top-level
// handler (see cmd/main) classifies every error returned from a command into one
// of these categories so that CI/CD pipelines can branch on the process exit
// code reliably:
//
//	0      pass (handled by the command returning nil)
//	1-9    operational errors: the tool could NOT complete the scan
//	>=10   policy verdicts: the scan completed and a gate decided fail/block
//
// Keeping "the scan found problems" (>=10) distinct from "the tool failed"
// (1-9) is the core requirement: a pipeline must be able to fail a stage on
// findings without confusing that with an infrastructure/auth/network failure.
package clierror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

// Exit codes. These values are a public contract consumed by automation; do not
// renumber them without a corresponding documentation and major-version note.
const (
	// CodeOK is success / pass. Commands signal this by returning nil; it is
	// defined here for completeness and for tests.
	CodeOK = 0

	// CodeUnexpected is an uncategorized/internal error.
	CodeUnexpected = 1

	// CodeUsage is a usage error: bad flags or arguments, mutually exclusive
	// options, missing/invalid input, or a server-side HTTP 400.
	CodeUsage = 2

	// CodeAuth is an authentication/configuration error: missing or invalid
	// API key or base URL, a disallowed domain, or HTTP 401/403.
	CodeAuth = 3

	// CodeNetwork is a connectivity/timeout error: network failure, request
	// timeout, scan-poll timeout, or HTTP 429/5xx. This class is retryable.
	CodeNetwork = 4

	// CodeScanFailed is a scan-execution error: the scan could not produce a
	// verdict (server status FAILED/CANCELED, HTTP 404 for a bad id, or the
	// bundled aibom analyzer subprocess crashed).
	CodeScanFailed = 5

	// CodeFail is a policy verdict: findings at/above --fail-on but below
	// --block-on.
	CodeFail = 10

	// CodeBlock is a policy verdict: findings at/above --block-on (hard stop).
	CodeBlock = 20
)

// Error is a typed CLI error carrying an exit code and an optional remediation
// hint. The top-level handler prints Message (and Hint, RequestID when present)
// to stderr and exits with Code.
type Error struct {
	Code      int
	Message   string
	Hint      string
	RequestID string
	Cause     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap exposes the underlying cause for errors.Is/errors.As traversal.
func (e *Error) Unwrap() error { return e.Cause }

// New builds an *Error with an explicit code, message, and hint.
func New(code int, message, hint string, cause error) *Error {
	return &Error{Code: code, Message: message, Hint: hint, Cause: cause}
}

// Usage builds a usage error (exit 2).
func Usage(message string) *Error {
	return &Error{Code: CodeUsage, Message: message}
}

// Auth builds an auth/config error (exit 3).
func Auth(message, hint string, cause error) *Error {
	return &Error{Code: CodeAuth, Message: message, Hint: hint, Cause: cause}
}

// Network builds a connectivity/timeout error (exit 4).
func Network(message, hint string, cause error) *Error {
	return &Error{Code: CodeNetwork, Message: message, Hint: hint, Cause: cause}
}

// ScanFailed builds a scan-execution error (exit 5).
func ScanFailed(message, hint string, cause error) *Error {
	return &Error{Code: CodeScanFailed, Message: message, Hint: hint, Cause: cause}
}

// Unexpected builds an uncategorized error (exit 1).
func Unexpected(message string, cause error) *Error {
	return &Error{Code: CodeUnexpected, Message: message, Cause: cause}
}

// Gate builds a policy-verdict error (exit 10 or 20). It carries a one-line
// summary and intentionally no hint: the verdict envelope is already printed to
// stdout, so the only job here is to set the exit code and echo a short reason.
func Gate(code int, summary string) *Error {
	return &Error{Code: code, Message: summary}
}

// Hints used by Classify, kept as constants so tests and messages stay in sync.
const (
	hintAuth = "Provide credentials via --api-key, the AI_DEFENSE_API_KEY env var, or api_key in the config file, " +
		"and a valid --base-url (a *.cisco.com host)."
	hintNetwork = "Check network connectivity and the --base-url. If you are on a corporate network, ensure the VPN is " +
		"connected and the Cisco CA is trusted (e.g. set SSL_CERT_FILE to your CA bundle)."
	hintRateLimited     = "The service is rate-limiting or temporarily unavailable. Retry with backoff."
	hintNotFound        = "Verify the id is correct and that it belongs to your tenant."
	hintBadRequest      = "The request was rejected as invalid. Check the command arguments and try again."
	hintUploadTransient = "Uploading to the object store kept failing. Check network connectivity and any " +
		"proxy between you and AWS S3, then run the command again."
	hintUploadRejected = "The object store rejected the upload. This is not an AI Defense API key problem; " +
		"run the command again, and contact support if it keeps failing."
)

// Classify maps an arbitrary error returned by a command into a typed *Error
// with the appropriate exit code and a remediation hint. If err is already an
// *Error (e.g. a gate verdict or a hint-carrying failure raised by a command),
// it is returned unchanged.
func Classify(err error) *Error {
	if err == nil {
		return nil
	}

	// Already classified (gate verdicts, command-raised typed errors).
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}

	// Object-store upload failures wrap S3 ApiErrors/TransportErrors, so they
	// must be classified before the generic API checks below.
	var uploadErr *api.UploadError
	if errors.As(err, &uploadErr) {
		return classifyUploadError(err, uploadErr)
	}

	// Server-side API errors carry an HTTP status we can map precisely.
	var apiErr *api.ApiError
	if errors.As(err, &apiErr) {
		return classifyAPIError(apiErr)
	}

	// Client-side validation failures are usage errors.
	var valErr *api.ValidationError
	if errors.As(err, &valErr) {
		return &Error{Code: CodeUsage, Message: valErr.Error(), Cause: err}
	}

	// Scan poll budget exhausted.
	var pollErr *api.PollTimeoutError
	if errors.As(err, &pollErr) {
		if pollErr.StatusCommand == "" {
			return Network(
				pollErr.Error(),
				"Increase the polling budget via AI_DEFENSE_RETRY_COUNT_FOR_SCANNING and/or "+
					"AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK, then re-run.",
				err,
			)
		}
		return Network(pollErr.Error(), pollTimeoutHint(pollErr), err)
	}

	// HTTP client transport failures (timeout, DNS, TLS, connection refused).
	var transportErr *api.TransportError
	if errors.As(err, &transportErr) {
		return Network(transportErr.Error(), hintNetwork, err)
	}

	// Fall back to uncategorized.
	return &Error{Code: CodeUnexpected, Message: err.Error(), Cause: err}
}

// pollTimeoutHint never suggests re-running the scan command: that registers a
// new scan and, for file scans, uploads the file again.
func pollTimeoutHint(e *api.PollTimeoutError) string {
	var b strings.Builder
	b.WriteString("The scan is still running on the server and was not cancelled. ")
	fmt.Fprintf(&b, "Check its result with: %s", e.StatusCommand)
	b.WriteString(" (running the scan command again starts a new scan).")
	if len(e.BudgetEnvVars) > 0 {
		fmt.Fprintf(&b, " To wait longer next time, raise %s.", strings.Join(e.BudgetEnvVars, " and/or "))
	}
	return b.String()
}

// classifyUploadError keeps the wrapped message (e.g. "failed to upload file:
// multipart upload part 3 failed") and appends the sanitized object-store detail.
func classifyUploadError(err error, uploadErr *api.UploadError) *Error {
	msg := err.Error()
	if detail := uploadErr.Detail(); detail != "" {
		msg += " (" + detail + ")"
	}
	if uploadErr.Transient {
		return Network(msg, hintUploadTransient, err)
	}
	return &Error{Code: CodeUnexpected, Message: msg, Hint: hintUploadRejected, Cause: err}
}

func classifyAPIError(apiErr *api.ApiError) *Error {
	code := CodeUnexpected
	hint := ""
	switch {
	case apiErr.StatusCode == 400:
		code, hint = CodeUsage, hintBadRequest
	case apiErr.StatusCode == 401, apiErr.StatusCode == 403:
		code, hint = CodeAuth, hintAuth
	case apiErr.StatusCode == 404:
		code, hint = CodeScanFailed, hintNotFound
	case apiErr.StatusCode == 429:
		code, hint = CodeNetwork, hintRateLimited
	case apiErr.StatusCode >= 500:
		code, hint = CodeNetwork, hintRateLimited
	default:
		code = CodeUnexpected
	}
	return &Error{
		Code:      code,
		Message:   fmt.Sprintf("API error (status %d): %s", apiErr.StatusCode, apiErr.Message),
		Hint:      hint,
		RequestID: apiErr.RequestID,
		Cause:     apiErr,
	}
}
