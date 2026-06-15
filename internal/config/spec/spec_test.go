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
	if len(s.Sections) != 1 || s.Sections[0].ID != "default" {
		t.Fatalf("expected single default root section")
	}
	groups := spec.GroupFields(s.Sections[0])
	if len(groups) < 2 {
		t.Fatalf("expected group fields under default")
	}
	req := spec.RequiredSection()
	var baseURLField spec.FieldSpec
	for _, f := range req.Fields {
		if f.Key == "base_url" {
			baseURLField = f
			break
		}
	}
	if baseURLField.Key != "base_url" || len(baseURLField.Options) != 4 || !baseURLField.AllowCustom {
		t.Fatalf("base_url field: %+v", baseURLField)
	}
}

func TestOptionalSectionsFromSpec(t *testing.T) {
	optional := spec.OptionalSections()
	if len(optional) != 2 {
		t.Fatalf("got %d optional sections", len(optional))
	}
	hf, ok := spec.ByID("huggingface")
	if !ok || hf.ID != "huggingface" || hf.Fields[0].Key != "token" {
		t.Fatalf("huggingface spec: %+v", hf)
	}
	aibom := spec.MustByID("aibom")
	if len(aibom.Presets) != 4 {
		t.Fatalf("expected 4 aibom presets, got %d", len(aibom.Presets))
	}
}

func TestBaseURLOptions(t *testing.T) {
	req := spec.RequiredSection()
	var baseURL spec.FieldSpec
	for _, f := range req.Fields {
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

func TestValidateBaseURLHost(t *testing.T) {
	t.Parallel()
	ok := []string{
		"https://us.api.aidefense.security.cisco.com",
		"https://sam.example.aiteam.cisco.com/",
	}
	for _, u := range ok {
		if err := spec.ValidateBaseURLHost(u); err != nil {
			t.Errorf("%q: %v", u, err)
		}
	}
	bad := []string{"", "not-a-url", "https://evil.example.com"}
	for _, u := range bad {
		if err := spec.ValidateBaseURLHost(u); err == nil {
			t.Errorf("%q: expected error", u)
		}
	}
	if err := spec.ValidateBaseURLHost("https://evil.example.com"); err == nil || err.Error() != "base URL host must be a *.cisco.com domain" {
		t.Fatalf("got %v", err)
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

func TestTomlPathInference(t *testing.T) {
	hf := spec.MustByID("huggingface")
	if len(hf.TomlPath) != 2 || hf.TomlPath[0] != "default" || hf.TomlPath[1] != "huggingface" {
		t.Fatalf("got %v", hf.TomlPath)
	}
	def := spec.RequiredSection()
	if len(def.TomlPath) != 1 || def.TomlPath[0] != "default" {
		t.Fatalf("got %v", def.TomlPath)
	}
}
