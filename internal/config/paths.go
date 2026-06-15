package config

import (
	"os"
	"path/filepath"
)

const appConfigDir = "aid-cisco-cli"

// LocationKind identifies where the user chose to store config.
type LocationKind string

const (
	LocationXDG     LocationKind = "xdg"
	LocationProject LocationKind = "project"
	LocationCustom  LocationKind = "custom"
)

// ResolveConfigPath returns the path used to load config (first existing candidate).
func ResolveConfigPath() string {
	if p := os.Getenv("AI_DEFENSE_CONFIG_PATH"); p != "" {
		return absPath(p)
	}
	if cwd, err := os.Getwd(); err == nil {
		local := filepath.Join(cwd, ".config.toml")
		if fileExists(local) {
			return local
		}
	}
	if xdg := XDGConfigPath(); fileExists(xdg) {
		return xdg
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, ".config.toml")
	}
	return XDGConfigPath()
}

// PathForLocation maps a wizard location choice to a write target.
func PathForLocation(kind LocationKind, customPath string) (string, error) {
	switch kind {
	case LocationXDG:
		return XDGConfigPath(), nil
	case LocationProject:
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".config.toml"), nil
	case LocationCustom:
		if customPath == "" {
			return "", errCustomPathRequired
		}
		if !filepath.IsAbs(customPath) {
			return "", errCustomPathAbsolute
		}
		return customPath, nil
	default:
		return "", errUnknownLocation
	}
}

// XDGConfigPath returns ~/.config/aid-cisco-cli/config.toml (or XDG_CONFIG_HOME).
func XDGConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".config", appConfigDir, "config.toml")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appConfigDir, "config.toml")
}

func absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	return filepath.Join(cwd, p)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

var (
	errCustomPathRequired  = errorString("custom config path is required")
	errCustomPathAbsolute  = errorString("custom config path must be absolute")
	errUnknownLocation     = errorString("unknown config location")
)

type errorString string

func (e errorString) Error() string { return string(e) }
