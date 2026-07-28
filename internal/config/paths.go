// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	homeConfigDirName  = ".ai-defense"
	configFileName     = "config.toml"
	legacyAppConfigDir = "cisco-aid-cli"
)

// HomeConfigDir returns ~/.ai-defense (CLI state directory, like ~/.aws or ~/.ssh).
func HomeConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return homeConfigDirName
	}
	return filepath.Join(home, homeConfigDirName)
}

// DefaultConfigPath returns ~/.ai-defense/config.toml (default profile).
func DefaultConfigPath() string {
	return filepath.Join(HomeConfigDir(), configFileName)
}

// ConfigPathForWrite returns where the wizard writes for the active profile.
func ConfigPathForWrite() string {
	if p := os.Getenv("AI_DEFENSE_CONFIG_PATH"); p != "" {
		return absPath(p)
	}
	return ProfileConfigPath(ActiveProfile())
}

// ResolveConfigPath returns the path used to load config.
func ResolveConfigPath() string {
	if p := os.Getenv("AI_DEFENSE_CONFIG_PATH"); p != "" {
		return absPath(p)
	}
	profile := ActiveProfile()
	path := ProfileConfigPath(profile)
	if fileExists(path) {
		return path
	}
	if profile == DefaultProfileName {
		if p := legacyXDGConfigPath(); fileExists(p) {
			return p
		}
		if p := legacyProjectConfigPath(); p != "" && fileExists(p) {
			return p
		}
	}
	return path
}

// ExistingConfigPath returns a config file on disk for resume prompts, if any.
func ExistingConfigPath() (string, bool) {
	if p := os.Getenv("AI_DEFENSE_CONFIG_PATH"); p != "" {
		p = absPath(p)
		return p, fileExists(p)
	}
	profile := ActiveProfile()
	path := ProfileConfigPath(profile)
	if fileExists(path) {
		return path, true
	}
	if profile == DefaultProfileName {
		for _, p := range []string{legacyXDGConfigPath(), legacyProjectConfigPath()} {
			if p != "" && fileExists(p) {
				return p, true
			}
		}
	}
	return "", false
}

// ConfigLocationLabel returns a short human label for a config file path.
func ConfigLocationLabel(path string) string {
	profile := profileNameForPath(path)
	if profile != "" {
		if profile == DefaultProfileName {
			return "Profile default (" + shortenHome(path) + ")"
		}
		return "Profile " + profile + " (" + shortenHome(path) + ")"
	}
	return "Config (" + shortenHome(path) + ")"
}

// ProfileNameForPath returns the profile name for a config file path, or "" if unknown.
func ProfileNameForPath(path string) string {
	return profileNameForPath(path)
}

func profileNameForPath(path string) string {
	if path == DefaultConfigPath() {
		return DefaultProfileName
	}
	dir := ProfilesDir()
	if strings.HasPrefix(path, dir+string(filepath.Separator)) && strings.HasSuffix(path, ".toml") {
		base := filepath.Base(path)
		return strings.TrimSuffix(base, ".toml")
	}
	return ""
}

func legacyXDGConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".config", legacyAppConfigDir, configFileName)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, legacyAppConfigDir, configFileName)
}

func legacyProjectConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".config.toml")
}

func absPath(p string) string {
	p = expandHome(p)
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	return filepath.Join(cwd, p)
}

// expandHome replaces a leading ~ with the user home directory (macOS, Linux, and Windows).
func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// FileExists reports whether path is a regular file.
func FileExists(path string) bool {
	return fileExists(path)
}

// ShortenHome replaces the user home directory prefix with ~ for display.
func ShortenHome(path string) string {
	return shortenHome(path)
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return filepath.Join("~", strings.TrimPrefix(path, prefix))
	}
	return path
}
