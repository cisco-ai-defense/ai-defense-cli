package aibom

import (
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

// envBuilder computes aid_cli-injected env for a passthrough invocation. It is
// nil for pure passthroughs and supplied for commands that bridge credentials
// (analyze: LLM settings; report: upload target).
type envBuilder func(cli *cmdutil.CLI, args []string) ([]string, error)