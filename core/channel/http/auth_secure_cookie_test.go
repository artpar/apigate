package http

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAuthHandler_SetSessionCookie_HTTP verifies cookie is NOT secure over HTTP
func TestAuthHandler_SetSessionCookie_HTTP(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	// r.TLS is nil, simulating HTTP

	session := Session{
		UserID:    "user123",
		Email:     "test@example.com",
		Name:      "Test User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}

	cookie := cookies[0]
	if cookie.Secure {
		t.Error("cookie should NOT be secure over HTTP")
	}
	t.Logf("✓ Cookie Secure flag = false for HTTP request")
}

// TestAuthHandler_SetSessionCookie_HTTPS verifies cookie IS secure over HTTPS
func TestAuthHandler_SetSessionCookie_HTTPS(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	// Set TLS to non-nil, simulating HTTPS
	r.TLS = &tls.ConnectionState{}

	session := Session{
		UserID:    "user123",
		Email:     "test@example.com",
		Name:      "Test User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}

	cookie := cookies[0]
	if !cookie.Secure {
		t.Error("cookie SHOULD be secure over HTTPS")
	}
	t.Logf("✓ Cookie Secure flag = true for HTTPS request")
}

// TestAuthHandler_SetSessionCookie_ProxyHTTPS verifies cookie IS secure behind HTTPS proxy
func TestAuthHandler_SetSessionCookie_ProxyHTTPS(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	// Simulate reverse proxy forwarding HTTPS request
	r.Header.Set("X-Forwarded-Proto", "https")

	session := Session{
		UserID:    "user123",
		Email:     "test@example.com",
		Name:      "Test User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}

	cookie := cookies[0]
	if !cookie.Secure {
		t.Error("cookie SHOULD be secure when X-Forwarded-Proto is https")
	}
	t.Logf("✓ Cookie Secure flag = true for proxied HTTPS request")
}
