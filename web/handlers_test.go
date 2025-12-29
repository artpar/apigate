package web

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// Test mocks

type mockUsers struct {
	users map[string]ports.User
}

func newMockUsers() *mockUsers {
	return &mockUsers{users: make(map[string]ports.User)}
}

func (m *mockUsers) Get(ctx context.Context, id string) (ports.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return ports.User{}, errors.New("not found")
}

func (m *mockUsers) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return ports.User{}, errors.New("not found")
}

func (m *mockUsers) Create(ctx context.Context, u ports.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUsers) Update(ctx context.Context, u ports.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return errors.New("not found")
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUsers) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUsers) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	var result []ports.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

func (m *mockUsers) Count(ctx context.Context) (int, error) {
	return len(m.users), nil
}

type mockKeys struct {
	keys map[string]key.Key
}

func newMockKeys() *mockKeys {
	return &mockKeys{keys: make(map[string]key.Key)}
}

func (m *mockKeys) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	var result []key.Key
	for _, k := range m.keys {
		if strings.HasPrefix(k.ID, prefix) {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockKeys) Create(ctx context.Context, k key.Key) error {
	m.keys[k.ID] = k
	return nil
}

func (m *mockKeys) Revoke(ctx context.Context, id string, at time.Time) error {
	if k, ok := m.keys[id]; ok {
		k.RevokedAt = &at
		m.keys[id] = k
		return nil
	}
	return errors.New("not found")
}

func (m *mockKeys) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	var result []key.Key
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *mockKeys) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	if k, ok := m.keys[id]; ok {
		k.LastUsed = &at
		m.keys[id] = k
		return nil
	}
	return errors.New("not found")
}

type mockUsage struct{}

func (m *mockUsage) RecordBatch(ctx context.Context, events []usage.Event) error { return nil }
func (m *mockUsage) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	return usage.Summary{}, nil
}
func (m *mockUsage) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	return nil, nil
}
func (m *mockUsage) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	return nil, nil
}

type mockPlans struct {
	plans map[string]ports.Plan
}

func newMockPlans() *mockPlans {
	return &mockPlans{plans: make(map[string]ports.Plan)}
}

func (m *mockPlans) Get(ctx context.Context, id string) (ports.Plan, error) {
	if p, ok := m.plans[id]; ok {
		return p, nil
	}
	return ports.Plan{}, errors.New("not found")
}

func (m *mockPlans) Create(ctx context.Context, p ports.Plan) error {
	if _, exists := m.plans[p.ID]; exists {
		return errors.New("already exists")
	}
	m.plans[p.ID] = p
	return nil
}

func (m *mockPlans) Update(ctx context.Context, p ports.Plan) error {
	if _, ok := m.plans[p.ID]; !ok {
		return errors.New("not found")
	}
	m.plans[p.ID] = p
	return nil
}

func (m *mockPlans) Delete(ctx context.Context, id string) error {
	delete(m.plans, id)
	return nil
}

func (m *mockPlans) List(ctx context.Context) ([]ports.Plan, error) {
	var result []ports.Plan
	for _, p := range m.plans {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockPlans) GetDefault(ctx context.Context) (ports.Plan, error) {
	for _, p := range m.plans {
		if p.IsDefault {
			return p, nil
		}
	}
	return ports.Plan{}, errors.New("not found")
}

type mockRoutes struct {
	routes map[string]route.Route
}

func newMockRoutes() *mockRoutes {
	return &mockRoutes{routes: make(map[string]route.Route)}
}

func (m *mockRoutes) Get(ctx context.Context, id string) (route.Route, error) {
	if r, ok := m.routes[id]; ok {
		return r, nil
	}
	return route.Route{}, errors.New("not found")
}

func (m *mockRoutes) Create(ctx context.Context, r route.Route) error {
	m.routes[r.ID] = r
	return nil
}

func (m *mockRoutes) Update(ctx context.Context, r route.Route) error {
	m.routes[r.ID] = r
	return nil
}

func (m *mockRoutes) Delete(ctx context.Context, id string) error {
	delete(m.routes, id)
	return nil
}

func (m *mockRoutes) List(ctx context.Context) ([]route.Route, error) {
	var result []route.Route
	for _, r := range m.routes {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRoutes) ListEnabled(ctx context.Context) ([]route.Route, error) {
	return m.List(ctx)
}

type mockUpstreams struct {
	upstreams map[string]route.Upstream
}

func newMockUpstreams() *mockUpstreams {
	return &mockUpstreams{upstreams: make(map[string]route.Upstream)}
}

func (m *mockUpstreams) Get(ctx context.Context, id string) (route.Upstream, error) {
	if u, ok := m.upstreams[id]; ok {
		return u, nil
	}
	return route.Upstream{}, errors.New("not found")
}

func (m *mockUpstreams) Create(ctx context.Context, u route.Upstream) error {
	m.upstreams[u.ID] = u
	return nil
}

func (m *mockUpstreams) Update(ctx context.Context, u route.Upstream) error {
	m.upstreams[u.ID] = u
	return nil
}

func (m *mockUpstreams) Delete(ctx context.Context, id string) error {
	delete(m.upstreams, id)
	return nil
}

func (m *mockUpstreams) List(ctx context.Context) ([]route.Upstream, error) {
	var result []route.Upstream
	for _, u := range m.upstreams {
		result = append(result, u)
	}
	return result, nil
}

func (m *mockUpstreams) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	return m.List(ctx)
}

type mockHash struct{}

func (m *mockHash) Hash(plaintext string) ([]byte, error) {
	return []byte("hash_" + plaintext), nil
}

func (m *mockHash) Compare(hash []byte, plaintext string) bool {
	return string(hash) == "hash_"+plaintext
}

type mockExprValidator struct{}

func (m *mockExprValidator) ValidateExpr(expression, context string) app.ExprValidationResult {
	return app.ExprValidationResult{Valid: true}
}

type mockRouteTester struct{}

func (m *mockRouteTester) TestRoute(req app.RouteTestRequest) app.RouteTestResult {
	return app.RouteTestResult{Matched: true}
}

// Helper to create test handler
func newTestHandler() (*Handler, *mockUsers, *mockKeys, *mockPlans) {
	users := newMockUsers()
	keys := newMockKeys()
	plans := newMockPlans()
	routes := newMockRoutes()
	upstreams := newMockUpstreams()

	h := &Handler{
		templates:     make(map[string]*template.Template), // Empty for now
		tokens:        auth.NewTokenService("test-secret", 24*time.Hour),
		users:         users,
		keys:          keys,
		usage:         &mockUsage{},
		routes:        routes,
		upstreams:     upstreams,
		plans:         plans,
		appSettings:   AppSettings{UpstreamURL: "http://localhost:8000"},
		logger:        zerolog.Nop(),
		hasher:        &mockHash{},
		isSetup:       func() bool { return true },
		exprValidator: &mockExprValidator{},
		routeTester:   &mockRouteTester{},
	}
	return h, users, keys, plans
}

func TestHandler_AuthMiddleware_NoToken(t *testing.T) {
	h, _, _, _ := newTestHandler()

	protected := h.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	location := w.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %s, want /login", location)
	}
}

func TestHandler_AuthMiddleware_InvalidToken(t *testing.T) {
	h, _, _, _ := newTestHandler()

	protected := h.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token"})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestHandler_AuthMiddleware_ValidToken(t *testing.T) {
	h, _, _, _ := newTestHandler()

	token, _, _ := h.tokens.GenerateToken("user1", "user@example.com", "admin")

	var gotContext bool
	protected := h.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := getClaims(r.Context())
		gotContext = claims != nil && claims.UserID == "user1"
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if !gotContext {
		t.Error("Claims should be in context")
	}
}

func TestHandler_AuthMiddleware_HTMX(t *testing.T) {
	h, _, _, _ := newTestHandler()

	protected := h.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	if w.Header().Get("HX-Redirect") != "/login" {
		t.Error("HX-Redirect should be set to /login")
	}
}

func TestHandler_SetupRequired_NotSetup(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.isSetup = func() bool { return false }

	wrapped := h.SetupRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if w.Header().Get("Location") != "/setup" {
		t.Error("Should redirect to /setup")
	}
}

func TestHandler_SetupRequired_AlreadySetup(t *testing.T) {
	h, _, _, _ := newTestHandler()

	wrapped := h.SetupRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_SetupRequired_SetupPath(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.isSetup = func() bool { return false }

	wrapped := h.SetupRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/setup", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Setup paths should pass through even when not setup
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_SetupRequired_StaticPath(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.isSetup = func() bool { return false }

	wrapped := h.SetupRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/static/style.css", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Static paths should pass through even when not setup
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_countActiveKeys(t *testing.T) {
	h, users, keys, _ := newTestHandler()

	// Create users
	users.users["user1"] = ports.User{ID: "user1"}
	users.users["user2"] = ports.User{ID: "user2"}

	// Create keys - 2 active, 1 revoked
	now := time.Now()
	keys.keys["key1"] = key.Key{ID: "key1", UserID: "user1", RevokedAt: nil}
	keys.keys["key2"] = key.Key{ID: "key2", UserID: "user1", RevokedAt: &now}
	keys.keys["key3"] = key.Key{ID: "key3", UserID: "user2", RevokedAt: nil}

	userList := []ports.User{users.users["user1"], users.users["user2"]}
	count := h.countActiveKeys(context.Background(), userList)

	if count != 2 {
		t.Errorf("countActiveKeys() = %d, want 2", count)
	}
}

func TestHandler_getPlans(t *testing.T) {
	h, _, _, plans := newTestHandler()

	// No plans
	result := h.getPlans()
	if len(result) != 0 {
		t.Errorf("getPlans() with empty store = %d, want 0", len(result))
	}

	// Add plans
	plans.plans["free"] = ports.Plan{
		ID:                 "free",
		Name:               "Free Plan",
		RateLimitPerMinute: 60,
		RequestsPerMonth:   1000,
		PriceMonthly:       0,
		IsDefault:          true,
		Enabled:            true,
	}
	plans.plans["pro"] = ports.Plan{
		ID:                 "pro",
		Name:               "Pro Plan",
		RateLimitPerMinute: 1000,
		RequestsPerMonth:   100000,
		PriceMonthly:       2999,
		IsDefault:          false,
		Enabled:            true,
	}

	result = h.getPlans()
	if len(result) != 2 {
		t.Errorf("getPlans() = %d, want 2", len(result))
	}
}

func TestHandler_getPlans_NilStore(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.plans = nil

	result := h.getPlans()
	if len(result) != 0 {
		t.Errorf("getPlans() with nil store = %d, want 0", len(result))
	}
}

func TestPlanToInfo(t *testing.T) {
	p := ports.Plan{
		ID:                 "pro",
		Name:               "Pro Plan",
		Description:        "For professionals",
		RateLimitPerMinute: 1000,
		RequestsPerMonth:   100000,
		PriceMonthly:       2999,
		OveragePrice:       1,
		StripePriceID:      "price_abc",
		PaddlePriceID:      "123",
		LemonVariantID:     "var_xyz",
		IsDefault:          false,
		Enabled:            true,
	}

	info := planToInfo(p)

	if info.ID != "pro" {
		t.Errorf("ID = %s, want pro", info.ID)
	}
	if info.Name != "Pro Plan" {
		t.Errorf("Name = %s, want Pro Plan", info.Name)
	}
	if info.PriceMonthly != 29.99 {
		t.Errorf("PriceMonthly = %f, want 29.99", info.PriceMonthly)
	}
	if info.OveragePrice != 0.01 {
		t.Errorf("OveragePrice = %f, want 0.01", info.OveragePrice)
	}
	if info.StripePriceID != "price_abc" {
		t.Errorf("StripePriceID = %s, want price_abc", info.StripePriceID)
	}
	if !info.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		n    float64
		unit string
		want string
	}{
		{1, "minute", "1 minute ago"},
		{5, "minute", "5 minutes ago"},
		{1, "hour", "1 hour ago"},
		{3, "hour", "3 hours ago"},
		{1, "day", "1 day ago"},
		{7, "day", "7 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.n, tt.unit)
			if got != tt.want {
				t.Errorf("formatDuration(%f, %s) = %s, want %s", tt.n, tt.unit, got, tt.want)
			}
		})
	}
}

func TestGenerateUserID(t *testing.T) {
	id := generateUserID()

	if !strings.HasPrefix(id, "user_") {
		t.Errorf("ID should start with 'user_', got %s", id)
	}

	if len(id) != 21 { // "user_" + 16 hex chars
		t.Errorf("ID length = %d, want 21", len(id))
	}

	// Test uniqueness
	id2 := generateUserID()
	if id == id2 {
		t.Error("Generated IDs should be unique")
	}
}

func TestHandler_Logout(t *testing.T) {
	h, _, _, _ := newTestHandler()

	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if w.Header().Get("Location") != "/login" {
		t.Error("Should redirect to /login")
	}

	// Check cookie is cleared
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "token" && c.Value == "" && c.Expires.Before(time.Now()) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Token cookie should be cleared")
	}
}

func TestHandler_UserDelete_HTMX(t *testing.T) {
	h, users, _, _ := newTestHandler()
	// Add simple template for partial
	h.templates["dashboard"] = template.Must(template.New("dashboard").Parse(`{{define "partial_users"}}users list{{end}}`))

	users.users["user1"] = ports.User{ID: "user1", Email: "user@test.com"}

	r := chi.NewRouter()
	r.Delete("/users/{id}", h.UserDelete)

	req := httptest.NewRequest("DELETE", "/users/user1", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	if _, ok := users.users["user1"]; ok {
		t.Error("User should be deleted")
	}
}

func TestHandler_UserDelete_Regular(t *testing.T) {
	h, users, _, _ := newTestHandler()

	users.users["user1"] = ports.User{ID: "user1", Email: "user@test.com"}

	r := chi.NewRouter()
	r.Delete("/users/{id}", h.UserDelete)

	req := httptest.NewRequest("DELETE", "/users/user1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if w.Header().Get("Location") != "/users" {
		t.Error("Should redirect to /users")
	}
}

func TestHandler_KeyRevoke_Regular(t *testing.T) {
	h, _, keys, _ := newTestHandler()

	keys.keys["key1"] = key.Key{ID: "key1", UserID: "user1"}

	r := chi.NewRouter()
	r.Delete("/keys/{id}", h.KeyRevoke)

	req := httptest.NewRequest("DELETE", "/keys/key1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if keys.keys["key1"].RevokedAt == nil {
		t.Error("Key should be revoked")
	}
}

func TestHandler_KeyRevoke_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Delete("/keys/{id}", h.KeyRevoke)

	req := httptest.NewRequest("DELETE", "/keys/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_PlanDelete_WithUsers(t *testing.T) {
	h, users, _, plans := newTestHandler()

	plans.plans["pro"] = ports.Plan{ID: "pro", Name: "Pro Plan"}
	users.users["user1"] = ports.User{ID: "user1", PlanID: "pro"}

	r := chi.NewRouter()
	r.Delete("/plans/{id}", h.PlanDelete)

	req := httptest.NewRequest("DELETE", "/plans/pro", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Plan should NOT be deleted
	if _, ok := plans.plans["pro"]; !ok {
		t.Error("Plan should not be deleted when users are assigned")
	}
}

func TestHandler_PlanDelete_NoUsers(t *testing.T) {
	h, _, _, plans := newTestHandler()

	plans.plans["pro"] = ports.Plan{ID: "pro", Name: "Pro Plan"}

	r := chi.NewRouter()
	r.Delete("/plans/{id}", h.PlanDelete)

	req := httptest.NewRequest("DELETE", "/plans/pro", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if _, ok := plans.plans["pro"]; ok {
		t.Error("Plan should be deleted")
	}
}

func TestHandler_UserUpdate_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Post("/users/{id}", h.UserUpdate)

	form := url.Values{"plan_id": {"pro"}, "status": {"active"}}
	req := httptest.NewRequest("POST", "/users/nonexistent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_UserEditPage_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Get("/users/{id}", h.UserEditPage)

	req := httptest.NewRequest("GET", "/users/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_PlanEditPage_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Get("/plans/{id}", h.PlanEditPage)

	req := httptest.NewRequest("GET", "/plans/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_PlanUpdate_NotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	r := chi.NewRouter()
	r.Post("/plans/{id}", h.PlanUpdate)

	form := url.Values{"name": {"Updated"}}
	req := httptest.NewRequest("POST", "/plans/nonexistent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_KeyCreate_UserNotFound(t *testing.T) {
	h, _, _, _ := newTestHandler()

	form := url.Values{"user_id": {"nonexistent"}}
	req := httptest.NewRequest("POST", "/keys", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.KeyCreate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_Router_Routes(t *testing.T) {
	h, _, _, _ := newTestHandler()
	// Create minimal templates
	tmpl := template.Must(template.New("test").Parse(`{{define "base"}}ok{{end}}`))
	h.templates["login"] = tmpl
	h.templates["dashboard"] = tmpl
	h.templates["setup"] = tmpl

	router := h.Router()

	// Test that basic routes exist
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/login"},
		{"POST", "/login"},
		{"POST", "/logout"},
		{"GET", "/setup"},
		{"POST", "/setup"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not be 404 (route exists)
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s %s should exist", route.method, route.path)
			}
		})
	}
}

func TestHandler_Router_ProtectedRoutes(t *testing.T) {
	h, _, _, _ := newTestHandler()

	router := h.Router()

	// These routes should redirect to login (no auth)
	protectedRoutes := []string{
		"/dashboard",
		"/users",
		"/keys",
		"/plans",
		"/usage",
		"/settings",
		"/system",
	}

	for _, path := range protectedRoutes {
		t.Run("GET "+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusFound {
				t.Errorf("Route %s should redirect without auth, got %d", path, w.Code)
			}
			if w.Header().Get("Location") != "/login" {
				t.Errorf("Route %s should redirect to /login", path)
			}
		})
	}
}

func TestHandler_PartialUsers_Limit(t *testing.T) {
	h, users, _, _ := newTestHandler()
	h.templates["dashboard"] = template.Must(template.New("dashboard").Parse(`{{define "partial_users"}}{{len .Users}} users{{end}}`))

	// Add users
	for i := 0; i < 10; i++ {
		users.users[string(rune('a'+i))] = ports.User{ID: string(rune('a' + i))}
	}

	req := httptest.NewRequest("GET", "/partials/users?limit=5", nil)
	w := httptest.NewRecorder()

	h.PartialUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_SetupPage_AlreadySetup(t *testing.T) {
	h, _, _, _ := newTestHandler()
	h.isSetup = func() bool { return true }

	req := httptest.NewRequest("GET", "/setup", nil)
	w := httptest.NewRecorder()

	h.SetupPage(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusFound)
	}

	if w.Header().Get("Location") != "/login" {
		t.Error("Should redirect to /login when already setup")
	}
}

func TestAppSettings(t *testing.T) {
	s := AppSettings{
		UpstreamURL:     "http://api.example.com",
		UpstreamTimeout: "30s",
		AuthMode:        "apikey",
		AuthHeader:      "X-API-Key",
		DatabaseDSN:     "sqlite:///data.db",
	}

	if s.UpstreamURL != "http://api.example.com" {
		t.Error("UpstreamURL mismatch")
	}
	if s.UpstreamTimeout != "30s" {
		t.Error("UpstreamTimeout mismatch")
	}
	if s.AuthMode != "apikey" {
		t.Error("AuthMode mismatch")
	}
}

func TestPlanInfo(t *testing.T) {
	p := PlanInfo{
		ID:             "pro",
		Name:           "Pro Plan",
		Description:    "For professionals",
		RateLimit:      1000,
		MonthlyQuota:   100000,
		PriceMonthly:   29.99,
		OveragePrice:   0.01,
		StripePriceID:  "price_abc",
		PaddlePriceID:  "123",
		LemonVariantID: "var_xyz",
		IsDefault:      false,
		Enabled:        true,
	}

	if p.ID != "pro" {
		t.Error("ID mismatch")
	}
	if p.PriceMonthly != 29.99 {
		t.Error("PriceMonthly mismatch")
	}
}
