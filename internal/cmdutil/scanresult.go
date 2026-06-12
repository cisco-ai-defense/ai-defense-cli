package cmdutil

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/verdict"
)

// ScanResultOptions holds the gating and output flags shared by every command
// that produces a scan verdict. Using one helper across model-scan and mcp-scan
// keeps the automation UX identical between scan types.
type ScanResultOptions struct {
	FailOn  string
	BlockOn string
	Output  string
	Raw     bool
}

// AddScanResultFlags registers --fail-on/--block-on/--output/--raw on cmd and
// returns the bound options. Defaults are fail-closed: any detected threat fails
// (--fail-on=low) and a critical finding blocks (--block-on=critical).
func AddScanResultFlags(cmd *cobra.Command) *ScanResultOptions {
	o := &ScanResultOptions{}
	cmd.Flags().StringVar(&o.FailOn, "fail-on", "low",
		"Minimum severity that fails the scan with exit 10 (none|low|medium|high|critical)")
	cmd.Flags().StringVar(&o.BlockOn, "block-on", "critical",
		"Minimum severity that blocks with exit 20 (none|low|medium|high|critical)")
	cmd.Flags().StringVar(&o.Output, "output", "json",
		"Verdict envelope format: json|text")
	cmd.Flags().BoolVar(&o.Raw, "raw", false,
		"Print the raw API response instead of the normalized verdict envelope")
	return o
}

// Validate checks the gating and output flags up front so a misconfigured
// invocation fails fast with a usage error (exit 2) before any network work.
func (o *ScanResultOptions) Validate() error {
	if _, _, err := o.Thresholds(); err != nil {
		return err
	}
	if _, err := o.OutputFormat(); err != nil {
		return err
	}
	return nil
}

// Thresholds parses --fail-on/--block-on into verdict thresholds, returning a
// usage error (exit 2) for invalid values.
func (o *ScanResultOptions) Thresholds() (failOn, blockOn verdict.Threshold, err error) {
	failOn, err = verdict.ParseThreshold(o.FailOn)
	if err != nil {
		return failOn, blockOn, clierror.Usage("invalid --fail-on: " + err.Error())
	}
	blockOn, err = verdict.ParseThreshold(o.BlockOn)
	if err != nil {
		return failOn, blockOn, clierror.Usage("invalid --block-on: " + err.Error())
	}
	return failOn, blockOn, nil
}

// OutputFormat returns the validated output format ("json" or "text"), or a
// usage error for anything else.
func (o *ScanResultOptions) OutputFormat() (string, error) {
	switch strings.ToLower(strings.TrimSpace(o.Output)) {
	case "", "json":
		return "json", nil
	case "text":
		return "text", nil
	default:
		return "", clierror.Usage("invalid --output " + o.Output + " (want json|text)")
	}
}
