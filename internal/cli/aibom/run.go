package aibom

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	aibomsdk "github.com/cisco-ai-defense/ai-defense-cli/api/aibom"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/output"
)

func newRunCmd(cli *cmdutil.CLI) *cobra.Command {
	var sources []string
	var outputFile string
	var customCatalog string
	var llm llmFlagVars

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Generate and submit an AIBOM in one step",
		Long: `Analyze local source paths, generate an AIBOM, and submit it to AI Defense.

An LLM classifies scanner candidates, so --llm-model (or AIBOM_LLM_MODEL) and
credentials for that model are required.

Examples:
  cisco-aid-cli ai-bom run --source ./my-app --llm-model gpt-5.4 --llm-api-key sk-...
  cisco-aid-cli ai-bom run --source ./app --source ./lib --output-file report.json --llm-model gpt-5.4 --llm-api-key sk-...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved := llm.resolve(cli)
			if err := ValidateLLM(resolved); err != nil {
				return err
			}

			apiKey, baseURL, err := cli.ResolveAuth()
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Generating AIBOM...")
			result, err := generateReport(sources, customCatalog, resolved)
			if err != nil {
				return err
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, result, 0o644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
			}

			var report map[string]interface{}
			if err := json.Unmarshal(result, &report); err != nil {
				return fmt.Errorf("failed to parse generated AIBOM: %w", err)
			}

			config := cli.APIConfig(baseURL)
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

	cmd.Flags().StringArrayVar(&sources, "source", nil, "Local path to analyze (repeatable)")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Path to save the generated AIBOM JSON")
	cmd.Flags().StringVar(&customCatalog, "custom-catalog", "", "Path to a custom DuckDB catalog for component matching")
	addLLMFlags(cmd, &llm)
	_ = cmd.MarkFlagRequired("source")

	return cmd
}
