package install

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/selfupdate"
)

// NewCmd returns the install subcommand.
func NewCmd(currentVersion string) *cobra.Command {
	var (
		version string
		check   bool
		prefix  string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Download and install a release build of cisco-aid-cli",
		Long: `Install or change the running CLI by downloading a verified release
binary from GitHub Releases. Use after the initial manual bootstrap from the
Releases page.

By default the running binary is replaced in the same directory as the
executable you invoked (or --prefix). On Unix you can install while the CLI
is running; the new file is written atomically and takes effect on the next
invocation. Close other running instances if your platform locks executables.

Examples:
  cisco-aid-cli install
  cisco-aid-cli install --version 0.1.2
  cisco-aid-cli install --check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return clierror.Usage("install accepts no positional arguments")
			}
			installPrefix := prefix
			if installPrefix == "" {
				var err error
				installPrefix, err = selfupdate.DefaultPrefix()
				if err != nil {
					return clierror.Usage(fmt.Sprintf("could not determine install prefix: %v (use --prefix)", err))
				}
			}

			opts := selfupdate.Options{
				Version: version,
				Prefix:  installPrefix,
			}

			if check {
				res, err := selfupdate.Check(opts, currentVersion)
				if err != nil {
					return classifyInstallErr(err)
				}
				if res.Changed {
					fmt.Fprintf(os.Stdout, "Update available: %s -> %s (%s)\n",
						res.CurrentVersion, res.TargetVersion, res.Artifact)
				} else {
					fmt.Fprintf(os.Stdout, "Already at %s (%s)\n", res.TargetVersion, res.Artifact)
				}
				fmt.Fprintf(os.Stdout, "Install path: %s\n", res.InstallPath)
				return nil
			}

			res, err := selfupdate.Install(opts)
			if err != nil {
				return classifyInstallErr(err)
			}
			fmt.Fprintf(os.Stdout, "Installed %s to %s\n", res.TargetVersion, res.InstallPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "latest", "Release version to install (default: latest)")
	cmd.Flags().BoolVar(&check, "check", false, "Report available update without installing")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Directory to install cisco-aid-cli (default: directory of running binary)")

	return cmd
}

func classifyInstallErr(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unsupported platform"),
		strings.Contains(msg, "checksum"),
		strings.Contains(msg, "asset"),
		strings.Contains(msg, "GitHub API returned 404"):
		return clierror.Usage(msg)
	case strings.Contains(msg, "fetch release"),
		strings.Contains(msg, "download"),
		strings.Contains(msg, "HTTP"),
		strings.Contains(msg, "network"),
		strings.Contains(msg, "timeout"):
		return clierror.Network(msg, "Check network connectivity and try again.", err)
	default:
		return clierror.Unexpected(msg, err)
	}
}
