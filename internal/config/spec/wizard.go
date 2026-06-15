package spec

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/huh"
)

const wizardCustomValue = "__custom__"

// CollectCoreFields prompts for all core fields in spec order (onboard / config init).
func CollectCoreFields(existing map[string]string) (map[string]string, error) {
	values := copyValues(existing)
	for _, f := range CoreFields() {
		current := values[f.Key]
		v, err := f.CollectWizardValue(current)
		if err != nil {
			return nil, err
		}
		values[f.Key] = v
	}
	return values, nil
}

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
		v, err := field.CollectWizardValue(current)
		if err != nil {
			return nil, err
		}
		values[key] = v
	}
	return values, nil
}

// CollectWizardValue prompts for one field (plain input, or select from options + optional custom).
func (f FieldSpec) CollectWizardValue(current string) (string, error) {
	if len(f.Options) > 0 {
		return f.collectWithOptions(current)
	}
	return f.collectPlainInput(current)
}

func (f FieldSpec) collectWithOptions(current string) (string, error) {
	choice := matchOptionValue(f.Options, current)
	if choice == "" && current != "" {
		choice = wizardCustomValue
	}

	opts := make([]huh.Option[string], 0, len(f.Options)+1)
	for _, o := range f.Options {
		opts = append(opts, huh.NewOption(o.Label, o.Value))
	}
	if f.AllowCustom {
		opts = append(opts, huh.NewOption("Custom…", wizardCustomValue))
	}

	selectField := huh.NewSelect[string]().
		Title(f.Title).
		Options(opts...).
		Value(&choice)
	if f.Description != "" {
		selectField = selectField.Description(f.Description)
	}
	if err := huh.NewForm(huh.NewGroup(selectField)).Run(); err != nil {
		return "", err
	}

	if choice != wizardCustomValue {
		return choice, nil
	}
	custom := current
	if matchOptionValue(f.Options, current) != "" {
		custom = ""
	}
	return f.collectPlainInput(custom)
}

func (f FieldSpec) collectPlainInput(current string) (string, error) {
	input := huh.NewInput().
		Title(f.Title).
		Value(&current)
	if f.Description != "" {
		input = input.Description(f.Description)
	}
	if f.Kind == FieldKindSecret {
		input = input.EchoMode(huh.EchoModePassword)
	}
	if f.Required {
		input = input.Validate(func(v string) error {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("%s is required", f.Title)
			}
			return nil
		})
	}
	if f.Key == "base_url" {
		input = input.Validate(validateBaseURLHost)
	}
	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(current), nil
}

func matchOptionValue(options []FieldOption, value string) string {
	for _, o := range options {
		if o.Value == value {
			return o.Value
		}
	}
	return ""
}

func validateBaseURLHost(s string) error {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid base URL")
	}
	host := u.Hostname()
	if host != "cisco.com" && !strings.HasSuffix(host, ".cisco.com") {
		return fmt.Errorf("base URL host must be a *.cisco.com domain")
	}
	return nil
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
