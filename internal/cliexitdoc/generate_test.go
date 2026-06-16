package cliexitdoc

import "testing"

func TestValidateRegistry(t *testing.T) {
	if err := ValidateRegistry(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateHandbookNonEmpty(t *testing.T) {
	md := GenerateHandbook()
	if len(md) < 1000 {
		t.Fatalf("handbook too short: %d bytes", len(md))
	}
	if !contains(md, "internal/cliexitdoc/registry.go") {
		t.Fatal("expected generator attribution in handbook")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
