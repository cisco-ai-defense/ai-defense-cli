//go:build !embed_python

package python

import (
	"fmt"
	"os"
)

// Interpreter is the stub for builds without the embedded Python bundle. Set
// AID_CLI_PYTHON to a local interpreter with cisco-aibom installed for dev.
func Interpreter() (string, error) {
	if py := os.Getenv("AID_CLI_PYTHON"); py != "" {
		if _, err := os.Stat(py); err != nil {
			return "", fmt.Errorf("AID_CLI_PYTHON %q: %w", py, err)
		}
		return py, nil
	}

	return "", fmt.Errorf(
		"embedded Python is not available in this build.\n" +
			"Build with the embedded analyzer (make bundle && make build-embed, or " +
			"make build BUILD_MODE=full), or set AID_CLI_PYTHON to a python " +
			"interpreter with cisco-aibom installed for local development")
}
