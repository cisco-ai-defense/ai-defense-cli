package aibom

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

type llmFlagVars struct {
	byKey map[string]*string
}

func addLLMFlags(cmd *cobra.Command, v *llmFlagVars) {
	ig := spec.MustByID(aibomIntegrationID)
	v.byKey = make(map[string]*string, len(ig.Fields))
	for _, f := range ig.Fields {
		if f.Flag == "" {
			continue
		}
		defaultVal := ""
		if f.EnvVar != "" {
			defaultVal = os.Getenv(f.EnvVar)
		}
		desc := f.Title
		if f.EnvVar != "" {
			desc += " (env: " + f.EnvVar + ")"
		}
		val := defaultVal
		v.byKey[f.Key] = &val
		cmd.Flags().StringVar(v.byKey[f.Key], f.Flag, defaultVal, desc)
	}
}

func (v llmFlagVars) resolve(cli *cmdutil.CLI) LLMConfig {
	flags := make(map[string]string, len(v.byKey))
	for key, ptr := range v.byKey {
		flags[key] = *ptr
	}
	return mapToLLMConfig(aibomSpec().Resolve(flags, cli.ConfigManager()))
}
