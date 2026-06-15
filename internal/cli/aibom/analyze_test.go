package aibom

import (
	"slices"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

func TestLLMOptionsValidate(t *testing.T) {
	t.Run("missing model", func(t *testing.T) {
		err := (llmOptions{}).validate()
		if err == nil {
			t.Fatal("expected error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})

	t.Run("missing credential", func(t *testing.T) {
		for _, env := range providerCredentialEnvVars {
			t.Setenv(env, "")
		}
		err := (llmOptions{model: "gpt-4"}).validate()
		if err == nil {
			t.Fatal("expected error")
		}
		if code := clierror.Classify(err).Code; code != clierror.CodeUsage {
			t.Fatalf("exit code = %d, want %d", code, clierror.CodeUsage)
		}
	})

	t.Run("valid with api key", func(t *testing.T) {
		err := (llmOptions{model: "gpt-4", apiKey: "sk-test"}).validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestHasResolvableCredential(t *testing.T) {
	for _, env := range providerCredentialEnvVars {
		t.Setenv(env, "")
	}

	tests := []struct {
		name string
		o    llmOptions
		env  map[string]string
		want bool
	}{
		{"api key flag", llmOptions{apiKey: "sk-test"}, nil, true},
		{"api base", llmOptions{apiBase: "http://localhost:11434"}, nil, true},
		{"bedrock provider", llmOptions{provider: "bedrock"}, nil, true},
		{"bedrock model", llmOptions{model: "anthropic.claude-bedrock-v2"}, nil, true},
		{"provider env", llmOptions{}, map[string]string{"OPENAI_API_KEY": "k"}, true},
		{"none", llmOptions{}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, env := range providerCredentialEnvVars {
				t.Setenv(env, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := tt.o.hasResolvableCredential(); got != tt.want {
				t.Fatalf("hasResolvableCredential() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBedrock(t *testing.T) {
	if !isBedrock("Bedrock") || !isBedrock("aws-bedrock-runtime") {
		t.Fatal("expected bedrock detection")
	}
	if isBedrock("openai") {
		t.Fatal("openai should not match bedrock")
	}
}

func TestLLMOptionsCliArgs(t *testing.T) {
	o := llmOptions{
		model:      "gpt-4",
		provider:   "openai",
		apiKey:     "secret-key",
		apiBase:    "https://api.example.com",
		apiVersion: "2024-01",
	}
	args := o.cliArgs()
	if slices.Contains(args, "secret-key") {
		t.Fatal("api key must not appear in subprocess argv")
	}
	for _, pair := range [][2]string{
		{"--llm-model", "gpt-4"},
		{"--llm-provider", "openai"},
		{"--llm-api-base", "https://api.example.com"},
		{"--llm-api-version", "2024-01"},
	} {
		if idx := slices.Index(args, pair[0]); idx < 0 || args[idx+1] != pair[1] {
			t.Fatalf("expected %s %s in args, got %v", pair[0], pair[1], args)
		}
	}
	if slices.Contains(args, "--llm-api-key") {
		t.Fatal("--llm-api-key must not be passed on subprocess argv")
	}
}
