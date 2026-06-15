package aibom

import (
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

const aibomIntegrationID = "aibom"

// LLMConfig holds LLM settings for local AIBOM analysis (view over spec field map).
type LLMConfig struct {
	Model      string
	Provider   string
	APIKey     string
	APIBase    string
	APIVersion string
}

// ResolveLLM applies flags → env → TOML using the aibom integration spec.
func ResolveLLM(flags LLMConfig, mgr *config.Manager) LLMConfig {
	return mapToLLMConfig(aibomSpec().Resolve(llmConfigToMap(flags), mgr))
}

// ValidateLLM enforces analyzer requirements before spawning subprocess.
func ValidateLLM(o LLMConfig) error {
	return aibomSpec().Validate(llmConfigToMap(o))
}

func aibomSpec() spec.IntegrationSpec {
	return spec.MustByID(aibomIntegrationID)
}

func llmConfigToMap(o LLMConfig) map[string]string {
	return map[string]string{
		"llm_model":       o.Model,
		"llm_provider":    o.Provider,
		"llm_api_key":     o.APIKey,
		"llm_api_base":    o.APIBase,
		"llm_api_version": o.APIVersion,
	}
}

func mapToLLMConfig(vals map[string]string) LLMConfig {
	return LLMConfig{
		Model:      vals["llm_model"],
		Provider:   vals["llm_provider"],
		APIKey:     vals["llm_api_key"],
		APIBase:    vals["llm_api_base"],
		APIVersion: vals["llm_api_version"],
	}
}

func (o LLMConfig) cliArgs() []string {
	var a []string
	if o.Model != "" {
		a = append(a, "--llm-model", o.Model)
	}
	if o.Provider != "" {
		a = append(a, "--llm-provider", o.Provider)
	}
	if o.APIBase != "" {
		a = append(a, "--llm-api-base", o.APIBase)
	}
	if o.APIVersion != "" {
		a = append(a, "--llm-api-version", o.APIVersion)
	}
	return a
}
