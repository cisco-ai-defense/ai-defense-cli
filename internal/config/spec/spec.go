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

package spec

import (
	"fmt"
	"os"
	"strings"
)

// FieldKind describes how a config field is stored and collected.
type FieldKind int

const (
	FieldKindString FieldKind = iota
	FieldKindSecret
)

// FieldOption is a preset value for wizard select fields (e.g. regional base URLs).
type FieldOption struct {
	Label string
	Value string
}

// FieldSpec declares one TOML key with env/flag bindings and wizard metadata.
type FieldSpec struct {
	Key         string
	EnvVar      string
	Flag        string // CLI flag name without leading dashes (e.g. llm-model)
	Title       string
	Description string
	Kind        FieldKind
	Required    bool // required when section is enabled in wizard
	Options     []FieldOption
	AllowCustom bool // when true, wizard offers "Custom…" after options
}

// WizardPreset groups fields for provider-style onboarding flows.
type WizardPreset struct {
	ID          string
	Label       string
	SetFields   map[string]string
	RequireKeys []string
	OmitKeys    []string
}

// SectionSpec describes one config section (TOML path, fields, wizard metadata).
type SectionSpec struct {
	ID          string
	Label       string
	Description string
	TomlPath    []string
	Required    bool // always collected in onboard (not in optional capability hub)
	Optional    bool // shown in capability hub; omitted from TOML when not selected
	Fields      []FieldSpec
	Presets     []WizardPreset
	MissingHint string
	NextSteps   []string
	Validate    func(values map[string]string) error
}

// IntegrationSpec is an alias for optional/historical call sites (e.g. ByID("aibom")).
type IntegrationSpec = SectionSpec

// FieldByKey returns the field spec for key or nil.
func (s SectionSpec) FieldByKey(key string) *FieldSpec {
	for i := range s.Fields {
		if s.Fields[i].Key == key {
			return &s.Fields[i]
		}
	}
	return nil
}

// ValueSource reads nested TOML values (implemented by config.Manager).
type ValueSource interface {
	NestedString(keys ...string) string
}

// Resolve applies flag → env → TOML precedence for all fields in the section.
func (s SectionSpec) Resolve(flagValues map[string]string, src ValueSource) map[string]string {
	out := make(map[string]string, len(s.Fields))
	for _, f := range s.Fields {
		v := ""
		if flagValues != nil {
			v = flagValues[f.Key]
		}
		if v == "" && f.EnvVar != "" {
			v = os.Getenv(f.EnvVar)
		}
		if v == "" && src != nil {
			keys := append(append([]string{}, s.TomlPath...), f.Key)
			v = src.NestedString(keys...)
		}
		out[f.Key] = v
	}
	return out
}

// ResolveField applies flag → env → TOML for one field on the section.
func (s SectionSpec) ResolveField(fieldKey, flagValue string, src ValueSource) string {
	flags := map[string]string{fieldKey: flagValue}
	return s.Resolve(flags, src)[fieldKey]
}

// IsConfigured reports whether resolved values pass validation.
func (s SectionSpec) IsConfigured(src ValueSource) bool {
	if s.Validate == nil {
		return s.hasAnyValue(src)
	}
	return s.Validate(s.Resolve(nil, src)) == nil
}

func (s SectionSpec) hasAnyValue(src ValueSource) bool {
	vals := s.Resolve(nil, src)
	for _, v := range vals {
		if v != "" {
			return true
		}
	}
	return false
}

// Apply writes non-empty field values into a TOML root map at TomlPath.
func (s SectionSpec) Apply(data map[string]interface{}, values map[string]string) {
	section := map[string]interface{}{}
	for _, f := range s.Fields {
		if v := strings.TrimSpace(values[f.Key]); v != "" {
			section[f.Key] = v
		}
	}
	if len(section) == 0 {
		return
	}
	cur := data
	for i, key := range s.TomlPath {
		if i == len(s.TomlPath)-1 {
			existing, _ := cur[key].(map[string]interface{})
			if existing == nil {
				existing = map[string]interface{}{}
			}
			for k, v := range section {
				existing[k] = v
			}
			cur[key] = existing
			return
		}
		next, ok := cur[key].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			cur[key] = next
		}
		cur = next
	}
}

// SecretTomlPaths returns TOML paths for secret fields (for redaction).
func (s SectionSpec) SecretTomlPaths() [][]string {
	var paths [][]string
	for _, f := range s.Fields {
		if f.Kind == FieldKindSecret {
			path := append(append([]string{}, s.TomlPath...), f.Key)
			paths = append(paths, path)
		}
	}
	return paths
}

// FlagUsage builds help text listing env vars for the section.
func (s SectionSpec) FlagUsage() string {
	var b strings.Builder
	for _, f := range s.Fields {
		if f.EnvVar != "" {
			fmt.Fprintf(&b, "%s (env: %s); ", f.Key, f.EnvVar)
		}
	}
	return strings.TrimSuffix(b.String(), "; ")
}

// RequiredFieldKeys returns keys marked Required on fields.
func (s SectionSpec) RequiredFieldKeys() []string {
	var keys []string
	for _, f := range s.Fields {
		if f.Required {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// KeysForPreset returns which field keys to collect for a wizard preset.
func (s SectionSpec) KeysForPreset(presetID string) []string {
	for _, p := range s.Presets {
		if p.ID == presetID {
			if len(p.RequireKeys) > 0 {
				return subtractKeys(p.RequireKeys, p.OmitKeys)
			}
			break
		}
	}
	return s.RequiredFieldKeys()
}

func subtractKeys(keys, omit []string) []string {
	omitSet := map[string]bool{}
	for _, k := range omit {
		omitSet[k] = true
	}
	var out []string
	for _, k := range keys {
		if !omitSet[k] {
			out = append(out, k)
		}
	}
	return out
}
