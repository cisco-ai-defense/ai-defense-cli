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

import "fmt"

const (
	MCPServers = "mcp/servers"
	MCPScan    = "scan"
)

func ScanStart() string {
	return fmt.Sprintf("%s/%s", MCPServers, MCPScan)
}

func ScanStatus(scanID string) string {
	return fmt.Sprintf("%s/%s/%s", MCPServers, MCPScan, scanID)
}

func ServersRegister() string {
	return MCPServers
}

func ServerDelete(serverID string) string {
	return fmt.Sprintf("%s/%s", MCPServers, serverID)
}

func ServerGet(serverID string) string {
	return fmt.Sprintf("%s/%s", MCPServers, serverID)
}

func ServersList() string {
	return MCPServers
}

func ServerUpdateAuthConfig(serverID string) string {
	return fmt.Sprintf("%s/%s/auth", MCPServers, serverID)
}

func ServerCapabilities(serverID string) string {
	return fmt.Sprintf("%s/%s/capabilities", MCPServers, serverID)
}

func ServerScan(serverID string) string {
	return fmt.Sprintf("%s/%s/%s", MCPServers, serverID, MCPScan)
}

func ServerScanReport(serverID string) string {
	return fmt.Sprintf("%s/%s/%s/report", MCPServers, serverID, MCPScan)
}

func ServerScanSummary(serverID string) string {
	return fmt.Sprintf("%s/%s/%s/summary", MCPServers, serverID, MCPScan)
}

func ServerThreats(serverID string) string {
	return fmt.Sprintf("%s/%s/threats", MCPServers, serverID)
}

func ServersValidate() string {
	return fmt.Sprintf("%s:validate", MCPServers)
}
