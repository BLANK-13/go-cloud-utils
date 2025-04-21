package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// KVClient provides access to Cloudflare KV storage
type KVClient struct {
	config *CloudflareConfig
	client *http.Client
}

// KVKey represents a key in KV storage
type KVKey struct {
	Name       string `json:"name"`
	Expiration int64  `json:"expiration,omitempty"`
}

// NewKVClient creates a new KVClient with the provided configuration
func NewKVClient(config *CloudflareConfig) *KVClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.cloudflare.com/client/v4"
	}

	return &KVClient{
		config: config,
		client: createHTTPClient(config),
	}
}

// ListKeys lists keys in a KV namespace with an optional prefix
func (k *KVClient) ListKeys(ctx context.Context, namespaceID, prefix string) ([]KVKey, error) {
	urlPath := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/keys",
		k.config.BaseURL, k.config.AccountID, namespaceID)

	if prefix != "" {
		urlPath = fmt.Sprintf("%s?prefix=%s", urlPath, url.QueryEscape(prefix))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+k.config.APIToken)

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool    `json:"success"`
		Result  []KVKey `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("request was not successful")
	}

	return response.Result, nil
}

// WriteValue writes a value to a KV namespace with an optional expiration
func (k *KVClient) WriteValue(ctx context.Context, namespaceID, key string, value []byte, expiration *int64) error {
	urlPath := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		k.config.BaseURL, k.config.AccountID, namespaceID, url.PathEscape(key))

	if expiration != nil {
		urlPath = fmt.Sprintf("%s?expiration_ttl=%d", urlPath, *expiration)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, urlPath, bytes.NewReader(value))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+k.config.APIToken)
	req.Header.Add("Content-Type", "application/octet-stream")

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("request was not successful")
	}

	return nil
}

// ReadValue reads a value from a KV namespace
func (k *KVClient) ReadValue(ctx context.Context, namespaceID, key string) ([]byte, error) {
	urlPath := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		k.config.BaseURL, k.config.AccountID, namespaceID, url.PathEscape(key))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+k.config.APIToken)

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

// DeleteValue deletes a value from a KV namespace
func (k *KVClient) DeleteValue(ctx context.Context, namespaceID, key string) error {
	urlPath := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s",
		k.config.BaseURL, k.config.AccountID, namespaceID, url.PathEscape(key))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, urlPath, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+k.config.APIToken)

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("request was not successful")
	}

	return nil
}

// WriteJSON writes a JSON value to a KV namespace
func (k *KVClient) WriteJSON(ctx context.Context, namespaceID, key string, value interface{}, expiration *int64) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error encoding value to JSON: %w", err)
	}

	return k.WriteValue(ctx, namespaceID, key, jsonData, expiration)
}

// ReadJSON reads a JSON value from a KV namespace and unmarshals it into the target
func (k *KVClient) ReadJSON(ctx context.Context, namespaceID, key string, target interface{}) error {
	data, err := k.ReadValue(ctx, namespaceID, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("error decoding JSON value: %w", err)
	}

	return nil
}
