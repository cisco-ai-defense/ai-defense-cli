package aibom

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
	embeddedpython "github.com/cisco-ai-defense/ai-defense-cli/internal/embedded/python"
)

// aibomSpec returns the embedded [default.aibom] section spec (LLM env/TOML keys).
func aibomSpec() spec.SectionSpec {
	return spec.MustByID(aibomIntegrationID)
}

// runAibom execs the embedded aibom CLI for the given aibom subcommand,
// forwarding args verbatim (including --help). stdio is wired to the real
// terminal so Rich renders identically and interactive prompts work. extraEnv
// holds aid_cli-injected credentials (only values the user didn't already set).
//
// Abbreviated help flags are normalized to --help first: Typer/Click does not
// abbreviate long options, so "--h" would otherwise error instead of showing
// help. This also lets the bootstrap's help-panel framing fire for them.
func runAibom(aibomCmd string, args, extraEnv []string) error {
	py, err := embeddedpython.Interpreter()
	if err != nil {
		return err
	}

	args = normalizeHelpFlags(args)
	cmdArgs := append([]string{"-c", aibomBootstrap, aibomCmd}, args...)
	c := exec.Command(py, cmdArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), envProgName+"="+progName)
	// The preamble/epilogue framing is off by default (see showHelpFraming): when
	// disabled we omit the env vars entirely and the bootstrap renders no panels,
	// matching the unframed parent `ai-bom --help`.
	if showHelpFraming {
		c.Env = append(c.Env,
			envHelpPreamble+"="+helpPreamble,
			envHelpEpilogue+"="+helpEpilogue,
		)
	}
	c.Env = append(c.Env, extraEnv...)
	return c.Run()
}

// runAibomPassthrough forwards a subcommand to the embedded analyzer and mirrors
// its exact exit code. The analyzer renders its own help/error UI, so aid_cli
// must not print a second wrapper. Non-exec errors (interpreter missing, spawn
// failure) are aid_cli's to report.
func runAibomPassthrough(aibomCmd string, args, extraEnv []string) error {
	err := runAibom(aibomCmd, args, extraEnv)
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	return err
}

// newPassthroughCmd builds a thin command that forwards everything (including
// -h/--help) to the embedded aibom subcommand. DisableFlagParsing keeps the
// analyzer's flag grammar authoritative; the annotation lets the smoke test skip
// it (its help is the analyzer's, available only in embedded builds).
func newPassthroughCmd(cli *cmdutil.CLI, use, short, aibomCmd string, buildEnv envBuilder) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		DisableFlagParsing: true,
		Annotations:        map[string]string{PassthroughAnnotation: "true"},
		RunE: func(_ *cobra.Command, args []string) error {
			var extraEnv []string
			if buildEnv != nil {
				var err error
				if extraEnv, err = buildEnv(cli, args); err != nil {
					return err
				}
			}
			return runAibomPassthrough(aibomCmd, args, extraEnv)
		},
	}
}

// aibomLLMEnv injects [default.aibom] LLM settings as AIBOM_LLM_* env vars so
// credentials stored in config.toml reach the analyzer. Values already present
// in the environment (and CLI flags, which pass through verbatim) win.
func aibomLLMEnv(cli *cmdutil.CLI, _ []string) ([]string, error) {
	return aibomEnvInjection(aibomSpec(), configSource(cli)), nil
}

// aibomEnvInjection resolves a config section (env → TOML) and returns KEY=VALUE
// entries for every field whose analyzer env var is currently unset but has a
// resolved (config-sourced) value. Env vars the user already exported are left
// untouched so explicit input always wins.
func aibomEnvInjection(sec spec.SectionSpec, src spec.ValueSource) []string {
	resolved := sec.Resolve(nil, src)
	var env []string
	for _, f := range sec.Fields {
		if f.EnvVar == "" || os.Getenv(f.EnvVar) != "" {
			continue
		}
		if v := resolved[f.Key]; v != "" {
			env = append(env, f.EnvVar+"="+v)
		}
	}
	return env
}

// configSource adapts the CLI's config manager to spec.ValueSource, tolerating a
// nil manager (e.g. before PersistentPreRun in some tests).
func configSource(cli *cmdutil.CLI) spec.ValueSource {
	if cli == nil || cli.ConfigManager == nil {
		return nil
	}
	if m := cli.ConfigManager(); m != nil {
		return m
	}
	return nil
}

// hasFlag reports whether args contains the long flag name in either the
// "--flag value" or "--flag=value" form.
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name || strings.HasPrefix(a, name+"=") {
			return true
		}
	}
	return false
}

// normalizeHelpFlags rewrites an abbreviated long help flag to the canonical
// "--help" the analyzer understands. Typer/Click does not abbreviate long
// options, so "ai-bom analyze --h" would otherwise error; we accept the strict
// prefixes of "--help" ("--h", "--he", "--hel") as aliases. Other tokens
// (including "--help" itself, "-h", and unrelated flags like "--host") pass
// through untouched. A copy is returned so the caller's slice is not mutated.
func normalizeHelpFlags(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		switch a {
		case "--h", "--he", "--hel":
			out[i] = "--help"
		default:
			out[i] = a
		}
	}
	return out
}

// firstNonFlag returns the first positional argument (first token that does not
// start with "-"), or "" if there is none.
func firstNonFlag(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}