package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// SmokeCommandPaths returns CLI command paths exercised by post-build smoke tests
// (each path is run with --help). Root is the empty string.
func SmokeCommandPaths() []string {
	paths := []string{""}
	var walk func(*cobra.Command, []string)
	walk = func(c *cobra.Command, prefix []string) {
		for _, child := range c.Commands() {
			if !smokeIncludeCommand(child) {
				continue
			}
			parts := append(append([]string{}, prefix...), child.Name())
			paths = append(paths, strings.Join(parts, " "))
			walk(child, parts)
		}
	}
	walk(rootCmd, nil)
	return paths
}

func smokeIncludeCommand(c *cobra.Command) bool {
	if c == nil || c.Name() == "" {
		return false
	}
	// Hidden deprecated alias (scan) remains user-facing; skip cobra help topics.
	if c.IsAdditionalHelpTopicCommand() {
		return false
	}
	return c.IsAvailableCommand() || c.Hidden
}
