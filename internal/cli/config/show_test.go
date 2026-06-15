package configcmd

import (
	"strings"
	"testing"

	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestFormatConfigOverview(t *testing.T) {
	t.Parallel()
	m := appconfig.NewManagerFromPath("")
	mData := map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  "secret",
			"base_url": "https://us.api.aidefense.security.cisco.com",
		},
		"onboard": map[string]interface{}{
			"step": "complete",
		},
	}
	// inject via path write
	dir := t.TempDir()
	path := dir + "/config.toml"
	if err := appconfig.WriteConfig(path, mData); err != nil {
		t.Fatal(err)
	}
	m = appconfig.NewManagerFromPath(path)

	text := formatConfigOverview(m, appconfig.RedactSecrets(m.Data()))
	if !strings.Contains(text, "API key: ***") {
		t.Fatalf("expected redacted api key, got:\n%s", text)
	}
	if !strings.Contains(text, "us.api.aidefense") {
		t.Fatalf("expected base url in overview, got:\n%s", text)
	}
	if !strings.Contains(text, "Onboarding step: complete") {
		t.Fatalf("expected onboard step, got:\n%s", text)
	}
}
