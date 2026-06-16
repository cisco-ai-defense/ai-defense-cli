package configcmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

// RunOptions controls wizard execution (onboard / config init).
type RunOptions struct {
	Force      bool // start over and replace an existing config file
	ConfigPath string // output from last run
	Profile    string // profile name (empty = active profile)
}

func profileForWizard(opts RunOptions) string {
	if opts.Profile != "" {
		return opts.Profile
	}
	return appconfig.ActiveProfile()
}

// RunWizard executes the onboarding TUI and writes config.
func RunWizard(opts RunOptions) (*capabilities.WizardContext, error) {
	if !isTTY() {
		return nil, clierror.Usage(
			"interactive config wizard requires a terminal; set AI_DEFENSE_API_KEY and AI_DEFENSE_BASE_URL " +
				"or create ~/.ai-defense/config.toml manually — see " + cmdutil.ClientDocsURL,
		)
	}

	tui.PrintBanner(os.Stdout, "Setup", "Configure API credentials and optional integrations.\n↑/↓ move · Enter select · esc back · Ctrl+C cancel")

	for {
		profileName, err := resolveWizardProfile(opts)
		if err != nil {
			return nil, err
		}
		opts.Profile = profileName

		ctx := &capabilities.WizardContext{
			Selected:   map[string]bool{},
			Skipped:    map[string]bool{},
			ConfigPath: appconfig.ProfileConfigPath(profileName),
		}
		var existing *appconfig.Manager

		if !opts.Force {
			if path, ok := appconfig.ExistingConfigPathForProfile(profileName); ok {
				resume, err := promptContinueOrFresh(path, profileName)
				if err != nil {
					return nil, err
				}
				if !resume {
					opts.Force = true
					ctx.StartFresh = true
				} else {
					existing = appconfig.NewManagerFromPath(path)
					seedFromExisting(ctx, existing)
				}
			}
		}

		if opts.Force && existing == nil {
			ctx.StartFresh = true
		}

		completed := ""
		if existing != nil {
			completed = completedOnboardStep(existing, ctx)
		}

		saver := newDraftSaver(&existing)
		ctx.SaveProgress = func(step string) error {
			ctx.OnboardStep = step
			return saver.save(ctx)
		}

		if err := runCoreForms(ctx, saver, completed); errors.Is(err, tui.ErrFormBack) {
			opts.Profile = ""
			continue
		} else if err != nil {
			return nil, err
		}
		if err := runOptionalCapabilities(ctx, saver, completed); errors.Is(err, tui.ErrFormBack) {
			opts.Profile = ""
			continue
		} else if err != nil {
			return nil, err
		}

		if !stepDone(ctx.OnboardStep, stepComplete) {
			ctx.OnboardStep = stepComplete
			if err := saver.save(ctx); err != nil {
				return nil, err
			}
		}

		if name := profileForWizard(opts); name != "" {
			if err := appconfig.WriteActiveProfile(name); err != nil {
				return nil, clierror.Unexpected("set active profile: "+err.Error(), err)
			}
		}

		return ctx, nil
	}
}

type draftSaver struct {
	existing **appconfig.Manager
}

func newDraftSaver(existing **appconfig.Manager) *draftSaver {
	return &draftSaver{existing: existing}
}

func (d *draftSaver) save(ctx *capabilities.WizardContext) error {
	path := ctx.ConfigPath
	if path == "" {
		path = appconfig.ConfigPathForWrite()
	}

	var prior *appconfig.Manager
	if ctx.StartFresh {
		prior = nil
	} else {
		prior = d.managerAt(path)
		if prior == nil {
			profile := appconfig.ProfileNameForPath(path)
			if profile == "" {
				profile = appconfig.DefaultProfileName
			}
			if readPath, ok := appconfig.ExistingConfigPathForProfile(profile); ok {
				prior = d.managerAt(readPath)
			}
		}
	}
	data := buildConfigData(ctx, prior)

	// Start over before core fields: persist only onboard progress (replaces prior sections).
	if ctx.StartFresh && ctx.APIKey == "" && ctx.BaseURL == "" {
		data = map[string]interface{}{}
		applyOnboardMeta(data, ctx.OnboardStep)
	}

	if err := appconfig.WriteConfig(path, data); err != nil {
		return clierror.Unexpected("save config: "+err.Error(), err)
	}
	ctx.ConfigPath = path
	m := appconfig.NewManagerFromPath(path)
	*d.existing = m
	return nil
}

func (d *draftSaver) managerAt(path string) *appconfig.Manager {
	if m := d.manager(); m != nil && m.ConfigPath() == path {
		return m
	}
	if appconfig.FileExists(path) {
		return appconfig.NewManagerFromPath(path)
	}
	return nil
}

func (d *draftSaver) manager() *appconfig.Manager {
	if d.existing == nil || *d.existing == nil {
		return nil
	}
	return *d.existing
}

func buildConfigData(ctx *capabilities.WizardContext, existing *appconfig.Manager) map[string]interface{} {
	data := map[string]interface{}{}
	spec.RequiredSection().Apply(data, map[string]string{
		"api_key":  ctx.APIKey,
		"base_url": ctx.BaseURL,
	})
	for _, cap := range capabilities.All() {
		if ctx.Selected[cap.ID()] {
			cap.Apply(ctx, data)
		}
	}
	if existing != nil {
		mergePreservedCore(data, existing, ctx)
		mergePreservedCapabilities(data, existing, ctx)
	}
	applyOnboardMeta(data, ctx.OnboardStep)
	return data
}

func promptContinueOrFresh(_ string, profileName string) (resume bool, err error) {
	choice := "continue"
	desc := fmt.Sprintf("Profile %q has saved settings.\n\nContinue where you left off, or start over?", profileName)

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Existing configuration").
				Description(desc).
				Options(
					huh.NewOption("Continue", "continue"),
					huh.NewOption("Start over", "fresh"),
				).
				Value(&choice),
		),
	)
	if err := tui.Run(form); err != nil {
		return false, err
	}
	return choice == "continue", nil
}

func runCoreForms(ctx *capabilities.WizardContext, saver *draftSaver, completed string) error {
	return collectCoreFields(ctx, saver, completed)
}

func collectCoreFields(ctx *capabilities.WizardContext, saver *draftSaver, completed string) error {
	sec := spec.RequiredSection()
	values := map[string]string{
		"api_key":  ctx.APIKey,
		"base_url": ctx.BaseURL,
	}
	skip := func(key string) bool {
		return stepDone(completed, key)
	}
	after := func(key string, vals map[string]string) error {
		ctx.APIKey = vals["api_key"]
		ctx.BaseURL = vals["base_url"]
		ctx.OnboardStep = key
		return saver.save(ctx)
	}
	_, err := sec.CollectWizardFunc(values, after, skip)
	return err
}

func mergePreservedCore(data map[string]interface{}, existing *appconfig.Manager, ctx *capabilities.WizardContext) {
	def, _ := data["default"].(map[string]interface{})
	if def == nil {
		def = map[string]interface{}{}
	}
	if ctx.APIKey == "" {
		if v := existing.NestedString("default", "api_key"); v != "" {
			def["api_key"] = v
		}
	}
	if ctx.BaseURL == "" {
		if v := existing.NestedString("default", "base_url"); v != "" {
			def["base_url"] = v
		}
	}
	if len(def) > 0 {
		data["default"] = def
	}
}

func mergePreservedCapabilities(data map[string]interface{}, existing *appconfig.Manager, ctx *capabilities.WizardContext) {
	for _, cap := range capabilities.All() {
		if ctx.Selected[cap.ID()] {
			continue
		}
		if ctx.Skipped != nil && ctx.Skipped[cap.ID()] {
			continue
		}
		if !cap.IsConfigured(existing) {
			continue
		}
		if sc, ok := cap.(capabilities.SpecCapability); ok {
			sc.Spec.Apply(data, sc.Spec.LoadFromManager(existing))
		}
	}
}

const (
	capabilityConfigure = "configure"
	capabilitySkip      = "skip"
)

// runOptionalCapabilities walks each optional capability one at a time.
func runOptionalCapabilities(ctx *capabilities.WizardContext, saver *draftSaver, completed string) error {
	caps := capabilities.All()
	for i := 0; i < len(caps); {
		cap := caps[i]
		if stepDone(completed, cap.ID()) {
			i++
			continue
		}
		id := cap.ID()
		if strings.HasPrefix(completed, id+".") {
			ctx.Selected[id] = true
			sc, ok := cap.(capabilities.SpecCapability)
			if !ok {
				i++
				continue
			}
			if err := sc.ResumeCollect(ctx, completed); errors.Is(err, tui.ErrFormBack) {
				restart, err := wizardBackFrom(ctx, saver, id)
				if errors.Is(err, tui.ErrFormBack) {
					return err
				}
				if err != nil {
					return err
				}
				completed = ctx.OnboardStep
				i = restart
				continue
			} else if err != nil {
				return err
			}
			if err := cap.Validate(ctx); err != nil {
				return clierror.Usage(err.Error())
			}
			ctx.OnboardStep = id
			if err := saver.save(ctx); err != nil {
				return err
			}
			completed = ctx.OnboardStep
			i++
			continue
		}
		if err := promptOptionalCapability(ctx, cap, saver); errors.Is(err, tui.ErrFormBack) {
			restart, err := wizardBackFrom(ctx, saver, id)
			if errors.Is(err, tui.ErrFormBack) {
				return err
			}
			if err != nil {
				return err
			}
			completed = ctx.OnboardStep
			i = restart
			continue
		} else if err != nil {
			return err
		}
		completed = ctx.OnboardStep
		i++
	}
	return nil
}

func recollectCoreField(ctx *capabilities.WizardContext, saver *draftSaver, key string) error {
	sec := spec.RequiredSection()
	values := map[string]string{
		"api_key":  ctx.APIKey,
		"base_url": ctx.BaseURL,
	}
	after := func(fieldKey string, vals map[string]string) error {
		ctx.APIKey = vals["api_key"]
		ctx.BaseURL = vals["base_url"]
		ctx.OnboardStep = fieldKey
		return saver.save(ctx)
	}
	if err := sec.CollectFieldKey(key, values, after); err != nil {
		return err
	}
	ctx.APIKey = values["api_key"]
	ctx.BaseURL = values["base_url"]
	return nil
}

func promptOptionalCapability(ctx *capabilities.WizardContext, cap capabilities.Capability, saver *draftSaver) error {
	for {
		choice := capabilitySkip
		selectField := huh.NewSelect[string]().
			Title(cap.Label()).
			Description(cap.Description() + "\nPress esc to go back.").
			Options(
				huh.NewOption("Configure", capabilityConfigure),
				huh.NewOption("Skip for now", capabilitySkip),
			).
			Value(&choice)
		if err := tui.RunFormEscBack(huh.NewGroup(selectField)); errors.Is(err, tui.ErrFormBack) {
			return tui.ErrFormBack
		} else if err != nil {
			return err
		}

		switch choice {
		case capabilitySkip:
			ctx.Selected[cap.ID()] = false
			if ctx.Skipped == nil {
				ctx.Skipped = map[string]bool{}
			}
			ctx.Skipped[cap.ID()] = true
			if ctx.IntegrationValues != nil {
				delete(ctx.IntegrationValues, cap.ID())
			}
			ctx.OnboardStep = cap.ID()
			return saver.save(ctx)
		case capabilityConfigure:
			ctx.Selected[cap.ID()] = true
			if err := cap.Collect(ctx); errors.Is(err, tui.ErrFormBack) {
				continue
			} else if err != nil {
				return err
			}
			if err := cap.Validate(ctx); err != nil {
				return clierror.Usage(err.Error())
			}
			ctx.OnboardStep = cap.ID()
			return saver.save(ctx)
		}
	}
}

func seedFromExisting(ctx *capabilities.WizardContext, m *appconfig.Manager) {
	ctx.APIKey = m.NestedString("default", "api_key")
	ctx.BaseURL = m.NestedString("default", "base_url")
	ctx.OnboardStep = m.NestedString("onboard", "step")
	capabilities.SeedIntegrationValues(ctx, m)
	for _, cap := range capabilities.All() {
		if cap.IsConfigured(m) {
			ctx.Selected[cap.ID()] = true
		}
	}
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PrintNextSteps writes post-onboard guidance to stdout.
func PrintNextSteps(ctx *capabilities.WizardContext) {
	profile := appconfig.ActiveProfile()
	var body strings.Builder
	fmt.Fprintf(&body, "Config saved to %s", appconfig.ShortenHome(ctx.ConfigPath))
	if profile != "" {
		fmt.Fprintf(&body, " (profile: %s)", profile)
	}
	active := appconfig.ResolveConfigPath()
	if active != "" && ctx.ConfigPath != "" && active != ctx.ConfigPath {
		fmt.Fprintf(&body, "\n\nNote: commands load config from %s.", appconfig.ShortenHome(active))
	}
	body.WriteString("\n\nTry:\n  aid-cisco-cli model-scan list --limit 5")
	for _, cap := range capabilities.All() {
		if ctx.Selected[cap.ID()] {
			for _, step := range cap.NextSteps() {
				fmt.Fprintf(&body, "\n  %s", step)
			}
		}
	}
	body.WriteString("\n  aid-cisco-cli config show")
	tui.PrintBanner(os.Stdout, "Complete", body.String())
	fmt.Fprintf(os.Stdout, "\nUser guide: %s\n", cmdutil.ClientDocsURL)
}
