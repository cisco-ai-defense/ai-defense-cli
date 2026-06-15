// Package diffcover computes statement coverage on Go lines changed vs a git base ref.
package diffcover

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Config controls diff coverage analysis.
type Config struct {
	ProfilePath string
	BaseRef     string
	MinPercent  float64
	SkipPath    func(rel string) bool
}

// Result holds diff coverage metrics.
type Result struct {
	CoveredStmts   int
	UncoveredStmts int
	ChangedFiles   int
}

// Percent returns covered / (covered + uncovered) * 100, or 100 when nothing to measure.
func (r Result) Percent() float64 {
	total := r.CoveredStmts + r.UncoveredStmts
	if total == 0 {
		return 100
	}
	return float64(r.CoveredStmts) / float64(total) * 100
}

type coverBlock struct {
	startLine int
	endLine   int
	stmts     int
	count     int
}

// Check compares coverage.out against changed production Go lines since merge-base(BaseRef).
func Check(cfg Config) (Result, error) {
	if err := validateConfig(cfg); err != nil {
		return Result{}, err
	}
	if cfg.SkipPath == nil {
		cfg.SkipPath = func(string) bool { return false }
	}

	mergeBase, err := gitOutput("merge-base", "HEAD", cfg.BaseRef)
	if err != nil {
		return Result{}, fmt.Errorf("merge-base HEAD %s: %w", cfg.BaseRef, err)
	}

	diff, err := gitOutput("diff", "-U0", mergeBase, "HEAD", "--", "*.go")
	if err != nil {
		return Result{}, fmt.Errorf("git diff: %w", err)
	}
	return checkDiff(cfg, diff)
}

func validateConfig(cfg Config) error {
	if cfg.ProfilePath == "" {
		return fmt.Errorf("profile path is required")
	}
	if cfg.BaseRef == "" {
		return fmt.Errorf("base ref is required")
	}
	if cfg.MinPercent < 0 {
		return fmt.Errorf("min percent must be >= 0")
	}
	return nil
}

func checkDiff(cfg Config, diff string) (Result, error) {
	changedLines, changedFiles, err := parseChangedLines(diff, cfg.SkipPath)
	if err != nil {
		return Result{}, err
	}

	profile, err := parseCoverProfile(cfg.ProfilePath)
	if err != nil {
		return Result{}, err
	}

	res := evaluateCoverage(changedLines, profile)
	res.ChangedFiles = changedFiles

	total := res.CoveredStmts + res.UncoveredStmts
	if total == 0 {
		return res, nil
	}
	if res.Percent()+1e-9 < cfg.MinPercent {
		return res, fmt.Errorf("diff coverage %.1f%% is below minimum %.1f%% (%d covered, %d uncovered statements on changed lines in %d files)",
			res.Percent(), cfg.MinPercent, res.CoveredStmts, res.UncoveredStmts, res.ChangedFiles)
	}
	return res, nil
}

func evaluateCoverage(changedLines map[string]map[int]struct{}, profile map[string][]coverBlock) Result {
	var res Result
	for file, lines := range changedLines {
		blocks := profile[file]
		if len(blocks) == 0 {
			continue
		}
		seen := make(map[int]struct{})
		for line := range lines {
			for i, b := range blocks {
				if line < b.startLine || line > b.endLine {
					continue
				}
				if _, ok := seen[i]; ok {
					continue
				}
				seen[i] = struct{}{}
				if b.count > 0 {
					res.CoveredStmts += b.stmts
				} else {
					res.UncoveredStmts += b.stmts
				}
			}
		}
	}
	return res
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseChangedLines(diff string, skipPath func(string) bool) (map[string]map[int]struct{}, int, error) {
	changed := make(map[string]map[int]struct{})
	var currentFile string
	var curNew int
	var filesWithLines int

	flushFile := func() {
		if currentFile == "" {
			return
		}
		if skipPath(currentFile) || strings.HasSuffix(currentFile, "_test.go") {
			delete(changed, currentFile)
			return
		}
		if len(changed[currentFile]) > 0 {
			filesWithLines++
		}
	}

	sc := bufio.NewScanner(strings.NewReader(diff))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flushFile()
			currentFile = ""
		case strings.HasPrefix(line, "+++ b/"):
			currentFile = strings.TrimPrefix(line, "+++ b/")
			if changed[currentFile] == nil {
				changed[currentFile] = make(map[int]struct{})
			}
		case strings.HasPrefix(line, "@@"):
			start, ok := parseHunkNewStart(line)
			if !ok {
				return nil, 0, fmt.Errorf("parse hunk header: %q", line)
			}
			curNew = start
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			if currentFile != "" {
				changed[currentFile][curNew] = struct{}{}
			}
			curNew++
		case strings.HasPrefix(line, " "):
			curNew++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// deletions only affect the old side
		}
	}
	if err := sc.Err(); err != nil {
		return nil, 0, err
	}
	flushFile()

	for f, lines := range changed {
		if len(lines) == 0 {
			delete(changed, f)
		}
	}
	return changed, filesWithLines, nil
}

func parseHunkNewStart(header string) (int, bool) {
	at := strings.Index(header, " +")
	if at < 0 {
		return 0, false
	}
	rest := header[at+2:]
	end := strings.Index(rest, " @@")
	if end < 0 {
		return 0, false
	}
	part := rest[:end]
	if comma := strings.Index(part, ","); comma >= 0 {
		part = part[:comma]
	}
	n, err := strconv.Atoi(strings.TrimSpace(part))
	return n, err == nil
}

func parseCoverProfile(path string) (map[string][]coverBlock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}

	profile := make(map[string][]coverBlock)
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		file, block, ok := parseCoverLine(line)
		if !ok {
			continue
		}
		file = repoRelativeCoverPath(file)
		profile[file] = append(profile[file], block)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return profile, nil
}

// repoRelativeCoverPath strips the module prefix from go cover profile paths so they
// match git diff paths (e.g. api/foo.go). Intentionally keyed on the repo directory
// name; update if github.com/cisco-ai-defense/ai-defense-cli is renamed in go.mod.
func repoRelativeCoverPath(profilePath string) string {
	const marker = "ai-defense-cli/"
	if i := strings.Index(profilePath, marker); i >= 0 {
		return profilePath[i+len(marker):]
	}
	return profilePath
}

func parseCoverLine(line string) (string, coverBlock, bool) {
	colon := strings.Index(line, ":")
	if colon < 0 {
		return "", coverBlock{}, false
	}
	file := line[:colon]
	rest := line[colon+1:]
	sp := strings.Fields(rest)
	if len(sp) < 3 {
		return "", coverBlock{}, false
	}
	rangePart := sp[0]
	stmts, err1 := strconv.Atoi(sp[1])
	count, err2 := strconv.Atoi(sp[2])
	if err1 != nil || err2 != nil {
		return "", coverBlock{}, false
	}
	dot := strings.Index(rangePart, ".")
	dash := strings.Index(rangePart, ",")
	if dot < 0 || dash < 0 {
		return "", coverBlock{}, false
	}
	startLine, err1 := strconv.Atoi(rangePart[:dot])
	endPart := rangePart[dash+1:]
	endDot := strings.Index(endPart, ".")
	if endDot < 0 {
		return "", coverBlock{}, false
	}
	endLine, err2 := strconv.Atoi(endPart[:endDot])
	if err1 != nil || err2 != nil {
		return "", coverBlock{}, false
	}
	return file, coverBlock{startLine: startLine, endLine: endLine, stmts: stmts, count: count}, true
}

func stmtCountForLine(blocks []coverBlock, line int) (covered, uncovered int, ok bool) {
	for _, b := range blocks {
		if line < b.startLine || line > b.endLine {
			continue
		}
		ok = true
		if b.count > 0 {
			covered += b.stmts
		} else {
			uncovered += b.stmts
		}
	}
	return covered, uncovered, ok
}

// DefaultSkipPath matches GO_COVER_PKGS exclusions plus non-production paths.
func DefaultSkipPath(rel string) bool {
	rel = strings.ReplaceAll(rel, "\\", "/")
	switch {
	case rel == "main.go":
		return true
	case strings.HasPrefix(rel, "scripts/"):
		return true
	case strings.HasPrefix(rel, "internal/testutil/"):
		return true
	case strings.HasPrefix(rel, "internal/tui/"):
		return true
	case strings.HasPrefix(rel, "internal/selfupdate/"):
		return true
	case strings.HasPrefix(rel, "internal/cli/config/"):
		return true
	case strings.HasPrefix(rel, "internal/cli/install/"):
		return true
	case strings.HasPrefix(rel, "internal/cli/onboard/"):
		return true
	case strings.HasPrefix(rel, "internal/config/spec/"):
		return true
	case strings.HasPrefix(rel, "cmd/"):
		return true
	default:
		return false
	}
}
