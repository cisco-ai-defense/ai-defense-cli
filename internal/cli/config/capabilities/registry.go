package capabilities

import (
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

// WizardContext holds wizard state shared across core and capability modules.
type WizardContext struct {
	ConfigPath string
	BaseURL    string
	APIKey     string

	Selected map[string]bool

	// Skipped marks capabilities explicitly skipped this session (omit from TOML).
	Skipped map[string]bool

	// StartFresh is true when the user chose to replace an existing config file.
	StartFresh bool

	// OnboardStep is the last completed onboarding step (persisted under [onboard] in config.toml).
	OnboardStep string

	// SaveProgress persists ctx after a wizard step (set by RunWizard).
	SaveProgress func(step string) error

	// IntegrationValues holds field values keyed by integration id (from spec).
	IntegrationValues map[string]map[string]string
}

// Capability is one optional config module in the onboarding hub.
type Capability interface {
	ID() string
	Label() string
	Description() string
	IsConfigured(m *config.Manager) bool
	Collect(ctx *WizardContext) error
	Validate(ctx *WizardContext) error
	Apply(ctx *WizardContext, data map[string]interface{})
	NextSteps() []string
	MissingHint() string
}

// All returns registered capabilities in stable display order (spec-driven).
func All() []Capability {
	specs := spec.Integrations()
	caps := make([]Capability, len(specs))
	for i, s := range specs {
		caps[i] = SpecCapability{Spec: s}
	}
	return caps
}

func byID(id string) Capability {
	if s, ok := spec.ByID(id); ok {
		return SpecCapability{Spec: s}
	}
	return nil
}

// MissingHint returns the hint for a capability id.
func MissingHint(id string) string {
	if c := byID(id); c != nil {
		return c.MissingHint()
	}
	return ""
}
