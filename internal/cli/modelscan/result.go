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
