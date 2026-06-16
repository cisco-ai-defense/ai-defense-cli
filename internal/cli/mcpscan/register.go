package mcpscan

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newRegisterCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		url, name, transport, description string
		scanEnabled                       bool
		noScanEnabled                     bool
		repositoryURL, sourceIdentifier   string
		subfolder                         string
		authType, authHeader, authKey     string
		oauthClientID, oauthClientSecret  string
		oauthServerURL                    string
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register an MCP server for ongoing security monitoring",
		Long: `Register an MCP server for ongoing security monitoring.

Examples:
  cisco-aid-cli mcp-scan register --url https://mcp.example.com/sse --name "My Server"
  cisco-aid-cli mcp-scan register --repository-url https://github.com/org/repo --name "Repo Server"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" && repositoryURL == "" {
				return clierror.Usage("at least one of --url or --repository-url must be provided")
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			authCfg, err := buildAuthConfig(authType, authHeader, authKey, oauthClientID, oauthClientSecret, oauthServerURL)
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("scan-enabled") && cmd.Flags().Changed("no-scan-enabled") {
				return clierror.Usage("use either --scan-enabled or --no-scan-enabled, not both")
			}
			if cmd.Flags().Changed("no-scan-enabled") {
				scanEnabled = !noScanEnabled
			}

			transportEnum, err := parseTransport(transport)
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			req := &mcpscansdk.RegisterMCPServerRequest{
				Name:             name,
				URL:              url,
				Description:      description,
				ConnectionType:   transportEnum,
				ScanEnabled:      scanEnabled,
				AuthConfig:       authCfg,
				RepositoryURL:    repositoryURL,
				SourceIdentifier: sourceIdentifier,
				Subfolder:        subfolder,
			}

			fmt.Fprintln(os.Stderr, "Registering MCP server...")
			resp, err := client.RegisterServer(req)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "URL endpoint of the MCP server")
	cmd.Flags().StringVar(&name, "name", "", "Name for the MCP server")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&transport, "transport", "SSE", "Transport type: SSE, STREAMABLE")
	cmd.Flags().StringVar(&description, "description", "", "Description for the MCP server")
	cmd.Flags().BoolVar(&scanEnabled, "scan-enabled", true, "Whether to enable scanning")
	cmd.Flags().BoolVar(&noScanEnabled, "no-scan-enabled", false, "Deprecated alias for --scan-enabled=false")
	cmd.Flags().StringVar(&repositoryURL, "repository-url", "", "Repository URL for source scanning")
	cmd.Flags().StringVar(&sourceIdentifier, "source-identifier", "", "Source identifier")
	cmd.Flags().StringVar(&subfolder, "subfolder", "", "Subfolder within repository")
	addAuthFlags(cmd, &authType, &authHeader, &authKey, &oauthClientID, &oauthClientSecret, &oauthServerURL)

	return cmd
}
