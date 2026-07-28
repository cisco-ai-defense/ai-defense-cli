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

package mcpscan

import (
	"encoding/json"
	"time"
)

// TransportType represents the transport protocol used by the MCP server.
type TransportType string

const (
	TransportSSE        TransportType = "SSE"
	TransportStreamable TransportType = "STREAMABLE"
)

// AuthType represents the authentication type for MCP servers.
type AuthType string

const (
	AuthTypeNoAuth AuthType = "NO_AUTH"
	AuthTypeAPIKey AuthType = "API_KEY"
	AuthTypeOAuth  AuthType = "OAUTH"
)

// MCPScanStatus represents the status of an MCP scan.
type MCPScanStatus string

const (
	MCPScanStatusPending    MCPScanStatus = "PENDING"
	MCPScanStatusInProgress MCPScanStatus = "IN_PROGRESS"
	MCPScanStatusCompleted  MCPScanStatus = "COMPLETED"
	MCPScanStatusFailed     MCPScanStatus = "FAILED"
	MCPScanStatusCancelled  MCPScanStatus = "CANCELLED"
)

// OnboardingStatus represents the onboarding state of a server.
type OnboardingStatus string

const (
	OnboardingCompleted  OnboardingStatus = "COMPLETED"
	OnboardingInProgress OnboardingStatus = "INPROGRESS"
	OnboardingFailed     OnboardingStatus = "FAILED"
)

// ServerType represents the type of MCP server.
type ServerType string

const (
	ServerTypeRemote ServerType = "REMOTE"
	ServerTypeLocal  ServerType = "LOCAL"
)

// CapabilityType represents the type of MCP capability.
type CapabilityType string

const (
	CapabilityTool     CapabilityType = "TOOL"
	CapabilityPrompt   CapabilityType = "PROMPT"
	CapabilityResource CapabilityType = "RESOURCE"
)

// ApiKeyConfig holds API key auth configuration.
type ApiKeyConfig struct {
	HeaderName string `json:"header_name"`
	APIKey     string `json:"api_key"`
}

// OAuthConfig holds OAuth auth configuration.
type OAuthConfig struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret,omitempty"`
	AuthServerURL string `json:"auth_server_url"`
}

// AuthConfig holds authentication configuration for an MCP server.
type AuthConfig struct {
	AuthType AuthType      `json:"auth_type"`
	APIKey   *ApiKeyConfig `json:"api_key,omitempty"`
	OAuth    *OAuthConfig  `json:"oauth,omitempty"`
}

// RegisterMCPServerRequest is sent to register a new MCP server.
type RegisterMCPServerRequest struct {
	Name             string        `json:"name"`
	URL              string        `json:"url,omitempty"`
	Description      string        `json:"description,omitempty"`
	ConnectionType   TransportType `json:"connection_type"`
	ScanEnabled      bool          `json:"scan_enabled"`
	AuthConfig       AuthConfig    `json:"auth_config"`
	RepositoryURL    string        `json:"repository_url,omitempty"`
	SourceIdentifier string        `json:"source_identifier,omitempty"`
	Subfolder        string        `json:"subfolder,omitempty"`
}

// RemoteServerInput holds remote server connection details for scanning.
type RemoteServerInput struct {
	URL            string        `json:"url"`
	ConnectionType TransportType `json:"connection_type"`
}

// StartMCPServerScanRequest is sent to start a scan on an unregistered MCP server.
type StartMCPServerScanRequest struct {
	Name       string             `json:"name"`
	ServerType ServerType         `json:"server_type"`
	Remote     *RemoteServerInput `json:"remote,omitempty"`
	AuthConfig AuthConfig         `json:"auth_config"`
}

// StartMCPServerScanResponse is returned when a scan is started.
type StartMCPServerScanResponse struct {
	ScanID string `json:"scan_id"`
}

// UnmarshalJSON accepts both snake_case and lowerCamel response fields for scan IDs.
func (s *StartMCPServerScanResponse) UnmarshalJSON(data []byte) error {
	var decoded struct {
		ScanID      string `json:"scan_id"`
		ScanIDCamel string `json:"scanId"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	if decoded.ScanID != "" {
		s.ScanID = decoded.ScanID
		return nil
	}

	s.ScanID = decoded.ScanIDCamel
	return nil
}

// Paging holds pagination information.
type Paging struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// MCPServerSummary holds summary info for a registered MCP server.
type MCPServerSummary struct {
	ServerID         string           `json:"server_id"`
	Name             string           `json:"name"`
	URL              string           `json:"url,omitempty"`
	Description      string           `json:"description,omitempty"`
	ConnectionType   TransportType    `json:"connection_type"`
	OnboardingStatus OnboardingStatus `json:"onboarding_status"`
	ScanEnabled      bool             `json:"scan_enabled"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// ListMCPServersResponse contains a paginated list of MCP servers.
type ListMCPServersResponse struct {
	Items  []MCPServerSummary `json:"items"`
	Paging Paging             `json:"paging"`
}

// MCPScanStatusResponse holds the status of an MCP scan.
type MCPScanStatusResponse struct {
	ScanID string        `json:"scan_id"`
	Status MCPScanStatus `json:"status"`
}

// Capability holds a single MCP server capability entry.
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SeverityLevel represents a threat severity level.
type SeverityLevel string

const (
	SeverityUnknown SeverityLevel = "SEVERITY_UNKNOWN"
	SeveritySafe    SeverityLevel = "SAFE"
	SeverityLow     SeverityLevel = "LOW"
	SeverityMedium  SeverityLevel = "MEDIUM"
	SeverityHigh    SeverityLevel = "HIGH"
)

// ThreatSeverityLevel represents a threat severity level for filtering.
type ThreatSeverityLevel string

const (
	ThreatSeverityUnknown  ThreatSeverityLevel = "THREAT_SEVERITY_UNKNOWN"
	ThreatSeverityLow      ThreatSeverityLevel = "LOW"
	ThreatSeverityMedium   ThreatSeverityLevel = "MEDIUM"
	ThreatSeverityHigh     ThreatSeverityLevel = "HIGH"
	ThreatSeverityCritical ThreatSeverityLevel = "CRITICAL"
)

// ServersSortBy represents sort criteria for MCP server listing.
type ServersSortBy string

const (
	SortUnspecified    ServersSortBy = "SORT_UNSPECIFIED"
	SortThreatSeverity ServersSortBy = "THREATS_SEVERITY"
	SortScanDate       ServersSortBy = "SCAN_DATE"
)

// SortOrder represents sort direction.
type SortOrder string

const (
	SortOrderUnspecified SortOrder = "SortOrder_Unspecified"
	SortOrderAsc         SortOrder = "asc"
	SortOrderDesc        SortOrder = "desc"
)

// FilterOptions holds filter options for scan report requests.
type FilterOptions struct {
	CapabilityType *CapabilityType       `json:"capabilityType,omitempty"`
	ThreatSeverity []ThreatSeverityLevel `json:"threatSeverity,omitempty"`
}

// GetMCPServerScanReportRequest is the POST body for scan report retrieval.
type GetMCPServerScanReportRequest struct {
	ServerID      string         `json:"serverId,omitempty"`
	Offset        *int           `json:"offset,omitempty"`
	FilterOptions *FilterOptions `json:"filterOptions,omitempty"`
}

// ValidateMCPServersRequest is sent to validate MCP server URLs.
type ValidateMCPServersRequest struct {
	URLs          []string       `json:"urls"`
	TransportType *TransportType `json:"transportType,omitempty"`
	AuthConfig    *AuthConfig    `json:"authConfig,omitempty"`
}

// ErrorInfo holds error information from server validation.
type ErrorInfo struct {
	Message         string   `json:"message"`
	ErrorMessage    string   `json:"error_message,omitempty"`
	RemediationTips []string `json:"remediation_tips,omitempty"`
}

// InvalidURLInfo describes a URL that failed validation.
type InvalidURLInfo struct {
	URL       string    `json:"url"`
	ErrorInfo ErrorInfo `json:"errorInfo"`
}

// ValidateMCPServersResponse holds validation results.
type ValidateMCPServersResponse struct {
	ValidURLs   []string         `json:"validUrls"`
	InvalidURLs []InvalidURLInfo `json:"invalidUrls"`
}

// CapabilitiesResponse contains a paginated list of capabilities.
type CapabilitiesResponse struct {
	Items  []Capability `json:"items"`
	Paging Paging       `json:"paging"`
}
