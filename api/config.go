package api

import "net/http"

const (
	DefaultAPIVersion = "v1"
	APIPrefix         = "api/ai-defense"
	DefaultTimeout    = 30

	// AuthHeader is the tenant-scoped API key header. The CLI targets auth-manager
	// (which fronts scan-manager, mcp-scanning, and ai-bom); its header matcher
	// forwards x-cisco-ai-defense-tenant-api-key (auth_manager/http.go).
	AuthHeader = "X-Cisco-AI-Defense-Tenant-API-Key"
)

var ManagementRegionEndpoints = map[string]string{
	"us-west-2":      "https://us.api.aidefense.security.cisco.com",
	"eu-central-1":   "https://eu.api.aidefense.security.cisco.com",
	"ap-northeast-1": "https://ap.api.aidefense.security.cisco.com",
	"me-central-1":   "https://uae.api.aidefense.security.cisco.com",
}

// Config holds SDK configuration for connecting to the AI Defense API.
type Config struct {
	ManagementBaseURL string
	Timeout           int
	UploadTimeout     int
	APIVersion        string
	// HTTPClient, when set, overrides the default client (used in tests to route
	// requests to an httptest server while keeping a valid *.cisco.com base URL).
	HTTPClient *http.Client
}

func NewConfig(managementBaseURL string) *Config {
	return &Config{
		ManagementBaseURL: managementBaseURL,
		Timeout:           DefaultTimeout,
		UploadTimeout:     0,
		APIVersion:        DefaultAPIVersion,
	}
}
