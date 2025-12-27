package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/ports"
)

// UpstreamClient forwards requests to the upstream service.
type UpstreamClient struct {
	client  *http.Client
	baseURL *url.URL
}

// UpstreamConfig contains configuration for the upstream client.
type UpstreamConfig struct {
	BaseURL        string
	Timeout        time.Duration
	MaxIdleConns   int
	IdleConnTimeout time.Duration
}

// NewUpstreamClient creates a new upstream HTTP client.
func NewUpstreamClient(cfg UpstreamConfig) (*UpstreamClient, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 100
	}

	idleConnTimeout := cfg.IdleConnTimeout
	if idleConnTimeout == 0 {
		idleConnTimeout = 90 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConns,
		IdleConnTimeout:     idleConnTimeout,
		DisableCompression:  false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &UpstreamClient{
		client:  client,
		baseURL: baseURL,
	}, nil
}

// Forward sends a request to the upstream and returns the response.
func (u *UpstreamClient) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
	start := time.Now()

	// Build upstream URL
	upstreamURL := u.baseURL.ResolveReference(&url.URL{
		Path:     req.Path,
		RawQuery: req.Query,
	})

	// Create HTTP request
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, upstreamURL.String(), body)
	if err != nil {
		return proxy.Response{}, fmt.Errorf("create request: %w", err)
	}

	// Copy headers (except those we don't want to forward)
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Add forwarding headers
	httpReq.Header.Set("X-Forwarded-For", req.RemoteIP)
	httpReq.Header.Set("X-Forwarded-Proto", "https")
	if req.TraceID != "" {
		httpReq.Header.Set("X-Request-ID", req.TraceID)
	}

	// Execute request
	resp, err := u.client.Do(httpReq)
	if err != nil {
		return proxy.Response{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20)) // 50MB limit
	if err != nil {
		return proxy.Response{}, fmt.Errorf("read response: %w", err)
	}

	// Extract response headers
	headers := make(map[string]string)
	for k, v := range resp.Header {
		// Skip hop-by-hop headers
		lower := strings.ToLower(k)
		if lower == "connection" || lower == "keep-alive" ||
			lower == "proxy-authenticate" || lower == "proxy-authorization" ||
			lower == "te" || lower == "trailers" || lower == "transfer-encoding" ||
			lower == "upgrade" {
			continue
		}
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return proxy.Response{
		Status:       resp.StatusCode,
		Headers:      headers,
		Body:         respBody,
		LatencyMs:    time.Since(start).Milliseconds(),
		UpstreamAddr: u.baseURL.Host,
	}, nil
}

// HealthCheck verifies the upstream is reachable.
func (u *UpstreamClient) HealthCheck(ctx context.Context) error {
	// Try a HEAD request to the base URL
	req, err := http.NewRequestWithContext(ctx, "HEAD", u.baseURL.String(), nil)
	if err != nil {
		return err
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Any response (even 404) means upstream is reachable
	return nil
}

// Close closes the upstream client.
func (u *UpstreamClient) Close() error {
	u.client.CloseIdleConnections()
	return nil
}

// Ensure interface compliance.
var _ ports.Upstream = (*UpstreamClient)(nil)
