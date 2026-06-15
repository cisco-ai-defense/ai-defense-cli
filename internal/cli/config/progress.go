package configcmd

import (
	"strings"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

const (
	stepComplete   = "complete"
	legacyStepCore = "core" // older drafts used a single core step
	legacyStepLoc  = "location"
)

func wizardStepOrder() []string {
	order := []string{}
	for _, key := range spec.RequiredSection().RequiredFieldKeys() {
		order = append(order, key)
	}
	for _, c := range capabilities.All() {
		order = append(order, c.ID())
	}
	return append(order, stepComplete)
}

func stepIndex(step string) int {
	step = normalizeCompletedStep(step)
	for i, s := range wizardStepOrder() {
		if s == step {
			return i
		}
	}
	return -1
}

// stepDone reports whether the wizard has finished target (inclusive).
func stepDone(completed, target string) bool {
	if completed == stepComplete {
		return true
	}
	if target == legacyStepLoc {
		if completed == "" {
			return false
		}
		return completed == legacyStepLoc || stepIndex(normalizeCompletedStep(completed)) >= 0
	}
	completed = normalizeCompletedStep(completed)
	target = normalizeCompletedStep(target)

	if strings.HasPrefix(completed, target+".") {
		return false
	}
	if completed == target {
		return true
	}

	ci, ti := stepIndex(completedBase(completed)), stepIndex(target)
	if ci < 0 || ti < 0 {
		return false
	}
	return ci >= ti
}

func normalizeCompletedStep(step string) string {
	if step == legacyStepCore {
		keys := spec.RequiredSection().RequiredFieldKeys()
		if len(keys) > 0 {
			return keys[len(keys)-1]
		}
	}
	if step == legacyStepLoc {
		return ""
	}
	return step
}

func completedBase(step string) string {
	if i := strings.Index(step, "."); i > 0 {
		return step[:i]
	}
	return step
}

func completedOnboardStep(m *appconfig.Manager, ctx *capabilities.WizardContext) string {
	if m != nil {
		if s := m.NestedString("onboard", "step"); s != "" {
			return s
		}
	}
	keys := spec.RequiredSection().RequiredFieldKeys()
	if len(keys) > 0 && hasCoreField(ctx, keys[len(keys)-1]) {
		return keys[len(keys)-1]
	}
	return ""
}

func hasCoreField(ctx *capabilities.WizardContext, key string) bool {
	switch key {
	case "api_key":
		return ctx.APIKey != ""
	case "base_url":
		return ctx.BaseURL != ""
	default:
		return false
	}
}

func applyOnboardMeta(data map[string]interface{}, step string) {
	if step == "" {
		return
	}
	data["onboard"] = map[string]interface{}{
		"step": step,
	}
}
