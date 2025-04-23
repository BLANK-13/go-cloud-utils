package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// D1Client provides access to Cloudflare D1 SQL database
type D1Client struct {
	config *CloudflareConfig
	client *http.Client
}

// D1ResponseItem represents a single item in the result array
type D1ResponseItem struct {
    Meta struct {
        ChangedDB        bool   `json:"changed_db"`
        Changes          int    `json:"changes"`
        Duration         float32    `json:"duration"`
        LastRowID        int    `json:"last_row_id"`
        RowsRead         int    `json:"rows_read"`
        RowsWritten      int    `json:"rows_written"`
        ServedByPrimary  bool   `json:"served_by_primary"`
        ServedByRegion   string `json:"served_by_region"`
        SizeAfter        int    `json:"size_after"`
        Timings struct {
            SQLDurationMS float32 `json:"sql_duration_ms"`
        } `json:"timings"`
    } `json:"meta"`
    Results []map[string]interface{} `json:"results"`
    Success bool                     `json:"success"`
}

// D1Error represents an error in the response
type D1Error struct {
    Code             int    `json:"code"`
    Message          string `json:"message"`
    DocumentationURL string `json:"documentation_url"`
    Source struct {
        Pointer string `json:"pointer"`
    } `json:"source"`
}

/*
* https://developers.cloudflare.com/api/resources/d1/subresources/database/methods/query/
*/

// D1Response represents the full API response
type D1Response struct {
    Errors   []D1Error       `json:"errors"`
    Messages []D1Error       `json:"messages"`
    Result   []D1ResponseItem `json:"result"`
    Success  bool            `json:"success"`
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
func (d *D1Client) ExecuteQuery(ctx context.Context, databaseID, query string, params []interface{}) (*D1ResponseItem, error) {
    // Create request URL
    url := fmt.Sprintf("%s/accounts/%s/d1/database/%s/query", 
        d.config.BaseURL, d.config.AccountID, databaseID)
    
    // Format parameters as expected by the API
    type QueryParams struct {
        Params []interface{} `json:"params,omitempty"`
    }
    
    // Prepare request body
    requestBody := struct {
        SQL    string      `json:"sql"`
        Params interface{} `json:"params,omitempty"`
    }{
        SQL: query,
    }
    
    // Only add params if they exist
    if params != nil && len(params) > 0 {
        requestBody.Params = params
    }
    
    // Marshal request body
    jsonBody, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("error marshaling request: %w", err)
    }
    
    // Create request
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, fmt.Errorf("error creating request: %w", err)
    }
    
    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+d.config.APIToken)
    
    // Execute request
    resp, err := d.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()
    
    // Check status code
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }
    
    // Decode response
    var response D1Response
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("error decoding response: %w", err)
    }
    
    // Check for success
    if !response.Success {
        errMsg := "request was not successful"
        if len(response.Errors) > 0 {
            errMsg = fmt.Sprintf("%s: %s", errMsg, response.Errors[0].Message)
        }
        return nil, fmt.Errorf(errMsg)
    }
    
    // Return the first result item
    if len(response.Result) == 0 {
        return nil, fmt.Errorf("empty result set")
    }
    
    return &response.Result[0], nil
}
