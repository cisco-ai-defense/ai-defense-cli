package aibom

import (
	"path/filepath"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestValidateLLM(t *testing.T) {
	if err := ValidateLLM(LLMConfig{}); err == nil {
		t.Fatal("expected error for missing model")
	}
	if err := ValidateLLM(LLMConfig{Model: "gpt-5.4", APIKey: "sk-test"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLLM(LLMConfig{Model: "gpt-5.4", APIBase: "http://localhost:11434"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLLM(LLMConfig{Model: "bedrock/anthropic.claude", Provider: "bedrock"}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveLLMFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.toml")
	if err := config.WriteConfig(path, map[string]interface{}{
		"default": map[string]interface{}{
			"aibom": map[string]interface{}{
				"llm_model":   "gpt-5.4",
				"llm_api_key": "from-toml",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := config.NewManagerFromPath(path)
	got := ResolveLLM(LLMConfig{}, m)
	if got.Model != "gpt-5.4" || got.APIKey != "from-toml" {
		t.Fatalf("got %+v", got)
	}
}
