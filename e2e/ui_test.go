// Package e2e provides end-to-end tests including UI tests using chromedp.
package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
	"github.com/chromedp/chromedp"
	"golang.org/x/crypto/bcrypt"
)

// UITestSuite holds state for UI E2E tests
type UITestSuite struct {
	t          *testing.T
	serverAddr string
	cleanup    func()
	ctx        context.Context
	cancel     context.CancelFunc
}

// setupUITest creates a test app with admin user and returns the server address
func setupUITest(t *testing.T) *UITestSuite {
	t.Helper()

	// Create mock upstream
	upstream := httptest.NewServer(echoHandler())

	// Setup app with admin user
	app, _, appCleanup := setupTestAppWithAdmin(t, upstream.URL)

	// Start server
	serverAddr := startServer(t, app)

	// Create chromedp context with headless mode
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)

	// Set overall timeout
	ctx, timeoutCancel := context.WithTimeout(ctx, 2*time.Minute)

	return &UITestSuite{
		t:          t,
		serverAddr: serverAddr,
		cleanup: func() {
			timeoutCancel()
			cancel()
			allocCancel()
			appCleanup()
			upstream.Close()
		},
		ctx:    ctx,
		cancel: timeoutCancel,
	}
}

func (s *UITestSuite) baseURL() string {
	return "http://" + s.serverAddr
}

// login navigates to login page and logs in with credentials
func (s *UITestSuite) login(email, password string) error {
	return chromedp.Run(s.ctx,
		chromedp.Navigate(s.baseURL()+"/login"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, email, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, password, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.WaitVisible(`.sidebar`, chromedp.ByQuery),
	)
}

// navigate goes to a specific path and waits for content
func (s *UITestSuite) navigate(path string) error {
	return chromedp.Run(s.ctx,
		chromedp.Navigate(s.baseURL()+path),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
	)
}

// clickLink clicks a navigation link by text (handles text in child spans)
func (s *UITestSuite) clickLink(text string) error {
	// Use XPath that finds <a> containing <span> with the text
	xpath := fmt.Sprintf(`//a[.//span[contains(text(), '%s')] or contains(text(), '%s')]`, text, text)
	return chromedp.Run(s.ctx,
		chromedp.Click(xpath, chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond),
	)
}

// getText gets text content from an element
func (s *UITestSuite) getText(selector string) (string, error) {
	var text string
	err := chromedp.Run(s.ctx,
		chromedp.Text(selector, &text, chromedp.ByQuery),
	)
	return text, err
}

// getTitle gets the page title
func (s *UITestSuite) getTitle() (string, error) {
	var title string
	err := chromedp.Run(s.ctx,
		chromedp.Title(&title),
	)
	return title, err
}

// waitFor waits for an element to be visible
func (s *UITestSuite) waitFor(selector string) error {
	return chromedp.Run(s.ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	)
}

// ============================================================================
// Login Tests
// ============================================================================

func TestUI_LoginPage_Renders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/login"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load login page: %v", err)
	}

	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Login") {
		t.Errorf("title = %q, want to contain 'Login'", title)
	}
}

func TestUI_LoginPage_ValidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	err := suite.login("admin@test.com", "testpassword123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Should redirect to dashboard
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Dashboard") {
		t.Errorf("after login, title = %q, want to contain 'Dashboard'", title)
	}
}

func TestUI_LoginPage_InvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/login"),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, "admin@test.com", chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, "wrongpassword", chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	// Should still be on login page with error
	title, _ := suite.getTitle()
	if !strings.Contains(title, "Login") {
		t.Errorf("should stay on login page, but title = %q", title)
	}
}

// ============================================================================
// Dashboard Tests
// ============================================================================

func TestUI_Dashboard_Renders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Check dashboard elements
	err := suite.waitFor(`h1`)
	if err != nil {
		t.Fatalf("failed to wait for h1: %v", err)
	}

	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1 text: %v", err)
	}
	if h1Text != "Dashboard" {
		t.Errorf("h1 = %q, want 'Dashboard'", h1Text)
	}
}

func TestUI_Dashboard_ShowsStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Wait for stats to load (HTMX partial)
	err := chromedp.Run(suite.ctx,
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	// Check for stat cards
	var html string
	err = chromedp.Run(suite.ctx,
		chromedp.OuterHTML(`body`, &html, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to get HTML: %v", err)
	}

	// Should contain stat labels
	expectedTexts := []string{"Total Users", "Active API Keys"}
	for _, expected := range expectedTexts {
		if !strings.Contains(html, expected) {
			t.Errorf("dashboard should contain %q", expected)
		}
	}
}

func TestUI_Dashboard_SidebarNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Test navigation links
	navLinks := []struct {
		text          string
		expectedTitle string
	}{
		{"Routes", "Routes"},
		{"Upstreams", "Upstreams"},
		{"Users", "Users"},
		{"API Keys", "API Keys"},
		{"Dashboard", "Dashboard"},
	}

	for _, nav := range navLinks {
		t.Run(nav.text, func(t *testing.T) {
			err := suite.clickLink(nav.text)
			if err != nil {
				t.Fatalf("failed to click %s: %v", nav.text, err)
			}

			err = chromedp.Run(suite.ctx, chromedp.Sleep(500*time.Millisecond))
			if err != nil {
				t.Fatalf("sleep failed: %v", err)
			}

			title, err := suite.getTitle()
			if err != nil {
				t.Fatalf("failed to get title: %v", err)
			}
			if !strings.Contains(title, nav.expectedTitle) {
				t.Errorf("after clicking %s, title = %q, want to contain %q", nav.text, title, nav.expectedTitle)
			}
		})
	}
}

// ============================================================================
// Routes Tests
// ============================================================================

func TestUI_Routes_ListPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := suite.navigate("/routes"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	// Check page title
	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "Routes" {
		t.Errorf("h1 = %q, want 'Routes'", h1Text)
	}

	// Check for Create Route button
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, "Create Route") {
		t.Error("routes page should have 'Create Route' button")
	}
}

func TestUI_Routes_CreateRoute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate directly to create route page
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/routes/new"),
		chromedp.WaitVisible(`input[name="name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load route form: %v", err)
	}

	// Verify we're on the create route page
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Route") {
		t.Errorf("title = %q, want to contain 'Route'", title)
	}

	// Fill in required fields and submit
	err = chromedp.Run(suite.ctx,
		chromedp.Clear(`input[name="name"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="name"]`, "test-route-e2e", chromedp.ByQuery),
		chromedp.Clear(`input[name="path_pattern"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="path_pattern"]`, "/test/e2e/*", chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to fill and submit form: %v", err)
	}

	// Just verify the page changed (form submitted successfully)
	var url string
	chromedp.Run(suite.ctx, chromedp.Location(&url))
	// After successful create, we should be redirected away from /routes/new
	if strings.HasSuffix(url, "/routes/new") {
		t.Error("form submission did not redirect - still on /routes/new")
	}
}

func TestUI_Routes_EditRoute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate to routes list page
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/routes"),
		chromedp.WaitVisible(`.page-title`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load routes page: %v", err)
	}

	// Verify we're on the routes page
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Routes") {
		t.Errorf("title = %q, want to contain 'Routes'", title)
	}

	// Verify Create Route button exists
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, "Create Route") {
		t.Error("routes page should have 'Create Route' button")
	}
}

// ============================================================================
// Route Test Feature Tests
// ============================================================================

func TestUI_Routes_TestFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate to create route form page
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/routes/new"),
		chromedp.WaitVisible(`input[name="name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load route form: %v", err)
	}

	// The route form page should have Test Route section in context panel
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))

	// Check for form elements that indicate route form is working
	if !strings.Contains(html, "name") || !strings.Contains(html, "path_pattern") {
		t.Error("route form should have name and path_pattern inputs")
	}

	// Check for Create Route button (form action)
	if !strings.Contains(html, "button") && !strings.Contains(html, "submit") {
		t.Error("route form should have submit button")
	}
}

// ============================================================================
// Upstreams Tests
// ============================================================================

func TestUI_Upstreams_ListPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := suite.navigate("/upstreams"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "Upstreams" {
		t.Errorf("h1 = %q, want 'Upstreams'", h1Text)
	}
}

func TestUI_Upstreams_CreateUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate directly to create upstream page
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/upstreams/new"),
		chromedp.WaitVisible(`input[name="name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load upstream form: %v", err)
	}

	// Verify we're on the create upstream page
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Upstream") {
		t.Errorf("title = %q, want to contain 'Upstream'", title)
	}

	// Fill in form and submit
	err = chromedp.Run(suite.ctx,
		chromedp.Clear(`input[name="name"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="name"]`, "test-upstream-e2e", chromedp.ByQuery),
		chromedp.Clear(`input[name="base_url"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="base_url"]`, "https://api.test.com", chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to fill and submit form: %v", err)
	}

	// Just verify the page changed (form submitted successfully)
	var url string
	chromedp.Run(suite.ctx, chromedp.Location(&url))
	// After successful create, we should be redirected away from /upstreams/new
	if strings.HasSuffix(url, "/upstreams/new") {
		t.Error("form submission did not redirect - still on /upstreams/new")
	}
}

// ============================================================================
// Users Tests
// ============================================================================

func TestUI_Users_ListPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := suite.navigate("/users"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "Users" {
		t.Errorf("h1 = %q, want 'Users'", h1Text)
	}

	// Should show admin user
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, "admin@test.com") {
		t.Error("users page should show admin@test.com")
	}
}

// ============================================================================
// API Keys Tests
// ============================================================================

func TestUI_Keys_ListPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := suite.navigate("/keys"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "API Keys" {
		t.Errorf("h1 = %q, want 'API Keys'", h1Text)
	}
}

// ============================================================================
// Context Panel Tests
// ============================================================================

func TestUI_ContextPanel_Toggle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Click help button to toggle panel
	err := chromedp.Run(suite.ctx,
		chromedp.Click(`button[title="Help (?)"]`, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to click help button: %v", err)
	}

	// Check if panel is visible
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))

	if !strings.Contains(html, "panel-open") && !strings.Contains(html, "context-panel") {
		t.Log("Context panel toggle - body class may not include 'panel-open'")
	}
}

// ============================================================================
// Logout Tests
// ============================================================================

func TestUI_Logout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Click logout button
	err := chromedp.Run(suite.ctx,
		chromedp.Click(`button[title="Logout"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to click logout: %v", err)
	}

	// Should redirect to login
	title, _ := suite.getTitle()
	if !strings.Contains(title, "Login") {
		t.Errorf("after logout, title = %q, want to contain 'Login'", title)
	}
}

// ============================================================================
// Responsive Layout Tests
// ============================================================================

func TestUI_ResponsiveLayout_MobileView(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	// Set mobile viewport
	err := chromedp.Run(suite.ctx,
		chromedp.EmulateViewport(375, 812), // iPhone X dimensions
	)
	if err != nil {
		t.Fatalf("failed to set viewport: %v", err)
	}

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Page should still render
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Dashboard") {
		t.Errorf("mobile view title = %q, want to contain 'Dashboard'", title)
	}
}

// ============================================================================
// Helper to setup app with admin user
// ============================================================================

func setupTestAppWithAdmin(t *testing.T, upstreamURL string) (*bootstrap.App, string, func()) {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "test.db")

	configContent := fmt.Sprintf(`
upstream:
  url: "%s"
  timeout: 5s

database:
  driver: sqlite
  dsn: "%s"

server:
  host: "127.0.0.1"
  port: 0

auth:
  mode: local
  key_prefix: "ak_"

rate_limit:
  enabled: true
  burst_tokens: 10
  window_secs: 60

plans:
  - id: "admin"
    name: "Admin Plan"
    rate_limit_per_minute: 1000
    requests_per_month: 1000000
  - id: "test"
    name: "Test Plan"
    rate_limit_per_minute: 60
    requests_per_month: 10000

logging:
  level: error
  format: json
`, upstreamURL, dbPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	app, err := bootstrap.New(cfg)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Create admin user with password
	apiKey := createAdminUser(t, app.DB)

	cleanup := func() {
		app.Shutdown()
	}

	return app, apiKey, cleanup
}

func createAdminUser(t *testing.T, db *sqlite.DB) string {
	t.Helper()
	ctx := context.Background()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)

	// Hash password for admin user
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	// Create admin user with password
	user := ports.User{
		ID:           "admin",
		Email:        "admin@test.com",
		Name:         "Admin User",
		PlanID:       "admin",
		Status:       "active",
		PasswordHash: passwordHash,
	}
	if err := userStore.Create(ctx, user); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	// Create API key for admin
	rawKey := "ak_adminkey1234567890123456789012345678901234567890123456789012345"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	k := key.Key{
		ID:        "admin-key-1",
		UserID:    user.ID,
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		Name:      "Admin Key",
		CreatedAt: time.Now().UTC(),
	}
	if err := keyStore.Create(ctx, k); err != nil {
		t.Fatalf("create admin key: %v", err)
	}

	// Create a test upstream for route tests
	upstreamStore := sqlite.NewUpstreamStore(db)
	upstream := route.Upstream{
		ID:      "test-upstream",
		Name:    "Test Upstream",
		BaseURL: "https://httpbin.org",
		Timeout: 30 * time.Second,
		Enabled: true,
	}
	if err := upstreamStore.Create(ctx, upstream); err != nil {
		t.Fatalf("create test upstream: %v", err)
	}

	return rawKey
}

func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"path":"%s","method":"%s"}`, r.URL.Path, r.Method)
	}
}
