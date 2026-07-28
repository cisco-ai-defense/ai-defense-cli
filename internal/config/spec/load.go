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
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.spec.yaml
var embeddedSpec []byte

// Schema is the parsed CLI config contract (from config.spec.yaml).
type Schema struct {
	Meta     metaDef      `yaml:"meta"`
	Sections []sectionDef `yaml:"sections"`
}

type metaDef struct {
	SchemaVersion string `yaml:"schema_version"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
}

type sectionDef struct {
	ID          string     `yaml:"id"`
	Label       string     `yaml:"label"`
	Description string     `yaml:"description"`
	TomlPath    []string   `yaml:"toml_path"` // optional; defaults to [id]
	Required    bool       `yaml:"required"`
	Fields      []fieldDef `yaml:"fields"`
}

type fieldOptionDef struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

type fieldDef struct {
	Key         string           `yaml:"key"`
	Title       string           `yaml:"title"`
	Description string           `yaml:"description"`
	Kind        string           `yaml:"kind"`
	Required    bool             `yaml:"required"`
	Optional    bool             `yaml:"optional"` // group fields only
	Env         string           `yaml:"env"`
	Flag        string           `yaml:"flag"`
	Options     []fieldOptionDef `yaml:"options"`
	AllowCustom bool             `yaml:"allow_custom"`
	// group field metadata (kind: group)
	Label       string      `yaml:"label"`
	MissingHint string      `yaml:"missing_hint"`
	NextSteps   []string    `yaml:"next_steps"`
	Validator   string      `yaml:"validator"`
	Presets     []presetDef `yaml:"presets"`
	Validate    []ruleDef   `yaml:"validate"`
	Fields      []fieldDef  `yaml:"fields"`
}

type presetDef struct {
	ID     string            `yaml:"id"`
	Label  string            `yaml:"label"`
	Set    map[string]string `yaml:"set"`
	Prompt []string          `yaml:"prompt"`
	Omit   []string          `yaml:"omit"`
}

type ruleDef struct {
	Type    string `yaml:"type"`
	Field   string `yaml:"field"`
	Message string `yaml:"message"`
}

var (
	schema          Schema
	sectionsByID    map[string]SectionSpec
	sectionsList    []SectionSpec
	requiredSection SectionSpec
)

func init() {
	if err := loadEmbedded(embeddedSpec); err != nil {
		panic("config spec: " + err.Error())
	}
}

// LoadSchema parses a config.spec.yaml document.
func LoadSchema(raw []byte) (Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return Schema{}, fmt.Errorf("parse config.spec.yaml: %w", err)
	}
	if s.Meta.SchemaVersion == "" {
		return Schema{}, fmt.Errorf("config.spec.yaml: meta.schema_version is required")
	}
	if len(s.Sections) == 0 {
		return Schema{}, fmt.Errorf("config.spec.yaml: at least one section is required")
	}
	return s, nil
}

func loadEmbedded(raw []byte) error {
	s, err := LoadSchema(raw)
	if err != nil {
		return err
	}
	schema = s

	sectionsList, err = buildSectionsList(s.Sections)
	if err != nil {
		return err
	}
	sectionsByID = make(map[string]SectionSpec, len(sectionsList))
	var foundRequired bool
	for _, sec := range sectionsList {
		sectionsByID[sec.ID] = sec
		if sec.Required {
			if foundRequired {
				return fmt.Errorf("config.spec.yaml: multiple required sections (only one allowed)")
			}
			requiredSection = sec
			foundRequired = true
		}
	}
	if !foundRequired {
		return fmt.Errorf("config.spec.yaml: exactly one section must have required: true")
	}
	return nil
}

func buildSectionsList(defs []sectionDef) ([]SectionSpec, error) {
	if len(defs) != 1 {
		return nil, fmt.Errorf("config.spec.yaml: exactly one root section is required")
	}
	root := defs[0]
	if !root.Required {
		return nil, fmt.Errorf("root section %q must have required: true", root.ID)
	}
	if root.ID == "" {
		return nil, fmt.Errorf("root section id is required")
	}

	rootPath, err := resolveTomlPath(root.ID, root.TomlPath)
	if err != nil {
		return nil, err
	}

	var scalarFields []FieldSpec
	var optional []SectionSpec

	for _, fd := range root.Fields {
		if fd.isGroup() {
			if !fd.Optional {
				return nil, fmt.Errorf("group field %q must have optional: true", fd.Key)
			}
			sec, err := fd.toGroupSectionSpec(root.ID)
			if err != nil {
				return nil, fmt.Errorf("group %q: %w", fd.Key, err)
			}
			optional = append(optional, sec)
			continue
		}
		if fd.Optional {
			return nil, fmt.Errorf("field %q: only group fields may be optional", fd.Key)
		}
		scalarFields = append(scalarFields, fd.toFieldSpec())
	}

	required := SectionSpec{
		ID:       root.ID,
		Label:    root.Label,
		TomlPath: rootPath,
		Required: true,
		Fields:   scalarFields,
	}

	out := make([]SectionSpec, 0, 1+len(optional))
	out = append(out, required)
	out = append(out, optional...)
	return out, nil
}

func resolveTomlPath(id string, explicit []string) ([]string, error) {
	if len(explicit) > 0 {
		return explicit, nil
	}
	if id == "" {
		return nil, fmt.Errorf("toml_path or id is required")
	}
	return []string{id}, nil
}

func (f fieldDef) isGroup() bool {
	return strings.EqualFold(f.Kind, "group")
}

func (g fieldDef) toGroupSectionSpec(parentID string) (SectionSpec, error) {
	if g.Key == "" {
		return SectionSpec{}, fmt.Errorf("key is required")
	}
	if len(g.Fields) == 0 {
		return SectionSpec{}, fmt.Errorf("group must declare fields")
	}
	for _, nf := range g.Fields {
		if nf.isGroup() {
			return SectionSpec{}, fmt.Errorf("nested groups are not supported")
		}
	}

	fields := make([]FieldSpec, len(g.Fields))
	for i, f := range g.Fields {
		fields[i] = f.toFieldSpec()
	}

	presets := make([]WizardPreset, len(g.Presets))
	for i, p := range g.Presets {
		presets[i] = WizardPreset{
			ID:          p.ID,
			Label:       p.Label,
			SetFields:   p.Set,
			RequireKeys: p.Prompt,
			OmitKeys:    p.Omit,
		}
	}

	validate, err := buildValidatorMeta(g.Validator, g.Validate)
	if err != nil {
		return SectionSpec{}, err
	}

	label := g.Label
	if label == "" {
		label = g.Title
	}

	return SectionSpec{
		ID:          g.Key,
		Label:       label,
		Description: g.Description,
		TomlPath:    []string{parentID, g.Key},
		Optional:    true,
		Fields:      fields,
		Presets:     presets,
		MissingHint: g.MissingHint,
		NextSteps:   g.NextSteps,
		Validate:    validate,
	}, nil
}

func (f fieldDef) toFieldSpec() FieldSpec {
	kind := FieldKindString
	if strings.EqualFold(f.Kind, "secret") {
		kind = FieldKindSecret
	}
	opts := make([]FieldOption, len(f.Options))
	for i, o := range f.Options {
		opts[i] = FieldOption{Label: o.Label, Value: o.Value}
	}
	return FieldSpec{
		Key:         f.Key,
		EnvVar:      f.Env,
		Flag:        f.Flag,
		Title:       f.Title,
		Description: f.Description,
		Kind:        kind,
		Required:    f.Required,
		Options:     opts,
		AllowCustom: f.AllowCustom,
	}
}

func buildValidatorMeta(validator string, rules []ruleDef) (func(map[string]string) error, error) {
	if validator != "" {
		fn, ok := namedValidators[validator]
		if !ok {
			return nil, fmt.Errorf("unknown validator %q", validator)
		}
		return fn, nil
	}
	if len(rules) > 0 {
		return declarativeValidator(rules), nil
	}
	return nil, nil
}

func declarativeValidator(rules []ruleDef) func(map[string]string) error {
	return func(values map[string]string) error {
		for _, r := range rules {
			switch r.Type {
			case "non_empty":
				if strings.TrimSpace(values[r.Field]) == "" {
					msg := r.Message
					if msg == "" {
						msg = fmt.Sprintf("%s is required", r.Field)
					}
					return fmt.Errorf("%s", msg)
				}
			default:
				return fmt.Errorf("unknown validate rule type %q", r.Type)
			}
		}
		return nil
	}
}

// Sections returns all config sections from the spec.
func Sections() []SectionSpec {
	return sectionsList
}

// OptionalSections returns optional group fields (capability hub entries).
func OptionalSections() []SectionSpec {
	out := make([]SectionSpec, 0)
	for _, s := range sectionsList {
		if s.Optional {
			out = append(out, s)
		}
	}
	return out
}

// Integrations is an alias for OptionalSections (capability hub entries).
func Integrations() []SectionSpec {
	return OptionalSections()
}

// RequiredSection returns the single required section (e.g. default).
func RequiredSection() SectionSpec {
	return requiredSection
}

// ByID returns a section spec by id (root id or group field key).
func ByID(id string) (SectionSpec, bool) {
	sec, ok := sectionsByID[id]
	return sec, ok
}

// MustByID returns a section spec or panics if the id is absent from config.spec.yaml.
func MustByID(id string) SectionSpec {
	sec, ok := ByID(id)
	if !ok {
		panic("config spec: section " + id + " not defined in config.spec.yaml")
	}
	return sec
}

// EmbeddedSpecRaw returns the embedded config.spec.yaml bytes (for docs/tests).
func EmbeddedSpecRaw() []byte {
	return embeddedSpec
}

// GroupFields returns kind: group field defs from the root section (schema shape).
func GroupFields(root sectionDef) []fieldDef {
	var groups []fieldDef
	for _, f := range root.Fields {
		if f.isGroup() {
			groups = append(groups, f)
		}
	}
	return groups
}
