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
	"fmt"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/verdict"
)

// emitScanResult prints an MCP scan outcome and returns the gating error that
// drives the exit code, mirroring the model-scan contract so automation is
// identical across scan types.
//
// Note: the unregistered run/info(scan-id) flow's status endpoint returns only
// scan_id + status. Severity extraction is therefore best-effort and schema
// tolerant (verdict.ExtractMCPSeverities); when the payload carries no severity
// signal a COMPLETED scan reports PASS.
func emitScanResult(scanID, status string, raw json.RawMessage, opts *cmdutil.ScanResultOptions) error {
	failOn, blockOn, err := opts.Thresholds()
	if err != nil {
		return err
	}
	format, err := opts.OutputFormat()
	if err != nil {
		return err
	}

	switch mcpscansdk.MCPScanStatus(status) {
	case mcpscansdk.MCPScanStatusFailed, mcpscansdk.MCPScanStatusCancelled:
		if opts.PrintRawAPI() {
			output.PrintRawJSON(raw)
		}
		return clierror.ScanFailed(
			fmt.Sprintf("MCP scan %s did not complete (status: %s)", scanID, status),
			"Re-run the scan and verify the MCP server URL, transport, and auth are correct.",
			nil)
	}

	severities := verdict.ExtractMCPSeverities(raw)
	result := verdict.FromMCPScan(scanID, status, severities)
	result.Decide(failOn, blockOn)

	if opts.PrintRawAPI() {
		output.PrintRawJSON(raw)
	} else {
		fmt.Println(result.Render(format))
	}
	return result.GateError()
}
