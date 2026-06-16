// gen-exit-docs writes docs/exit-codes-handbook.md from internal/cliexitdoc.
//
// Usage:
//
//	go run ./cmd/gen-exit-docs [-o docs/exit-codes-handbook.md]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cliexitdoc"
)

func main() {
	out := flag.String("o", "docs/exit-codes-handbook.md", "output markdown path")
	flag.Parse()

	if err := cliexitdoc.ValidateRegistry(); err != nil {
		fmt.Fprintf(os.Stderr, "registry validation failed: %v\n", err)
		os.Exit(1)
	}

	content := cliexitdoc.NormalizeGenerated(cliexitdoc.GenerateHandbook())
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *out)
}
