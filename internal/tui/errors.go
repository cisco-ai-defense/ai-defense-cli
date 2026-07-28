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

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// ErrFormBack means the user pressed esc to return to the previous wizard step.
var ErrFormBack = errors.New("wizard step back")

// NormalizeFormErr maps interactive form errors to CLI contract errors.
// Ctrl+C is intentional user cancellation (exit 1), not a usage error (exit 2).
func NormalizeFormErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrFormBack) {
		return ErrFormBack
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return clierror.Unexpected("Cancelled.", err)
	}
	return err
}
