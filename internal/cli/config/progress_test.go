package configcmd

import "testing"

func TestStepDone(t *testing.T) {
	t.Parallel()
	cases := []struct {
		completed, target string
		want              bool
	}{
		{"", "api_key", false},
		{"base_url", "api_key", true},
		{"base_url", "base_url", true},
		{"api_key", "base_url", false},
		{"base_url", "huggingface", false},
		{"huggingface", "huggingface", true},
		{"aibom.llm_model", "aibom", false},
		{"huggingface", "aibom", false},
		{stepComplete, "aibom", true},
		{legacyStepCore, "base_url", true},
		{legacyStepLoc, "api_key", false},
		{"base_url", legacyStepLoc, true},
	}
	for _, tc := range cases {
		if got := stepDone(tc.completed, tc.target); got != tc.want {
			t.Errorf("stepDone(%q, %q) = %v, want %v", tc.completed, tc.target, got, tc.want)
		}
	}
}
