package spec

import "testing"

func TestValidateWizardBaseURL(t *testing.T) {
	t.Parallel()
	if err := validateWizardBaseURL("Base URL", ""); err == nil {
		t.Fatal("expected required error for empty value")
	}
	if err := validateWizardBaseURL("Base URL", "https://us.api.aidefense.security.cisco.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateWizardBaseURL("Base URL", "https://evil.example.com"); err == nil {
		t.Fatal("expected host validation error")
	}
}
