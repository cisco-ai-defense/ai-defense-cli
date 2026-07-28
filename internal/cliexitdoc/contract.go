// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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
