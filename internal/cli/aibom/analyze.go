package aibom

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
	embeddedpython "github.com/cisco-ai-defense/ai-defense-cli/internal/embedded/python"
)

// generateReport runs `aibom analyze` via the bundled interpreter and returns
// the AIBOM JSON report. aibom's own stdout/stderr is forwarded to our stderr
// so the returned bytes contain only the report.
func generateReport(sources []string, customCatalog string, llm LLMConfig) ([]byte, error) {
	py, err := embeddedpython.Interpreter()
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "aibom-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	args := []string{"-m", "aibom", "analyze"}
	args = append(args, sources...)
	args = append(args, "--output-format", "json", "--output-file", tmpPath)
	if customCatalog != "" {
		args = append(args, "--custom-catalog", customCatalog)
	}
	args = append(args, llm.cliArgs()...)

	cmd := exec.Command(py, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if llm.APIKey != "" {
		cmd.Env = append(cmd.Env, "AIBOM_LLM_API_KEY="+llm.APIKey)
	}

	if err := cmd.Run(); err != nil {
		return nil, clierror.ScanFailed(
			"aibom analyze failed",
			"Check LLM credentials and model settings, then retry.",
			err,
		)
	}

	return os.ReadFile(tmpPath)
}
