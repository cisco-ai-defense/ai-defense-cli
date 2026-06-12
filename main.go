package main

import (
	"os"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
