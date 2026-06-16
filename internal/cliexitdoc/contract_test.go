package cliexitdoc

import "testing"

func TestSingleExitCode(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in   string
		ok   bool
		want int
	}{
		"zero":    {"0", true, 0},
		"usage":   {"2", true, 2},
		"composite": {"10 / 20", false, 0},
		"ellipsis":  {"3 / 4 / 5 / …", false, 0},
		"see onboard": {"*see onboard*", false, 0},
		"empty":   {"", false, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := SingleExitCode(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("SingleExitCode(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestConfigValidateContractMatchesRegistry(t *testing.T) {
	t.Parallel()
	rows := ConfigValidateContract()
	if len(rows) < 3 {
		t.Fatalf("expected at least 3 single-code validate rows, got %d", len(rows))
	}
	if code, ok := ContractCode(rows, "Valid"); !ok || code != 0 {
		t.Fatalf("Valid row: (%d, %v)", code, ok)
	}
	if code, ok := ContractCode(rows, "read/parse"); !ok || code != 2 {
		t.Fatalf("parse row: (%d, %v)", code, ok)
	}
	if code, ok := ContractCode(rows, "Missing API key"); !ok || code != 3 {
		t.Fatalf("auth row: (%d, %v)", code, ok)
	}
}

func TestModelScanContractIncludesBareParent(t *testing.T) {
	t.Parallel()
	rows := ModelScanContract()
	code, ok := ContractCode(rows, "No subcommand")
	if !ok || code != 0 {
		t.Fatalf("bare model-scan row: (%d, %v)", code, ok)
	}
}

func TestMcpScanContractNonEmpty(t *testing.T) {
	t.Parallel()
	if len(McpScanContract()) < 5 {
		t.Fatalf("expected several mcp-scan contract rows, got %d", len(McpScanContract()))
	}
}

func TestAibomContractNonEmpty(t *testing.T) {
	t.Parallel()
	if len(AibomContract()) < 5 {
		t.Fatalf("expected several ai-bom contract rows, got %d", len(AibomContract()))
	}
}
