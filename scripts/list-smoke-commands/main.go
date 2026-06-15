// Command list-smoke-commands prints CLI paths for scripts/smoke-test-cli.sh.
package main

import (
	"fmt"
	"os"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli"
)

func main() {
	for _, path := range cli.SmokeCommandPaths() {
		if _, err := fmt.Fprintln(os.Stdout, path); err != nil {
			os.Exit(1)
		}
	}
}
