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

package verdict

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	modelscansdk "github.com/cisco-ai-defense/ai-defense-cli/api/modelscan"
	"github.com/cisco-ai-defense/ai-defense-cli/internal/clierror"
)

// EnvelopeVersion is the schema version of the verdict JSON envelope.
const EnvelopeVersion = "1"

// Decision is the policy outcome of a completed scan.
type Decision string

const (
	DecisionPass  Decision = "PASS"
	DecisionFail  Decision = "FAIL"
	DecisionBlock Decision = "BLOCK"
)

// Threat is a single normalized finding in the verdict envelope.
type Threat struct {
	Target     string `json:"target,omitempty"`
	Type       string `json:"type,omitempty"`
	Severity   string `json:"severity"`
	Title      string `json:"title,omitempty"`
	Detail     string `json:"detail,omitempty"`
	severityID Severity
}

// Result is the unified, machine-readable verdict envelope emitted for both
// model and MCP scans. It is the stable CI/CD contract: automation reads
// `decision`/`highest_severity`/`counts_by_severity` and branches on the process
// exit code rather than parsing scan-type-specific payloads.
type Result struct {
	EnvelopeVersion  string         `json:"envelope_version"`
	ScanType         string         `json:"scan_type"`
	ScanID           string         `json:"scan_id,omitempty"`
	Status           string         `json:"status,omitempty"`
	Decision         Decision       `json:"decision"`
	ExitCode         int            `json:"exit_code"`
	HighestSeverity  string         `json:"highest_severity"`
	ThreatCount      int            `json:"threat_count"`
	CountsBySeverity map[string]int `json:"counts_by_severity"`
	Threats          []Threat       `json:"threats,omitempty"`

	highest Severity
}

// FromModelScan builds a Result from a completed model-scan status payload.
func FromModelScan(info modelscansdk.ScanStatusInfo) Result {
	r := Result{
		EnvelopeVersion: EnvelopeVersion,
		ScanType:        "model-scan",
		ScanID:   info.ScanID,
		Status:   string(info.Status),
	}
	for _, file := range info.AnalysisResults.Items {
		for _, tech := range file.Threats.Items {
			for _, sub := range tech.Items {
				for _, item := range sub.Items {
					r.add(Threat{
						Target:   file.Name,
						Type:     item.ThreatType,
						Severity: string(item.Severity),
						Title:    sub.SubTechniqueName,
						Detail:   item.Details,
					})
				}
			}
		}
	}
	return r
}

// FromMCPScan builds a Result from an MCP scan. The unregistered run/info flow
// surfaces a status plus (best-effort) a set of threat severities extracted by
// the caller; pass the raw severity strings in threatSeverities.
func FromMCPScan(scanID, status string, threatSeverities []string) Result {
	r := Result{
		EnvelopeVersion: EnvelopeVersion,
		ScanType:        "mcp-scan",
		ScanID:   scanID,
		Status:   status,
	}
	for _, sev := range threatSeverities {
		r.add(Threat{Severity: sev})
	}
	return r
}

func (r *Result) add(t Threat) {
	t.severityID = ParseSeverity(t.Severity)
	if t.Severity == "" {
		t.Severity = t.severityID.String()
	}
	r.Threats = append(r.Threats, t)
	if t.severityID > r.highest {
		r.highest = t.severityID
	}
}

// Decide applies the gating thresholds, finalizes the envelope's decision/exit
// fields, and returns the exit code (0 pass, 10 fail, 20 block).
func (r *Result) Decide(failOn, blockOn Threshold) int {
	r.ThreatCount = len(r.Threats)
	r.CountsBySeverity = map[string]int{}
	for _, t := range r.Threats {
		r.CountsBySeverity[t.severityID.String()]++
	}
	r.HighestSeverity = r.highest.String()

	switch {
	case r.ThreatCount > 0 && blockOn.triggers(r.highest):
		r.Decision = DecisionBlock
		r.ExitCode = clierror.CodeBlock
	case r.ThreatCount > 0 && failOn.triggers(r.highest):
		r.Decision = DecisionFail
		r.ExitCode = clierror.CodeFail
	default:
		r.Decision = DecisionPass
		r.ExitCode = clierror.CodeOK
	}
	return r.ExitCode
}

// GateError returns a clierror.Error for a non-pass decision, or nil for pass.
// The returned error sets the process exit code; the envelope itself is printed
// separately to stdout.
func (r *Result) GateError() error {
	if r.ExitCode == clierror.CodeOK {
		return nil
	}
	return clierror.Gate(r.ExitCode, fmt.Sprintf(
		"%s: scan decision %s (highest severity %s, %d finding(s))",
		r.ScanType, r.Decision, r.HighestSeverity, r.ThreatCount))
}

// Render returns the envelope in the requested format ("json" or "text").
func (r *Result) Render(format string) string {
	if strings.EqualFold(format, "text") {
		return r.renderText()
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\":\"failed to render result: %v\"}", err)
	}
	return string(data)
}

func (r *Result) renderText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Scan type:        %s\n", r.ScanType)
	if r.ScanID != "" {
		fmt.Fprintf(&b, "Scan ID:          %s\n", r.ScanID)
	}
	if r.Status != "" {
		fmt.Fprintf(&b, "Status:           %s\n", r.Status)
	}
	fmt.Fprintf(&b, "Decision:         %s\n", r.Decision)
	fmt.Fprintf(&b, "Highest severity: %s\n", r.HighestSeverity)
	fmt.Fprintf(&b, "Threats found:    %d\n", r.ThreatCount)
	if len(r.CountsBySeverity) > 0 {
		keys := make([]string, 0, len(r.CountsBySeverity))
		for k := range r.CountsBySeverity {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			if r.CountsBySeverity[k] > 0 {
				parts = append(parts, fmt.Sprintf("%s=%d", k, r.CountsBySeverity[k]))
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(&b, "By severity:      %s\n", strings.Join(parts, " "))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// MCPSeverityFields are the JSON keys we look for when best-effort extracting
// threat severities from an MCP scan payload whose schema may vary.
var MCPSeverityFields = []string{"severity", "threat_severity", "threatSeverity", "max_severity", "maxSeverity"}

// ExtractMCPSeverities walks an arbitrary MCP scan JSON payload and collects any
// severity-like string values it finds. This is intentionally schema-tolerant:
// the unregistered scan-status payload shape is not guaranteed, so we gather
// whatever severity signals are present rather than binding to one layout.
func ExtractMCPSeverities(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	var out []string
	walk(root, &out)
	return out
}

func walk(node interface{}, out *[]string) {
	switch v := node.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if isSeverityKey(key) {
				if s, ok := val.(string); ok && ParseSeverity(s) > SevNone {
					*out = append(*out, s)
				}
			}
			walk(val, out)
		}
	case []interface{}:
		for _, item := range v {
			walk(item, out)
		}
	}
}

func isSeverityKey(key string) bool {
	for _, k := range MCPSeverityFields {
		if strings.EqualFold(key, k) {
			return true
		}
	}
	return false
}
