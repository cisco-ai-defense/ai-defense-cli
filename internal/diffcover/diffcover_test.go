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

package diffcover

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckDiff(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "coverage.out")
	diff := `diff --git a/api/foo.go b/api/foo.go
--- a/api/foo.go
+++ b/api/foo.go
@@ -0,0 +1,2 @@
+func added() {}
+func covered() {}
`
	if err := os.WriteFile(profile, []byte(`mode: atomic
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:1.1,2.2 2 1
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{ProfilePath: profile, BaseRef: "origin/main", MinPercent: 75, SkipPath: DefaultSkipPath}
	res, err := checkDiff(cfg, diff)
	if err != nil {
		t.Fatalf("checkDiff: %v", err)
	}
	if res.CoveredStmts != 2 || res.ChangedFiles != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestCheckDiff_BelowMinimum(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "coverage.out")
	diff := `diff --git a/api/foo.go b/api/foo.go
--- a/api/foo.go
+++ b/api/foo.go
@@ -0,0 +1,2 @@
+func added() {}
+func missed() {}
`
	if err := os.WriteFile(profile, []byte(`mode: atomic
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:1.1,1.10 1 1
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:2.1,2.10 1 0
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{ProfilePath: profile, BaseRef: "origin/main", MinPercent: 75, SkipPath: DefaultSkipPath}
	_, err := checkDiff(cfg, diff)
	if err == nil {
		t.Fatal("expected below-minimum error")
	}
}

func TestCheckDiff_NoCoverableChanges(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(profile, []byte("mode: atomic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{ProfilePath: profile, BaseRef: "origin/main", MinPercent: 75, SkipPath: DefaultSkipPath}
	res, err := checkDiff(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.CoveredStmts+res.UncoveredStmts != 0 {
		t.Fatalf("expected skip, got %+v", res)
	}
}

func TestValidateConfig(t *testing.T) {
	if err := validateConfig(Config{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultSkipPath(t *testing.T) {
	cases := map[string]bool{
		"main.go":                          true,
		"scripts/x.go":                     true,
		"internal/tui/x.go":                true,
		"cmd/check-diff-coverage/main.go": true,
		"cmd/other/main.go":               true,
		"api/client.go":                      false,
	}
	for path, want := range cases {
		if got := DefaultSkipPath(path); got != want {
			t.Fatalf("%s: got %v want %v", path, got, want)
		}
	}
}

func TestCheck_EndToEndGit(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README")
	runGit(t, root, "commit", "-m", "base")

	apiDir := filepath.Join(root, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goSrc := "package api\n\nfunc Added() {}\n"
	if err := os.WriteFile(filepath.Join(apiDir, "foo.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "api/foo.go")
	runGit(t, root, "commit", "-m", "add foo")

	profile := filepath.Join(root, "coverage.out")
	if err := os.WriteFile(profile, []byte(`mode: atomic
github.com/cisco-ai-defense/ai-defense-cli/api/foo.go:3.1,3.15 1 1
`), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	res, err := Check(Config{
		ProfilePath: profile,
		BaseRef:     "HEAD~1",
		MinPercent:  75,
		SkipPath:    DefaultSkipPath,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.CoveredStmts == 0 {
		t.Fatalf("expected covered statements, got %+v", res)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

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
