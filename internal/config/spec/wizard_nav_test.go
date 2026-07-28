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

package spec

import "testing"

func TestSectionSpec_previousRequiredKey(t *testing.T) {
	sec := RequiredSection()
	keys := sec.RequiredFieldKeys()
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 required keys, got %v", keys)
	}

	prev, ok := sec.previousRequiredKey(keys[0])
	if ok {
		t.Fatalf("first key %q should have no previous", keys[0])
	}
	if prev != "" {
		t.Fatalf("expected empty previous, got %q", prev)
	}

	prev, ok = sec.previousRequiredKey(keys[1])
	if !ok {
		t.Fatalf("expected previous for %q", keys[1])
	}
	if prev != keys[0] {
		t.Fatalf("previous of %q = %q, want %q", keys[1], prev, keys[0])
	}
}
