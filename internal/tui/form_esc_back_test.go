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

package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestRunFormEscBackCancelMapsToCLIError(t *testing.T) {
	t.Parallel()
	// Simulate the cancel branch without a TTY.
	err := NormalizeFormErr(huh.ErrUserAborted)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected *clierror.Error, got %T", err)
	}
	if cliErr.Code != clierror.CodeUnexpected {
		t.Fatalf("code = %d, want %d", cliErr.Code, clierror.CodeUnexpected)
	}
	if cliErr.Message != "Cancelled." {
		t.Fatalf("message = %q", cliErr.Message)
	}
}
