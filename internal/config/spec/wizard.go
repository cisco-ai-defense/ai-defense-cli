package spec

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// CollectWizard prompts for integration fields using the spec (and presets when defined).
func (s IntegrationSpec) CollectWizard(existing map[string]string) (map[string]string, error) {
	values := copyValues(existing)

	if len(s.Presets) > 0 {
		return s.collectWithPresets(values)
	}
	return s.collectFields(values, s.RequiredFieldKeys())
}

func (s IntegrationSpec) collectWithPresets(values map[string]string) (map[string]string, error) {
	options := make([]huh.Option[string], len(s.Presets))
	for i, p := range s.Presets {
		options[i] = huh.NewOption(p.Label, p.ID)
	}
	var presetID string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(s.Label).
				Description(s.Description).
				Options(options...).
				Value(&presetID),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	for k, v := range s.presetByID(presetID).SetFields {
		values[k] = v
	}
	keys := s.KeysForPreset(presetID)
	return s.collectFields(values, keys)
}

func (s IntegrationSpec) presetByID(id string) WizardPreset {
	for _, p := range s.Presets {
		if p.ID == id {
			return p
		}
	}
	return WizardPreset{}
}

func (s IntegrationSpec) collectFields(values map[string]string, keys []string) (map[string]string, error) {
	for _, key := range keys {
		field := s.FieldByKey(key)
		if field == nil {
			continue
		}
		current := values[key]
		input := huh.NewInput().
			Title(field.Title).
			Value(&current)
		if field.Description != "" {
			input = input.Description(field.Description)
		}
		if field.Kind == FieldKindSecret {
			input = input.EchoMode(huh.EchoModePassword)
		}
		if field.Required {
			input = input.Validate(func(v string) error {
				if strings.TrimSpace(v) == "" {
					return fmt.Errorf("%s is required", field.Title)
				}
				return nil
			})
		}
		form := huh.NewForm(huh.NewGroup(input))
		if err := form.Run(); err != nil {
			return nil, err
		}
		values[key] = current
	}
	return values, nil
}

func copyValues(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// LoadFromManager reads TOML-only values (no env) for wizard seeding.
func (s IntegrationSpec) LoadFromManager(src ValueSource) map[string]string {
	out := make(map[string]string)
	for _, f := range s.Section.Fields {
		keys := append(append([]string{}, s.Section.TomlPath...), f.Key)
		out[f.Key] = src.NestedString(keys...)
	}
	return out
}
