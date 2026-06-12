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
