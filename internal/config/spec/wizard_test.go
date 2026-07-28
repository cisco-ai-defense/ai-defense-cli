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

func TestValidateWizardBaseURL(t *testing.T) {
	t.Parallel()
	if err := validateWizardBaseURL("Base URL", ""); err == nil {
		t.Fatal("expected required error for empty value")
	}
	if err := validateWizardBaseURL("Base URL", "https://us.api.aidefense.security.cisco.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateWizardBaseURL("Base URL", "https://evil.example.com"); err == nil {
		t.Fatal("expected host validation error")
	}
}
