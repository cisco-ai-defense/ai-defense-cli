package configcmd

import (
	"strings"
	"testing"

	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func TestProfileSelectLabel(t *testing.T) {
	t.Parallel()
	got := profileSelectLabel(appconfig.ProfileInfo{
		Name:   "work",
		Active: true,
		Exists: true,
	})
	if !strings.Contains(got, "active") || !strings.Contains(got, "ready") {
		t.Fatalf("got %q", got)
	}
}

func TestFormatProfileOverview(t *testing.T) {
	t.Parallel()
	profiles := []appconfig.ProfileInfo{
		{Name: "default", Active: false, Exists: true, Path: "/home/u/.ai-defense/config.toml"},
		{Name: "work", Active: true, Exists: true, Path: "/home/u/.ai-defense/profiles/work.toml"},
	}
	out := formatProfileOverview(profiles, "work")
	if !strings.Contains(out, "Stored active profile: work") {
		t.Fatalf("missing active line: %q", out)
	}
	if !strings.Contains(out, "* work") {
		t.Fatalf("missing active marker: %q", out)
	}
	if strings.Contains(out, "/home/") {
		t.Fatal("overview should not include file paths")
	}
}
