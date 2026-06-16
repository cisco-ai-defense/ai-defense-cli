package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	appconfig "github.com/cisco-ai-defense/ai-defense-cli/internal/config"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "List and switch configuration profiles",
		Long:  "Manage named configuration profiles under ~/.ai-defense (like AWS profiles).",
	}
	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileCurrentCmd())
	cmd.AddCommand(newProfileUseCmd())
	cmd.AddCommand(newProfileCreateCmd())
	cmd.AddCommand(newProfileDeleteCmd())
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configuration profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := appconfig.ListProfiles()
			if err != nil {
				return clierror.Unexpected("list profiles: "+err.Error(), err)
			}
			session := appconfig.ActiveProfile()
			stored := appconfig.ReadStoredActiveProfile()
			for _, p := range profiles {
				marker := " "
				if p.Active {
					marker = "*"
				}
				status := "missing"
				if p.Exists {
					status = "ready"
				}
				fmt.Fprintf(os.Stdout, "%s %-12s %-7s %s\n", marker, p.Name, status, p.Path)
			}
			if session != stored {
				fmt.Fprintf(os.Stdout, "\nSession profile (--profile / %s): %s\n", "AI_DEFENSE_PROFILE", session)
			}
			return nil
		},
	}
}

func newProfileCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the active profile and config path",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := appconfig.ActiveProfile()
			path := appconfig.ResolveConfigPath()
			fmt.Fprintf(os.Stdout, "profile: %s\npath:    %s\n", name, path)
			return nil
		},
	}
}

func newProfileUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active configuration profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if err := appconfig.ValidateProfileName(name); err != nil {
				return clierror.Usage(err.Error())
			}
			if err := appconfig.WriteActiveProfile(name); err != nil {
				return clierror.Unexpected("set profile: "+err.Error(), err)
			}
			appconfig.SetProfileOverride("")
			path := appconfig.ProfileConfigPath(name)
			if !appconfig.FileExists(path) && name != appconfig.DefaultProfileName {
				fmt.Fprintf(os.Stdout, "Active profile set to %q (not configured yet).\n", name)
				fmt.Fprintf(os.Stdout, "Run: cisco-aid-cli config profile create %s\n", name)
				return nil
			}
			fmt.Fprintf(os.Stdout, "Active profile: %s\nConfig path:   %s\n", name, appconfig.ResolveConfigPath())
			return nil
		},
	}
}

func newProfileCreateCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a profile with the onboarding wizard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if err := appconfig.ValidateProfileName(name); err != nil {
				return clierror.Usage(err.Error())
			}
			path := appconfig.ProfileConfigPath(name)
			if !force && appconfig.FileExists(path) {
				return clierror.Usage(fmt.Sprintf("profile %q already exists at %s; use --force to replace", name, path))
			}
			if err := appconfig.WriteActiveProfile(name); err != nil {
				return clierror.Unexpected("set profile: "+err.Error(), err)
			}
			appconfig.SetProfileOverride(name)
			ctx, err := RunWizard(RunOptions{
				Force:   force,
				Profile: name,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Profile %q saved to %s\n", name, ctx.ConfigPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Replace an existing profile file")
	return cmd
}

func newProfileDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a named profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if err := appconfig.DeleteProfile(name); err != nil {
				return clierror.Usage(err.Error())
			}
			fmt.Fprintf(os.Stdout, "Deleted profile %q\n", name)
			return nil
		},
	}
}

// ProfileFlag registers --profile on a command (empty = active profile).
func ProfileFlag(cmd *cobra.Command, dest *string) {
	cmd.Flags().StringVar(dest, "profile", "", "Configuration profile name (env: AI_DEFENSE_PROFILE)")
}

// ApplyProfileFlag applies a command-local --profile value for this run.
func ApplyProfileFlag(name string) {
	if name != "" {
		appconfig.SetProfileOverride(name)
	}
}
