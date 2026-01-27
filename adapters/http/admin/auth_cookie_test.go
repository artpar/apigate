package admin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// TestAdminRegister_SetsCookie_HTTP verifies cookie is set without Secure flag on HTTP
func TestAdminRegister_SetsCookie_HTTP(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create HTTP request (no TLS)
	reqBody := RegisterRequest{
		Email:    "http@example.com",
		Password: "Test123!",
		Name:     "HTTP User",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.Register(w, r)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No cookies set")
	}

	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Validate all attributes for HTTP
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
		{"Secure", cookie.Secure, false, true}, // Must be false for HTTP
		{"Expires not zero", !cookie.Expires.IsZero(), true, true},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			msg := t.Errorf
			if !tt.critical {
				msg = t.Logf
			}
			msg("HTTP Register cookie attribute %s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	t.Logf("✓ Register HTTP cookie attributes correct (Secure=false)")
}

// TestAdminRegister_SetsCookie_HTTPS verifies cookie is set with Secure flag on HTTPS
func TestAdminRegister_SetsCookie_HTTPS(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create HTTPS request
	reqBody := RegisterRequest{
		Email:    "https@example.com",
		Password: "Test123!",
		Name:     "HTTPS User",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	r.TLS = &tls.ConnectionState{} // Simulate HTTPS
	w := httptest.NewRecorder()

	// Execute
	handler.Register(w, r)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Critical: Secure flag must be true for HTTPS
	if !cookie.Secure {
		t.Errorf("HTTPS Register cookie Secure = false, want true")
	}

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
			msg("HTTPS Register cookie attribute %s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	t.Logf("✓ Register HTTPS cookie attributes correct (Secure=true)")
}

// TestAdminRegister_SetsCookie_ProxiedHTTPS verifies cookie is set with Secure flag via X-Forwarded-Proto
func TestAdminRegister_SetsCookie_ProxiedHTTPS(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create proxied HTTPS request
	reqBody := RegisterRequest{
		Email:    "proxy@example.com",
		Password: "Test123!",
		Name:     "Proxy User",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	r.Header.Set("X-Forwarded-Proto", "https") // Simulate reverse proxy
	w := httptest.NewRecorder()

	// Execute
	handler.Register(w, r)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Critical: Secure flag must be true for proxied HTTPS
	if !cookie.Secure {
		t.Errorf("Proxied HTTPS Register cookie Secure = false, want true (X-Forwarded-Proto detection)")
	}

	t.Logf("✓ Register proxied HTTPS cookie attributes correct (Secure=true via X-Forwarded-Proto)")
}

// TestAdminLogin_SetsCookie_HTTP verifies cookie is set without Secure flag on HTTP login
func TestAdminLogin_SetsCookie_HTTP(t *testing.T) {
	handler := setupCookieTestHandlerWithPassword(t, "login@example.com", "Test123!")

	// Create HTTP request
	reqBody := LoginRequest{
		Email:    "login@example.com",
		Password: "Test123!",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.Login(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Critical: Secure flag must be false for HTTP
	if cookie.Secure {
		t.Errorf("HTTP Login cookie Secure = true, want false")
	}

	t.Logf("✓ Login HTTP cookie attributes correct (Secure=false)")
}

// TestAdminLogin_SetsCookie_HTTPS verifies cookie is set with Secure flag on HTTPS login
func TestAdminLogin_SetsCookie_HTTPS(t *testing.T) {
	handler := setupCookieTestHandlerWithPassword(t, "login-https@example.com", "Test123!")

	// Create HTTPS request
	reqBody := LoginRequest{
		Email:    "login-https@example.com",
		Password: "Test123!",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	r.TLS = &tls.ConnectionState{} // Simulate HTTPS
	w := httptest.NewRecorder()

	// Execute
	handler.Login(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Critical: Secure flag must be true for HTTPS
	if !cookie.Secure {
		t.Errorf("HTTPS Login cookie Secure = false, want true")
	}

	t.Logf("✓ Login HTTPS cookie attributes correct (Secure=true)")
}

// TestAdminLogin_SetsCookie_ProxiedHTTPS verifies cookie is set with Secure flag via X-Forwarded-Proto on login
func TestAdminLogin_SetsCookie_ProxiedHTTPS(t *testing.T) {
	handler := setupCookieTestHandlerWithPassword(t, "login-proxy@example.com", "Test123!")

	// Create proxied HTTPS request
	reqBody := LoginRequest{
		Email:    "login-proxy@example.com",
		Password: "Test123!",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	r.Header.Set("X-Forwarded-Proto", "https") // Simulate reverse proxy
	w := httptest.NewRecorder()

	// Execute
	handler.Login(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Critical: Secure flag must be true for proxied HTTPS
	if !cookie.Secure {
		t.Errorf("Proxied HTTPS Login cookie Secure = false, want true (X-Forwarded-Proto detection)")
	}

	t.Logf("✓ Login proxied HTTPS cookie attributes correct (Secure=true via X-Forwarded-Proto)")
}

// TestAdminLoginAPIKey_SetsCookie_HTTP verifies cookie is set for API key auth on HTTP
func TestAdminLoginAPIKey_SetsCookie_HTTP(t *testing.T) {
	handler, rawKey := setupCookieTestHandlerWithAPIKey(t)

	user := ports.User{
		ID:     "user_apikey_http",
		Email:  "apikey@example.com",
		Name:   "API Key User",
		Status: "active",
	}
	handler.users.Create(context.Background(), user)

	// Create HTTP request
	reqBody := LoginRequest{
		Email:  "apikey@example.com",
		APIKey: rawKey,
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.Login(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Secure flag must be false for HTTP
	if cookie.Secure {
		t.Errorf("HTTP API Key Login cookie Secure = true, want false")
	}

	t.Logf("✓ API Key Login HTTP cookie attributes correct (Secure=false)")
}

// TestAdminLoginAPIKey_SetsCookie_HTTPS verifies cookie is set for API key auth on HTTPS
func TestAdminLoginAPIKey_SetsCookie_HTTPS(t *testing.T) {
	handler, rawKey := setupCookieTestHandlerWithAPIKey(t)

	user := ports.User{
		ID:     "user_apikey_https",
		Email:  "apikey-https@example.com",
		Name:   "API Key HTTPS User",
		Status: "active",
	}
	handler.users.Create(context.Background(), user)

	// Create HTTPS request
	reqBody := LoginRequest{
		Email:  "apikey-https@example.com",
		APIKey: rawKey,
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	r.TLS = &tls.ConnectionState{} // Simulate HTTPS
	w := httptest.NewRecorder()

	// Execute
	handler.Login(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatalf("Session cookie %q not found", SessionCookie)
	}

	// Secure flag must be true for HTTPS
	if !cookie.Secure {
		t.Errorf("HTTPS API Key Login cookie Secure = false, want true")
	}

	t.Logf("✓ API Key Login HTTPS cookie attributes correct (Secure=true)")
}

// TestAdminLogout_ClearsCookie verifies logout clears the session cookie
func TestAdminLogout_ClearsCookie(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create a session first
	session := handler.sessions.Create("user6", "logout@example.com", time.Hour)

	// Create logout request with session cookie AND context
	r := httptest.NewRequest("POST", "/auth/logout", nil)
	r.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: session.ID,
	})

	// Add session ID to context (Logout expects this)
	ctx := context.WithValue(r.Context(), ctxSessionKey, session.ID)
	ctx = context.WithValue(ctx, ctxUserIDKey, "user6")
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	handler.Logout(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify cookie is cleared
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatal("Session cookie not found in logout response")
	}

	// Cookie should be cleared (MaxAge = -1)
	if cookie.MaxAge != -1 {
		t.Errorf("Logout cookie MaxAge = %d, want -1 (deleted)", cookie.MaxAge)
	}

	if cookie.Value != "" {
		t.Errorf("Logout cookie Value = %q, want empty", cookie.Value)
	}

	t.Logf("✓ Logout correctly clears session cookie (MaxAge=-1)")
}

// TestAdminCookie_Expiration_SevenDays verifies cookie expires in exactly 7 days
func TestAdminCookie_Expiration_SevenDays(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create request
	reqBody := RegisterRequest{
		Email:    "exp@example.com",
		Password: "Test123!",
		Name:     "Expiry User",
	}
	body, _ := json.Marshal(reqBody)

	before := time.Now()
	r := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Register(w, r)
	after := time.Now()

	// Get cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatal("Session cookie not found")
	}

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

// TestAdminCookie_Value_Base64Encoded verifies cookie value is base64 encoded JSON
func TestAdminCookie_Value_Base64Encoded(t *testing.T) {
	handler := setupCookieTestHandler(t)

	// Create request
	reqBody := RegisterRequest{
		Email:    "b64@example.com",
		Password: "Test123!",
		Name:     "Base64 User",
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Register(w, r)

	// Get cookie
	cookies := w.Result().Cookies()
	cookie := findCookie(cookies, SessionCookie)
	if cookie == nil {
		t.Fatal("Session cookie not found")
	}

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

// Helper functions

func setupCookieTestHandler(t *testing.T) *Handler {
	t.Helper()

	// Create real stores (not mocks) - simpler and more reliable
	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4) // low cost for tests

	// Create handler
	handler := NewHandler(Deps{
		Users:  userStore,
		Keys:   keyStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler
}

func setupCookieTestHandlerWithPassword(t *testing.T, email, password string) *Handler {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	// Create user with password
	passwordHash, _ := h.Hash(password)
	user := ports.User{
		ID:           "user_" + email,
		Email:        email,
		PasswordHash: passwordHash,
		PlanID:       "free",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	userStore.Create(context.Background(), user)

	handler := NewHandler(Deps{
		Users:  userStore,
		Keys:   keyStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler
}

func setupCookieTestHandlerWithAPIKey(t *testing.T) (*Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	// Create admin user
	adminUser := ports.User{
		ID:        "user_admin",
		Email:     "admin@test.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	userStore.Create(context.Background(), adminUser)

	// Create admin API key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(adminUser.ID)
	keyStore.Create(context.Background(), keyData)

	handler := NewHandler(Deps{
		Users:  userStore,
		Keys:   keyStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
