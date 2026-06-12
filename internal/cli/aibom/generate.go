package aibom

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/cmdutil"
)

func newGenerateCmd(cli *cmdutil.CLI) *cobra.Command {
	var sources []string
	var outputFile string
	var customCatalog string
	var llm llmOptions

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate an AIBOM locally from source paths",
		Long: `Analyze local source paths and output the AIBOM as JSON. The report is
generated locally; use "ai-bom run" to generate and submit it in one step.

An LLM classifies scanner candidates, so --llm-model (or AIBOM_LLM_MODEL) and
credentials for that model are required.

Examples:
  aid-cisco-cli ai-bom generate --source ./my-app --llm-model gpt-5.4 --llm-api-key sk-...
  aid-cisco-cli ai-bom generate --source ./app --source ./lib --output-file report.json --llm-model gpt-5.4 --llm-api-key sk-...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := llm.validate(); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Generating AIBOM...")
			result, err := generateReport(sources, customCatalog, llm)
			if err != nil {
				return fmt.Errorf("AIBOM generation failed: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, result, 0o644); err != nil {
					return fmt.Errorf("write output file: %w", err)
				}
			}

			fmt.Println(string(result))
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
