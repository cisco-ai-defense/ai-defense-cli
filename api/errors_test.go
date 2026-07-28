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

func TestPollTimeoutError_String(t *testing.T) {
	e := &PollTimeoutError{ScanID: "scan-1", BudgetSecs: 60}
	got := e.Error()
	if !strings.Contains(got, "scan-1") || !strings.Contains(got, "60") {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestIsTimeout_NetError(t *testing.T) {
	if !isTimeout(&timeoutNetError{}) {
		t.Fatal("expected net timeout to be detected")
	}
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return false }
