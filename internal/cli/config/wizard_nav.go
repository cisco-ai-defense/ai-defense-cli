package configcmd

import (
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

// wizardBackFrom rewinds the wizard to the step before fromStep. It returns the
// capability loop index to resume at (0 = first optional capability), or -1 when
// the caller should propagate ErrFormBack (e.g. to the profile picker).
func wizardBackFrom(ctx *capabilities.WizardContext, saver *draftSaver, fromStep string) (capIndex int, err error) {
	prev := previousWizardStep(fromStep)
	if prev == "" {
		return -1, tui.ErrFormBack
	}

	switch {
	case isCoreStep(prev):
		clearCapabilitiesFrom(ctx, firstCapabilityID())
		if err := recollectCoreField(ctx, saver, prev); err != nil {
			return -1, err
		}
		return 0, nil
	default:
		// Re-prompt the previous capability; clear its answer and anything after it.
		clearCapabilitiesFrom(ctx, prev)
		before := previousWizardStep(prev)
		if before == "" {
			return -1, tui.ErrFormBack
		}
		ctx.OnboardStep = before
		if err := saver.save(ctx); err != nil {
			return -1, err
		}
		idx := capabilityIndex(prev)
		if idx < 0 {
			return -1, clierror.Unexpected("unknown capability step: "+prev, nil)
		}
		return idx, nil
	}
}

func isCoreStep(step string) bool {
	firstCap := firstCapabilityID()
	idx := stepIndex(step)
	if idx < 0 {
		return false
	}
	if firstCap == "" {
		return true
	}
	return idx < stepIndex(firstCap)
}

func firstCapabilityID() string {
	caps := capabilities.All()
	if len(caps) == 0 {
		return ""
	}
	return caps[0].ID()
}

func capabilityIndex(id string) int {
	for i, c := range capabilities.All() {
		if c.ID() == id {
			return i
		}
	}
	return -1
}

func previousWizardStep(step string) string {
	base := completedBase(normalizeCompletedStep(step))
	order := wizardStepOrder()
	for i, s := range order {
		if s == base && i > 0 {
			return order[i-1]
		}
	}
	return ""
}

// clearCapabilitiesFrom removes wizard answers for fromID and every later capability.
func clearCapabilitiesFrom(ctx *capabilities.WizardContext, fromID string) {
	fromIdx := stepIndex(fromID)
	if fromIdx < 0 {
		return
	}
	for _, c := range capabilities.All() {
		if stepIndex(c.ID()) < fromIdx {
			continue
		}
		delete(ctx.Selected, c.ID())
		if ctx.Skipped != nil {
			delete(ctx.Skipped, c.ID())
		}
		if ctx.IntegrationValues != nil {
			delete(ctx.IntegrationValues, c.ID())
		}
	}
}
