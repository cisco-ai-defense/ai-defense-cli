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

package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON_Map(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(map[string]string{"key": "value"})
	})

	if !strings.Contains(got, `"key": "value"`) {
		t.Errorf("expected JSON with key/value, got %q", got)
	}
}

func TestPrintJSON_Nil(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(nil)
	})
	if strings.TrimSpace(got) != "null" {
		t.Errorf("expected 'null', got %q", got)
	}
}

func TestPrintJSON_Indented(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(map[string]interface{}{"a": 1, "b": "two"})
	})
	if !strings.Contains(got, "  ") {
		t.Error("expected indented output")
	}
}

func TestPrintRawJSON_ValidJSON(t *testing.T) {
	raw := json.RawMessage(`{"status":"ok"}`)
	got := captureStdout(t, func() {
		PrintRawJSON(raw)
	})
	if !strings.Contains(got, `"status": "ok"`) {
		t.Errorf("expected re-indented JSON, got %q", got)
	}
}

func TestPrintRawJSON_Nil(t *testing.T) {
	got := captureStdout(t, func() {
		PrintRawJSON(nil)
	})
	if got != "" {
		t.Errorf("expected empty output for nil, got %q", got)
	}
}

func TestPrintRawJSON_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not json`)
	got := captureStdout(t, func() {
		PrintRawJSON(raw)
	})
	if !strings.Contains(got, "not json") {
		t.Errorf("expected raw passthrough for invalid JSON, got %q", got)
	}
}
