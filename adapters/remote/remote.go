// Package remote provides adapters that delegate to external HTTP services.
// This enables customers to use their existing auth, billing, and usage systems.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides HTTP communication with external services.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	headers    map[string]string
}

// ClientConfig configures the remote client.
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
	Headers map[string]string
}

// NewClient creates a new remote HTTP client.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		headers:    cfg.Headers,
	}
}

// Request sends an HTTP request to the remote service.
func (c *Client) Request(ctx context.Context, method, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &RemoteError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// RemoteError represents an error from the remote service.
type RemoteError struct {
	StatusCode int
	Message    string
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("remote error %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404.
func IsNotFound(err error) bool {
	if re, ok := err.(*RemoteError); ok {
		return re.StatusCode == 404
	}
	return false
}
