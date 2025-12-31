package proxy

import (
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	req := Request{
		APIKey:    "test-key",
		Method:    "GET",
		Path:      "/api/v1/users",
		Query:     "page=1",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      []byte(`{"test": true}`),
		RemoteIP:  "192.168.1.1",
		UserAgent: "test-agent",
		TraceID:   "trace-123",
	}

	if req.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", req.APIKey)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %s, want GET", req.Method)
	}
	if req.Path != "/api/v1/users" {
		t.Errorf("Path = %s, want /api/v1/users", req.Path)
	}
	if req.Query != "page=1" {
		t.Errorf("Query = %s, want page=1", req.Query)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("Headers[Content-Type] = %s, want application/json", req.Headers["Content-Type"])
	}
	if string(req.Body) != `{"test": true}` {
		t.Errorf("Body = %s, want {\"test\": true}", string(req.Body))
	}
	if req.RemoteIP != "192.168.1.1" {
		t.Errorf("RemoteIP = %s, want 192.168.1.1", req.RemoteIP)
	}
	if req.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %s, want test-agent", req.UserAgent)
	}
	if req.TraceID != "trace-123" {
		t.Errorf("TraceID = %s, want trace-123", req.TraceID)
	}
}

func TestResponse(t *testing.T) {
	resp := Response{
		Status:       200,
		Headers:      map[string]string{"Content-Type": "application/json"},
		Body:         []byte(`{"success": true}`),
		LatencyMs:    50,
		UpstreamAddr: "https://api.example.com",
	}

	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Headers[Content-Type] = %s, want application/json", resp.Headers["Content-Type"])
	}
	if string(resp.Body) != `{"success": true}` {
		t.Errorf("Body = %s, want {\"success\": true}", string(resp.Body))
	}
	if resp.LatencyMs != 50 {
		t.Errorf("LatencyMs = %d, want 50", resp.LatencyMs)
	}
	if resp.UpstreamAddr != "https://api.example.com" {
		t.Errorf("UpstreamAddr = %s, want https://api.example.com", resp.UpstreamAddr)
	}
}

func TestAuthContext(t *testing.T) {
	auth := AuthContext{
		KeyID:     "key-123",
		UserID:    "user-456",
		PlanID:    "pro",
		RateLimit: 1000,
		Scopes:    []string{"read", "write"},
	}

	if auth.KeyID != "key-123" {
		t.Errorf("KeyID = %s, want key-123", auth.KeyID)
	}
	if auth.UserID != "user-456" {
		t.Errorf("UserID = %s, want user-456", auth.UserID)
	}
	if auth.PlanID != "pro" {
		t.Errorf("PlanID = %s, want pro", auth.PlanID)
	}
	if auth.RateLimit != 1000 {
		t.Errorf("RateLimit = %d, want 1000", auth.RateLimit)
	}
	if len(auth.Scopes) != 2 {
		t.Errorf("len(Scopes) = %d, want 2", len(auth.Scopes))
	}
}

func TestRequestContext(t *testing.T) {
	now := time.Now()
	ctx := RequestContext{
		Request: Request{
			Method: "POST",
			Path:   "/api/data",
		},
		Auth: AuthContext{
			KeyID:  "key-789",
			UserID: "user-abc",
		},
		Timestamp: now,
	}

	if ctx.Request.Method != "POST" {
		t.Errorf("Request.Method = %s, want POST", ctx.Request.Method)
	}
	if ctx.Auth.KeyID != "key-789" {
		t.Errorf("Auth.KeyID = %s, want key-789", ctx.Auth.KeyID)
	}
	if ctx.Timestamp != now {
		t.Errorf("Timestamp = %v, want %v", ctx.Timestamp, now)
	}
}

func TestErrorResponse(t *testing.T) {
	err := ErrorResponse{
		Status:  400,
		Code:    "bad_request",
		Message: "Invalid input",
	}

	if err.Status != 400 {
		t.Errorf("Status = %d, want 400", err.Status)
	}
	if err.Code != "bad_request" {
		t.Errorf("Code = %s, want bad_request", err.Code)
	}
	if err.Message != "Invalid input" {
		t.Errorf("Message = %s, want Invalid input", err.Message)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    ErrorResponse
		status int
		code   string
	}{
		{"ErrMissingKey", ErrMissingKey, 401, "missing_api_key"},
		{"ErrInvalidKey", ErrInvalidKey, 401, "invalid_api_key"},
		{"ErrRateLimited", ErrRateLimited, 429, "rate_limit_exceeded"},
		{"ErrQuotaExceeded", ErrQuotaExceeded, 402, "quota_exceeded"},
		{"ErrUpstreamError", ErrUpstreamError, 502, "upstream_error"},
		{"ErrTimeout", ErrTimeout, 504, "upstream_timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status {
				t.Errorf("%s.Status = %d, want %d", tt.name, tt.err.Status, tt.status)
			}
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %s, want %s", tt.name, tt.err.Code, tt.code)
			}
			if tt.err.Message == "" {
				t.Errorf("%s.Message should not be empty", tt.name)
			}
		})
	}
}
