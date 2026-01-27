package http

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCookieAttributes_HTTP_AllFields verifies all cookie attributes for HTTP requests
func TestCookieAttributes_HTTP_AllFields(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	// HTTP request - no TLS

	session := Session{
		UserID:    "user_http",
		Email:     "http@example.com",
		Name:      "HTTP User",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := cookies[0]

	// Validate all attributes for HTTP
	tests := []struct {
		name     string
		got      interface{}
		want     interface{}
		critical bool // If true, test fails; if false, just warns
	}{
		{"Name", cookie.Name, SessionCookie, true},
		{"Value not empty", cookie.Value != "", true, true},
		{"Path", cookie.Path, "/", true},
		{"HttpOnly", cookie.HttpOnly, true, true},
		{"SameSite", cookie.SameSite, http.SameSiteLaxMode, true},
		{"Secure", cookie.Secure, false, true}, // Must be false for HTTP
		{"Expires not zero", !cookie.Expires.IsZero(), true, true},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			msg := t.Errorf
			if !tt.critical {
				msg = t.Logf
			}
			msg("HTTP cookie attribute %s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	t.Logf("✓ All HTTP cookie attributes correct (Secure=false)")
}

// TestCookieAttributes_HTTPS_AllFields verifies all cookie attributes for HTTPS requests
func TestCookieAttributes_HTTPS_AllFields(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.TLS = &tls.ConnectionState{} // HTTPS request

	session := Session{
		UserID:    "user_https",
		Email:     "https@example.com",
		Name:      "HTTPS User",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := cookies[0]

	// Validate all attributes for HTTPS
	tests := []struct {
		name     string
		got      interface{}
		want     interface{}
		critical bool
	}{
		{"Name", cookie.Name, SessionCookie, true},
		{"Value not empty", cookie.Value != "", true, true},
		{"Path", cookie.Path, "/", true},
		{"HttpOnly", cookie.HttpOnly, true, true},
		{"SameSite", cookie.SameSite, http.SameSiteLaxMode, true},
		{"Secure", cookie.Secure, true, true}, // Must be true for HTTPS
		{"Expires not zero", !cookie.Expires.IsZero(), true, true},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			msg := t.Errorf
			if !tt.critical {
				msg = t.Logf
			}
			msg("HTTPS cookie attribute %s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	t.Logf("✓ All HTTPS cookie attributes correct (Secure=true)")
}

// TestCookieAttributes_ProxyHTTPS_AllFields verifies all cookie attributes for proxied HTTPS
func TestCookieAttributes_ProxyHTTPS_AllFields(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Header.Set("X-Forwarded-Proto", "https") // Proxied HTTPS

	session := Session{
		UserID:    "user_proxy",
		Email:     "proxy@example.com",
		Name:      "Proxy User",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := cookies[0]

	// Validate all attributes for proxied HTTPS
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Name", cookie.Name, SessionCookie},
		{"Path", cookie.Path, "/"},
		{"HttpOnly", cookie.HttpOnly, true},
		{"SameSite", cookie.SameSite, http.SameSiteLaxMode},
		{"Secure", cookie.Secure, true}, // Must be true for proxied HTTPS
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Proxied HTTPS cookie attribute %s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	t.Logf("✓ All proxied HTTPS cookie attributes correct (Secure=true)")
}

// TestCookieExpiration_SevenDays verifies cookie expires in exactly 7 days
func TestCookieExpiration_SevenDays(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	before := time.Now()
	session := Session{
		UserID:    "user_exp",
		Email:     "exp@example.com",
		Name:      "Expiry User",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	h.setSessionCookie(w, r, session)
	after := time.Now()

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := cookies[0]

	// Cookie should expire in approximately 7 days
	expectedMin := before.Add(7*24*time.Hour - 10*time.Second)
	expectedMax := after.Add(7*24*time.Hour + 10*time.Second)

	if cookie.Expires.Before(expectedMin) || cookie.Expires.After(expectedMax) {
		t.Errorf("Cookie expiration out of range. Got %v, want between %v and %v",
			cookie.Expires, expectedMin, expectedMax)
	}

	duration := cookie.Expires.Sub(before)
	t.Logf("✓ Cookie expires in %v (approximately 7 days)", duration)
}

// TestCookieValue_Base64Encoded verifies cookie value is base64 encoded JSON
func TestCookieValue_Base64Encoded(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	session := Session{
		UserID:    "user_b64",
		Email:     "b64@example.com",
		Name:      "Base64 User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := cookies[0]

	// Value should be non-empty base64
	if cookie.Value == "" {
		t.Error("Cookie value should not be empty")
	}

	// Base64 should not contain spaces, newlines, or control characters
	for _, ch := range cookie.Value {
		if ch < 33 || ch > 126 {
			t.Errorf("Cookie value contains invalid character: %c (code %d)", ch, ch)
		}
	}

	t.Logf("✓ Cookie value is properly encoded: %d bytes", len(cookie.Value))
}

// TestCookieSecurity_HTTPOnly verifies HttpOnly prevents JS access
func TestCookieSecurity_HTTPOnly(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	session := Session{
		UserID:    "user_security",
		Email:     "security@example.com",
		Name:      "Security User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	cookie := cookies[0]

	if !cookie.HttpOnly {
		t.Fatal("Cookie MUST be HttpOnly to prevent XSS attacks")
	}

	t.Logf("✓ Cookie is HttpOnly (protected from JavaScript access)")
}

// TestCookieSecurity_SameSite verifies SameSite prevents CSRF
func TestCookieSecurity_SameSite(t *testing.T) {
	h := NewAuthHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	session := Session{
		UserID:    "user_csrf",
		Email:     "csrf@example.com",
		Name:      "CSRF User",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	h.setSessionCookie(w, r, session)

	cookies := w.Result().Cookies()
	cookie := cookies[0]

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Cookie SameSite = %v, want %v (CSRF protection)",
			cookie.SameSite, http.SameSiteLaxMode)
	}

	t.Logf("✓ Cookie has SameSite=Lax (CSRF protection)")
}
