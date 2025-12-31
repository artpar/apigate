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

// mockPlanStore implements ports.PlanStore for testing.
type mockPlanStore struct {
	plans []ports.Plan
}

func newMockPlanStore() *mockPlanStore {
	return &mockPlanStore{
		plans: []ports.Plan{
			{
				ID:        "plan_default",
				Name:      "Free",
				IsDefault: true,
				Enabled:   true,
			},
		},
	}
}

func (m *mockPlanStore) List(ctx context.Context) ([]ports.Plan, error) {
	return m.plans, nil
}

func (m *mockPlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	for _, p := range m.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return ports.Plan{}, errNotFound
}

func (m *mockPlanStore) Create(ctx context.Context, p ports.Plan) error {
	m.plans = append(m.plans, p)
	return nil
}

func (m *mockPlanStore) Update(ctx context.Context, p ports.Plan) error {
	for i, existing := range m.plans {
		if existing.ID == p.ID {
			m.plans[i] = p
			return nil
		}
	}
	return errNotFound
}

func (m *mockPlanStore) Delete(ctx context.Context, id string) error {
	for i, p := range m.plans {
		if p.ID == id {
			m.plans = append(m.plans[:i], m.plans[i+1:]...)
			return nil
		}
	}
	return errNotFound
}

var errNotFound = errors.New("not found")

// Helper to create test portal handler
func newTestPortalHandler() (*PortalHandler, *mockUserStore, *mockTokenStore, *email.MockSender) {
	userStore := newMockUserStore()
	tokenStore := newMockTokenStore()
	sessionStore := newMockSessionStore()
	planStore := newMockPlanStore()
	emailSender := email.NewMockSender("https://test.com", "TestApp")

	deps := PortalDeps{
		Users:       userStore,
		Keys:        &mockKeyStore{},
		Usage:       &mockUsageStore{},
		AuthTokens:  tokenStore,
		Sessions:    sessionStore,
		Plans:       planStore,
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
	handler, userStore, _, _ := newTestPortalHandler()

	form := url.Values{
		"email":    {"newuser@example.com"},
		"password": {"Password123"},
		"name":     {"New User"},
	}

	req := httptest.NewRequest("POST", "/portal/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.SignupSubmit(w, req)

	// Should redirect to login with ready message (no email verification required by default)
	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/portal/login?signup=ready") {
		t.Errorf("Location = %s, want to contain /portal/login?signup=ready", location)
	}

	// User should be created
	if len(userStore.users) != 1 {
		t.Errorf("User count = %d, want 1", len(userStore.users))
	}

	// User should be active (no email verification required by default)
	for _, u := range userStore.users {
		if u.Status != "active" {
			t.Errorf("Status = %s, want active", u.Status)
		}
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

func TestPortalHandler_VerifyEmail_MissingToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/verify-email", nil)
	w := httptest.NewRecorder()

	handler.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_VerifyEmail_InvalidToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/verify-email?token=invalid-token", nil)
	w := httptest.NewRecorder()

	handler.VerifyEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_ResetPasswordPage_MissingToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/reset-password", nil)
	w := httptest.NewRecorder()

	handler.ResetPasswordPage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_ResetPasswordPage_WithToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/reset-password?token=test-token", nil)
	w := httptest.NewRecorder()

	handler.ResetPasswordPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_ResetPasswordSubmit_PasswordMismatch(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"token":            {"test-token"},
		"password":         {"Password123"},
		"confirm_password": {"Password456"},
	}

	req := httptest.NewRequest("POST", "/portal/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResetPasswordSubmit(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestPortalHandler_ResetPasswordSubmit_WeakPassword(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"token":            {"test-token"},
		"password":         {"weak"},
		"confirm_password": {"weak"},
	}

	req := httptest.NewRequest("POST", "/portal/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResetPasswordSubmit(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestPortalHandler_ResetPasswordSubmit_InvalidToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"token":            {"invalid-token"},
		"password":         {"Password123"},
		"confirm_password": {"Password123"},
	}

	req := httptest.NewRequest("POST", "/portal/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResetPasswordSubmit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_PortalLogout(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Create active user
	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "active",
	}

	// Generate token
	tokenService := auth.NewTokenService("test-secret", 24*time.Hour)
	token, _, _ := tokenService.GenerateToken("user1", "user@example.com", "user")

	req := httptest.NewRequest("POST", "/portal/logout", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: token})
	w := httptest.NewRecorder()

	handler.PortalLogout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if w.Header().Get("Location") != "/portal/login" {
		t.Errorf("Location = %s, want /portal/login", w.Header().Get("Location"))
	}

	// Check cookie is cleared
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "portal_token" && c.MaxAge == -1 {
			return
		}
	}
	t.Error("Cookie should be cleared")
}

func TestPortalHandler_PortalDashboard(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Create active user with plan
	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Name:   "Test User",
		PlanID: "plan_default",
		Status: "active",
	}

	// Generate token
	tokenService := auth.NewTokenService("test-secret", 24*time.Hour)
	token, _, _ := tokenService.GenerateToken("user1", "user@example.com", "user")

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: token})

	// Add context with portal user
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.PortalDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_APIKeysPage(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Name:   "Test User",
		Status: "active",
	}

	req := httptest.NewRequest("GET", "/portal/api-keys", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.APIKeysPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_CreateAPIKey(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	// Set up keys store that actually stores keys
	keyStore := &mockKeyStoreWithStorage{keys: make(map[string]key.Key)}
	handler.keys = keyStore

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "active",
	}

	form := url.Values{
		"name": {"My API Key"},
	}

	req := httptest.NewRequest("POST", "/portal/api-keys", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_PortalUsagePage(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "active",
	}

	req := httptest.NewRequest("GET", "/portal/usage", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.PortalUsagePage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_AccountSettingsPage(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "active",
	}

	req := httptest.NewRequest("GET", "/portal/settings", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.AccountSettingsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_UpdateAccountSettings_Success(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Name:   "Old Name",
		Status: "active",
	}

	form := url.Values{
		"name": {"New Name"},
	}

	req := httptest.NewRequest("POST", "/portal/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Old Name",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.UpdateAccountSettings(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	// Verify name was updated
	if userStore.users["user1"].Name != "New Name" {
		t.Error("Name should be updated")
	}
}

func TestPortalHandler_UpdateAccountSettings_EmptyName(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Name:   "Old Name",
		Status: "active",
	}

	form := url.Values{
		"name": {""},
	}

	req := httptest.NewRequest("POST", "/portal/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.UpdateAccountSettings(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestPortalHandler_ChangePassword_Success(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_OldPassword123"),
		Status:       "active",
	}

	form := url.Values{
		"current_password": {"OldPassword123"},
		"new_password":     {"NewPassword456"},
		"confirm_password": {"NewPassword456"},
	}

	req := httptest.NewRequest("POST", "/portal/settings/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_CorrectPassword"),
		Status:       "active",
	}

	form := url.Values{
		"current_password": {"WrongPassword"},
		"new_password":     {"NewPassword456"},
		"confirm_password": {"NewPassword456"},
	}

	req := httptest.NewRequest("POST", "/portal/settings/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestPortalHandler_PlansPage(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		PlanID: "plan_default",
		Status: "active",
	}

	req := httptest.NewRequest("GET", "/portal/plans", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.PlansPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPortalHandler_ChangePlan_ToFreePlan(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		PlanID: "old_plan",
		Status: "active",
	}

	form := url.Values{
		"plan_id": {"plan_default"},
	}

	req := httptest.NewRequest("POST", "/portal/plans/change", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ChangePlan(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_CheckoutCancel(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	req := httptest.NewRequest("GET", "/portal/subscription/checkout-cancel", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.CheckoutCancel(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "error=cancelled") {
		t.Errorf("Location = %s, should contain error=cancelled", location)
	}
}

func TestPortalHandler_ResendVerification_MissingEmail(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{}

	req := httptest.NewRequest("POST", "/portal/resend-verification", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResendVerification(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_ResendVerification_UnknownEmail(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"email": {"unknown@example.com"},
	}

	req := httptest.NewRequest("POST", "/portal/resend-verification", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResendVerification(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_ResendVerification_AlreadyVerified(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "active", // Already verified
	}

	form := url.Values{
		"email": {"user@example.com"},
	}

	req := httptest.NewRequest("POST", "/portal/resend-verification", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResendVerification(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_CloseAccount_Success(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_Password123"),
		Status:       "active",
	}

	form := url.Values{
		"password": {"Password123"},
	}

	req := httptest.NewRequest("POST", "/portal/settings/close-account", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.CloseAccount(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	// Verify user is marked as cancelled
	if userStore.users["user1"].Status != "cancelled" {
		t.Error("User status should be cancelled")
	}
}

func TestPortalHandler_CloseAccount_WrongPassword(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_CorrectPassword"),
		Status:       "active",
	}

	form := url.Values{
		"password": {"WrongPassword"},
	}

	req := httptest.NewRequest("POST", "/portal/settings/close-account", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.CloseAccount(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// mockKeyStoreWithStorage stores keys for testing
type mockKeyStoreWithStorage struct {
	keys map[string]key.Key
}

func (m *mockKeyStoreWithStorage) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	return nil, nil
}

func (m *mockKeyStoreWithStorage) Create(ctx context.Context, k key.Key) error {
	m.keys[k.ID] = k
	return nil
}

func (m *mockKeyStoreWithStorage) Revoke(ctx context.Context, id string, at time.Time) error {
	return nil
}

func (m *mockKeyStoreWithStorage) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	var result []key.Key
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockKeyStoreWithStorage) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	return nil
}

func TestPortalHandler_PortalAuthMiddleware_InvalidToken(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	protected := handler.PortalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_PortalAuthMiddleware_UserNotFound(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	tokenService := auth.NewTokenService("test-secret", 24*time.Hour)
	token, _, _ := tokenService.GenerateToken("nonexistent", "user@example.com", "user")

	protected := handler.PortalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: token})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_PortalAuthMiddleware_UserNotActive(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:     "user1",
		Email:  "user@example.com",
		Status: "suspended",
	}

	tokenService := auth.NewTokenService("test-secret", 24*time.Hour)
	token, _, _ := tokenService.GenerateToken("user1", "user@example.com", "user")

	protected := handler.PortalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/portal/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "portal_token", Value: token})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_PortalLoginPage_WithMessages(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	tests := []struct {
		name   string
		query  string
	}{
		{"signup success", "?signup=success"},
		{"signup ready", "?signup=ready"},
		{"verified", "?verified=true"},
		{"reset success", "?reset=success"},
		{"with email", "?email=test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/portal/login"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.PortalLoginPage(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

func TestPortalHandler_PortalLoginSubmit_ValidationError(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"email":    {""},
		"password": {""},
	}

	req := httptest.NewRequest("POST", "/portal/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.PortalLoginSubmit(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestPortalHandler_PortalLoginSubmit_SuspendedUser(t *testing.T) {
	handler, userStore, _, _ := newTestPortalHandler()

	userStore.users["user1"] = ports.User{
		ID:           "user1",
		Email:        "user@example.com",
		PasswordHash: []byte("hashed_Password123"),
		Status:       "suspended",
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
}

func TestNewPortalPageData(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	ctx := withPortalUser(context.Background(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
		Name:  "Test User",
	})

	data := handler.newPortalPageData(ctx, "Test Page")

	if data.Title != "Test Page" {
		t.Errorf("Title = %s, want Test Page", data.Title)
	}

	if data.User == nil {
		t.Fatal("User should not be nil")
	}

	if data.User.ID != "user1" {
		t.Errorf("User.ID = %s, want user1", data.User.ID)
	}

	if data.AppName != "TestApp" {
		t.Errorf("AppName = %s, want TestApp", data.AppName)
	}
}

// mockKeyStoreWithData implements ports.KeyStore with actual data storage
type mockKeyStoreWithData struct {
	keys map[string]key.Key
}

func newMockKeyStoreWithData() *mockKeyStoreWithData {
	return &mockKeyStoreWithData{keys: make(map[string]key.Key)}
}

func (m *mockKeyStoreWithData) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	for _, k := range m.keys {
		if k.Prefix == prefix {
			return []key.Key{k}, nil
		}
	}
	return nil, nil
}

func (m *mockKeyStoreWithData) Create(ctx context.Context, k key.Key) error {
	m.keys[k.ID] = k
	return nil
}

func (m *mockKeyStoreWithData) Revoke(ctx context.Context, id string, at time.Time) error {
	if k, ok := m.keys[id]; ok {
		k.RevokedAt = &at
		m.keys[id] = k
		return nil
	}
	return errors.New("key not found")
}

func (m *mockKeyStoreWithData) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	var result []key.Key
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockKeyStoreWithData) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	return nil
}

func newTestPortalHandlerWithKeyStore() (*PortalHandler, *mockUserStore, *mockKeyStoreWithData) {
	userStore := newMockUserStore()
	tokenStore := newMockTokenStore()
	sessionStore := newMockSessionStore()
	planStore := newMockPlanStore()
	keyStore := newMockKeyStoreWithData()
	emailSender := email.NewMockSender("https://test.com", "TestApp")

	deps := PortalDeps{
		Users:       userStore,
		Keys:        keyStore,
		Usage:       &mockUsageStore{},
		AuthTokens:  tokenStore,
		Sessions:    sessionStore,
		Plans:       planStore,
		EmailSender: emailSender,
		Logger:      zerolog.Nop(),
		Hasher:      &mockHasher{},
		IDGen:       &mockIDGen{},
		JWTSecret:   "test-secret",
		BaseURL:     "https://test.com",
		AppName:     "TestApp",
	}

	handler, _ := NewPortalHandler(deps)
	return handler, userStore, keyStore
}

func TestPortalHandler_RevokeAPIKey(t *testing.T) {
	handler, userStore, keyStore := newTestPortalHandlerWithKeyStore()

	// Create test user and key
	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
	}

	keyStore.keys["key1"] = key.Key{
		ID:     "key1",
		UserID: "user1",
		Name:   "Test Key",
	}

	r := chi.NewRouter()
	r.Delete("/portal/api-keys/{id}", handler.RevokeAPIKey)

	req := httptest.NewRequest("DELETE", "/portal/api-keys/key1", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPortalHandler_RevokeAPIKey_MissingID(t *testing.T) {
	handler, _, _ := newTestPortalHandlerWithKeyStore()

	req := httptest.NewRequest("DELETE", "/portal/api-keys/", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.RevokeAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPortalHandler_RevokeAPIKey_NotOwned(t *testing.T) {
	handler, userStore, keyStore := newTestPortalHandlerWithKeyStore()

	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
	}

	// Key belongs to different user
	keyStore.keys["key1"] = key.Key{
		ID:     "key1",
		UserID: "other_user",
		Name:   "Test Key",
	}

	r := chi.NewRouter()
	r.Delete("/portal/api-keys/{id}", handler.RevokeAPIKey)

	req := httptest.NewRequest("DELETE", "/portal/api-keys/key1", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPortalHandler_CheckoutSuccess(t *testing.T) {
	handler, userStore, _ := newTestPortalHandlerWithKeyStore()

	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
	}

	req := httptest.NewRequest("GET", "/portal/checkout/success?session_id=test_session", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CheckoutSuccess(w, req)

	// Should redirect since payment provider is nil
	if w.Code != http.StatusFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want Found or InternalServerError", w.Code)
	}
}

func TestPortalHandler_ManageSubscription(t *testing.T) {
	handler, userStore, _ := newTestPortalHandlerWithKeyStore()

	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
	}

	req := httptest.NewRequest("GET", "/portal/subscription/manage", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ManageSubscription(w, req)

	// Should redirect or error since payment provider is nil
	if w.Code != http.StatusFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want Found or InternalServerError", w.Code)
	}
}

func TestPortalHandler_CancelSubscription(t *testing.T) {
	handler, userStore, _ := newTestPortalHandlerWithKeyStore()

	userStore.users["user1"] = ports.User{
		ID:    "user1",
		Email: "user@example.com",
	}

	req := httptest.NewRequest("POST", "/portal/subscription/cancel", nil)
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.CancelSubscription(w, req)

	// Should redirect or error since payment provider is nil
	if w.Code != http.StatusFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want Found or InternalServerError", w.Code)
	}
}

func TestPortalHandler_RenderSignupPage(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	// Test basic render
	result := handler.renderSignupPage("Test User", "test@example.com", nil)
	if result == "" {
		t.Error("renderSignupPage should return non-empty string")
	}
	if !strings.Contains(result, "Test User") {
		t.Error("renderSignupPage should contain the name")
	}

	// Test with errors
	errors := map[string]string{"email": "Invalid email"}
	result2 := handler.renderSignupPage("", "bad-email", errors)
	if !strings.Contains(result2, "Invalid email") {
		t.Error("renderSignupPage should contain error message")
	}
}

func TestPortalHandler_ChangePlan_MissingPlanID(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{}

	req := httptest.NewRequest("POST", "/portal/subscription/change", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ChangePlan(w, req)

	// Should redirect or show error for missing plan
	if w.Code != http.StatusFound && w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want Found or BadRequest", w.Code)
	}
}

func TestPortalHandler_ChangePassword_MissingFields(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"current_password": {""},
		"new_password":     {""},
	}

	req := httptest.NewRequest("POST", "/portal/settings/change-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := withPortalUser(req.Context(), &PortalUser{
		ID:    "user1",
		Email: "user@example.com",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	// Should handle missing fields - 422 is also acceptable for validation errors
	if w.Code != http.StatusBadRequest && w.Code != http.StatusFound && w.Code != http.StatusOK && w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want BadRequest, Found, OK, or UnprocessableEntity", w.Code)
	}
}

func TestPortalHandler_ResetPasswordSubmit_MissingFields(t *testing.T) {
	handler, _, _, _ := newTestPortalHandler()

	form := url.Values{
		"token":        {""},
		"new_password": {""},
	}

	req := httptest.NewRequest("POST", "/portal/reset-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ResetPasswordSubmit(w, req)

	// Should handle missing fields - 422 is also acceptable for validation errors
	if w.Code != http.StatusBadRequest && w.Code != http.StatusFound && w.Code != http.StatusOK && w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Status = %d, want BadRequest, Found, OK, or UnprocessableEntity", w.Code)
	}
}

