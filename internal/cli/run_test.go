package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// runWithArgs invokes Run with the given args while suppressing the handler's
// stderr output, restoring global command state afterward.
func runWithArgs(t *testing.T, args ...string) int {
	t.Helper()
	origErr := os.Stderr
	devnull, _ := os.Open(os.DevNull)
	os.Stderr = devnull
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		os.Stderr = origErr
		_ = devnull.Close()
		rootCmd.SetArgs(nil)
	})
	return Run()
}

func TestRunVersionIsOK(t *testing.T) {
	if code := runWithArgs(t, "--version"); code != clierror.CodeOK {
		t.Fatalf("--version exit code = %d, want %d", code, clierror.CodeOK)
	}
}

func TestRunUnknownCommandIsUsageError(t *testing.T) {
	if code := runWithArgs(t, "definitely-not-a-command"); code != clierror.CodeUsage {
		t.Fatalf("unknown command exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRunUnknownFlagIsUsageError(t *testing.T) {
	if code := runWithArgs(t, "model-scan", "--nope"); code != clierror.CodeUsage {
		t.Fatalf("unknown flag exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRunModelScanUnknownSubcommandIsUsageError(t *testing.T) {
	if code := runWithArgs(t, "model-scan", "definitely-not-a-subcommand"); code != clierror.CodeUsage {
		t.Fatalf("unknown model-scan subcommand exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

func TestRunModelScanRunDeprecatedIsUsageError(t *testing.T) {
	if code := runWithArgs(t, "model-scan", "run"); code != clierror.CodeUsage {
		t.Fatalf("deprecated model-scan run exit code = %d, want %d", code, clierror.CodeUsage)
	}
}

// runCaptureStderr invokes Run with args, returning (exit code, stderr text).
// Unlike runWithArgs it captures stderr through an os.Pipe (a real *os.File) so
// the help color gate behaves as it would on a terminal under FORCE_COLOR.
func runCaptureStderr(t *testing.T, args ...string) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origErr := os.Stderr
	os.Stderr = w
	rootCmd.SetArgs(args)
	code := Run()
	_ = w.Close()
	os.Stderr = origErr
	t.Cleanup(func() { rootCmd.SetArgs(nil); _ = r.Close() })

	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	return code, buf.String()
}

func TestRunUsageErrorEmitsColoredUsage(t *testing.T) {
	// Regression: the unknown-subcommand usage block must honor the stderr color
	// gate. It previously rendered via cobra's buffer-backed UsageString(), which
	// is not a TTY, so ANSI was stripped even on a terminal.
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")
	code, errOut := runCaptureStderr(t, "ai-bom", "definitely-not-a-subcommand")
	if code != clierror.CodeUsage {
		t.Fatalf("exit = %d, want %d", code, clierror.CodeUsage)
	}
	if !strings.Contains(errOut, "\x1b[") {
		t.Fatalf("expected ANSI color in usage output, got plain:\n%s", errOut)
	}
}

func TestRunUsageErrorShowsErrorPanelAndSuggestion(t *testing.T) {
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("NO_COLOR", "1")
	// A near-miss subcommand should produce the aibom-style Error panel with a
	// "Did you mean" suggestion, followed by the full command list.
	code, errOut := runCaptureStderr(t, "ai-bom", "analyz")
	if code != clierror.CodeUsage {
		t.Fatalf("exit = %d, want %d", code, clierror.CodeUsage)
	}
	for _, want := range []string{
		"Error",
		"No such command 'analyz'.",
		"Did you mean 'analyze'?",
		"Commands", // full help still follows the error panel
	} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("usage output missing %q:\n%s", want, errOut)
		}
	}
}

func TestRunUsageErrorNoColorStaysPlain(t *testing.T) {
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("NO_COLOR", "1")
	_, errOut := runCaptureStderr(t, "ai-bom", "definitely-not-a-subcommand")
	if strings.Contains(errOut, "\x1b[") {
		t.Fatalf("NO_COLOR set but usage output contains ANSI:\n%s", errOut)
	}
}

func TestIsUsageError(t *testing.T) {
	cases := map[string]bool{
		"unknown command \"x\"":          true,
		"unknown flag: --x":              true,
		"required flag(s) \"y\" not set": true,
		"accepts 1 arg(s), received 0":   true,
		"some random runtime failure":    false,
		"API error (status 500): boom":   false,
	}
	for msg, want := range cases {
		if got := isUsageError(errTest(msg)); got != want {
			t.Errorf("isUsageError(%q) = %v, want %v", msg, got, want)
		}
	}
}

func TestCleanUsageMessage(t *testing.T) {
	// Unknown command with a cobra-embedded suggestion (root-command style).
	rootStyle := "unknown command \"lst\" for \"cisco-aid-cli\"\n\nDid you mean this?\n\tlist\n"
	if got := cleanUsageMessage(rootStyle, nil); got != "No such command 'lst'. Did you mean 'list'?" {
		t.Fatalf("root-style: got %q", got)
	}

	// Unknown command without an embedded suggestion and no command to query:
	// just the concise "No such command" line.
	plain := "unknown command \"zzzz\" for \"cisco-aid-cli ai-bom\""
	if got := cleanUsageMessage(plain, nil); got != "No such command 'zzzz'." {
		t.Fatalf("plain: got %q", got)
	}

	// Non-unknown-command usage errors collapse to a single line, unchanged.
	flag := "unknown flag: --nope"
	if got := cleanUsageMessage(flag, nil); got != flag {
		t.Fatalf("flag: got %q", got)
	}
	multiline := "required flag(s)\n\t\"url\" not set"
	if got := cleanUsageMessage(multiline, nil); got != "required flag(s) \"url\" not set" {
		t.Fatalf("multiline: got %q", got)
	}
}

func TestCleanUsageMessage_SuggestsFromCommand(t *testing.T) {
	// When cobra did not embed a suggestion (subcommand path), the suggestion is
	// computed from the resolved command's subcommands.
	parent := &cobra.Command{Use: "ai-bom"}
	parent.AddCommand(&cobra.Command{Use: "analyze", RunE: func(*cobra.Command, []string) error { return nil }})
	parent.AddCommand(&cobra.Command{Use: "report", RunE: func(*cobra.Command, []string) error { return nil }})
	setSuggestionDistance(parent, 2)

	msg := "unknown command \"analyz\" for \"ai-bom\""
	if got := cleanUsageMessage(msg, parent); got != "No such command 'analyz'. Did you mean 'analyze'?" {
		t.Fatalf("got %q", got)
	}

	// Pure garbage yields no suggestion.
	if got := cleanUsageMessage("unknown command \"qqqqqq\" for \"ai-bom\"", parent); got != "No such command 'qqqqqq'." {
		t.Fatalf("garbage: got %q", got)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
