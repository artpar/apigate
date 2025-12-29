package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/adapters/email"
	domainAuth "github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// mockUserStore implements ports.UserStore for testing.
type mockUserStore struct {
	users map[string]ports.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]ports.User)}
}

func (m *mockUserStore) Get(ctx context.Context, id string) (ports.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return ports.User{}, errNotFound
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return ports.User{}, errNotFound
}

func (m *mockUserStore) Create(ctx context.Context, u ports.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserStore) Update(ctx context.Context, u ports.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return errNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserStore) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	var result []ports.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

func (m *mockUserStore) Count(ctx context.Context) (int, error) {
	return len(m.users), nil
}

// mockTokenStore implements ports.TokenStore for testing.
type mockTokenStore struct {
	tokens map[string]domainAuth.Token
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]domainAuth.Token)}
}

func (m *mockTokenStore) Create(ctx context.Context, token domainAuth.Token) error {
	m.tokens[token.ID] = token
	return nil
}

func (m *mockTokenStore) GetByHash(ctx context.Context, hash []byte) (domainAuth.Token, error) {
	for _, t := range m.tokens {
		if string(t.Hash) == string(hash) {
			return t, nil
		}
	}
	return domainAuth.Token{}, errNotFound
}

func (m *mockTokenStore) GetByUserAndType(ctx context.Context, userID string, tokenType domainAuth.TokenType) (domainAuth.Token, error) {
	for _, t := range m.tokens {
		if t.UserID == userID && t.Type == tokenType {
			return t, nil
		}
	}
	return domainAuth.Token{}, errNotFound
}

func (m *mockTokenStore) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	if t, ok := m.tokens[id]; ok {
		t.UsedAt = &usedAt
		m.tokens[id] = t
		return nil
	}
	return errNotFound
}

func (m *mockTokenStore) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockTokenStore) DeleteByUser(ctx context.Context, userID string) error {
	for id, t := range m.tokens {
		if t.UserID == userID {
			delete(m.tokens, id)
		}
	}
	return nil
}

// mockSessionStore implements ports.SessionStore for testing.
type mockSessionStore struct {
	sessions map[string]domainAuth.Session
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]domainAuth.Session)}
}

func (m *mockSessionStore) Create(ctx context.Context, session domainAuth.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionStore) Get(ctx context.Context, id string) (domainAuth.Session, error) {
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return domainAuth.Session{}, errNotFound
}

func (m *mockSessionStore) Delete(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	for id, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

// mockHasher implements ports.Hasher for testing.
type mockHasher struct{}

func (m *mockHasher) Hash(plaintext string) ([]byte, error) {
	return []byte("hashed_" + plaintext), nil
}

func (m *mockHasher) Compare(hash []byte, plaintext string) bool {
	return string(hash) == "hashed_"+plaintext
}

// mockIDGen implements ports.IDGenerator for testing.
type mockIDGen struct {
	counter int
}

func (m *mockIDGen) New() string {
	m.counter++
	return "id_" + string(rune('0'+m.counter))
}

// mockKeyStore implements ports.KeyStore for testing.
type mockKeyStore struct{}

func (m *mockKeyStore) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	return nil, nil
}
func (m *mockKeyStore) Create(ctx context.Context, k key.Key) error                { return nil }
func (m *mockKeyStore) Revoke(ctx context.Context, id string, at time.Time) error  { return nil }
func (m *mockKeyStore) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	return nil, nil
}
func (m *mockKeyStore) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	return nil
}

// mockUsageStore implements ports.UsageStore for testing.
type mockUsageStore struct{}

func (m *mockUsageStore) RecordBatch(ctx context.Context, events []usage.Event) error { return nil }
func (m *mockUsageStore) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	return usage.Summary{}, nil
}
func (m *mockUsageStore) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	return nil, nil
}
func (m *mockUsageStore) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	return nil, nil
}

var errNotFound = errors.New("not found")

// Helper to create test portal handler
func newTestPortalHandler() (*PortalHandler, *mockUserStore, *mockTokenStore, *email.MockSender) {
	userStore := newMockUserStore()
	tokenStore := newMockTokenStore()
	sessionStore := newMockSessionStore()
	emailSender := email.NewMockSender("https://test.com", "TestApp")

	deps := PortalDeps{
		Users:       userStore,
		AuthTokens:  tokenStore,
		Sessions:    sessionStore,
		EmailSender: emailSender,
		Logger:      zerolog.Nop(),
		Hasher:      &mockHasher{},
		IDGen:       &mockIDGen{},
		JWTSecret:   "test-secret",
		BaseURL:     "https://test.com",
		AppName:     "TestApp",
	}

	handler, _ := NewPortalHandler(deps)
	return handler, userStore, tokenStore, emailSender
}

func TestPortalHandler_SignupPage(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/signup", nil)
	w := httptest.NewRecorder()

	handler.SignupPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Create your account") {
		t.Error("Page should contain 'Create your account'")
	}
	if !strings.Contains(body, "TestApp") {
		t.Error("Page should contain app name")
	}
}

func TestPortalHandler_SignupSubmit_Success(t *testing.T) {
	handler, userStore, _, emailSender := newTestPortalHandler()

	form := url.Values{
		"email":    {"newuser@example.com"},
		"password": {"Password123"},
		"name":     {"New User"},
	}

	req := httptest.NewRequest("POST", "/portal/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.SignupSubmit(w, req)

	// Should redirect to login with success message
	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/portal/login?signup=success") {
		t.Errorf("Location = %s, want /portal/login?signup=success", location)
	}

	// User should be created
	if len(userStore.users) != 1 {
		t.Errorf("User count = %d, want 1", len(userStore.users))
	}

	// User should be pending (not active until email verified)
	for _, u := range userStore.users {
		if u.Status != "pending" {
			t.Errorf("Status = %s, want pending", u.Status)
		}
	}

	// Verification email should be sent
	if emailSender.Count() != 1 {
		t.Errorf("Email count = %d, want 1", emailSender.Count())
	}
}

func TestPortalHandler_SignupSubmit_ValidationError(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	tests := []struct {
		name     string
		email    string
		password string
		userName string
		wantErr  string
	}{
		{
			name:     "empty email",
			email:    "",
			password: "Password123",
			userName: "Test",
			wantErr:  "Email is required",
		},
		{
			name:     "invalid email",
			email:    "notanemail",
			password: "Password123",
			userName: "Test",
			wantErr:  "Invalid email",
		},
		{
			name:     "short password",
			email:    "test@example.com",
			password: "short",
			userName: "Test",
			wantErr:  "at least 8 characters",
		},
		{
			name:     "weak password",
			email:    "test@example.com",
			password: "password",
			userName: "Test",
			wantErr:  "uppercase, lowercase, and number",
		},
		{
			name:     "empty name",
			email:    "test@example.com",
			password: "Password123",
			userName: "",
			wantErr:  "Name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{
				"email":    {tt.email},
				"password": {tt.password},
				"name":     {tt.userName},
			}

			req := httptest.NewRequest("POST", "/portal/signup", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.SignupSubmit(w, req)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.wantErr) {
				t.Errorf("Body should contain %q", tt.wantErr)
			}
		})
	}
}

func TestPortalHandler_SignupSubmit_DuplicateEmail(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Create existing user
	userStore.users["existing"] = ports.User{
		ID:    "existing",
		Email: "existing@example.com",
	}

	form := url.Values{
		"email":    {"existing@example.com"},
		"password": {"Password123"},
		"name":     {"New User"},
	}

	req := httptest.NewRequest("POST", "/portal/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.SignupSubmit(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusConflict)
	}

	body := w.Body.String()
	if !strings.Contains(body, "already registered") {
		t.Error("Body should contain 'already registered'")
	}
}

func TestPortalHandler_PortalLoginPage(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/login", nil)
	w := httptest.NewRecorder()

	handler.PortalLoginPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Log in to your account") {
		t.Error("Page should contain 'Log in to your account'")
	}
}

func TestPortalHandler_PortalLoginSubmit_Success(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Create active user with hashed password
	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_Password123"),
		Status:       "active",
	}

	form := url.Values{
		"email":    {"user@example.com"},
		"password": {"Password123"},
	}

	req := httptest.NewRequest("POST", "/portal/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.PortalLoginSubmit(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location != "/portal/dashboard" {
		t.Errorf("Location = %s, want /portal/dashboard", location)
	}

	// Should set cookie
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "portal_token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("portal_token cookie should be set")
	}
}

func TestPortalHandler_PortalLoginSubmit_WrongPassword(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_CorrectPassword"),
		Status:       "active",
	}

	form := url.Values{
		"email":    {"user@example.com"},
		"password": {"WrongPassword"},
	}

	req := httptest.NewRequest("POST", "/portal/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.PortalLoginSubmit(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid email or password") {
		t.Error("Body should contain error message")
	}
}

func TestPortalHandler_PortalLoginSubmit_PendingUser(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_Password123"),
		Status:       "pending",
	}

	form := url.Values{
		"email":    {"user@example.com"},
		"password": {"Password123"},
	}

	req := httptest.NewRequest("POST", "/portal/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.PortalLoginSubmit(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusForbidden)
	}

	body := w.Body.String()
	if !strings.Contains(body, "verify your email") {
		t.Error("Body should mention email verification")
	}
}

func TestPortalHandler_ForgotPasswordPage(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/forgot-password", nil)
	w := httptest.NewRecorder()

	handler.ForgotPasswordPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Reset your password") {
		t.Error("Page should contain 'Reset your password'")
	}
}

func TestPortalHandler_ForgotPasswordSubmit(t *testing.T) {
	handler, userStore, _, emailSender := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	}

	form := url.Values{
		"email": {"user@example.com"},
	}

	req := httptest.NewRequest("POST", "/portal/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ForgotPasswordSubmit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	// Should always show success (prevent enumeration)
	body := w.Body.String()
	if !strings.Contains(body, "receive a password reset link") {
		t.Error("Body should show success message")
	}

	// Reset email should be sent
	emails := emailSender.FindByType("password_reset")
	if len(emails) != 1 {
		t.Errorf("Password reset email count = %d, want 1", len(emails))
	}
}

func TestPortalHandler_ForgotPasswordSubmit_UnknownEmail(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"email": {"unknown@example.com"},
	}

	req := httptest.NewRequest("POST", "/portal/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ForgotPasswordSubmit(w, req)

	// Should still show success (prevent email enumeration)
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "receive a password reset link") {
		t.Error("Body should show success message even for unknown email")
	}
}

func TestPortalHandler_PortalAuthMiddleware_NoToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	// Create a protected handler
	protected := handler.PortalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location != "/portal/login" {
		t.Errorf("Location = %s, want /portal/login", location)
	}
}

func TestPortalHandler_PortalAuthMiddleware_ValidToken(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Create active user
	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Name:   "Test User",
		Status: "active",
	}

	// Generate a valid token
	tokenService := auth.NewTokenService("test-secret", 24*time.Hour)
	token, _, _ := tokenService.GenerateToken("user1", "user@example.com", "user")

	// Create a protected handler
	var gotUser *PortalUser
	protected := handler.PortalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = getPortalUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: token})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if gotUser == nil {
		t.Fatal("User should be in context")
	}
	if gotUser.ID != "user1" {
		t.Errorf("User ID = %s, want user1", gotUser.ID)
	}
}

func TestPortalHandler_Router(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()
	router := handler.Router()

	// Test that routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/signup"},
		{"POST", "/signup"},
		{"GET", "/login"},
		{"POST", "/login"},
		{"GET", "/forgot-password"},
		{"POST", "/forgot-password"},
		{"GET", "/reset-password"},
		{"POST", "/reset-password"},
		{"GET", "/verify-email"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			r := chi.NewRouter()
			r.Mount("/portal", router)

			req := httptest.NewRequest(route.method, "/portal"+route.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// Should not be 404 (route exists)
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s should exist", route.method, route.path)
			}
		})
	}
}
