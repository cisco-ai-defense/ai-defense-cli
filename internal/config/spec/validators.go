package spec

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// providerCredentialEnvVars lists native API-key env vars the aibom analyzer reads.
var providerCredentialEnvVars = []string{
	"OPENAI_API_KEY",
	"AZURE_OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GOOGLE_API_KEY",
	"GEMINI_API_KEY",
	"MISTRAL_API_KEY",
	"COHERE_API_KEY",
}

// namedValidators maps config.spec.yaml validator names to Go implementations.
var namedValidators = map[string]func(map[string]string) error{
	"aibom_llm": validateAIBOMLLM,
}

func validateAIBOMLLM(values map[string]string) error {
	model := values["llm_model"]
	if strings.TrimSpace(model) == "" {
		return clierror.Usage("--llm-model is required (or set AIBOM_LLM_MODEL): the analyzer uses an LLM to classify scanner candidates")
	}
	if !hasAIBOMCredential(values) {
		e := clierror.Usage("no LLM credential found: pass --llm-api-key (or set AIBOM_LLM_API_KEY), " +
			"use --llm-api-base for a local endpoint (e.g. Ollama), " +
			"set --llm-provider bedrock to use the AWS credential chain, " +
			"or export the provider's native key (e.g. OPENAI_API_KEY)")
		e.Hint = "Run 'cisco-aid-cli config edit' and enable Local AIBOM analysis"
		return e
	}
	return nil
}

func hasAIBOMCredential(values map[string]string) bool {
	if values["llm_api_key"] != "" {
		return true
	}
	if values["llm_api_base"] != "" {
		return true
	}
	if isBedrock(values["llm_provider"]) || isBedrock(values["llm_model"]) {
		return true
	}
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

// ValidateBaseURLHost checks that baseURL is HTTPS-capable and on a *.cisco.com host.
func ValidateBaseURLHost(baseURL string) error {
	s := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid base URL")
	}
	host := u.Hostname()
	if host != "cisco.com" && !strings.HasSuffix(host, ".cisco.com") {
		return fmt.Errorf("base URL host must be a *.cisco.com domain")
	}
	return nil
}
