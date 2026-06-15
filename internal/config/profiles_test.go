package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	t.Parallel()
	valid := []string{"default", "work", "staging-1", "a"}
	for _, name := range valid {
		if err := ValidateProfileName(name); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	invalid := []string{"", "-bad", "has space", stringsRepeat("a", 65)}
	for _, name := range invalid {
		if err := ValidateProfileName(name); err == nil {
			t.Errorf("%q: expected error", name)
		}
	}
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestProfileConfigPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user")
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "")

	if got := ProfileConfigPath(DefaultProfileName); got != DefaultConfigPath() {
		t.Fatalf("default: got %q", got)
	}
	wantWork := filepath.Join(home, homeConfigDirName, profilesSubdir, "work.toml")
	if got := ProfileConfigPath("work"); got != wantWork {
		t.Fatalf("work: got %q want %q", got, wantWork)
	}
}

func TestActiveProfilePriority(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user")
	if err := os.MkdirAll(filepath.Join(home, homeConfigDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_PROFILE", "")
	SetProfileOverride("")

	if err := WriteActiveProfile("work"); err != nil {
		t.Fatal(err)
	}
	if got := ReadStoredActiveProfile(); got != "work" {
		t.Fatalf("stored: got %q", got)
	}
	if got := ActiveProfile(); got != "work" {
		t.Fatalf("active: got %q", got)
	}

	t.Setenv("AI_DEFENSE_PROFILE", "staging")
	if got := ActiveProfile(); got != "staging" {
		t.Fatalf("env: got %q", got)
	}

	SetProfileOverride("cli")
	if got := ActiveProfile(); got != "cli" {
		t.Fatalf("flag: got %q", got)
	}
}

func TestResolveConfigPathWithProfile(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user")
	if err := os.MkdirAll(filepath.Join(home, homeConfigDirName, profilesSubdir), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AI_DEFENSE_CONFIG_PATH", "")
	SetProfileOverride("")

	workPath := ProfileConfigPath("work")
	if err := os.WriteFile(workPath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteActiveProfile("work"); err != nil {
		t.Fatal(err)
	}
	if got := ResolveConfigPath(); got != workPath {
		t.Fatalf("got %q want %q", got, workPath)
	}
}

func TestListAndDeleteProfile(t *testing.T) {
	home := filepath.Join(t.TempDir(), "user")
	if err := os.MkdirAll(filepath.Join(home, homeConfigDirName, profilesSubdir), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	SetProfileOverride("")

	defPath := DefaultConfigPath()
	if err := os.WriteFile(defPath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	workPath := ProfileConfigPath("work")
	if err := os.WriteFile(workPath, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteActiveProfile("work"); err != nil {
		t.Fatal(err)
	}

	list, err := ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("expected at least 2 profiles, got %d", len(list))
	}

	if err := DeleteProfile("work"); err != nil {
		t.Fatal(err)
	}
	if fileExists(workPath) {
		t.Fatal("work profile should be deleted")
	}
	if ReadStoredActiveProfile() != DefaultProfileName {
		t.Fatalf("active should reset to default, got %q", ReadStoredActiveProfile())
	}
	if err := DeleteProfile(DefaultProfileName); err == nil {
		t.Fatal("expected error deleting default")
	}
}
