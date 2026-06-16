// Package cliexitdoc holds the machine-readable catalog of CLI exit-code
// scenarios and generates docs/exit-codes-handbook.md. When you change command
// error behavior, update registry.go and run: make gen-exit-docs
package cliexitdoc

import (
	"fmt"
	"strings"

	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// Scenario is one documented exit outcome for a command or global rule.
type Scenario struct {
	When  string
	Code  int // clierror.Code* constant; use -1 for display-only multi-code rows
	Label string // optional override when Code is -1 (e.g. "10 / 20")
	Notes string
	Retry string // "Yes", "No", "Maybe", "—", or empty (omit column)
}

// CommandGroup is a handbook section with a markdown table.
type CommandGroup struct {
	Title   string // heading without ### (e.g. "`install`")
	Intro   string // optional paragraph before the table
	Columns []string
	Rows    []Scenario
}

// MultiRow is a table row with extra leading columns (e.g. subcommand).
type MultiRow struct {
	Cols []string // len must match len(Columns)
}

// MultiCommandGroup uses heterogeneous row strings (config profile style).
type MultiCommandGroup struct {
	Title   string
	Columns []string
	Rows    []MultiRow
}

// CodeLabel formats an exit code for markdown tables.
func CodeLabel(code int, override string) string {
	if override != "" {
		return override
	}
	if code < 0 {
		return ""
	}
	return fmt.Sprintf("%d", code)
}

// ValidContractCodes are the public exit codes consumers may rely on.
var ValidContractCodes = map[int]bool{
	clierror.CodeOK:          true,
	clierror.CodeUnexpected:  true,
	clierror.CodeUsage:       true,
	clierror.CodeAuth:        true,
	clierror.CodeNetwork:     true,
	clierror.CodeScanFailed:  true,
	clierror.CodeFail:        true,
	clierror.CodeBlock:       true,
}

// ValidateRegistry returns an error if any scenario uses an undocumented code.
func ValidateRegistry() error {
	check := func(title string, rows []Scenario) error {
		for _, r := range rows {
			if r.Label != "" {
				continue
			}
			if !ValidContractCodes[r.Code] {
				return fmt.Errorf("%s: invalid code %d for %q", title, r.Code, r.When)
			}
		}
		return nil
	}
	for _, g := range GlobalScenarios() {
		if err := check(g.Title, g.Rows); err != nil {
			return err
		}
	}
	for _, g := range CommandScenarios() {
		if err := check(g.Title, g.Rows); err != nil {
			return err
		}
	}
	return nil
}

func renderTable(columns []string, rows [][]string) string {
	var b strings.Builder
	b.WriteString("| ")
	b.WriteString(strings.Join(columns, " | "))
	b.WriteString(" |\n|")
	for range columns {
		b.WriteString("----------|")
	}
	b.WriteString("\n")
	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}

func scenarioRows(columns []string, scenarios []Scenario) [][]string {
	out := make([][]string, 0, len(scenarios))
	for _, s := range scenarios {
		code := CodeLabel(s.Code, s.Label)
		switch len(columns) {
		case 4:
			out = append(out, []string{s.When, code, s.Notes, s.Retry})
		case 3:
			out = append(out, []string{s.When, code, s.Notes})
		default:
			out = append(out, []string{s.When, code})
		}
	}
	return out
}
