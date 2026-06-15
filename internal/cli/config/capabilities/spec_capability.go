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
	existing := ctx.IntegrationValues[c.Spec.ID]
	vals, err := c.Spec.CollectWizard(existing)
	if err != nil {
		return err
	}
	if ctx.IntegrationValues == nil {
		ctx.IntegrationValues = map[string]map[string]string{}
	}
	ctx.IntegrationValues[c.Spec.ID] = vals
	return nil
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
