package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// mockInviteStore implements ports.InviteStore for testing.
type mockInviteStore struct {
	invites      map[string]ports.AdminInvite
	byTokenHash  map[string]ports.AdminInvite // keyed by hex of hash
	createErr    error
	getErr       error
	deleteErr    error
	markUsedErr  error
}

func newMockInviteStore() *mockInviteStore {
	return &mockInviteStore{
		invites:     make(map[string]ports.AdminInvite),
		byTokenHash: make(map[string]ports.AdminInvite),
	}
}

func (m *mockInviteStore) Create(ctx context.Context, invite ports.AdminInvite) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.invites[invite.ID] = invite
	// Store by token hash as hex string for lookup
	hashKey := string(invite.TokenHash)
	m.byTokenHash[hashKey] = invite
	return nil
}

func (m *mockInviteStore) GetByTokenHash(ctx context.Context, hash []byte) (ports.AdminInvite, error) {
	if m.getErr != nil {
		return ports.AdminInvite{}, m.getErr
	}
	hashKey := string(hash)
	if inv, ok := m.byTokenHash[hashKey]; ok {
		return inv, nil
	}
	return ports.AdminInvite{}, errors.New("not found")
}

func (m *mockInviteStore) List(ctx context.Context, limit, offset int) ([]ports.AdminInvite, error) {
	var result []ports.AdminInvite
	for _, inv := range m.invites {
		result = append(result, inv)
	}
	return result, nil
}

func (m *mockInviteStore) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	if m.markUsedErr != nil {
		return m.markUsedErr
	}
	if inv, ok := m.invites[id]; ok {
		inv.UsedAt = &usedAt
		m.invites[id] = inv
		// Also update byTokenHash
		hashKey := string(inv.TokenHash)
		m.byTokenHash[hashKey] = inv
		return nil
	}
	return errors.New("not found")
}

func (m *mockInviteStore) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if inv, ok := m.invites[id]; ok {
		delete(m.invites, id)
		hashKey := string(inv.TokenHash)
		delete(m.byTokenHash, hashKey)
	}
	return nil
}

func (m *mockInviteStore) DeleteExpired(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now()
	for id, inv := range m.invites {
		if inv.ExpiresAt.Before(now) && inv.UsedAt == nil {
			delete(m.invites, id)
			count++
		}
	}
	return count, nil
}

func (m *mockInviteStore) Count(ctx context.Context) (int, error) {
	return len(m.invites), nil
}

// addInvite is a helper to add an invite with a known token
func (m *mockInviteStore) addInvite(id, email, token, createdBy string, expiresAt time.Time, usedAt *time.Time) {
	hash := sha256.Sum256([]byte(token))
	invite := ports.AdminInvite{
		ID:        id,
		Email:     email,
		TokenHash: hash[:],
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		UsedAt:    usedAt,
	}
	m.invites[id] = invite
	m.byTokenHash[string(hash[:])] = invite
}

// Create test handler with invites
func newTestHandlerWithInvites() (*Handler, *mockUsers, *mockInviteStore) {
	users := newMockUsers()
	invites := newMockInviteStore()

	h := &Handler{
		templates: make(map[string]*template.Template),
		tokens:    auth.NewTokenService("test-secret", 24*time.Hour),
		users:     users,
		invites:   invites,
		logger:    zerolog.Nop(),
		hasher:    &mockHash{},
		isSetup:   func() bool { return true },
	}

	return h, users, invites
}

func TestHandler_InvitesPage(t *testing.T) {
	h, users, invites := newTestHandlerWithInvites()

	// Add a user who created invites
	users.users["admin-1"] = ports.User{ID: "admin-1", Email: "admin@example.com"}

	// Add an invite
	invites.addInvite("inv-1", "newadmin@example.com", "testtoken123", "admin-1", time.Now().Add(24*time.Hour), nil)

	// Create a simple template for testing
	tmpl, _ := template.New("base").Parse(`{{define "invites"}}invites page{{end}}`)
	h.templates["invites"] = tmpl

	// Create authenticated request
	token, _, _ := h.tokens.GenerateToken("admin-1", "admin@example.com", "admin")

	req := httptest.NewRequest("GET", "/invites", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	req = req.WithContext(withClaims(req.Context(), &auth.Claims{
		UserID: "admin-1",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
	w := httptest.NewRecorder()

	h.InvitesPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_InviteCreate_Success(t *testing.T) {
	h, users, _ := newTestHandlerWithInvites()

	// Add admin user
	users.users["admin-1"] = ports.User{ID: "admin-1", Email: "admin@example.com"}

	form := url.Values{}
	form.Set("email", "newadmin@example.com")

	req := httptest.NewRequest("POST", "/invites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(withClaims(req.Context(), &auth.Claims{
		UserID: "admin-1",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
	w := httptest.NewRecorder()

	h.InviteCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "success=created") {
		t.Errorf("expected redirect to success, got %s", location)
	}
}

func TestHandler_InviteCreate_EmptyEmail(t *testing.T) {
	h, _, _ := newTestHandlerWithInvites()

	form := url.Values{}
	form.Set("email", "")

	req := httptest.NewRequest("POST", "/invites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(withClaims(req.Context(), &auth.Claims{
		UserID: "admin-1",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
	w := httptest.NewRecorder()

	h.InviteCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "error=email_required") {
		t.Errorf("expected error=email_required, got %s", location)
	}
}

func TestHandler_InviteCreate_UserExists(t *testing.T) {
	h, users, _ := newTestHandlerWithInvites()

	// Add existing user
	users.users["user-1"] = ports.User{ID: "user-1", Email: "existing@example.com"}

	form := url.Values{}
	form.Set("email", "existing@example.com")

	req := httptest.NewRequest("POST", "/invites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(withClaims(req.Context(), &auth.Claims{
		UserID: "admin-1",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
	w := httptest.NewRecorder()

	h.InviteCreate(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "error=user_exists") {
		t.Errorf("expected error=user_exists, got %s", location)
	}
}

func TestHandler_InviteDelete_Success(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	// Add an invite
	invites.addInvite("inv-1", "test@example.com", "token123", "admin-1", time.Now().Add(24*time.Hour), nil)

	// Create router to get URL params
	r := chi.NewRouter()
	r.Delete("/invites/{id}", h.InviteDelete)

	req := httptest.NewRequest("DELETE", "/invites/inv-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
}

func TestHandler_InviteDelete_HTMX(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	// Add an invite
	invites.addInvite("inv-1", "test@example.com", "token123", "admin-1", time.Now().Add(24*time.Hour), nil)

	r := chi.NewRouter()
	r.Delete("/invites/{id}", h.InviteDelete)

	req := httptest.NewRequest("DELETE", "/invites/inv-1", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for HTMX, got %d", w.Code)
	}
}

func TestHandler_InviteDelete_Error(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()
	invites.deleteErr = errors.New("delete failed")

	r := chi.NewRouter()
	r.Delete("/invites/{id}", h.InviteDelete)

	req := httptest.NewRequest("DELETE", "/invites/inv-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterPage_ValidToken(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "validtoken123"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), nil)

	// Create a simple template
	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}register page for {{.Email}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Get("/admin/register/{token}", h.AdminRegisterPage)

	req := httptest.NewRequest("GET", "/admin/register/"+token, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterPage_InvalidToken(t *testing.T) {
	h, _, _ := newTestHandlerWithInvites()

	// Create template
	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error: {{.Error}}{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Get("/admin/register/{token}", h.AdminRegisterPage)

	req := httptest.NewRequest("GET", "/admin/register/invalidtoken", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should still return 200 with error message
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterPage_ExpiredToken(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "expiredtoken"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(-24*time.Hour), nil)

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Get("/admin/register/{token}", h.AdminRegisterPage)

	req := httptest.NewRequest("GET", "/admin/register/"+token, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterPage_UsedToken(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "usedtoken"
	usedAt := time.Now()
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), &usedAt)

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Get("/admin/register/{token}", h.AdminRegisterPage)

	req := httptest.NewRequest("GET", "/admin/register/"+token, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterSubmit_Success(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "validtoken123"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), nil)

	r := chi.NewRouter()
	r.Post("/admin/register/{token}", h.AdminRegisterSubmit)

	form := url.Values{}
	form.Set("name", "New Admin")
	form.Set("password", "password123")
	form.Set("confirm_password", "password123")

	req := httptest.NewRequest("POST", "/admin/register/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "success=registered") {
		t.Errorf("expected redirect to success, got %s", location)
	}
}

func TestHandler_AdminRegisterSubmit_InvalidToken(t *testing.T) {
	h, _, _ := newTestHandlerWithInvites()

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Post("/admin/register/{token}", h.AdminRegisterSubmit)

	form := url.Values{}
	form.Set("name", "New Admin")
	form.Set("password", "password123")
	form.Set("confirm_password", "password123")

	req := httptest.NewRequest("POST", "/admin/register/invalidtoken", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should render error page
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterSubmit_PasswordMismatch(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "validtoken123"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), nil)

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error: {{.Error}}{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Post("/admin/register/{token}", h.AdminRegisterSubmit)

	form := url.Values{}
	form.Set("name", "New Admin")
	form.Set("password", "password123")
	form.Set("confirm_password", "differentpassword")

	req := httptest.NewRequest("POST", "/admin/register/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// The template renders the error - we just verify it returned 200 (form re-render)
}

func TestHandler_AdminRegisterSubmit_ShortPassword(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "validtoken123"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), nil)

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Post("/admin/register/{token}", h.AdminRegisterSubmit)

	form := url.Values{}
	form.Set("name", "New Admin")
	form.Set("password", "short")
	form.Set("confirm_password", "short")

	req := httptest.NewRequest("POST", "/admin/register/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AdminRegisterSubmit_EmptyName(t *testing.T) {
	h, _, invites := newTestHandlerWithInvites()

	token := "validtoken123"
	invites.addInvite("inv-1", "newadmin@example.com", token, "admin-1", time.Now().Add(24*time.Hour), nil)

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{if .Error}}error{{end}}{{end}}`)
	h.templates["admin_register"] = tmpl

	r := chi.NewRouter()
	r.Post("/admin/register/{token}", h.AdminRegisterSubmit)

	form := url.Values{}
	form.Set("name", "")
	form.Set("password", "password123")
	form.Set("confirm_password", "password123")

	req := httptest.NewRequest("POST", "/admin/register/"+token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_getBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		expected string
	}{
		{
			name:     "http request",
			req:      httptest.NewRequest("GET", "http://example.com/path", nil),
			expected: "http://example.com",
		},
		{
			name: "https forwarded",
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "http://example.com/path", nil)
				r.Header.Set("X-Forwarded-Proto", "https")
				return r
			}(),
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBaseURL(tt.req)
			if result != tt.expected {
				t.Errorf("getBaseURL() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestHandler_renderAdminRegisterError(t *testing.T) {
	h, _, _ := newTestHandlerWithInvites()

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{.Error}}{{end}}`)
	h.templates["admin_register"] = tmpl

	req := httptest.NewRequest("GET", "/admin/register/token", nil)
	w := httptest.NewRecorder()

	h.renderAdminRegisterError(w, req, "Test error", "test@example.com")

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_renderAdminRegisterFormError(t *testing.T) {
	h, _, _ := newTestHandlerWithInvites()

	tmpl, _ := template.New("base").Parse(`{{define "admin_register"}}{{.Error}}{{end}}`)
	h.templates["admin_register"] = tmpl

	req := httptest.NewRequest("GET", "/admin/register/token", nil)
	w := httptest.NewRecorder()

	h.renderAdminRegisterFormError(w, req, "Form error", "test@example.com", "token123")

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Suppress unused variable warning
var _ = bytes.Buffer{}
