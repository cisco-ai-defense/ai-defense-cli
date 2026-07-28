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

package modelscan

import (
	"encoding/json"
	"fmt"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/verdict"
)

// emitScanResult prints a scan outcome and returns the gating error that drives
// the process exit code. It is shared by `trigger` (which polls to a terminal
// status) and `info` so both behave identically for automation.
//
// raw carries the verbatim API payload for --raw mode; info is the parsed status
// used to compute the verdict.
func emitScanResult(info modelscansdk.ScanStatusInfo, raw json.RawMessage, opts *cmdutil.ScanResultOptions) error {
	failOn, blockOn, err := opts.Thresholds()
	if err != nil {
		return err
	}
	format, err := opts.OutputFormat()
	if err != nil {
		return err
	}

	// A scan that ended without producing a verdict is an execution failure
	// (exit 5), not a clean pass.
	switch info.Status {
	case modelscansdk.ScanStatusFailed, modelscansdk.ScanStatusCanceled:
		if opts.PrintRawAPI() {
			output.PrintRawJSON(raw)
		}
		return clierror.ScanFailed(
			fmt.Sprintf("model scan %s did not complete (status: %s)", info.ScanID, info.Status),
			"Re-run the scan; if the failure persists, verify the model file is valid and contact support.",
			nil)
	}

	result := verdict.FromModelScan(info)
	result.Decide(failOn, blockOn)

	if opts.PrintRawAPI() {
		output.PrintRawJSON(raw)
	} else {
		fmt.Println(result.Render(format))
	}
	return result.GateError()
}
