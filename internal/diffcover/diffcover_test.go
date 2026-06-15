package diffcover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChangedLines(t *testing.T) {
	diff := `diff --git a/api/foo.go b/api/foo.go
--- a/api/foo.go
+++ b/api/foo.go
@@ -1,1 +1,3 @@
 package api
+func added() {}
+func also() {}
diff --git a/internal/tui/x.go b/internal/tui/x.go
--- a/internal/tui/x.go
+++ b/internal/tui/x.go
@@ -0,0 +1,2 @@
+package tui
+func skip() {}
diff --git a/api/foo_test.go b/api/foo_test.go
--- a/api/foo_test.go
+++ b/api/foo_test.go
@@ -0,0 +1,2 @@
+package api
+func TestX(t *testing.T) {}
`

	lines, files, err := parseChangedLines(diff, DefaultSkipPath)
	if err != nil {
		t.Fatal(err)
	}
	if files != 1 {
		t.Fatalf("expected 1 file, got %d", files)
	}
	if len(lines["api/foo.go"]) != 2 {
		t.Fatalf("expected 2 changed lines in api/foo.go, got %d", len(lines["api/foo.go"]))
	}
	if _, ok := lines["internal/tui/x.go"]; ok {
		t.Fatal("expected tui file skipped")
	}
	if _, ok := lines["api/foo_test.go"]; ok {
		t.Fatal("expected test file skipped")
	}
}

func TestParseCoverProfileAndStmtCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.out")
	content := `mode: atomic
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:10.1,12.2 2 1
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:14.1,14.10 1 0
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	profile, err := parseCoverProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	blocks := profile["api/foo.go"]
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	c, u, ok := stmtCountForLine(blocks, 11)
	if !ok || c != 2 || u != 0 {
		t.Fatalf("line 11: covered=%d uncovered=%d ok=%v", c, u, ok)
	}
	c, u, ok = stmtCountForLine(blocks, 14)
	if !ok || c != 0 || u != 1 {
		t.Fatalf("line 14: covered=%d uncovered=%d ok=%v", c, u, ok)
	}
}

func TestResultPercent(t *testing.T) {
	if got := (Result{CoveredStmts: 3, UncoveredStmts: 1}).Percent(); got != 75 {
		t.Fatalf("got %v", got)
	}
	if got := (Result{}).Percent(); got != 100 {
		t.Fatalf("empty result should be 100%%, got %v", got)
	}
}

func TestRepoRelativeCoverPath(t *testing.T) {
	in := "github.com/cisco-ai-defense/ai-defense-cli/internal/cli/root.go"
	want := "internal/cli/root.go"
	if got := repoRelativeCoverPath(in); got != want {
		t.Fatalf("got %q", got)
	}
}
