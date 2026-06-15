package capabilities

import (
	"fmt"
	"strings"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

// SpecCapability implements Capability from a declarative IntegrationSpec.
type SpecCapability struct {
	Spec spec.IntegrationSpec
}

func (c SpecCapability) ID() string          { return c.Spec.ID }
func (c SpecCapability) Label() string       { return c.Spec.Label }
func (c SpecCapability) Description() string { return c.Spec.Description }

func (c SpecCapability) IsConfigured(m *config.Manager) bool {
	return c.Spec.IsConfigured(m)
}

func (c SpecCapability) Collect(ctx *WizardContext) error {
	return c.collectFields(ctx, nil)
}

// ResumeCollect continues prompts after a partial save (onboard.step = id.field).
func (c SpecCapability) ResumeCollect(ctx *WizardContext, completedStep string) error {
	skip := func(fieldKey string) bool {
		return fieldStepDone(completedStep, c.Spec, fieldKey)
	}
	return c.collectFields(ctx, skip)
}

func (c SpecCapability) collectFields(ctx *WizardContext, skipField func(string) bool) error {
	existing := ctx.IntegrationValues[c.Spec.ID]
	after := func(fieldKey string, values map[string]string) error {
		if ctx.IntegrationValues == nil {
			ctx.IntegrationValues = map[string]map[string]string{}
		}
		ctx.IntegrationValues[c.Spec.ID] = values
		if ctx.SaveProgress == nil {
			return nil
		}
		return ctx.SaveProgress(c.Spec.ID + "." + fieldKey)
	}
	vals, err := c.Spec.CollectWizardFunc(existing, after, skipField)
	if err != nil {
		return err
	}
	if ctx.IntegrationValues == nil {
		ctx.IntegrationValues = map[string]map[string]string{}
	}
	ctx.IntegrationValues[c.Spec.ID] = vals
	return nil
}

func fieldStepDone(completed string, sec spec.SectionSpec, fieldKey string) bool {
	fieldStep := sec.ID + "." + fieldKey
	if completed == fieldStep {
		return true
	}
	if !strings.HasPrefix(completed, sec.ID+".") {
		return false
	}
	doneField := strings.TrimPrefix(completed, sec.ID+".")
	keys := sec.RequiredFieldKeys()
	doneIdx, fieldIdx := indexOf(keys, doneField), indexOf(keys, fieldKey)
	if doneIdx < 0 || fieldIdx < 0 {
		return completed == fieldStep
	}
	return fieldIdx <= doneIdx
}

func indexOf(keys []string, key string) int {
	for i, k := range keys {
		if k == key {
			return i
		}
	}
	return -1
}

func (c SpecCapability) Validate(ctx *WizardContext) error {
	vals := ctx.IntegrationValues[c.Spec.ID]
	if c.Spec.Validate == nil {
		return nil
	}
	return c.Spec.Validate(vals)
}

func (c SpecCapability) Apply(ctx *WizardContext, data map[string]interface{}) {
	c.Spec.Apply(data, ctx.IntegrationValues[c.Spec.ID])
}

func (c SpecCapability) NextSteps() []string {
	return c.Spec.NextSteps
}

func (c SpecCapability) MissingHint() string {
	return c.Spec.MissingHint
}

// SeedIntegrationValues loads TOML values into ctx for edit mode.
func SeedIntegrationValues(ctx *WizardContext, m *config.Manager) {
	if ctx.IntegrationValues == nil {
		ctx.IntegrationValues = map[string]map[string]string{}
	}
	for _, s := range spec.Integrations() {
		ctx.IntegrationValues[s.ID] = s.LoadFromManager(m)
	}
}

// FormatIntegrationSummary returns redacted field lines for enabled integrations.
func FormatIntegrationSummary(ctx *WizardContext) string {
	var lines []string
	for _, cap := range All() {
		if !ctx.Selected[cap.ID()] {
			continue
		}
		sc, ok := cap.(SpecCapability)
		if !ok {
			continue
		}
		vals := ctx.IntegrationValues[sc.Spec.ID]
		for _, f := range sc.Spec.Fields {
			if vals[f.Key] == "" {
				continue
			}
			display := vals[f.Key]
			if f.Kind == spec.FieldKindSecret {
				display = "***"
			}
			lines = append(lines, fmt.Sprintf("  %s: %s", f.Title, display))
		}
	}
	return strings.Join(lines, "\n")
}
