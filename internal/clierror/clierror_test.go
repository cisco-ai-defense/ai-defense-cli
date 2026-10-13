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
	t.Run("model scan", func(t *testing.T) {
		got := Classify(&api.PollTimeoutError{
			ScanID:        "scan-1",
			BudgetSecs:    600,
			StatusCommand: "cisco-aid-cli model-scan info --scan-id scan-1",
			BudgetEnvVars: []string{"AI_DEFENSE_RETRY_COUNT_FOR_SCANNING", "AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK"},
		})
		if got.Code != CodeNetwork {
			t.Fatalf("want CodeNetwork, got %d", got.Code)
		}
		for _, s := range []string{
			"still running on the server and was not cancelled",
			"cisco-aid-cli model-scan info --scan-id scan-1",
			"AI_DEFENSE_RETRY_COUNT_FOR_SCANNING and/or AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK",
		} {
			if !strings.Contains(got.Hint, s) {
				t.Errorf("hint %q missing %q", got.Hint, s)
			}
		}
		if strings.Contains(strings.ToLower(got.Hint), "re-run") {
			t.Errorf("hint must not suggest re-running: %q", got.Hint)
		}
	})

	t.Run("without status command keeps the existing hint", func(t *testing.T) {
		got := Classify(&api.PollTimeoutError{ScanID: "scan-2", BudgetSecs: 300})
		if got.Code != CodeNetwork {
			t.Fatalf("want CodeNetwork, got %d", got.Code)
		}
		if !strings.Contains(got.Hint, "Increase the polling budget") {
			t.Errorf("hint = %q, want the existing polling-budget hint", got.Hint)
		}
	})
}

func TestClassifyUploadError(t *testing.T) {
	s3Denied := &api.ApiError{StatusCode: 403, Message: "AccessDenied: Access Denied"}
	tests := []struct {
		name      string
		err       error
		wantCode  int
		wantInMsg []string
		wantHint  string
	}{
		{
			name:      "rejected part keeps part number",
			err:       fmt.Errorf("failed to upload file: %w", &api.UploadError{Part: 3, Cause: s3Denied}),
			wantCode:  CodeUnexpected,
			wantInMsg: []string{"failed to upload file: multipart upload part 3 failed", "object store returned status 403: AccessDenied: Access Denied"},
			wantHint:  hintUploadRejected,
		},
		{
			name:      "exhausted transient failures",
			err:       fmt.Errorf("failed to upload file: %w", &api.UploadError{Part: 1, Transient: true, Cause: &api.ApiError{StatusCode: 503, Message: "SlowDown"}}),
			wantCode:  CodeNetwork,
			wantInMsg: []string{"multipart upload part 1 failed", "status 503: SlowDown"},
			wantHint:  hintUploadTransient,
		},
		{
			name: "transport timeout",
			err: &api.UploadError{Part: 2, Transient: true, Cause: &api.TransportError{
				Method: "PUT", URL: "https://bucket.s3.amazonaws.com/p?X-Amz-Signature=fake-signature-value", Cause: context.DeadlineExceeded,
			}},
			wantCode:  CodeNetwork,
			wantInMsg: []string{"multipart upload part 2 failed", "request to the object store timed out"},
			wantHint:  hintUploadTransient,
		},
		{
			name:      "single upload",
			err:       &api.UploadError{Cause: &api.ApiError{StatusCode: 400, Message: "EntityTooLarge"}},
			wantCode:  CodeUnexpected,
			wantInMsg: []string{"file upload failed", "status 400: EntityTooLarge"},
			wantHint:  hintUploadRejected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.err)
			if got.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", got.Code, tt.wantCode)
			}
			for _, s := range tt.wantInMsg {
				if !strings.Contains(got.Message, s) {
					t.Errorf("message %q missing %q", got.Message, s)
				}
			}
			if got.Hint != tt.wantHint {
				t.Errorf("hint = %q, want %q", got.Hint, tt.wantHint)
			}
			if strings.Contains(got.Message, "API error") || strings.Contains(got.Hint, "--api-key") {
				t.Errorf("object-store failure reported as an AI Defense API failure: %q / %q", got.Message, got.Hint)
			}
			if strings.Contains(got.Message, "fake-signature-value") {
				t.Errorf("message leaks the presigned signature: %q", got.Message)
			}
		})
	}

	t.Run("AI Defense API 403 is still an auth failure", func(t *testing.T) {
		if got := Classify(fmt.Errorf("failed to get multipart upload URLs: %w", s3Denied)); got.Code != CodeAuth {
			t.Fatalf("code = %d, want %d", got.Code, CodeAuth)
		}
	})
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
