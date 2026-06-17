package aibom

import (
	"strings"
	"testing"
)

func TestKBNeedsCredentials(t *testing.T) {
	cases := map[string]bool{
		"request --sdk x --version 1": true,
		"request-status abc":          true,
		"list-requests":               true,
		"download":                    false,
		"check":                       false,
		"info":                        false,
		"verify":                      false,
		"--help":                      false,
	}
	for in, want := range cases {
		if got := kbNeedsCredentials(strings.Fields(in)); got != want {
			t.Errorf("kbNeedsCredentials(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestKBCredentialEnv_NotBuildCommand(t *testing.T) {
	env, err := kbCredentialEnv(testCLI(t), []string{"download"})
	if err != nil || env != nil {
		t.Fatalf("local kb command should be a no-op, got env=%v err=%v", env, err)
	}
}

func TestKBCredentialEnv_UserFlagWins(t *testing.T) {
	t.Setenv("CISCO_AI_DEFENSE_API_KEY", "")
	env, err := kbCredentialEnv(testCLI(t), []string{"request", "--api-key", "k", "--sdk", "x", "--version", "1"})
	if err != nil || env != nil {
		t.Fatalf("user --api-key should suppress injection, got env=%v err=%v", env, err)
	}
}

func TestKBCredentialEnv_UserEnvWins(t *testing.T) {
	t.Setenv("CISCO_AI_DEFENSE_API_KEY", "from-env")
	env, err := kbCredentialEnv(testCLI(t), []string{"request", "--sdk", "x", "--version", "1"})
	if err != nil || env != nil {
		t.Fatalf("preset CISCO_AI_DEFENSE_API_KEY should suppress injection, got env=%v err=%v", env, err)
	}
}

func TestKBCredentialEnv_DerivesFromConfig(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("CISCO_AI_DEFENSE_API_KEY", "")
	cli := cliWithConfig(t, map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  "tenant-key",
			"base_url": "https://us.api.aidefense.security.cisco.com",
		},
	})
	env, err := kbCredentialEnv(cli, []string{"request", "--sdk", "x", "--version", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 1 || env[0] != "CISCO_AI_DEFENSE_API_KEY=tenant-key" {
		t.Fatalf("expected only the tenant key injected, got %v", env)
	}
}

func TestKBCredentialEnv_AuthError(t *testing.T) {
	t.Setenv("AI_DEFENSE_API_KEY", "")
	t.Setenv("CISCO_AI_DEFENSE_API_KEY", "")
	_, err := kbCredentialEnv(testCLI(t), []string{"request", "--sdk", "x", "--version", "1"})
	if err == nil {
		t.Fatal("expected auth error when no tenant key is configured")
	}
}
