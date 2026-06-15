package configcmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cli/config/capabilities"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/config/spec"
)

// RunOptions controls wizard execution.
type RunOptions struct {
	Mode       string // init, edit
	Force      bool
	Onboard    bool
	ConfigPath string // output from last run
}

// RunWizard executes the core + capability hub TUI and writes config.
func RunWizard(opts RunOptions) (*capabilities.WizardContext, error) {
	if !isTTY() {
		return nil, clierror.Usage(
			"interactive config wizard requires a terminal; set AI_DEFENSE_API_KEY and AI_DEFENSE_BASE_URL " +
				"or create a .config.toml manually (see docs/getting-started-client.md)",
		)
	}

	ctx := &capabilities.WizardContext{Selected: map[string]bool{}}
	var existing *appconfig.Manager
	if opts.Mode == "edit" {
		path := appconfig.ResolveConfigPath()
		existing = appconfig.NewManagerFromPath(path)
		if existing.ConfigPath() != "" {
			ctx.ConfigPath = existing.ConfigPath()
			seedFromExisting(ctx, existing)
		}
	}

	if err := runCoreForms(ctx, opts); err != nil {
		return nil, err
	}
	if err := runCapabilityHub(ctx, existing, opts.Mode == "edit"); err != nil {
		return nil, err
	}
	for _, cap := range capabilities.All() {
		if !ctx.Selected[cap.ID()] {
			continue
		}
		if err := cap.Collect(ctx); err != nil {
			return nil, err
		}
		if err := cap.Validate(ctx); err != nil {
			return nil, clierror.Usage(err.Error())
		}
	}

	data := map[string]interface{}{
		"default": map[string]interface{}{
			"api_key":  ctx.APIKey,
			"base_url": ctx.BaseURL,
		},
	}
	for _, cap := range capabilities.All() {
		if ctx.Selected[cap.ID()] {
			cap.Apply(ctx, data)
		}
	}

	if err := confirmSummary(ctx); err != nil {
		return nil, err
	}

	targetPath, err := appconfig.PathForLocation(ctx.LocationKind, ctx.CustomPath)
	if err != nil {
		return nil, clierror.Usage(err.Error())
	}
	if opts.Mode == "init" && !opts.Force {
		if _, err := os.Stat(targetPath); err == nil {
			return nil, clierror.Usage(fmt.Sprintf("config already exists at %s (use --force to overwrite)", targetPath))
		}
	}

	if err := appconfig.WriteConfig(targetPath, data); err != nil {
		return nil, clierror.Unexpected("write config: "+err.Error(), err)
	}
	ctx.ConfigPath = targetPath
	return ctx, nil
}

func runCoreForms(ctx *capabilities.WizardContext, opts RunOptions) error {
	var locationChoice string
	if ctx.ConfigPath == "" {
		locationChoice = "xdg"
	} else if ctx.LocationKind != "" {
		switch ctx.LocationKind {
		case appconfig.LocationXDG:
			locationChoice = "xdg"
		case appconfig.LocationProject:
			locationChoice = "project"
		default:
			locationChoice = "custom"
		}
	}

	customPath := ctx.CustomPath

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Config file location").
				Description("Where should settings be saved?").
				Options(
					huh.NewOption("User config (~/.config/aid-cisco-cli/config.toml)", "xdg"),
					huh.NewOption("Project local (./.config.toml in current directory)", "project"),
					huh.NewOption("Custom absolute path", "custom"),
				).
				Value(&locationChoice),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	switch locationChoice {
	case "xdg":
		ctx.LocationKind = appconfig.LocationXDG
	case "project":
		ctx.LocationKind = appconfig.LocationProject
	case "custom":
		ctx.LocationKind = appconfig.LocationCustom
		customForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom config path").
					Value(&customPath).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("path is required")
						}
						return nil
					}),
			),
		)
		if err := customForm.Run(); err != nil {
			return err
		}
		ctx.CustomPath = customPath
	}

	existing := map[string]string{
		"api_key":  ctx.APIKey,
		"base_url": ctx.BaseURL,
	}
	core, err := spec.CollectCoreFields(existing)
	if err != nil {
		return err
	}
	ctx.APIKey = core["api_key"]
	ctx.BaseURL = core["base_url"]
	return nil
}

func runCapabilityHub(ctx *capabilities.WizardContext, existing *appconfig.Manager, editMode bool) error {
	caps := capabilities.All()
	options := make([]huh.Option[string], 0, len(caps))
	for _, c := range caps {
		options = append(options, huh.NewOption(c.Label(), c.ID()))
	}

	selected := make([]string, 0)
	if editMode && existing != nil {
		for _, c := range caps {
			if c.IsConfigured(existing) {
				selected = append(selected, c.ID())
				ctx.Selected[c.ID()] = true
			}
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Optional capabilities").
				Description("Select any you need now; you can add more later with config edit").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	ctx.Selected = map[string]bool{}
	for _, id := range selected {
		ctx.Selected[id] = true
	}
	return nil
}

func confirmSummary(ctx *capabilities.WizardContext) error {
	var enabled []string
	for _, c := range capabilities.All() {
		if ctx.Selected[c.ID()] {
			enabled = append(enabled, c.Label())
		}
	}
	summary := fmt.Sprintf("Base URL: %s\nAPI key: ***\n", ctx.BaseURL)
	if len(enabled) > 0 {
		summary += "Optional capabilities:\n"
		for _, e := range enabled {
			summary += "  - " + e + "\n"
		}
		if detail := capabilities.FormatIntegrationSummary(ctx); detail != "" {
			summary += detail + "\n"
		}
	} else {
		summary += "Optional capabilities: none"
	}

	var proceed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save configuration?").
				Description(summary).
				Value(&proceed),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	if !proceed {
		return clierror.Usage("configuration not saved")
	}
	return nil
}

func seedFromExisting(ctx *capabilities.WizardContext, m *appconfig.Manager) {
	ctx.APIKey = m.NestedString("default", "api_key")
	ctx.BaseURL = m.NestedString("default", "base_url")
	capabilities.SeedIntegrationValues(ctx, m)
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PrintNextSteps writes post-onboard guidance to stdout.
func PrintNextSteps(ctx *capabilities.WizardContext) {
	fmt.Fprintf(os.Stdout, "\nConfig saved to: %s\n\nNext steps:\n", ctx.ConfigPath)
	fmt.Fprintf(os.Stdout, "  aid-cisco-cli model-scan list --limit 5\n")
	for _, cap := range capabilities.All() {
		if ctx.Selected[cap.ID()] {
			for _, step := range cap.NextSteps() {
				fmt.Fprintf(os.Stdout, "  %s\n", step)
			}
		}
	}
	fmt.Fprintf(os.Stdout, "\nDocs: docs/getting-started-client.md\n")
}
