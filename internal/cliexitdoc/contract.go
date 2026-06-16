package cliexitdoc

import (
	"strconv"
	"strings"
)

// ScenarioRow is one documented exit outcome with a single integer code.
type ScenarioRow struct {
	Command string // subcommand label from multi tables (empty for simple groups)
	When    string
	Code    int
}

// SingleExitCode parses a handbook code cell when it is exactly one integer.
// Returns false for composite rows ("10 / 20", "3 / 4 / …", "*see onboard*").
func SingleExitCode(cell string) (int, bool) {
	cell = strings.TrimSpace(cell)
	if cell == "" || strings.Contains(cell, "/") || strings.Contains(cell, "*") {
		return 0, false
	}
	n, err := strconv.Atoi(cell)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ConfigValidateContract returns validate rows with a single documented code.
func ConfigValidateContract() []ScenarioRow {
	return scenarioRowsFromGroup(configValidateScenarios())
}

// ModelScanContract returns model-scan rows with a single documented code.
func ModelScanContract() []ScenarioRow {
	return multiRowContract(ModelScanRows())
}

// McpScanContract returns mcp-scan rows with a single documented code.
func McpScanContract() []ScenarioRow {
	return multiRowContract(McpScanRows())
}

// AibomContract returns ai-bom rows with a single documented code.
func AibomContract() []ScenarioRow {
	return multiRowContract(AibomRows())
}

func multiRowContract(rows []MultiRow) []ScenarioRow {
	var out []ScenarioRow
	for _, r := range rows {
		if len(r.Cols) < 3 {
			continue
		}
		code, ok := SingleExitCode(r.Cols[2])
		if !ok {
			continue
		}
		out = append(out, ScenarioRow{
			Command: strings.Trim(r.Cols[0], "`"),
			When:    r.Cols[1],
			Code:    code,
		})
	}
	return out
}

func scenarioRowsFromGroup(g CommandGroup) []ScenarioRow {
	var out []ScenarioRow
	for _, s := range g.Rows {
		code, ok := SingleExitCode(CodeLabel(s.Code, s.Label))
		if !ok {
			continue
		}
		out = append(out, ScenarioRow{When: s.When, Code: code})
	}
	return out
}

// ContractCode returns the documented code for the first row whose When contains substr.
func ContractCode(rows []ScenarioRow, whenContains string) (int, bool) {
	for _, r := range rows {
		if strings.Contains(r.When, whenContains) {
			return r.Code, true
		}
	}
	return 0, false
}
