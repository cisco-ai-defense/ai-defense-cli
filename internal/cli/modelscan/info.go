package modelscan

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func newInfoCmd(cli *cmdutil.CLI) *cobra.Command {
	var scanID string
	var limit, offset int
	var resultOpts *cmdutil.ScanResultOptions

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get detailed information about a specific scan",
		Long: `Retrieve scan status, scanned files, detected threats, and severity breakdown.

Prints a normalized verdict envelope and applies the same exit-code contract as
'trigger' for completed scans (0 pass, 10 fail, 20 block; 5 if the scan failed).

Examples:
  aid-cisco-cli model-scan info --scan-id <uuid>
  aid-cisco-cli model-scan info --scan-id <uuid> --limit 50 --output text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := resultOpts.Validate(); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := api.NewConfig(baseURL)
			client := modelscansdk.NewClient(apiKey, config)

			resp, err := client.GetScan(scanID, &modelscansdk.GetScanStatusRequest{
				FileLimit:  limit,
				FileOffset: offset,
			})
			if err != nil {
				return err
			}

			raw, err := json.Marshal(resp)
			if err != nil {
				return clierror.Unexpected("failed to render scan result", err)
			}
			return emitScanResult(resp.ScanStatusInfo, raw, resultOpts)
		},
	}

	cmd.Flags().StringVar(&scanID, "scan-id", "", "UUID of the scan")
	_ = cmd.MarkFlagRequired("scan-id")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of file results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of file results to skip")
	resultOpts = cmdutil.AddScanResultFlags(cmd)

	return cmd
}
