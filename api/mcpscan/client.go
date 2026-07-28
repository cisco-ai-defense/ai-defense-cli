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
	"fmt"
	"net/url"
	"strconv"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

// Client provides access to the AI Defense MCP Scan API.
type Client struct {
	base *api.BaseClient
}

// NewClient creates a new MCP scan client.
func NewClient(apiKey string, config *api.Config) *Client {
	return &Client{base: api.NewBaseClient(apiKey, config)}
}

// RegisterServer registers a new MCP server.
func (c *Client) RegisterServer(req *RegisterMCPServerRequest) (json.RawMessage, error) {
	return c.base.Post(ServersRegister(), req)
}

// DeleteServer deletes a registered MCP server.
func (c *Client) DeleteServer(serverID string) error {
	_, err := c.base.Delete(ServerDelete(serverID))
	return err
}

// StartScan starts a scan on an unregistered MCP server.
func (c *Client) StartScan(req *StartMCPServerScanRequest) (*StartMCPServerScanResponse, error) {
	resp, err := c.base.Post(ScanStart(), req)
	if err != nil {
		return nil, err
	}
	var result StartMCPServerScanResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse start scan response: %w", err)
	}
	return &result, nil
}

// GetScanStatus retrieves the status of an MCP scan.
func (c *Client) GetScanStatus(scanID string) (*MCPScanStatusResponse, error) {
	resp, err := c.base.Get(ScanStatus(scanID), nil)
	if err != nil {
		return nil, err
	}
	var result MCPScanStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scan status response: %w", err)
	}
	return &result, nil
}

// GetScanStatusRaw retrieves the raw scan status JSON.
func (c *Client) GetScanStatusRaw(scanID string) (json.RawMessage, error) {
	return c.base.Get(ScanStatus(scanID), nil)
}

// ListServersParams holds all filter options for listing MCP servers.
type ListServersParams struct {
	Limit            int
	Offset           int
	NameSubstr       string
	OnboardingStatus []OnboardingStatus
	TransportType    []TransportType
	Severity         []SeverityLevel
	RegistryID       string
	SortBy           ServersSortBy
	SortOrder        SortOrder
}

// ListServers lists registered MCP servers with filters.
func (c *Client) ListServers(p ListServersParams) (json.RawMessage, error) {
	params := url.Values{}
	if p.Limit > 0 {
		params.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		params.Set("offset", strconv.Itoa(p.Offset))
	}
	if p.NameSubstr != "" {
		params.Set("server_name_substr", p.NameSubstr)
	}
	for _, s := range p.OnboardingStatus {
		params.Add("onboarding_status", string(s))
	}
	for _, t := range p.TransportType {
		params.Add("transport_type", string(t))
	}
	for _, s := range p.Severity {
		params.Add("severity", string(s))
	}
	if p.RegistryID != "" {
		params.Set("registry_id", p.RegistryID)
	}
	if p.SortBy != "" {
		params.Set("sort_by", string(p.SortBy))
	}
	if p.SortOrder != "" {
		params.Set("sort_order", string(p.SortOrder))
	}
	return c.base.Get(ServersList(), params)
}

// GetServer retrieves details of a registered MCP server.
func (c *Client) GetServer(serverID string) (json.RawMessage, error) {
	return c.base.Get(ServerGet(serverID), nil)
}

// TriggerServerScan triggers an on-demand scan for a registered server.
func (c *Client) TriggerServerScan(serverID string) error {
	_, err := c.base.Post(ServerScan(serverID), nil)
	return err
}

// GetServerCapabilities retrieves capabilities for a registered server.
func (c *Client) GetServerCapabilities(serverID string, capType CapabilityType, capName string, limit, offset int) (json.RawMessage, error) {
	params := url.Values{}
	params.Set("capability_type", string(capType))
	if capName != "" {
		params.Set("capability_name", capName)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	return c.base.Get(ServerCapabilities(serverID), params)
}

// GetServerScanReport retrieves the scan report for a server (POST with filters).
func (c *Client) GetServerScanReport(serverID string, req *GetMCPServerScanReportRequest) (json.RawMessage, error) {
	if req != nil {
		req.ServerID = serverID
	} else {
		req = &GetMCPServerScanReportRequest{ServerID: serverID}
	}
	return c.base.Post(ServerScanReport(serverID), req)
}

// GetServerScanSummary retrieves the scan summary for a server.
func (c *Client) GetServerScanSummary(serverID string) (json.RawMessage, error) {
	return c.base.Get(ServerScanSummary(serverID), nil)
}

// GetServerThreats retrieves threats detected on a server with optional filters.
func (c *Client) GetServerThreats(serverID string, capType *CapabilityType, threatSeverity []string, limit, offset int) (json.RawMessage, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if capType != nil {
		params.Set("capability_type", string(*capType))
	}
	for _, s := range threatSeverity {
		params.Add("threat_severity", s)
	}
	return c.base.Get(ServerThreats(serverID), params)
}

// ValidateServers validates connectivity and auth for MCP server URLs.
func (c *Client) ValidateServers(req *ValidateMCPServersRequest) (*ValidateMCPServersResponse, error) {
	resp, err := c.base.Post(ServersValidate(), req)
	if err != nil {
		return nil, err
	}
	var result ValidateMCPServersResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse validate servers response: %w", err)
	}
	return &result, nil
}

// UpdateAuthConfig updates the auth configuration for a server.
func (c *Client) UpdateAuthConfig(serverID string, authConfig *AuthConfig) error {
	_, err := c.base.Put(ServerUpdateAuthConfig(serverID), authConfig)
	return err
}
