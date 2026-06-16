package cli

import (
	"os"
	"testing"

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

type errTest string

func (e errTest) Error() string { return string(e) }
