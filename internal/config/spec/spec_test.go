package spec_test

import (
	"strings"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

func TestEmbeddedSpecLoads(t *testing.T) {
	s, err := spec.LoadSchema(spec.EmbeddedSpecRaw())
	if err != nil {
		t.Fatal(err)
	}
	if s.Meta.SchemaVersion != "1" {
		t.Fatalf("schema version %q", s.Meta.SchemaVersion)
	}
	if len(s.Core.Fields) < 2 {
		t.Fatalf("expected core fields")
	}
	var baseURLField spec.FieldSpec
	for _, f := range spec.CoreFields() {
		if f.Key == "base_url" {
			baseURLField = f
			break
		}
	}
	if baseURLField.Key != "base_url" || len(baseURLField.Options) != 4 || !baseURLField.AllowCustom {
		t.Fatalf("base_url field: %+v", baseURLField)
	}
	if len(s.Integrations) < 2 {
		t.Fatalf("expected integrations")
	}
}

func TestIntegrationsFromSpec(t *testing.T) {
	integrations := spec.Integrations()
	if len(integrations) != 2 {
		t.Fatalf("got %d integrations", len(integrations))
	}
	hf, ok := spec.ByID("huggingface")
	if !ok || hf.ID != "huggingface" || hf.Section.Fields[0].Key != "token" {
		t.Fatalf("huggingface spec: %+v", hf)
	}
	aibom := spec.MustByID("aibom")
	if len(aibom.Presets) != 4 {
		t.Fatalf("expected 4 aibom presets, got %d", len(aibom.Presets))
	}
}

func TestBaseURLOptions(t *testing.T) {
	fields := spec.CoreFields()
	var baseURL spec.FieldSpec
	for _, f := range fields {
		if f.Key == "base_url" {
			baseURL = f
			break
		}
	}
	if len(baseURL.Options) != 4 {
		t.Fatalf("got %d options", len(baseURL.Options))
	}
	if !strings.HasSuffix(baseURL.Options[0].Value, ".cisco.com") {
		t.Fatalf("got %v", baseURL.Options[0].Value)
	}
}

func TestHuggingFaceResolveFromEnv(t *testing.T) {
	hf := spec.MustByID("huggingface")
	vals := hf.Resolve(map[string]string{"token": "hf_abc"}, nil)
	if vals["token"] != "hf_abc" {
		t.Fatalf("got %q", vals["token"])
	}
}

func TestAIBOMResolvePrecedence(t *testing.T) {
	t.Setenv("AIBOM_LLM_MODEL", "env-model")
	aibom := spec.MustByID("aibom")
	vals := aibom.Resolve(map[string]string{"llm_model": "flag-model"}, nil)
	if vals["llm_model"] != "flag-model" {
		t.Fatalf("flag should win, got %q", vals["llm_model"])
	}
	vals = aibom.Resolve(nil, nil)
	if vals["llm_model"] != "env-model" {
		t.Fatalf("env expected, got %q", vals["llm_model"])
	}
}

func TestAIBOMValidate(t *testing.T) {
	aibom := spec.MustByID("aibom")
	if err := aibom.Validate(map[string]string{}); err == nil {
		t.Fatal("expected error")
	}
	if err := aibom.Validate(map[string]string{
		"llm_model":   "gpt-5.4",
		"llm_api_key": "sk-x",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSecretTomlPaths(t *testing.T) {
	paths := spec.MustByID("aibom").SecretTomlPaths()
	if len(paths) != 1 || paths[0][2] != "llm_api_key" {
		t.Fatalf("got %v", paths)
	}
}

func TestApplyOmitsEmpty(t *testing.T) {
	root := map[string]interface{}{}
	spec.MustByID("aibom").Apply(root, map[string]string{
		"llm_model":    "gpt-5.4",
		"llm_provider": "",
	})
	def := root["default"].(map[string]interface{})
	aibomSec := def["aibom"].(map[string]interface{})
	if _, ok := aibomSec["llm_provider"]; ok {
		t.Fatal("empty provider should be omitted")
	}
}
