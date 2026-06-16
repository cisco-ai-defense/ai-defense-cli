package configcmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/tui"
)

const (
	editActionCore   = "core"
	editActionDone   = "done"
	editCapConfigure = "configure"
	editCapEdit      = "edit"
	editCapRemove    = "remove"
	editCapBack      = "back"
)

// RunEditHub provides a menu-driven TUI for changing existing configuration.
func RunEditHub() (*capabilities.WizardContext, error) {
	if !isTTY() {
		return nil, clierror.Usage(
			"interactive config edit requires a terminal; set env vars or edit the TOML file directly — see " +
				cmdutil.ClientDocsURL,
		)
	}

	m := appconfig.NewManager()
	if m.ConfigPath() == "" {
		return nil, clierror.Usage("no config file found; run aid-cisco-cli onboard first")
	}
	if err := m.LoadError(); err != nil {
		return nil, clierror.Usage(err.Error())
	}

	ctx := &capabilities.WizardContext{
		Selected: map[string]bool{},
		Skipped:  map[string]bool{},
	}
	ctx.ConfigPath = m.ConfigPath()
	seedFromExisting(ctx, m)
	ctx.OnboardStep = stepComplete

	var existing *appconfig.Manager = m
	save := func() error {
		return saveConfigFromContext(ctx, &existing)
	}

	tui.PrintBanner(os.Stdout, "Configuration", "Edit saved settings.\n↑/↓ move · Enter select · esc back · Ctrl+C cancel")

	choice := editActionCore
	for choice != editActionDone {
		menuOpts := []huh.Option[string]{
			huh.NewOption("Core API settings", editActionCore),
		}
		for _, cap := range capabilities.All() {
			label := cap.Label()
			if cap.IsConfigured(m) {
				label += " (configured)"
			}
			menuOpts = append(menuOpts, huh.NewOption(label, cap.ID()))
		}
		menuOpts = append(menuOpts, huh.NewOption("Done", editActionDone))

		form := tui.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Edit configuration").
					Description(fmt.Sprintf("File: %s\nChoose a section to change.", appconfig.ShortenHome(ctx.ConfigPath))).
					Options(menuOpts...).
					Value(&choice),
			),
		)
		if err := tui.Run(form); err != nil {
			return nil, err
		}
		if choice == editActionDone {
			break
		}
		if choice == editActionCore {
			if err := editCoreSection(ctx, save); err != nil {
				return nil, err
			}
			continue
		}
		if err := editCapabilitySection(ctx, m, choice, save); err != nil {
			return nil, err
		}
		m = appconfig.NewManagerFromPath(ctx.ConfigPath)
	}

	return ctx, nil
}

func editCoreSection(ctx *capabilities.WizardContext, save func() error) error {
	sec := spec.RequiredSection()
	values := map[string]string{
		"api_key":  ctx.APIKey,
		"base_url": ctx.BaseURL,
	}
	after := func(_ string, vals map[string]string) error {
		ctx.APIKey = vals["api_key"]
		ctx.BaseURL = vals["base_url"]
		return save()
	}
	_, err := sec.CollectWizardFunc(values, after, nil)
	return err
}

func editCapabilitySection(ctx *capabilities.WizardContext, m *appconfig.Manager, capID string, save func() error) error {
	cap := capabilityByID(capID)
	if cap == nil {
		return clierror.Usage("unknown capability: " + capID)
	}

	configured := cap.IsConfigured(m)
	action := editCapConfigure
	if configured {
		action = editCapEdit
	}

	var options []huh.Option[string]
	if configured {
		options = []huh.Option[string]{
			huh.NewOption("Edit settings", editCapEdit),
			huh.NewOption("Remove", editCapRemove),
			huh.NewOption("Back", editCapBack),
		}
	} else {
		options = []huh.Option[string]{
			huh.NewOption("Configure", editCapConfigure),
			huh.NewOption("Back", editCapBack),
		}
	}

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(cap.Label()).
				Description(cap.Description()).
				Options(options...).
				Value(&action),
		),
	)
	if err := tui.Run(form); err != nil {
		return err
	}

	switch action {
	case editCapBack:
		return nil
	case editCapRemove:
		ctx.Selected[capID] = false
		ctx.Skipped[capID] = true
		if ctx.IntegrationValues != nil {
			delete(ctx.IntegrationValues, capID)
		}
		return save()
	case editCapConfigure, editCapEdit:
		ctx.Selected[capID] = true
		if err := cap.Collect(ctx); err != nil {
			return err
		}
		if err := cap.Validate(ctx); err != nil {
			return clierror.Usage(err.Error())
		}
		return save()
	}
	return nil
}

// saveConfigFromContext writes the full config from wizard state (no merge with prior partials).
func saveConfigFromContext(ctx *capabilities.WizardContext, existing **appconfig.Manager) error {
	if ctx.ConfigPath == "" {
		return clierror.Usage("config path is not set")
	}
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
	applyOnboardMeta(data, stepComplete)
	if err := appconfig.WriteConfig(ctx.ConfigPath, data); err != nil {
		return clierror.Unexpected("save config: "+err.Error(), err)
	}
	m := appconfig.NewManagerFromPath(ctx.ConfigPath)
	if existing != nil {
		*existing = m
	}
	return nil
}
