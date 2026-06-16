package configcmd

import (
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestPreviousWizardStep(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from, want string
	}{
		{"aibom", "huggingface"},
		{"huggingface", "base_url"},
		{"base_url", "api_key"},
		{"api_key", ""},
		{"aibom.llm_model", "huggingface"},
	}
	for _, tc := range cases {
		if got := previousWizardStep(tc.from); got != tc.want {
			t.Errorf("previousWizardStep(%q) = %q, want %q", tc.from, got, tc.want)
		}
	}
}

func TestClearCapabilitiesFrom(t *testing.T) {
	t.Parallel()
	ctx := &capabilities.WizardContext{
		Selected: map[string]bool{"huggingface": true, "aibom": true},
		Skipped:  map[string]bool{"huggingface": true},
		IntegrationValues: map[string]map[string]string{
			"huggingface": {"token": "x"},
			"aibom":       {"llm_model": "gpt-4"},
		},
	}
	clearCapabilitiesFrom(ctx, "huggingface")
	if ctx.Selected["huggingface"] || ctx.Selected["aibom"] {
		t.Fatal("expected capability selections cleared from huggingface onward")
	}
	if ctx.Skipped["huggingface"] {
		t.Fatal("expected huggingface skip cleared")
	}
	if ctx.IntegrationValues["huggingface"] != nil || ctx.IntegrationValues["aibom"] != nil {
		t.Fatal("expected integration values cleared from huggingface onward")
	}
}

func TestWizardBackFromAibomResetsProgress(t *testing.T) {
	t.Parallel()
	ctx := &capabilities.WizardContext{
		APIKey:      "k",
		BaseURL:     "https://us.api.aidefense.security.cisco.com",
		OnboardStep: "aibom",
		Selected:    map[string]bool{"huggingface": false, "aibom": true},
		Skipped:     map[string]bool{"huggingface": true},
		ConfigPath:  t.TempDir() + "/config.toml",
	}
	var existing *appconfig.Manager
	saver := newDraftSaver(&existing)
	idx, err := wizardBackFrom(ctx, saver, "aibom")
	if err != nil {
		t.Fatalf("wizardBackFrom: %v", err)
	}
	if idx != capabilityIndex("huggingface") {
		t.Fatalf("restart index = %d, want huggingface index %d", idx, capabilityIndex("huggingface"))
	}
	if ctx.OnboardStep != "base_url" {
		t.Fatalf("OnboardStep = %q, want base_url", ctx.OnboardStep)
	}
	if stepDone(ctx.OnboardStep, "huggingface") {
		t.Fatal("huggingface should not be marked done after rewind")
	}
	if stepDone(ctx.OnboardStep, "aibom") {
		t.Fatal("aibom should not be marked done after rewind")
	}
}
