package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSmokeCommandPathsNonEmpty(t *testing.T) {
	paths := SmokeCommandPaths()
	if len(paths) < 10 {
		t.Fatalf("expected many command paths, got %d: %v", len(paths), paths)
	}
}

func TestSmokeHelpAllCommands(t *testing.T) {
	for _, path := range SmokeCommandPaths() {
		path := path
		t.Run(smokeTestName(path), func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)
			args := smokeHelpArgs(path)
			rootCmd.SetArgs(args)
			t.Cleanup(func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
				rootCmd.SetArgs(nil)
			})

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", strings.Join(args, " "), err)
			}
			got := out.String()
			if got == "" {
				t.Fatal("expected non-empty help output")
			}
			if strings.Contains(got, "Error:") {
				t.Fatalf("help output contains error: %q", got)
			}
		})
	}
}

func smokeTestName(path string) string {
	if path == "" {
		return "root"
	}
	return strings.ReplaceAll(path, " ", "_")
}

func smokeHelpArgs(path string) []string {
	if path == "" {
		return []string{"--help"}
	}
	return append(strings.Split(path, " "), "--help")
}
