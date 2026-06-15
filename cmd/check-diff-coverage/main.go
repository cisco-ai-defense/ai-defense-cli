package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/diffcover"
)

func main() {
	profile := flag.String("profile", "coverage.out", "go cover profile from make test-cover")
	base := flag.String("base", "origin/main", "git ref to compare against (merge-base with HEAD)")
	min := flag.Float64("min", 75, "minimum diff coverage percent on changed production lines")
	flag.Parse()

	res, err := diffcover.Check(diffcover.Config{
		ProfilePath: *profile,
		BaseRef:     *base,
		MinPercent:  *min,
		SkipPath:    diffcover.DefaultSkipPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "diff coverage check failed: %v\n", err)
		os.Exit(1)
	}

	total := res.CoveredStmts + res.UncoveredStmts
	if total == 0 {
		fmt.Printf("diff coverage: skipped (no coverable production Go changes vs %s)\n", *base)
		return
	}
	fmt.Printf("diff coverage: %.1f%% (%d covered, %d uncovered statements on changed lines in %d files; minimum %.1f%%)\n",
		res.Percent(), res.CoveredStmts, res.UncoveredStmts, res.ChangedFiles, *min)
}
