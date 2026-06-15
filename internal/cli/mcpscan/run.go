package mcpscan

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	mcpscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/mcpscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func newRunCmd(cli *cmdutil.CLI) *cobra.Command {
	var (
		url, name, transport             string
		authType, authHeader, authKey    string
		oauthClientID, oauthClientSecret string
		oauthServerURL                   string
		resultOpts                       *cmdutil.ScanResultOptions
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Scan a remote MCP server for security threats",
		Long: `Start a scan of an MCP server and poll until completion.

Prints raw API JSON by default. Use --envelope for the normalized verdict.
Exit codes: 0 pass, 10 fail, 20 block; failed/cancelled is 5; poll timeout is 4.
Note: the scan-status payload may report status only, in which case a completed
scan reports PASS.

Examples:
  aid-cisco-cli mcp-scan run --url https://mcp.example.com/sse --name "My Server"
  aid-cisco-cli mcp-scan run --url https://mcp.example.com/sse --name "Server" --auth-type api-key --auth-header X-API-Key --auth-key sk_xxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := resultOpts.Validate(); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			authCfg, err := buildAuthConfig(authType, authHeader, authKey, oauthClientID, oauthClientSecret, oauthServerURL)
			if err != nil {
				return err
			}

			transportEnum, err := parseTransport(transport)
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := mcpscansdk.NewClient(apiKey, config)

			req := &mcpscansdk.StartMCPServerScanRequest{
				Name:       name,
				ServerType: mcpscansdk.ServerTypeRemote,
				Remote:     &mcpscansdk.RemoteServerInput{URL: url, ConnectionType: transportEnum},
				AuthConfig: authCfg,
			}

			fmt.Fprintln(os.Stderr, "Starting MCP scan...")
			startResp, err := client.StartScan(req)
			if err != nil {
				return err
			}
			scanID := startResp.ScanID

			retryCount := cmdutil.EnvIntOrDefault("AI_DEFENSE_MCP_RETRY_COUNT", 60)
			waitSecs := cmdutil.EnvIntOrDefault("AI_DEFENSE_MCP_WAIT_TIME_SECS", 10)

			terminalStatuses := map[mcpscansdk.MCPScanStatus]bool{
				mcpscansdk.MCPScanStatusCompleted: true,
				mcpscansdk.MCPScanStatusFailed:    true,
				mcpscansdk.MCPScanStatusCancelled: true,
			}

			var lastStatus mcpscansdk.MCPScanStatus
			timedOut := true
			for i := 0; i < retryCount; i++ {
				time.Sleep(time.Duration(waitSecs) * time.Second)
				statusResp, err := client.GetScanStatus(scanID)
				if err != nil {
					return err
				}
				lastStatus = statusResp.Status
				fmt.Fprintf(os.Stderr, "Scan status: %s\n", lastStatus)
				if terminalStatuses[lastStatus] {
					timedOut = false
					break
				}
			}

			if timedOut {
				return &api.PollTimeoutError{
					ScanID:     scanID,
					BudgetSecs: retryCount * waitSecs,
				}
			}

			raw, err := client.GetScanStatusRaw(scanID)
			if err != nil {
				return err
			}
			return emitScanResult(scanID, string(lastStatus), raw, resultOpts)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "URL of the MCP server to scan")
	_ = cmd.MarkFlagRequired("url")
	cmd.Flags().StringVar(&name, "name", "", "Display name for this scan")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&transport, "transport", "SSE", "Transport type: SSE, STREAMABLE")
	addAuthFlags(cmd, &authType, &authHeader, &authKey, &oauthClientID, &oauthClientSecret, &oauthServerURL)
	resultOpts = cmdutil.AddScanResultFlags(cmd)

	return cmd
}
