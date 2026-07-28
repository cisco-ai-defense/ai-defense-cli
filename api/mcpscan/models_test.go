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

package mcpscan

import (
	"encoding/json"
	"testing"
)

func TestStartMCPServerScanResponse_UnmarshalJSON(t *testing.T) {
	t.Run("snake_case", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{"scan_id":"s1"}`), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.ScanID != "s1" {
			t.Fatalf("got %q", resp.ScanID)
		}
	})

	t.Run("camelCase", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{"scanId":"s2"}`), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.ScanID != "s2" {
			t.Fatalf("got %q", resp.ScanID)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		var resp StartMCPServerScanResponse
		if err := json.Unmarshal([]byte(`{`), &resp); err == nil {
			t.Fatal("expected error")
		}
	})
}
