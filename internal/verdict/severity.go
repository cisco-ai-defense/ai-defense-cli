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

// Package verdict turns raw scan results into a single, normalized, machine
// readable envelope and a policy decision (pass/fail/block) that the CLI maps to
// an exit code. The same envelope shape is produced for both model scans and MCP
// scans so that automation never needs scan-type-specific parsing.
package verdict

import (
	"fmt"
	"strings"
)

// Severity is a normalized, ordered threat severity. The numeric ordering is the
// contract used by threshold comparisons; the AI Defense APIs use several
// spellings ("LOW", "SEVERITY_UNKNOWN", "NONE_SEVERITY", ...) which ParseSeverity
// folds into these values.
type Severity int

const (
	// SevNone covers safe/unknown/no-threat severities.
	SevNone Severity = iota
	SevLow
	SevMedium
	SevHigh
	SevCritical
)

// String returns the canonical upper-case name for a severity.
func (s Severity) String() string {
	switch s {
	case SevLow:
		return "LOW"
	case SevMedium:
		return "MEDIUM"
	case SevHigh:
		return "HIGH"
	case SevCritical:
		return "CRITICAL"
	default:
		return "NONE"
	}
}

// ParseSeverity normalizes an API severity string into a Severity. Unknown,
// safe, and empty values collapse to SevNone.
func ParseSeverity(raw string) Severity {
	s := strings.ToUpper(strings.TrimSpace(raw))
	// Strip common enum prefixes/suffixes ("THREAT_SEVERITY_HIGH", "NONE_SEVERITY").
	s = strings.TrimPrefix(s, "THREAT_SEVERITY_")
	s = strings.TrimSuffix(s, "_SEVERITY")
	switch s {
	case "LOW":
		return SevLow
	case "MEDIUM", "MED":
		return SevMedium
	case "HIGH":
		return SevHigh
	case "CRITICAL", "CRIT":
		return SevCritical
	default:
		// "NONE", "SAFE", "UNKNOWN", "SEVERITY_UNKNOWN" (-> "UNKNOWN"), "" all map here.
		return SevNone
	}
}

// Threshold is a gating threshold parsed from --fail-on / --block-on. A disabled
// threshold (from "none") never gates.
type Threshold struct {
	Disabled bool
	Sev      Severity
}

// ParseThreshold parses a user-supplied threshold value. The sentinel values
// "none", "off", and "never" disable gating; any other value must be a valid
// severity name.
func ParseThreshold(raw string) (Threshold, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	switch s {
	case "NONE", "OFF", "NEVER", "":
		return Threshold{Disabled: true}, nil
	case "LOW":
		return Threshold{Sev: SevLow}, nil
	case "MEDIUM", "MED":
		return Threshold{Sev: SevMedium}, nil
	case "HIGH":
		return Threshold{Sev: SevHigh}, nil
	case "CRITICAL", "CRIT":
		return Threshold{Sev: SevCritical}, nil
	default:
		return Threshold{}, fmt.Errorf("invalid severity %q (want one of: none, low, medium, high, critical)", raw)
	}
}

// triggers reports whether a finding at severity sev meets this threshold.
func (t Threshold) triggers(sev Severity) bool {
	if t.Disabled {
		return false
	}
	return sev >= t.Sev
}
