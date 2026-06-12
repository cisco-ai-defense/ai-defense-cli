package aibom

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newSubmitCmd(cli *cmdutil.CLI) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit an existing AIBOM report file to AI Defense",
		Long: `Upload a previously generated AIBOM JSON report to AI Defense.

Examples:
  aid-cisco-cli ai-bom submit --file report.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			raw, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read report file: %w", err)
			}

			var report map[string]interface{}
			if err := json.Unmarshal(raw, &report); err != nil {
				return fmt.Errorf("failed to parse report JSON: %w", err)
			}

			config := api.NewConfig(baseURL)
			client := aibomsdk.NewClient(apiKey, config)

			fmt.Fprintln(os.Stderr, "Submitting AIBOM report...")
			resp, err := client.SubmitReportFile(report)
			if err != nil {
				return err
			}
			output.PrintRawJSON(resp)
			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Path to the AIBOM report JSON file")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
