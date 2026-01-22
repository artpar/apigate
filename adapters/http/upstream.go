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
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
)

// UpstreamClient forwards requests to the upstream service.
type UpstreamClient struct {
	client          *http.Client // For buffered requests
	streamingClient *http.Client // For streaming requests (no timeout)
	baseURL         *url.URL
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

	// Streaming transport with same settings but no compression
	// (SSE shouldn't be compressed mid-stream)
	streamingTransport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConns,
		IdleConnTimeout:     idleConnTimeout,
		DisableCompression:  true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	// Streaming client has no timeout - streams can run indefinitely
	streamingClient := &http.Client{
		Transport: streamingTransport,
		Timeout:   0, // No timeout for streams
	}

	return &UpstreamClient{
		client:          client,
		streamingClient: streamingClient,
		baseURL:         baseURL,
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

	// Preserve original Host header for virtual hosting
	// Note: httpReq.Host is set to the URL's host by NewRequestWithContext,
	// and httpReq.Header.Set("Host", v) does NOT override httpReq.Host
	if host, ok := req.Headers["Host"]; ok && host != "" {
		httpReq.Host = host
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

// ForwardTo sends a request to a specific upstream URL (not the default).
// This is used when a route specifies a different upstream.
func (u *UpstreamClient) ForwardTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (proxy.Response, error) {
	start := time.Now()

	// Parse upstream base URL
	baseURL, err := url.Parse(upstream.BaseURL)
	if err != nil {
		return proxy.Response{}, fmt.Errorf("parse upstream URL: %w", err)
	}

	// Build upstream URL
	upstreamURL := baseURL.ResolveReference(&url.URL{
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

	// Copy headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Preserve original Host header for virtual hosting
	if host, ok := req.Headers["Host"]; ok && host != "" {
		httpReq.Host = host
	}

	// Add forwarding headers
	httpReq.Header.Set("X-Forwarded-For", req.RemoteIP)
	httpReq.Header.Set("X-Forwarded-Proto", "https")
	if req.TraceID != "" {
		httpReq.Header.Set("X-Request-ID", req.TraceID)
	}

	// Use appropriate client based on upstream timeout
	client := u.client
	if upstream.Timeout > 0 {
		// Create a client with the upstream's timeout
		client = &http.Client{
			Transport: u.client.Transport,
			Timeout:   upstream.Timeout,
		}
	}

	// Execute request
	resp, err := client.Do(httpReq)
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
		UpstreamAddr: baseURL.Host,
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
	u.streamingClient.CloseIdleConnections()
	return nil
}

// ForwardStreaming sends a request to the upstream and returns a streaming response.
// The caller is responsible for closing the response body.
func (u *UpstreamClient) ForwardStreaming(ctx context.Context, req proxy.Request) (ports.StreamingResponse, error) {
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
		return ports.StreamingResponse{}, fmt.Errorf("create streaming request: %w", err)
	}

	// Copy headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Preserve original Host header for virtual hosting
	if host, ok := req.Headers["Host"]; ok && host != "" {
		httpReq.Host = host
	}

	// Add forwarding headers
	httpReq.Header.Set("X-Forwarded-For", req.RemoteIP)
	httpReq.Header.Set("X-Forwarded-Proto", "https")
	if req.TraceID != "" {
		httpReq.Header.Set("X-Request-ID", req.TraceID)
	}

	// Execute request with streaming client (no timeout)
	resp, err := u.streamingClient.Do(httpReq)
	if err != nil {
		return ports.StreamingResponse{}, fmt.Errorf("execute streaming request: %w", err)
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

	contentType := resp.Header.Get("Content-Type")

	return ports.StreamingResponse{
		Status:       resp.StatusCode,
		Headers:      headers,
		Body:         resp.Body, // Caller must close
		IsStreaming:  true,
		ContentType:  contentType,
		LatencyMs:    time.Since(start).Milliseconds(),
		UpstreamAddr: u.baseURL.Host,
	}, nil
}

// ForwardStreamingTo sends a streaming request to a specific upstream (not the default).
func (u *UpstreamClient) ForwardStreamingTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (ports.StreamingResponse, error) {
	start := time.Now()

	// Parse upstream base URL
	baseURL, err := url.Parse(upstream.BaseURL)
	if err != nil {
		return ports.StreamingResponse{}, fmt.Errorf("parse upstream URL: %w", err)
	}

	// Build upstream URL
	upstreamURL := baseURL.ResolveReference(&url.URL{
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
		return ports.StreamingResponse{}, fmt.Errorf("create streaming request: %w", err)
	}

	// Copy headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Preserve original Host header for virtual hosting
	if host, ok := req.Headers["Host"]; ok && host != "" {
		httpReq.Host = host
	}

	// Add forwarding headers
	httpReq.Header.Set("X-Forwarded-For", req.RemoteIP)
	httpReq.Header.Set("X-Forwarded-Proto", "https")
	if req.TraceID != "" {
		httpReq.Header.Set("X-Request-ID", req.TraceID)
	}

	// Execute request with streaming client (no timeout)
	resp, err := u.streamingClient.Do(httpReq)
	if err != nil {
		return ports.StreamingResponse{}, fmt.Errorf("execute streaming request: %w", err)
	}

	// Extract response headers
	headers := make(map[string]string)
	for k, v := range resp.Header {
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

	contentType := resp.Header.Get("Content-Type")

	return ports.StreamingResponse{
		Status:       resp.StatusCode,
		Headers:      headers,
		Body:         resp.Body,
		IsStreaming:  true,
		ContentType:  contentType,
		LatencyMs:    time.Since(start).Milliseconds(),
		UpstreamAddr: baseURL.Host,
	}, nil
}

// ShouldStream determines if a request should use streaming based on protocol and request headers.
func (u *UpstreamClient) ShouldStream(req proxy.Request, protocol route.Protocol) bool {
	// Check protocol
	switch protocol {
	case route.ProtocolSSE, route.ProtocolHTTPStream, route.ProtocolWebSocket:
		return true
	}

	// Check Accept header for SSE
	if accept, ok := req.Headers["Accept"]; ok {
		if strings.Contains(accept, "text/event-stream") {
			return true
		}
	}

	return false
}

// Ensure interface compliance.
var _ ports.Upstream = (*UpstreamClient)(nil)
var _ ports.StreamingUpstream = (*UpstreamClient)(nil)
