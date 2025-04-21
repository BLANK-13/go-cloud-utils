package cloudflare

import (
	"net/http"
	"time"
)

// CloudflareConfig holds the credentials and endpoint information for Cloudflare services
type CloudflareConfig struct {
	// Cloudflare API token
	APIToken string

	// Account ID for Cloudflare
	AccountID string

	// Base URL for Cloudflare API
	BaseURL string

	// Timeout for HTTP requests
	Timeout time.Duration
}

// NewConfig creates a CloudflareConfig with sensible defaults
func NewConfig(apiToken, accountID string) *CloudflareConfig {
	return &CloudflareConfig{
		APIToken:  apiToken,
		AccountID: accountID,
		BaseURL:   "https://api.cloudflare.com/client/v4",
		Timeout:   30 * time.Second,
	}
}

// createHTTPClient creates an HTTP client with the config's timeout
func createHTTPClient(config *CloudflareConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
	}
}
