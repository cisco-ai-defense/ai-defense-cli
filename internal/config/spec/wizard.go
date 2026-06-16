package spec

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

const (
	wizardCustomValue = "__custom__"
)

// WizardFieldCallback runs after each field is collected (onboarding draft save).
type WizardFieldCallback func(fieldKey string, values map[string]string) error

// CollectRequiredSection prompts for the required section fields (onboard / config init).
func CollectRequiredSection(existing map[string]string) (map[string]string, error) {
	return RequiredSection().CollectWizard(existing)
}

// CollectWizard prompts for section fields using the spec (and presets when defined).
func (s SectionSpec) CollectWizard(existing map[string]string) (map[string]string, error) {
	return s.CollectWizardFunc(existing, nil, nil)
}

// CollectWizardFunc prompts field-by-field; after each field optionally calls after.
// skipField skips prompts for keys already completed when resuming onboarding.
func (s SectionSpec) CollectWizardFunc(existing map[string]string, after WizardFieldCallback, skipField func(string) bool) (map[string]string, error) {
	values := copyValues(existing)

	if len(s.Presets) > 0 {
		if hasAnyValue(values) {
			return s.collectFields(values, s.keysForResume(values), after, skipField)
		}
		return s.collectWithPresets(values, after, skipField)
	}
	return s.collectFields(values, s.RequiredFieldKeys(), after, skipField)
}

func hasAnyValue(values map[string]string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

// keysForResume returns field keys to collect when resuming a preset-based section.
func (s SectionSpec) keysForResume(values map[string]string) []string {
	for _, p := range s.Presets {
		if s.presetMatchesValues(p, values) {
			return s.KeysForPreset(p.ID)
		}
	}
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	for _, k := range s.RequiredFieldKeys() {
		add(k)
	}
	for _, p := range s.Presets {
		for _, k := range s.KeysForPreset(p.ID) {
			add(k)
		}
	}
	return keys
}

func (s SectionSpec) presetMatchesValues(p WizardPreset, values map[string]string) bool {
	for k, want := range p.SetFields {
		if values[k] != want {
			return false
		}
	}
	return true
}

func (s SectionSpec) collectWithPresets(values map[string]string, after WizardFieldCallback, skipField func(string) bool) (map[string]string, error) {
	options := make([]huh.Option[string], len(s.Presets))
	for i, p := range s.Presets {
		options[i] = huh.NewOption(p.Label, p.ID)
	}
	var presetID string
	form := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(s.Label).
				Description(s.Description).
				Options(options...).
				Value(&presetID),
		),
	)
	if err := tui.Run(form); err != nil {
		return nil, err
	}

	for k, v := range s.presetByID(presetID).SetFields {
		values[k] = v
	}
	keys := s.KeysForPreset(presetID)
	return s.collectFields(values, keys, after, skipField)
}

func (s SectionSpec) presetByID(id string) WizardPreset {
	for _, p := range s.Presets {
		if p.ID == id {
			return p
		}
	}
	return WizardPreset{}
}

func (s SectionSpec) collectFields(values map[string]string, keys []string, after WizardFieldCallback, skipField func(string) bool) (map[string]string, error) {
	forcePrompt := false
	for i := 0; i < len(keys); {
		key := keys[i]
		if skipField != nil && skipField(key) && !forcePrompt {
			i++
			continue
		}
		forcePrompt = false
		field := s.FieldByKey(key)
		if field == nil {
			i++
			continue
		}
		current := values[key]
		v, err := field.CollectWizardValue(current)
		if errors.Is(err, tui.ErrFormBack) {
			if i == 0 {
				return values, tui.ErrFormBack
			}
			i--
			forcePrompt = true
			continue
		}
		if err != nil {
			return nil, err
		}
		values[key] = v
		if after != nil {
			if err := after(key, values); err != nil {
				return nil, err
			}
		}
		i++
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

	customURL := strings.TrimSpace(current)
	if matchOptionValue(f.Options, customURL) != "" {
		customURL = ""
	}

	for {
		var presetChoice string
		if choice != "" {
			presetChoice = choice
		}

		selectField := huh.NewSelect[string]().
			Title(f.Title).
			Options(opts...).
			Value(&presetChoice)
		if f.Description != "" {
			selectField = selectField.Description(f.Description + "\nPress esc to go back.")
		} else {
			selectField = selectField.Description("Press esc to go back.")
		}

		if err := tui.RunFormEscBack(huh.NewGroup(selectField)); errors.Is(err, tui.ErrFormBack) {
			return "", tui.ErrFormBack
		} else if err != nil {
			return "", err
		}
		choice = presetChoice

		if choice != wizardCustomValue {
			return choice, nil
		}

		url, back, err := f.collectCustomOptionValue(customURL)
		if err != nil {
			return "", err
		}
		if back {
			choice = ""
			continue
		}
		return url, nil
	}
}

// collectCustomOptionValue prompts for a custom base URL. Press esc to return to
// the region preset list; enter submits (validation runs after submit).
func (f FieldSpec) collectCustomOptionValue(seed string) (url string, back bool, err error) {
	customURL := strings.TrimSpace(seed)
	validationHint := ""

	for {
		desc := "Enter a *.cisco.com AI Defense API endpoint. Press esc to go back."
		if validationHint != "" {
			desc = validationHint + "\n\n" + desc
		}
		customInput := huh.NewInput().
			Title("Custom base URL").
			Description(desc).
			Value(&customURL).
			Placeholder("https://us.api.aidefense.security.cisco.com")

		err := tui.RunFormEscBack(huh.NewGroup(customInput))
		if errors.Is(err, tui.ErrFormBack) {
			return "", true, nil
		}
		if err != nil {
			return "", false, err
		}

		trimmed := strings.TrimSpace(customURL)
		if err := validateWizardBaseURL(f.Title, trimmed); err != nil {
			validationHint = err.Error()
			continue
		}
		return trimmed, false, nil
	}
}

func validateWizardBaseURL(title, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", title)
	}
	return ValidateBaseURLHost(value)
}

func (f FieldSpec) newInputField(description string, value *string) *huh.Input {
	input := huh.NewInput().
		Title(f.Title).
		Value(value)
	if description != "" {
		input = input.Description(description)
	}
	if f.Kind == FieldKindSecret {
		input = input.EchoMode(huh.EchoModePassword)
	}
	return f.applyInputValidators(input)
}

func (f FieldSpec) applyInputValidators(input *huh.Input) *huh.Input {
	var validators []func(string) error
	if f.Required {
		validators = append(validators, func(v string) error {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("%s is required", f.Title)
			}
			return nil
		})
	}
	if f.Key == "base_url" {
		validators = append(validators, ValidateBaseURLHost)
	}
	if len(validators) == 0 {
		return input
	}
	return input.Validate(func(v string) error {
		for _, validate := range validators {
			if err := validate(v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (f FieldSpec) collectPlainInput(current string) (string, error) {
	desc := f.Description
	if desc != "" {
		desc += "\n\n"
	}
	desc += "Press esc to go back."
	input := f.newInputField(desc, &current)
	if err := tui.RunFormEscBack(huh.NewGroup(input)); errors.Is(err, tui.ErrFormBack) {
		return "", tui.ErrFormBack
	} else if err != nil {
		return "", err
	}
	return strings.TrimSpace(current), nil
}

// CollectFieldKey prompts for a single section field (used to revise a prior step).
func (s SectionSpec) CollectFieldKey(key string, values map[string]string, after WizardFieldCallback) error {
	field := s.FieldByKey(key)
	if field == nil {
		return fmt.Errorf("unknown field %q", key)
	}
	for {
		v, err := field.CollectWizardValue(values[key])
		if errors.Is(err, tui.ErrFormBack) {
			prev, ok := s.previousRequiredKey(key)
			if !ok {
				return tui.ErrFormBack
			}
			key = prev
			field = s.FieldByKey(key)
			continue
		}
		if err != nil {
			return err
		}
		values[key] = v
		if after != nil {
			if err := after(key, values); err != nil {
				return err
			}
		}
		return nil
	}
}

func (s SectionSpec) previousRequiredKey(key string) (string, bool) {
	keys := s.RequiredFieldKeys()
	for i, k := range keys {
		if k == key && i > 0 {
			return keys[i-1], true
		}
	}
	return "", false
}

func matchOptionValue(options []FieldOption, value string) string {
	for _, o := range options {
		if o.Value == value {
			return o.Value
		}
	}
	return ""
}

func copyValues(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// LoadFromManager reads TOML-only values (no env) for wizard seeding.
func (s SectionSpec) LoadFromManager(src ValueSource) map[string]string {
	out := make(map[string]string)
	for _, f := range s.Fields {
		keys := append(append([]string{}, s.TomlPath...), f.Key)
		out[f.Key] = src.NestedString(keys...)
	}
	return out
}
