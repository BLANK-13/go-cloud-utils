package cloudflare

import (
	"os"
)

// Storage is a convenient wrapper that provides access to all Cloudflare storage services
type Storage struct {
	// D1 SQL database client
	D1 *D1Client

	// R2 object storage client
	R2 *R2Client

	// KV key-value storage client
	KV *KVClient
}

// NewStorage creates a new Storage instance with all clients initialized
func NewStorage(config *CloudflareConfig) *Storage {
	return &Storage{
		D1: NewD1Client(config),
		R2: NewR2Client(config),
		KV: NewKVClient(config),
	}
}

// NewStorageFromEnv creates a new Storage instance using environment variables
// CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID
func NewStorageFromEnv() *Storage {
	config := &CloudflareConfig{
		APIToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
		AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		BaseURL:   "https://api.cloudflare.com/client/v4",
	}

	return NewStorage(config)
}
