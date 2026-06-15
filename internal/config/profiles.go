package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// DefaultProfileName is the root config.toml profile (backward compatible).
	DefaultProfileName = "default"
	activeProfileFile  = "profile"
	profilesSubdir     = "profiles"
	profileEnvVar      = "AI_DEFENSE_PROFILE"
)

var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// profileOverride is set by the global --profile flag before commands run.
var profileOverride string

// SetProfileOverride sets the profile from the CLI --profile flag (empty clears).
func SetProfileOverride(name string) {
	profileOverride = strings.TrimSpace(name)
}

// ResetProfileOverride clears any CLI --profile override (for tests).
func ResetProfileOverride() {
	profileOverride = ""
}

// ValidateProfileName returns an error if name is not a valid profile identifier.
func ValidateProfileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if name == DefaultProfileName || profileNamePattern.MatchString(name) {
		return nil
	}
	return fmt.Errorf("invalid profile name %q: use letters, digits, hyphen, underscore; max 64 characters", name)
}

// ActiveProfile returns the profile selected for this process.
// Priority: flag override → AI_DEFENSE_PROFILE → ~/.ai-defense/profile → default.
func ActiveProfile() string {
	if profileOverride != "" {
		return profileOverride
	}
	if p := strings.TrimSpace(os.Getenv(profileEnvVar)); p != "" {
		return p
	}
	return ReadStoredActiveProfile()
}

// ReadStoredActiveProfile reads the persisted active profile (default if missing).
func ReadStoredActiveProfile() string {
	path := ActiveProfileFilePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return DefaultProfileName
	}
	name := strings.TrimSpace(string(raw))
	if name == "" {
		return DefaultProfileName
	}
	if err := ValidateProfileName(name); err != nil {
		return DefaultProfileName
	}
	return name
}

// WriteActiveProfile persists the active profile name.
func WriteActiveProfile(name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if err := os.MkdirAll(HomeConfigDir(), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	path := ActiveProfileFilePath()
	tmp, err := os.CreateTemp(HomeConfigDir(), ".profile-*")
	if err != nil {
		return fmt.Errorf("create temp profile file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(name + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write profile file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod profile file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close profile file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace profile file: %w", err)
	}
	return nil
}

// ActiveProfileFilePath returns ~/.ai-defense/profile.
func ActiveProfileFilePath() string {
	return filepath.Join(HomeConfigDir(), activeProfileFile)
}

// ProfilesDir returns ~/.ai-defense/profiles.
func ProfilesDir() string {
	return filepath.Join(HomeConfigDir(), profilesSubdir)
}

// ProfileConfigPath returns the config file path for a profile name.
func ProfileConfigPath(name string) string {
	if name == "" || name == DefaultProfileName {
		return DefaultConfigPath()
	}
	return filepath.Join(ProfilesDir(), name+".toml")
}

// ProfileInfo describes one on-disk profile.
type ProfileInfo struct {
	Name   string
	Path   string
	Active bool
	Exists bool
}

// ListProfiles returns known profiles (default if config.toml exists, plus profiles/*.toml).
func ListProfiles() ([]ProfileInfo, error) {
	active := ReadStoredActiveProfile()
	flagActive := ActiveProfile()
	seen := map[string]bool{}
	var out []ProfileInfo

	add := func(name, path string) {
		if seen[name] {
			return
		}
		seen[name] = true
		exists := fileExists(path)
		if name == DefaultProfileName && !exists {
			// include default only when file exists or it is active
			if active != DefaultProfileName && flagActive != DefaultProfileName {
				return
			}
		}
		out = append(out, ProfileInfo{
			Name:   name,
			Path:   path,
			Active: name == active,
			Exists: exists,
		})
	}

	defPath := DefaultConfigPath()
	if fileExists(defPath) || active == DefaultProfileName || flagActive == DefaultProfileName {
		add(DefaultProfileName, defPath)
	}

	entries, err := os.ReadDir(ProfilesDir())
	if err != nil {
		if os.IsNotExist(err) {
			if len(out) == 0 {
				add(DefaultProfileName, defPath)
			}
			return out, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		if err := ValidateProfileName(name); err != nil {
			continue
		}
		add(name, filepath.Join(ProfilesDir(), e.Name()))
	}
	if len(out) == 0 {
		add(DefaultProfileName, defPath)
	}
	return out, nil
}

// ExistingConfigPathForProfile returns an on-disk config for a specific profile, if any.
func ExistingConfigPathForProfile(name string) (string, bool) {
	if err := ValidateProfileName(name); err != nil {
		return "", false
	}
	path := ProfileConfigPath(name)
	if fileExists(path) {
		return path, true
	}
	if name == DefaultProfileName {
		for _, p := range []string{legacyXDGConfigPath(), legacyProjectConfigPath()} {
			if p != "" && fileExists(p) {
				return p, true
			}
		}
	}
	return "", false
}

// ListProfilesForOnboarding returns profiles for the onboard picker (always includes default).
func ListProfilesForOnboarding() ([]ProfileInfo, error) {
	storedActive := ReadStoredActiveProfile()
	seen := map[string]bool{}
	var out []ProfileInfo

	add := func(name, path string) {
		if seen[name] {
			return
		}
		seen[name] = true
		out = append(out, ProfileInfo{
			Name:   name,
			Path:   path,
			Active: name == storedActive,
			Exists: fileExists(path),
		})
	}

	add(DefaultProfileName, DefaultConfigPath())

	entries, err := os.ReadDir(ProfilesDir())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		if err := ValidateProfileName(name); err != nil {
			continue
		}
		add(name, filepath.Join(ProfilesDir(), e.Name()))
	}
	return out, nil
}

// DeleteProfile removes a named profile file. default cannot be deleted.
func DeleteProfile(name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if name == DefaultProfileName {
		return fmt.Errorf("cannot delete the default profile; edit or remove %s manually", shortenHome(DefaultConfigPath()))
	}
	path := ProfileConfigPath(name)
	if !fileExists(path) {
		return fmt.Errorf("profile %q does not exist", name)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete profile %q: %w", name, err)
	}
	if ReadStoredActiveProfile() == name {
		if err := WriteActiveProfile(DefaultProfileName); err != nil {
			return err
		}
	}
	return nil
}
