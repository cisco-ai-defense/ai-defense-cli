package aibom

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	embeddedpython "github.com/cisco-ai-defense/ai-defense-cli/internal/embedded/python"
)

// providerCredentialEnvVars lists the native API-key environment variables that
// the cisco-aibom analyzer (via langchain/litellm) reads directly. The CLI
// treats any of them as a valid credential source even when --llm-api-key is
// unset, so we don't reject setups that work today (e.g. exported OPENAI_API_KEY).
var providerCredentialEnvVars = []string{
	"OPENAI_API_KEY",
	"AZURE_OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GOOGLE_API_KEY",
	"GEMINI_API_KEY",
	"MISTRAL_API_KEY",
	"COHERE_API_KEY",
}

// llmOptions holds the LLM agent configuration cisco-aibom requires for
// analysis. The bundled analyzer's `analyze` command hard-requires --llm-model
// (cli.py: it raises "--llm-model ... is required" and exits when missing) and
// uses the model to classify every scanner candidate.
type llmOptions struct {
	model      string
	provider   string
	apiKey     string
	apiBase    string
	apiVersion string
}

// addLLMFlags registers the --llm-* flags, defaulting from cisco-aibom's own
// AIBOM_LLM_* environment variables so either source works.
func addLLMFlags(cmd *cobra.Command, o *llmOptions) {
	cmd.Flags().StringVar(&o.model, "llm-model", os.Getenv("AIBOM_LLM_MODEL"),
		"LLM model used to classify scanner candidates (required) (env: AIBOM_LLM_MODEL)")
	cmd.Flags().StringVar(&o.provider, "llm-provider", os.Getenv("AIBOM_LLM_PROVIDER"),
		"LLM provider (env: AIBOM_LLM_PROVIDER)")
	cmd.Flags().StringVar(&o.apiKey, "llm-api-key", os.Getenv("AIBOM_LLM_API_KEY"),
		"LLM API key (env: AIBOM_LLM_API_KEY)")
	cmd.Flags().StringVar(&o.apiBase, "llm-api-base", os.Getenv("AIBOM_LLM_API_BASE"),
		"LLM API base URL (env: AIBOM_LLM_API_BASE)")
	cmd.Flags().StringVar(&o.apiVersion, "llm-api-version", os.Getenv("AIBOM_LLM_API_VERSION"),
		"LLM API version (env: AIBOM_LLM_API_VERSION)")
}

// validate enforces what the bundled aibom analyzer requires before we spawn it,
// so the CLI fails fast with a clear message instead of deferring to a subprocess
// error. The analyzer's `analyze` command hard-requires --llm-model, and a model
// without a resolvable credential would otherwise run, warn "Agentic
// classification failed", and quietly emit a degraded report.
func (o llmOptions) validate() error {
	if o.model == "" {
		return clierror.Usage("--llm-model is required (or set AIBOM_LLM_MODEL): the analyzer uses an LLM to classify scanner candidates")
	}
	if !o.hasResolvableCredential() {
		return clierror.Usage("no LLM credential found: pass --llm-api-key (or set AIBOM_LLM_API_KEY), " +
			"use --llm-api-base for a local endpoint (e.g. Ollama), " +
			"set --llm-provider bedrock to use the AWS credential chain, " +
			"or export the provider's native key (e.g. OPENAI_API_KEY)")
	}
	return nil
}

// hasResolvableCredential reports whether the analyzer will be able to find an
// LLM credential. The key is only conditionally required: cloud providers
// (OpenAI/Azure/Anthropic/Google) need an API key, but AWS Bedrock uses the AWS
// credential chain and local backends (Ollama) authenticate via a base URL. We
// deliberately err toward allowing the run (best-effort fail-fast) so we never
// block a valid setup we can't fully introspect.
func (o llmOptions) hasResolvableCredential() bool {
	if o.apiKey != "" {
		return true
	}
	// Local / self-hosted endpoints (e.g. Ollama) authenticate via the base URL.
	if o.apiBase != "" {
		return true
	}
	// AWS Bedrock relies on the ambient AWS credential chain, not --llm-api-key.
	if isBedrock(o.provider) || isBedrock(o.model) {
		return true
	}
	// The analyzer reads provider-native env vars directly.
	for _, env := range providerCredentialEnvVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

func isBedrock(s string) bool {
	return strings.Contains(strings.ToLower(s), "bedrock")
}

// cliArgs returns the non-secret LLM flags. The API key is intentionally
// excluded here and passed via the child environment instead, so it never
// appears in the subprocess argv (visible via ps).
func (o llmOptions) cliArgs() []string {
	var a []string
	if o.model != "" {
		a = append(a, "--llm-model", o.model)
	}
	if o.provider != "" {
		a = append(a, "--llm-provider", o.provider)
	}
	if o.apiBase != "" {
		a = append(a, "--llm-api-base", o.apiBase)
	}
	if o.apiVersion != "" {
		a = append(a, "--llm-api-version", o.apiVersion)
	}
	return a
}

// generateReport runs `aibom analyze` via the bundled interpreter and returns
// the AIBOM JSON report. aibom's own stdout/stderr is forwarded to our stderr
// so the returned bytes contain only the report.
func generateReport(sources []string, customCatalog string, llm llmOptions) ([]byte, error) {
	py, err := embeddedpython.Interpreter()
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "aibom-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{"-m", "aibom", "analyze"}
	args = append(args, sources...)
	args = append(args, "--output-format", "json", "--output-file", tmpPath)
	if customCatalog != "" {
		args = append(args, "--custom-catalog", customCatalog)
	}
	args = append(args, llm.cliArgs()...)

	cmd := exec.Command(py, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if llm.apiKey != "" {
		cmd.Env = append(cmd.Env, "AIBOM_LLM_API_KEY="+llm.apiKey)
	}

	if err := cmd.Run(); err != nil {
		return nil, clierror.ScanFailed(
			"aibom analyze failed",
			"Check LLM credentials and model settings, then retry.",
			err,
		)
	}

	return os.ReadFile(tmpPath)
}
