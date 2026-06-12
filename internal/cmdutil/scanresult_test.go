package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/verdict"
)

func TestAddScanResultFlagsDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	opts := AddScanResultFlags(cmd)
	if opts.FailOn != "low" || opts.BlockOn != "critical" || opts.Output != "json" || !opts.Raw || opts.Envelope {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
	if !opts.PrintRawAPI() {
		t.Fatal("default should print raw API JSON")
	}

	failOn, blockOn, err := opts.Thresholds()
	if err != nil {
		t.Fatalf("Thresholds() err: %v", err)
	}
	if failOn.Disabled || failOn.Sev != verdict.SevLow {
		t.Errorf("fail-on default = %+v", failOn)
	}
	if blockOn.Disabled || blockOn.Sev != verdict.SevCritical {
		t.Errorf("block-on default = %+v", blockOn)
	}
}

func TestThresholdsInvalid(t *testing.T) {
	opts := &ScanResultOptions{FailOn: "bogus", BlockOn: "critical"}
	_, _, err := opts.Thresholds()
	if err == nil {
		t.Fatal("expected error for invalid --fail-on")
	}
	if clierror.Classify(err).Code != clierror.CodeUsage {
		t.Errorf("invalid threshold should be a usage error")
	}

	opts = &ScanResultOptions{FailOn: "low", BlockOn: "nope"}
	if _, _, err := opts.Thresholds(); err == nil {
		t.Fatal("expected error for invalid --block-on")
	}
}

func TestPrintRawAPIDefaultsAndEnvelope(t *testing.T) {
	opts := &ScanResultOptions{Raw: true}
	if !opts.PrintRawAPI() {
		t.Fatal("expected raw API by default")
	}
	opts = &ScanResultOptions{Raw: true, Envelope: true}
	if opts.PrintRawAPI() {
		t.Fatal("--envelope should disable raw API stdout")
	}
	opts = &ScanResultOptions{Raw: false, Envelope: false}
	if opts.PrintRawAPI() {
		t.Fatal("--raw=false should select verdict envelope")
	}
}

func TestOutputFormat(t *testing.T) {
	for _, in := range []string{"", "json", "JSON", "text", "TEXT"} {
		opts := &ScanResultOptions{Output: in}
		if _, err := opts.OutputFormat(); err != nil {
			t.Errorf("OutputFormat(%q) unexpected err: %v", in, err)
		}
	}
	opts := &ScanResultOptions{Output: "yaml"}
	_, err := opts.OutputFormat()
	if err == nil {
		t.Fatal("expected error for unsupported output format")
	}
	if clierror.Classify(err).Code != clierror.CodeUsage {
		t.Errorf("bad output format should be a usage error")
	}
}
