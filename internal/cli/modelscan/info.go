package modelscan

import (
	"encoding/json"

	"github.com/spf13/cobra"

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

Prints raw API JSON by default. Use --envelope for the normalized verdict and
the same exit-code contract as 'trigger' (0 pass, 10 fail, 20 block; 5 if failed).

Examples:
  cisco-aid-cli model-scan info --scan-id <uuid>
  cisco-aid-cli model-scan info --scan-id <uuid> --limit 50 --output text`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := resultOpts.Validate(); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			config := cli.APIConfig(baseURL)
			client := modelscansdk.NewClient(apiKey, config)

			req := &modelscansdk.GetScanStatusRequest{
				FileLimit:  limit,
				FileOffset: offset,
			}
			raw, err := client.GetScanRaw(scanID, req)
			if err != nil {
				return err
			}
			var resp modelscansdk.GetScanStatusResponse
			if err := json.Unmarshal(raw, &resp); err != nil {
				return clierror.Unexpected("failed to parse scan result", err)
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
