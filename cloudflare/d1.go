package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// D1Client provides access to Cloudflare D1 SQL database
type D1Client struct {
	config *CloudflareConfig
	client *http.Client
}

// D1Result represents a result from a D1 query
type D1Result struct {
	Success bool                     `json:"success"`
	Errors  []string                 `json:"errors"`
	Results []map[string]interface{} `json:"results"`
	Meta    D1ResultMeta             `json:"meta"`
}

// D1ResultMeta contains metadata about the query result
type D1ResultMeta struct {
	Served      bool `json:"served"`
	Duration    int  `json:"duration"`
	RowsRead    int  `json:"rows_read"`
	RowsWritten int  `json:"rows_written"`
}

// NewD1Client creates a new D1Client with the provided configuration
func NewD1Client(config *CloudflareConfig) *D1Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.cloudflare.com/client/v4"
	}

	return &D1Client{
		config: config,
		client: createHTTPClient(config),
	}
}

// ExecuteQuery executes a SQL query on a D1 database
func (d *D1Client) ExecuteQuery(ctx context.Context, databaseID, query string, params []interface{}) (*D1Result, error) {
	url := fmt.Sprintf("%s/accounts/%s/d1/database/%s/query", d.config.BaseURL, d.config.AccountID, databaseID)

	requestBody := map[string]interface{}{
		"sql": query,
	}

	if len(params) > 0 {
		requestBody["params"] = params
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error encoding request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+d.config.APIToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Success bool      `json:"success"`
		Result  *D1Result `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !response.Success {
		return nil, errors.New("request was not successful")
	}

	return response.Result, nil
}
