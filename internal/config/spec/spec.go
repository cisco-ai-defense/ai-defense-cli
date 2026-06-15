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
	Required    bool // required when capability is enabled in wizard
	Options     []FieldOption
	AllowCustom bool // when true, wizard offers "Custom…" after options
}

// SectionSpec is a TOML section (e.g. [default.aibom]).
type SectionSpec struct {
	TomlPath []string // e.g. ["default", "aibom"]
	Fields   []FieldSpec
}

// WizardPreset groups fields for provider-style onboarding flows.
type WizardPreset struct {
	ID          string
	Label       string
	SetFields   map[string]string // preset values applied before field collection
	RequireKeys []string          // fields to prompt for (empty = infer from Required on fields)
	OmitKeys    []string          // fields skipped even if Required
}

// IntegrationSpec describes an optional capability or integration block.
type IntegrationSpec struct {
	ID          string
	Label       string
	Description string
	Section     SectionSpec
	Presets     []WizardPreset // when set, wizard starts with preset picker
	MissingHint string
	NextSteps   []string
	Validate    func(values map[string]string) error
}

// FieldByKey returns the field spec for key or nil.
func (s IntegrationSpec) FieldByKey(key string) *FieldSpec {
	for i := range s.Section.Fields {
		if s.Section.Fields[i].Key == key {
			return &s.Section.Fields[i]
		}
	}
	return nil
}

// ValueSource reads nested TOML values (implemented by config.Manager).
type ValueSource interface {
	NestedString(keys ...string) string
}

// Resolve applies flag → env → TOML precedence for all fields in the section.
func (s IntegrationSpec) Resolve(flagValues map[string]string, src ValueSource) map[string]string {
	out := make(map[string]string, len(s.Section.Fields))
	for _, f := range s.Section.Fields {
		v := ""
		if flagValues != nil {
			v = flagValues[f.Key]
		}
		if v == "" && f.EnvVar != "" {
			v = os.Getenv(f.EnvVar)
		}
		if v == "" && src != nil {
			keys := append(append([]string{}, s.Section.TomlPath...), f.Key)
			v = src.NestedString(keys...)
		}
		out[f.Key] = v
	}
	return out
}

// ResolveField applies flag → env → TOML for one field on an integration.
func (s IntegrationSpec) ResolveField(fieldKey, flagValue string, src ValueSource) string {
	flags := map[string]string{fieldKey: flagValue}
	return s.Resolve(flags, src)[fieldKey]
}

// IsConfigured reports whether resolved values pass validation.
func (s IntegrationSpec) IsConfigured(src ValueSource) bool {
	if s.Validate == nil {
		return s.hasAnyValue(src)
	}
	return s.Validate(s.Resolve(nil, src)) == nil
}

func (s IntegrationSpec) hasAnyValue(src ValueSource) bool {
	vals := s.Resolve(nil, src)
	for _, v := range vals {
		if v != "" {
			return true
		}
	}
	return false
}

// Apply writes non-empty field values into a TOML root map at Section.TomlPath.
func (s IntegrationSpec) Apply(data map[string]interface{}, values map[string]string) {
	section := map[string]interface{}{}
	for _, f := range s.Section.Fields {
		if v := strings.TrimSpace(values[f.Key]); v != "" {
			section[f.Key] = v
		}
	}
	if len(section) == 0 {
		return
	}
	cur := data
	for i, key := range s.Section.TomlPath {
		if i == len(s.Section.TomlPath)-1 {
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
func (s IntegrationSpec) SecretTomlPaths() [][]string {
	var paths [][]string
	for _, f := range s.Section.Fields {
		if f.Kind == FieldKindSecret {
			path := append(append([]string{}, s.Section.TomlPath...), f.Key)
			paths = append(paths, path)
		}
	}
	return paths
}

// FlagUsage builds help text listing env vars for the integration.
func (s IntegrationSpec) FlagUsage() string {
	var b strings.Builder
	for _, f := range s.Section.Fields {
		if f.EnvVar != "" {
			fmt.Fprintf(&b, "%s (env: %s); ", f.Key, f.EnvVar)
		}
	}
	return strings.TrimSuffix(b.String(), "; ")
}

// RequiredFieldKeys returns keys marked Required on fields (for simple integrations).
func (s IntegrationSpec) RequiredFieldKeys() []string {
	var keys []string
	for _, f := range s.Section.Fields {
		if f.Required {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// KeysForPreset returns which field keys to collect for a wizard preset.
func (s IntegrationSpec) KeysForPreset(presetID string) []string {
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
