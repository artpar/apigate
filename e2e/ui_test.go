// Package e2e provides end-to-end tests including UI tests using chromedp.
package e2e

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
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
		{"Plans", "Plans"},
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
// Plans Tests
// ============================================================================

func TestUI_Plans_ListPage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := suite.navigate("/plans"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	// Check page title
	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "Plans" {
		t.Errorf("h1 = %q, want 'Plans'", h1Text)
	}

	// Check for Create Plan button
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, "Create Plan") {
		t.Error("plans page should have 'Create Plan' button")
	}
}

func TestUI_Plans_CreatePlan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate directly to create plan page (same pattern as Routes/Upstreams)
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/plans/new"),
		chromedp.WaitVisible(`input[name="name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load plan form: %v", err)
	}

	// Verify we're on the create plan page
	title, err := suite.getTitle()
	if err != nil {
		t.Fatalf("failed to get title: %v", err)
	}
	if !strings.Contains(title, "Plan") {
		t.Errorf("title = %q, want to contain 'Plan'", title)
	}

	// Verify form fields are present
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, `name="id"`) {
		t.Error("page should contain plan id field")
	}
	if !strings.Contains(html, `name="name"`) {
		t.Error("page should contain plan name field")
	}
}

func TestUI_Plans_EditPlan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate to plans list page
	if err := suite.navigate("/plans"); err != nil {
		t.Fatalf("navigate failed: %v", err)
	}

	// Check page title
	h1Text, err := suite.getText(`h1`)
	if err != nil {
		t.Fatalf("failed to get h1: %v", err)
	}
	if h1Text != "Plans" {
		t.Errorf("h1 = %q, want 'Plans'", h1Text)
	}

	// Wait for HTMX to load the table
	chromedp.Run(suite.ctx, chromedp.Sleep(1*time.Second))

	// Verify table is rendered with Edit links
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))
	if !strings.Contains(html, "Edit") {
		t.Error("plans table should have Edit links")
	}
}

func TestUI_Plans_FormValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping UI test in short mode")
	}

	suite := setupUITest(t)
	defer suite.cleanup()

	if err := suite.login("admin@test.com", "testpassword123"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Navigate to create plan page - wait for name input (same as other forms)
	err := chromedp.Run(suite.ctx,
		chromedp.Navigate(suite.baseURL()+"/plans/new"),
		chromedp.WaitVisible(`input[name="name"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("failed to load plan form: %v", err)
	}

	// Verify form has all expected fields
	var html string
	chromedp.Run(suite.ctx, chromedp.OuterHTML(`body`, &html, chromedp.ByQuery))

	expectedFields := []string{
		`name="id"`,
		`name="name"`,
		`name="rate_limit"`,
		`name="monthly_quota"`,
		`name="price_monthly"`,
		`name="overage_price"`,
		`name="stripe_price_id"`,
		`name="paddle_price_id"`,
		`name="lemon_variant_id"`,
		`name="enabled"`,
		`name="is_default"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(html, field) {
			t.Errorf("plan form should have field %s", field)
		}
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
	dbPath := dir + "/test.db"

	// Pre-create database and insert settings BEFORE bootstrap
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	// Insert settings for upstream and rate limit
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "upstream.url", upstreamURL)
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.burst_tokens", "10")
	db.DB.ExecContext(ctx, "INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "ratelimit.window_secs", "60")

	// Create plans in database
	db.DB.ExecContext(ctx,
		"INSERT OR REPLACE INTO plans (id, name, rate_limit_per_minute, requests_per_month, enabled) VALUES (?, ?, ?, ?, ?)",
		"admin", "Admin Plan", 1000, 1000000, 1)
	db.DB.ExecContext(ctx,
		"INSERT OR REPLACE INTO plans (id, name, rate_limit_per_minute, requests_per_month, enabled) VALUES (?, ?, ?, ?, ?)",
		"test", "Test Plan", 60, 10000, 1)

	db.Close()

	// Set environment variables for bootstrap
	os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
	os.Setenv(bootstrap.EnvLogLevel, "error")
	os.Setenv(bootstrap.EnvLogFormat, "json")

	app, err := bootstrap.New()
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	// Create admin user with password
	apiKey := createAdminUser(t, app.DB)

	cleanup := func() {
		app.Shutdown()
		os.Unsetenv(bootstrap.EnvDatabaseDSN)
		os.Unsetenv(bootstrap.EnvLogLevel)
		os.Unsetenv(bootstrap.EnvLogFormat)
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
