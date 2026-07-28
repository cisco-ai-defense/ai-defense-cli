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

//go:build !embed_python

package python

import (
	"fmt"
	"os"
)

// Interpreter is the stub for builds without the embedded Python bundle. Set
// AID_CLI_PYTHON to a local interpreter with cisco-aibom installed for dev.
func Interpreter() (string, error) {
	if py := os.Getenv("AID_CLI_PYTHON"); py != "" {
		if _, err := os.Stat(py); err != nil {
			return "", fmt.Errorf("AID_CLI_PYTHON %q: %w", py, err)
		}
		return py, nil
	}

	return "", fmt.Errorf(
		"embedded Python is not available in this build.\n" +
			"Build with the embedded analyzer (make bundle && make build-embed, or " +
			"make build BUILD_MODE=full), or set AID_CLI_PYTHON to a python " +
			"interpreter with cisco-aibom installed for local development")
}
