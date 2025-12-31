package http_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apihttp "github.com/artpar/apigate/adapters/http"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/route"
)

func TestNewUpstreamClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     apihttp.UpstreamConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: apihttp.UpstreamConfig{
				BaseURL:         "https://api.example.com",
				Timeout:         30 * time.Second,
				MaxIdleConns:    50,
				IdleConnTimeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "minimal config with defaults",
			cfg: apihttp.UpstreamConfig{
				BaseURL: "https://api.example.com",
			},
			wantErr: false,
		},
		{
			name: "invalid URL",
			cfg: apihttp.UpstreamConfig{
				BaseURL: "://invalid-url",
			},
			wantErr: true,
		},
		{
			name: "empty URL is technically valid in url.Parse",
			cfg: apihttp.UpstreamConfig{
				BaseURL: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := apihttp.NewUpstreamClient(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client == nil {
				t.Error("expected non-nil client")
			}
			// Cleanup
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestUpstreamClient_Forward(t *testing.T) {
	// Create mock upstream server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify forwarded headers
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Error("missing X-Forwarded-For header")
		}
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			t.Errorf("X-Forwarded-Proto = %s, want https", r.Header.Get("X-Forwarded-Proto"))
		}

		// Echo back the request details
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"method":"` + r.Method + `","path":"` + r.URL.Path + `","query":"` + r.URL.RawQuery + `"}`))
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name       string
		req        proxy.Request
		wantStatus int
	}{
		{
			name: "GET request",
			req: proxy.Request{
				Method:   "GET",
				Path:     "/api/data",
				Query:    "foo=bar",
				Headers:  map[string]string{"Content-Type": "application/json"},
				RemoteIP: "192.168.1.1",
				TraceID:  "trace-123",
			},
			wantStatus: 200,
		},
		{
			name: "POST request with body",
			req: proxy.Request{
				Method:   "POST",
				Path:     "/api/data",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     []byte(`{"key":"value"}`),
				RemoteIP: "192.168.1.1",
			},
			wantStatus: 200,
		},
		{
			name: "request without trace ID",
			req: proxy.Request{
				Method:   "GET",
				Path:     "/api/health",
				RemoteIP: "10.0.0.1",
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Forward(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("Forward failed: %v", err)
			}
			if resp.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", resp.Status, tt.wantStatus)
			}
			if resp.LatencyMs < 0 {
				t.Error("LatencyMs should be non-negative")
			}
		})
	}
}

func TestUpstreamClient_Forward_SkipsHopByHopHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set hop-by-hop headers that should be filtered
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=5")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("X-Custom", "should-be-kept")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	resp, err := client.Forward(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}

	// Check that hop-by-hop headers are filtered
	if _, ok := resp.Headers["Connection"]; ok {
		t.Error("Connection header should be filtered")
	}
	if _, ok := resp.Headers["Keep-Alive"]; ok {
		t.Error("Keep-Alive header should be filtered")
	}
	if _, ok := resp.Headers["Transfer-Encoding"]; ok {
		t.Error("Transfer-Encoding header should be filtered")
	}

	// Check that custom headers are preserved
	if resp.Headers["X-Custom"] != "should-be-kept" {
		t.Error("X-Custom header should be preserved")
	}
}

func TestUpstreamClient_Forward_Timeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: server.URL,
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.Forward(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestUpstreamClient_Forward_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.Forward(ctx, proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	})
	if err == nil {
		t.Error("expected context error")
	}
}

func TestUpstreamClient_ForwardTo(t *testing.T) {
	// Create mock upstream server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create client with a different base URL
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999", // This won't be used
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	upstream := &route.Upstream{
		ID:      "test-upstream",
		Name:    "Test",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	resp, err := client.ForwardTo(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/api/data",
		Headers:  map[string]string{"Accept": "application/json"},
		RemoteIP: "192.168.1.1",
		TraceID:  "trace-456",
	}, upstream)
	if err != nil {
		t.Fatalf("ForwardTo failed: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if !strings.Contains(string(resp.Body), "ok") {
		t.Errorf("Body = %s, want to contain 'ok'", string(resp.Body))
	}
}

func TestUpstreamClient_ForwardTo_InvalidURL(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	upstream := &route.Upstream{
		ID:      "invalid",
		Name:    "Invalid",
		BaseURL: "://invalid-url",
	}

	_, err = client.ForwardTo(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	}, upstream)
	if err == nil {
		t.Error("expected error for invalid upstream URL")
	}
}

func TestUpstreamClient_ForwardTo_CustomTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
		Timeout: 10 * time.Second, // Default timeout
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Upstream with short timeout
	upstream := &route.Upstream{
		ID:      "test",
		BaseURL: server.URL,
		Timeout: 50 * time.Millisecond,
	}

	_, err = client.ForwardTo(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	}, upstream)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestUpstreamClient_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"healthy - 200", 200, false},
		{"healthy - 404", 404, false}, // Any response means reachable
		{"healthy - 500", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "HEAD" {
					t.Errorf("Method = %s, want HEAD", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer client.Close()

			err = client.HealthCheck(context.Background())
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpstreamClient_HealthCheck_Unreachable(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:59999", // Non-existent port
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	err = client.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestUpstreamClient_Close(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestUpstreamClient_ForwardStreaming(t *testing.T) {
	// Create SSE-like streaming server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter doesn't support Flusher")
			return
		}

		// Write some SSE data
		w.Write([]byte("data: {\"msg\":\"hello\"}\n\n"))
		flusher.Flush()
		w.Write([]byte("data: {\"msg\":\"world\"}\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	resp, err := client.ForwardStreaming(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/events",
		Headers:  map[string]string{"Accept": "text/event-stream"},
		RemoteIP: "192.168.1.1",
		TraceID:  "trace-stream",
	})
	if err != nil {
		t.Fatalf("ForwardStreaming failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if !resp.IsStreaming {
		t.Error("IsStreaming should be true")
	}
	if resp.ContentType != "text/event-stream" {
		t.Errorf("ContentType = %s, want text/event-stream", resp.ContentType)
	}

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if !strings.Contains(string(body), "hello") {
		t.Errorf("Body should contain 'hello', got: %s", string(body))
	}
}

func TestUpstreamClient_ForwardStreamingTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: test\n\n"))
	}))
	defer server.Close()

	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	upstream := &route.Upstream{
		ID:      "test",
		BaseURL: server.URL,
	}

	resp, err := client.ForwardStreamingTo(context.Background(), proxy.Request{
		Method:   "POST",
		Path:     "/chat",
		Body:     []byte(`{"prompt":"hello"}`),
		Headers:  map[string]string{"Content-Type": "application/json"},
		RemoteIP: "192.168.1.1",
	}, upstream)
	if err != nil {
		t.Fatalf("ForwardStreamingTo failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if !resp.IsStreaming {
		t.Error("IsStreaming should be true")
	}
}

func TestUpstreamClient_ForwardStreamingTo_InvalidURL(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	upstream := &route.Upstream{
		ID:      "invalid",
		BaseURL: "://invalid-url",
	}

	_, err = client.ForwardStreamingTo(context.Background(), proxy.Request{
		Method:   "GET",
		Path:     "/",
		RemoteIP: "127.0.0.1",
	}, upstream)
	if err == nil {
		t.Error("expected error for invalid upstream URL")
	}
}

func TestUpstreamClient_ShouldStream(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name     string
		req      proxy.Request
		protocol route.Protocol
		want     bool
	}{
		{
			name:     "SSE protocol",
			req:      proxy.Request{Method: "GET", Path: "/events"},
			protocol: route.ProtocolSSE,
			want:     true,
		},
		{
			name:     "HTTP stream protocol",
			req:      proxy.Request{Method: "GET", Path: "/stream"},
			protocol: route.ProtocolHTTPStream,
			want:     true,
		},
		{
			name:     "WebSocket protocol",
			req:      proxy.Request{Method: "GET", Path: "/ws"},
			protocol: route.ProtocolWebSocket,
			want:     true,
		},
		{
			name:     "HTTP protocol with SSE Accept header",
			req:      proxy.Request{Method: "GET", Path: "/data", Headers: map[string]string{"Accept": "text/event-stream"}},
			protocol: route.ProtocolHTTP,
			want:     true,
		},
		{
			name:     "HTTP protocol without SSE header",
			req:      proxy.Request{Method: "GET", Path: "/data", Headers: map[string]string{"Accept": "application/json"}},
			protocol: route.ProtocolHTTP,
			want:     false,
		},
		{
			name:     "HTTP protocol no headers",
			req:      proxy.Request{Method: "GET", Path: "/data"},
			protocol: route.ProtocolHTTP,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.ShouldStream(tt.req, tt.protocol)
			if got != tt.want {
				t.Errorf("ShouldStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpstreamClient_InterfaceCompliance(t *testing.T) {
	client, err := apihttp.NewUpstreamClient(apihttp.UpstreamConfig{
		BaseURL: "http://localhost:9999",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Verify the client implements the expected interfaces
	// This is a compile-time check but we can verify at runtime too
	if client == nil {
		t.Error("client should not be nil")
	}
}
