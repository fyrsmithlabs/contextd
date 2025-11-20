package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DaemonClient provides HTTP client for communicating with contextd daemon.
//
// This client is used by the stdio MCP server to delegate tool calls
// to the HTTP daemon running on localhost:9090.
type DaemonClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDaemonClient creates a new daemon HTTP client.
//
// The baseURL should point to the contextd HTTP daemon (e.g., "http://localhost:9090").
func NewDaemonClient(baseURL string) *DaemonClient {
	return &DaemonClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Post sends a POST request to the daemon endpoint.
//
// The path should be the endpoint path (e.g., "/mcp/checkpoint/save").
// The request body is JSON-encoded. The response is JSON-decoded into result.
func (c *DaemonClient) Post(ctx context.Context, path string, request interface{}, result interface{}) error {
	// Encode request body
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(request); err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	// Create HTTP request
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned status %d: %s", resp.StatusCode, string(body))
	}

	// Decode response
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// Get sends a GET request to the daemon endpoint.
//
// The response is JSON-decoded into result.
func (c *DaemonClient) Get(ctx context.Context, path string, result interface{}) error {
	// Create HTTP request
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned status %d: %s", resp.StatusCode, string(body))
	}

	// Decode response
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
