package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// R2Client provides access to Cloudflare R2 object storage
type R2Client struct {
	config *CloudflareConfig
	client *http.Client
}

// R2Object represents an object in R2 storage
type R2Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type"`
	LastModified string            `json:"last_modified"`
	Metadata     map[string]string `json:"metadata"`
}

// NewR2Client creates a new R2Client with the provided configuration
func NewR2Client(config *CloudflareConfig) *R2Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.cloudflare.com/client/v4"
	}

	return &R2Client{
		config: config,
		client: createHTTPClient(config),
	}
}

// ListObjects lists objects in an R2 bucket with an optional prefix
func (r *R2Client) ListObjects(ctx context.Context, bucketName, prefix string) ([]R2Object, error) {
	url := fmt.Sprintf("%s/accounts/%s/r2/buckets/%s/objects",
		r.config.BaseURL, r.config.AccountID, bucketName)

	if prefix != "" {
		url = fmt.Sprintf("%s?prefix=%s", url, prefix)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+r.config.APIToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool       `json:"success"`
		Result  []R2Object `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("request was not successful")
	}

	return response.Result, nil
}

// UploadObject uploads an object to an R2 bucket
func (r *R2Client) UploadObject(ctx context.Context, bucketName, key string, data io.Reader, contentType string, metadata map[string]string) (*R2Object, error) {
	url := fmt.Sprintf("%s/accounts/%s/r2/buckets/%s/objects/%s",
		r.config.BaseURL, r.config.AccountID, bucketName, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+r.config.APIToken)
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	// Add metadata headers if provided
	if metadata != nil {
		for k, v := range metadata {
			req.Header.Add("X-Metadata-"+k, v)
		}
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool     `json:"success"`
		Result  R2Object `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("request was not successful")
	}

	return &response.Result, nil
}

// GetObject retrieves an object from an R2 bucket
func (r *R2Client) GetObject(ctx context.Context, bucketName, key string) (io.ReadCloser, map[string]string, error) {
	url := fmt.Sprintf("%s/accounts/%s/r2/buckets/%s/objects/%s",
		r.config.BaseURL, r.config.AccountID, bucketName, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+r.config.APIToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Extract metadata from headers
	metadata := make(map[string]string)
	for k, v := range resp.Header {
		if strings.HasPrefix(k, "X-Metadata-") {
			metaKey := strings.TrimPrefix(k, "X-Metadata-")
			metadata[metaKey] = v[0]
		}
	}

	return resp.Body, metadata, nil
}

// DeleteObject deletes an object from an R2 bucket
func (r *R2Client) DeleteObject(ctx context.Context, bucketName, key string) error {
	url := fmt.Sprintf("%s/accounts/%s/r2/buckets/%s/objects/%s",
		r.config.BaseURL, r.config.AccountID, bucketName, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+r.config.APIToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
