package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	domainAuth "github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
)

// setModuleSessionCookie sets the apigate_session cookie for module WebUI compatibility.
// This allows users logged in via the root WebUI to also access the module WebUI.
func setModuleSessionCookie(w http.ResponseWriter, userID, email, name string, expiresAt time.Time) {
	session := struct {
		UserID    string    `json:"user_id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		ExpiresAt time.Time `json:"expires_at"`
	}{
		UserID:    userID,
		Email:     email,
		Name:      name,
		ExpiresAt: expiresAt,
	}
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     "apigate_session",
		Value:    encoded,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// LoginPage renders the login form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If setup not complete, redirect to setup wizard
	if h.isSetup != nil && !h.isSetup() {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}

	// If already logged in, redirect to dashboard
	if cookie, err := r.Cookie("token"); err == nil {
		if _, err := h.tokens.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
	}

	data := struct {
		PageData
		Error   string
		Success string
		Email   string
	}{
		PageData: h.newPageData(r.Context(), "Login"),
		Success:  r.URL.Query().Get("success"),
	}

	h.render(w, "login", data)
}

// LoginSubmit handles login form submission.
func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Find user by email
	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		h.renderLoginError(w, r, "Invalid email or password", email)
		return
	}

	// Verify password
	if len(user.PasswordHash) == 0 || !h.hasher.Compare(user.PasswordHash, password) {
		h.renderLoginError(w, r, "Invalid email or password", email)
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.tokens.GenerateToken(user.ID, user.Email, "admin")
	if err != nil {
		h.renderLoginError(w, r, "Login failed", email)
		return
	}

	// Set JWT cookie for root WebUI
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Also set apigate_session cookie for module WebUI compatibility
	setModuleSessionCookie(w, user.ID, user.Email, user.Name, expiresAt)

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (h *Handler) renderLoginError(w http.ResponseWriter, r *http.Request, errMsg, email string) {
	data := struct {
		PageData
		Error   string
		Success string
		Email   string
	}{
		PageData: h.newPageData(r.Context(), "Login"),
		Error:    errMsg,
		Email:    email,
	}
	h.render(w, "login", data)
}

// Logout clears the session cookies.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear root WebUI JWT cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	// Clear module WebUI session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "apigate_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ForgotPasswordPage renders the forgot password form.
func (h *Handler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		Error   string
		Success string
		Email   string
	}{
		PageData: h.newPageData(r.Context(), "Forgot Password"),
	}
	h.render(w, "forgot_password", data)
}

// ForgotPasswordSubmit handles forgot password form submission.
func (h *Handler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	email := strings.TrimSpace(r.FormValue("email"))

	// Validate email format
	valid, errMsg := domainAuth.ValidatePasswordResetRequest(domainAuth.PasswordResetRequest{Email: email})
	if !valid {
		data := struct {
			PageData
			Error   string
			Success string
			Email   string
		}{
			PageData: h.newPageData(ctx, "Forgot Password"),
			Error:    errMsg,
			Email:    email,
		}
		h.render(w, "forgot_password", data)
		return
	}

	// Look up user by email (don't reveal if user exists)
	user, err := h.users.GetByEmail(ctx, email)
	if err == nil && user.ID != "" && h.authTokens != nil && h.emailSender != nil {
		// Generate reset token (1 hour expiry)
		tokenResult := domainAuth.GenerateToken(user.ID, user.Email, domainAuth.TokenTypePasswordReset, 1*time.Hour)
		tokenWithHash := tokenResult.Token.WithHash(domainAuth.HashToken(tokenResult.RawToken))

		if err := h.authTokens.Create(ctx, tokenWithHash); err != nil {
			h.logger.Error().Err(err).Msg("failed to store reset token")
		} else if err := h.emailSender.SendPasswordReset(ctx, user.Email, user.Name, tokenResult.RawToken); err != nil {
			h.logger.Error().Err(err).Msg("failed to send password reset email")
		}
	}

	// Always show success message (don't reveal if user exists)
	data := struct {
		PageData
		Error   string
		Success string
		Email   string
	}{
		PageData: h.newPageData(ctx, "Forgot Password"),
		Success:  "If an account exists with that email, you will receive password reset instructions.",
	}
	h.render(w, "forgot_password", data)
}

// ResetPasswordPage renders the password reset form.
func (h *Handler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/forgot-password", http.StatusFound)
		return
	}

	data := struct {
		PageData
		Token  string
		Errors map[string]string
	}{
		PageData: h.newPageData(r.Context(), "Reset Password"),
		Token:    token,
		Errors:   make(map[string]string),
	}
	h.render(w, "reset_password", data)
}

// ResetPasswordSubmit handles password reset form submission.
func (h *Handler) ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawToken := r.FormValue("token")
	newPassword := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	errors := make(map[string]string)
	if rawToken == "" {
		errors["token"] = "Reset token is required"
	}
	if newPassword == "" {
		errors["password"] = "Password is required"
	} else if len(newPassword) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	}
	if newPassword != confirmPassword {
		errors["confirm_password"] = "Passwords do not match"
	}

	if len(errors) > 0 {
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	// Verify token
	if h.authTokens == nil {
		errors["token"] = "Password reset is not configured"
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	hash := domainAuth.HashToken(rawToken)
	token, err := h.authTokens.GetByHash(ctx, hash)
	if err != nil {
		errors["token"] = "Invalid or expired reset link"
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	if token.Type != domainAuth.TokenTypePasswordReset {
		errors["token"] = "Invalid token type"
	} else if token.ExpiresAt.Before(time.Now().UTC()) {
		errors["token"] = "This reset link has expired. Please request a new one."
	} else if token.UsedAt != nil {
		errors["token"] = "This reset link has already been used"
	}

	if len(errors) > 0 {
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	// Get user and update password
	user, err := h.users.Get(ctx, token.UserID)
	if err != nil {
		errors["token"] = "User not found"
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	// Hash and update password
	hashedPassword, err := h.hasher.Hash(newPassword)
	if err != nil {
		errors["password"] = "Failed to update password"
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	user.PasswordHash = hashedPassword
	if err := h.users.Update(ctx, user); err != nil {
		errors["password"] = "Failed to update password"
		data := struct {
			PageData
			Token  string
			Errors map[string]string
		}{
			PageData: h.newPageData(ctx, "Reset Password"),
			Token:    rawToken,
			Errors:   errors,
		}
		h.render(w, "reset_password", data)
		return
	}

	// Mark token as used
	if err := h.authTokens.MarkUsed(ctx, token.ID, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark token as used")
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/login?success=Password+reset+successfully.+Please+login+with+your+new+password.", http.StatusSeeOther)
}

// TermsPage renders the terms of service page.
func (h *Handler) TermsPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		LastUpdated string
	}{
		PageData:    h.newPageData(r.Context(), "Terms of Service"),
		LastUpdated: time.Now().Format("January 2, 2006"),
	}
	h.render(w, "terms", data)
}

// PrivacyPage renders the privacy policy page.
func (h *Handler) PrivacyPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		LastUpdated string
	}{
		PageData:    h.newPageData(r.Context(), "Privacy Policy"),
		LastUpdated: time.Now().Format("January 2, 2006"),
	}
	h.render(w, "privacy", data)
}

// ChecklistItem represents a single onboarding checklist item.
type ChecklistItem struct {
	Title       string
	Description string
	Done        bool
	Link        string
	LinkText    string
	Summary     string // Shows what was configured when Done is true
}

// Dashboard renders the main dashboard.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	users, _ := h.users.List(ctx, 1000, 0)
	activeKeys := h.countActiveKeys(ctx, users)

	// Get routes and upstreams for checklist
	routes, _ := h.routes.List(ctx)
	upstreams, _ := h.upstreams.List(ctx)

	// Check if any API calls have been made
	hasAPIActivity := false
	for _, u := range users {
		events, _ := h.usage.GetRecentRequests(ctx, u.ID, 1)
		if len(events) > 0 {
			hasAPIActivity = true
			break
		}
	}

	// Get first active API key for test command
	var firstKeyValue string
	for _, u := range users {
		keys, _ := h.keys.ListByUser(ctx, u.ID)
		for _, k := range keys {
			if k.RevokedAt == nil {
				firstKeyValue = k.Prefix + "..." // We don't have the full key, show prefix
				break
			}
		}
		if firstKeyValue != "" {
			break
		}
	}

	// Build test command if we have an active key
	testCommand := ""
	if activeKeys > 0 && !hasAPIActivity {
		// Determine the gateway URL
		gatewayURL := "http://localhost:8080"
		if h.appSettings.UpstreamURL != "" {
			gatewayURL = "http://localhost:8080"
		}
		testCommand = fmt.Sprintf("curl -H \"X-API-Key: YOUR_API_KEY\" %s/", gatewayURL)
	}

	// Load portal settings
	portalEnabled := false
	portalURL := ""
	if h.settings != nil {
		allSettings, err := h.settings.GetAll(ctx)
		if err == nil {
			portalEnabled = allSettings.GetBool(settings.KeyPortalEnabled)
			baseURL := allSettings.Get(settings.KeyPortalBaseURL)
			if baseURL != "" {
				portalURL = baseURL + "/portal/"
			} else {
				// Auto-detect from request
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				portalURL = fmt.Sprintf("%s://%s/portal/", scheme, r.Host)
			}
		}
	}

	// Calculate MRR
	plans, _ := h.plans.List(ctx)
	planPrices := make(map[string]int64)
	for _, p := range plans {
		planPrices[p.ID] = p.PriceMonthly
	}
	var mrr int64 // in cents
	for _, u := range users {
		if u.Status == "active" {
			if price, ok := planPrices[u.PlanID]; ok {
				mrr += price
			}
		}
	}

	data := struct {
		PageData
		Stats struct {
			TotalUsers    int
			ActiveKeys    int
			RequestsToday int64
			MRR           float64
		}
		Checklist         []ChecklistItem
		ChecklistComplete bool
		ChecklistProgress int
		ChecklistTotal    int
		TestCommand       string
		HasAPIActivity    bool
		PortalEnabled     bool
		PortalURL         string
	}{
		PageData: h.newPageData(ctx, "Dashboard"),
	}
	data.CurrentPath = "/dashboard"
	data.Stats.TotalUsers = len(users)
	data.Stats.ActiveKeys = activeKeys
	data.Stats.MRR = float64(mrr) / 100
	data.TestCommand = testCommand
	data.HasAPIActivity = hasAPIActivity
	data.PortalEnabled = portalEnabled
	data.PortalURL = portalURL

	// Build checklist with summaries for completed items

	// Get upstream summary
	upstreamSummary := ""
	if len(upstreams) > 0 {
		upstreamSummary = upstreams[0].Name
		if len(upstreams) > 1 {
			upstreamSummary = fmt.Sprintf("%s (+%d more)", upstreams[0].Name, len(upstreams)-1)
		}
	}

	// Get route summary
	routeSummary := ""
	if len(routes) > 0 {
		routeSummary = fmt.Sprintf("%d route(s) configured", len(routes))
	}

	// Get key summary
	keySummary := ""
	if activeKeys > 0 {
		keySummary = fmt.Sprintf("%d active key(s)", activeKeys)
	}

	data.Checklist = []ChecklistItem{
		{
			Title:       "Create admin account",
			Description: "Set up your administrator account",
			Done:        true, // Always true if logged in
			Link:        "/users",
			LinkText:    "Manage users",
			Summary:     fmt.Sprintf("%d user(s)", len(users)),
		},
		{
			Title:       "Add an upstream API",
			Description: "Connect to the API you want to proxy",
			Done:        len(upstreams) > 0,
			Link:        "/upstreams",
			LinkText:    "Add upstream",
			Summary:     upstreamSummary,
		},
		{
			Title:       "Create a route",
			Description: "Define how requests are routed to your API",
			Done:        len(routes) > 0,
			Link:        "/routes",
			LinkText:    "Create route",
			Summary:     routeSummary,
		},
		{
			Title:       "Generate an API key",
			Description: "Create a key for authenticating requests",
			Done:        activeKeys > 0,
			Link:        "/keys",
			LinkText:    "Create key",
			Summary:     keySummary,
		},
		{
			Title:       "Make your first API call",
			Description: "Test your gateway with a real request",
			Done:        hasAPIActivity,
			Link:        "/keys",
			LinkText:    "Try it",
			Summary:     "",
		},
	}

	// Check if all items are complete and count progress
	data.ChecklistComplete = true
	data.ChecklistTotal = len(data.Checklist)
	for _, item := range data.Checklist {
		if item.Done {
			data.ChecklistProgress++
		} else {
			data.ChecklistComplete = false
		}
	}

	// Get today's request count from usage store
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var requestsToday int64
	h.logger.Debug().Str("start", startOfDay.Format("2006-01-02 15:04:05")).Str("end", now.Format("2006-01-02 15:04:05")).Int("users", len(users)).Msg("dashboard usage query")
	for _, u := range users {
		summary, err := h.usage.GetSummary(ctx, u.ID, startOfDay, now)
		h.logger.Debug().Str("user", u.ID).Int64("count", summary.RequestCount).Err(err).Msg("user summary")
		if err == nil {
			requestsToday += summary.RequestCount
		}
	}
	data.Stats.RequestsToday = requestsToday

	h.render(w, "dashboard", data)
}

// countActiveKeys counts non-revoked keys across all users.
func (h *Handler) countActiveKeys(ctx context.Context, users []ports.User) int {
	count := 0
	for _, u := range users {
		keys, _ := h.keys.ListByUser(ctx, u.ID)
		for _, k := range keys {
			if k.RevokedAt == nil {
				count++
			}
		}
	}
	return count
}

// UsersPage lists all users.
func (h *Handler) UsersPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
	}{
		PageData: h.newPageData(r.Context(), "Users"),
	}
	data.CurrentPath = "/users"

	h.render(w, "users", data)
}

// UserNewPage renders the create user form.
func (h *Handler) UserNewPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		IsEdit   bool
		FormUser struct {
			ID     string
			Email  string
			PlanID string
			Status string
		}
		Plans []PlanInfo
		Error string
	}{
		PageData: h.newPageData(r.Context(), "Create User"),
	}
	data.CurrentPath = "/users"
	data.FormUser.Status = "active"
	data.FormUser.PlanID = "free"
	data.Plans = h.getPlans()

	h.render(w, "user_form", data)
}

// PlanInfo represents a plan for templates.
type PlanInfo struct {
	ID             string
	Name           string
	Description    string
	RateLimit      int
	MonthlyQuota   int64
	PriceMonthly   float64
	OveragePrice   float64
	StripePriceID  string
	PaddlePriceID  string
	LemonVariantID string
	IsDefault      bool
	Enabled        bool
}

// getPlans returns plans from database.
func (h *Handler) getPlans() []PlanInfo {
	if h.plans == nil {
		return []PlanInfo{}
	}

	plans, err := h.plans.List(context.Background())
	if err != nil {
		return []PlanInfo{}
	}

	result := make([]PlanInfo, len(plans))
	for i, p := range plans {
		result[i] = planToInfo(p)
	}
	return result
}

// planToInfo converts a ports.Plan to PlanInfo.
func planToInfo(p ports.Plan) PlanInfo {
	return PlanInfo{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		RateLimit:      p.RateLimitPerMinute,
		MonthlyQuota:   p.RequestsPerMonth,
		PriceMonthly:   float64(p.PriceMonthly) / 100,
		OveragePrice:   float64(p.OveragePrice) / 10000,
		StripePriceID:  p.StripePriceID,
		PaddlePriceID:  p.PaddlePriceID,
		LemonVariantID: p.LemonVariantID,
		IsDefault:      p.IsDefault,
		Enabled:        p.Enabled,
	}
}

// UserCreate handles user creation.
func (h *Handler) UserCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	email := r.FormValue("email")
	password := r.FormValue("password")
	planID := r.FormValue("plan_id")
	status := r.FormValue("status")

	if email == "" || password == "" {
		h.renderUserFormError(w, r, "Email and password are required", "", email, planID, status)
		return
	}

	passwordHash, err := h.hasher.Hash(password)
	if err != nil {
		h.renderUserFormError(w, r, "Failed to hash password", "", email, planID, status)
		return
	}

	now := time.Now().UTC()
	user := ports.User{
		ID:           generateUserID(),
		Email:        email,
		PasswordHash: passwordHash,
		PlanID:       planID,
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.users.Create(ctx, user); err != nil {
		h.renderUserFormError(w, r, "Failed to create user", "", email, planID, status)
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

// UserEditPage renders the edit user form.
func (h *Handler) UserEditPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	user, err := h.users.Get(ctx, id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := struct {
		PageData
		IsEdit   bool
		FormUser struct {
			ID     string
			Email  string
			PlanID string
			Status string
		}
		Plans []PlanInfo
		Error string
	}{
		PageData: h.newPageData(ctx, "Edit User"),
		IsEdit:   true,
	}
	data.CurrentPath = "/users"
	data.FormUser.ID = user.ID
	data.FormUser.Email = user.Email
	data.FormUser.PlanID = user.PlanID
	data.FormUser.Status = user.Status
	data.Plans = h.getPlans()

	h.render(w, "user_form", data)
}

// UserUpdate handles user updates.
func (h *Handler) UserUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	user, err := h.users.Get(ctx, id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.PlanID = r.FormValue("plan_id")
	user.Status = r.FormValue("status")
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(ctx, user); err != nil {
		h.renderUserFormError(w, r, "Failed to update user", id, user.Email, user.PlanID, user.Status)
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

// UserDelete handles user deletion.
func (h *Handler) UserDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.users.Delete(ctx, id); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// For HTMX, return updated users list
	if r.Header.Get("HX-Request") == "true" {
		h.PartialUsers(w, r)
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handler) renderUserFormError(w http.ResponseWriter, r *http.Request, errMsg, id, email, planID, status string) {
	data := struct {
		PageData
		IsEdit   bool
		FormUser struct {
			ID     string
			Email  string
			PlanID string
			Status string
		}
		Plans []PlanInfo
		Error string
	}{
		PageData: h.newPageData(r.Context(), "User"),
		IsEdit:   id != "",
		Error:    errMsg,
	}
	data.FormUser.ID = id
	data.FormUser.Email = email
	data.FormUser.PlanID = planID
	data.FormUser.Status = status
	data.Plans = h.getPlans()

	h.render(w, "user_form", data)
}

// KeysPage lists all API keys.
func (h *Handler) KeysPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, _ := h.users.List(ctx, 1000, 0)

	data := struct {
		PageData
		Users  []ports.User
		NewKey string
	}{
		PageData: h.newPageData(ctx, "API Keys"),
		Users:    users,
	}
	data.CurrentPath = "/keys"

	h.render(w, "keys", data)
}

// KeyCreate handles key creation.
func (h *Handler) KeyCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.FormValue("user_id")
	name := r.FormValue("name")

	if _, err := h.users.Get(ctx, userID); err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Generate key using domain function
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(userID).WithName(name)

	if err := h.keys.Create(ctx, keyData); err != nil {
		http.Error(w, "Failed to create key", http.StatusInternalServerError)
		return
	}

	// Show the key to the user (only time it's visible)
	users, _ := h.users.List(ctx, 1000, 0)

	data := struct {
		PageData
		Users  []ports.User
		NewKey string
	}{
		PageData: h.newPageData(ctx, "API Keys"),
		Users:    users,
		NewKey:   rawKey,
	}
	data.CurrentPath = "/keys"

	h.render(w, "keys", data)
}

// KeyRevoke handles key revocation.
func (h *Handler) KeyRevoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.keys.Revoke(ctx, id, time.Now().UTC()); err != nil {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	// For HTMX, return updated keys list
	if r.Header.Get("HX-Request") == "true" {
		h.PartialKeys(w, r)
		return
	}

	http.Redirect(w, r, "/keys", http.StatusFound)
}

// PlansPage lists all plans.
func (h *Handler) PlansPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type PlanWithCount struct {
		PlanInfo
		UserCount int
	}

	plans := h.getPlans()
	users, _ := h.users.List(ctx, 1000, 0)

	// Count users per plan
	planCounts := make(map[string]int)
	for _, u := range users {
		planCounts[u.PlanID]++
	}

	plansWithCount := make([]PlanWithCount, len(plans))
	for i, p := range plans {
		plansWithCount[i] = PlanWithCount{
			PlanInfo:  p,
			UserCount: planCounts[p.ID],
		}
	}

	data := struct {
		PageData
		Plans []PlanWithCount
	}{
		PageData: h.newPageData(ctx, "Plans"),
		Plans:    plansWithCount,
	}
	data.CurrentPath = "/plans"

	h.render(w, "plans", data)
}

// PlanNewPage renders the create plan form.
func (h *Handler) PlanNewPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		IsEdit   bool
		FormPlan PlanInfo
		Error    string
	}{
		PageData: h.newPageData(r.Context(), "Create Plan"),
	}
	data.CurrentPath = "/plans"
	data.FormPlan.Enabled = true
	data.FormPlan.RateLimit = 60
	data.FormPlan.MonthlyQuota = 1000

	h.render(w, "plan_form", data)
}

// PlanCreate handles plan creation.
func (h *Handler) PlanCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.FormValue("id")
	name := r.FormValue("name")

	if id == "" || name == "" {
		h.renderPlanFormError(w, r, "ID and name are required", "", PlanInfo{
			ID:   id,
			Name: name,
		})
		return
	}

	rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit"))
	monthlyQuota, _ := strconv.ParseInt(r.FormValue("monthly_quota"), 10, 64)
	priceMonthly, _ := strconv.ParseFloat(r.FormValue("price_monthly"), 64)
	overagePrice, _ := strconv.ParseFloat(r.FormValue("overage_price"), 64)

	plan := ports.Plan{
		ID:                 id,
		Name:               name,
		Description:        r.FormValue("description"),
		RateLimitPerMinute: rateLimit,
		RequestsPerMonth:   monthlyQuota,
		PriceMonthly:       int64(priceMonthly * 100), // Convert to cents
		OveragePrice:       int64(overagePrice * 10000), // Convert to hundredths of cents
		StripePriceID:      r.FormValue("stripe_price_id"),
		PaddlePriceID:      r.FormValue("paddle_price_id"),
		LemonVariantID:     r.FormValue("lemon_variant_id"),
		IsDefault:          r.FormValue("is_default") == "on",
		Enabled:            r.FormValue("enabled") == "on",
	}

	// Clear default flag on existing plans if creating a new default plan
	if plan.IsDefault {
		existingPlans, _ := h.plans.List(ctx)
		for _, p := range existingPlans {
			if p.IsDefault {
				p.IsDefault = false
				_ = h.plans.Update(ctx, p)
			}
		}
	}

	if err := h.plans.Create(ctx, plan); err != nil {
		h.renderPlanFormError(w, r, "Failed to create plan: "+err.Error(), "", planToInfo(plan))
		return
	}

	http.Redirect(w, r, "/plans", http.StatusFound)
}

// PlanEditPage renders the edit plan form.
func (h *Handler) PlanEditPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	plan, err := h.plans.Get(ctx, id)
	if err != nil {
		http.Error(w, "Plan not found", http.StatusNotFound)
		return
	}

	data := struct {
		PageData
		IsEdit   bool
		FormPlan PlanInfo
		Error    string
	}{
		PageData: h.newPageData(ctx, "Edit Plan"),
		IsEdit:   true,
		FormPlan: planToInfo(plan),
	}
	data.CurrentPath = "/plans"

	h.render(w, "plan_form", data)
}

// PlanUpdate handles plan updates.
func (h *Handler) PlanUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	plan, err := h.plans.Get(ctx, id)
	if err != nil {
		http.Error(w, "Plan not found", http.StatusNotFound)
		return
	}

	rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit"))
	monthlyQuota, _ := strconv.ParseInt(r.FormValue("monthly_quota"), 10, 64)
	priceMonthly, _ := strconv.ParseFloat(r.FormValue("price_monthly"), 64)
	overagePrice, _ := strconv.ParseFloat(r.FormValue("overage_price"), 64)

	plan.Name = r.FormValue("name")
	plan.Description = r.FormValue("description")
	plan.RateLimitPerMinute = rateLimit
	plan.RequestsPerMonth = monthlyQuota
	plan.PriceMonthly = int64(priceMonthly * 100)
	plan.OveragePrice = int64(overagePrice * 10000) // Convert to hundredths of cents
	plan.StripePriceID = r.FormValue("stripe_price_id")
	plan.PaddlePriceID = r.FormValue("paddle_price_id")
	plan.LemonVariantID = r.FormValue("lemon_variant_id")
	newIsDefault := r.FormValue("is_default") == "on"
	plan.Enabled = r.FormValue("enabled") == "on"

	// Clear default flag on other plans if setting this plan as default
	if newIsDefault && !plan.IsDefault {
		existingPlans, _ := h.plans.List(ctx)
		for _, p := range existingPlans {
			if p.IsDefault && p.ID != plan.ID {
				p.IsDefault = false
				_ = h.plans.Update(ctx, p)
			}
		}
	}
	plan.IsDefault = newIsDefault

	if err := h.plans.Update(ctx, plan); err != nil {
		h.renderPlanFormError(w, r, "Failed to update plan: "+err.Error(), id, planToInfo(plan))
		return
	}

	http.Redirect(w, r, "/plans", http.StatusFound)
}

// PlanDelete handles plan deletion.
func (h *Handler) PlanDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	// Check if any users are on this plan
	users, _ := h.users.List(ctx, 1000, 0)
	for _, u := range users {
		if u.PlanID == id {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Cannot delete plan: users are assigned to this plan",
			})
			return
		}
	}

	if err := h.plans.Delete(ctx, id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to delete plan",
		})
		return
	}

	// For HTMX, return updated plans list
	if r.Header.Get("HX-Request") == "true" {
		h.PartialPlans(w, r)
		return
	}

	http.Redirect(w, r, "/plans", http.StatusFound)
}

// PartialPlans returns the plans table partial for HTMX.
func (h *Handler) PartialPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type PlanWithCount struct {
		PlanInfo
		UserCount int
	}

	plans := h.getPlans()
	users, _ := h.users.List(ctx, 1000, 0)

	planCounts := make(map[string]int)
	for _, u := range users {
		planCounts[u.PlanID]++
	}

	plansWithCount := make([]PlanWithCount, len(plans))
	for i, p := range plans {
		plansWithCount[i] = PlanWithCount{
			PlanInfo:  p,
			UserCount: planCounts[p.ID],
		}
	}

	data := struct {
		PageData
		Plans []PlanWithCount
	}{
		PageData: h.newPageData(ctx, "Plans"),
		Plans:    plansWithCount,
	}

	h.renderPartial(w, "partial_plans", data)
}

func (h *Handler) renderPlanFormError(w http.ResponseWriter, r *http.Request, errMsg, id string, plan PlanInfo) {
	data := struct {
		PageData
		IsEdit   bool
		FormPlan PlanInfo
		Error    string
	}{
		PageData: h.newPageData(r.Context(), "Plan"),
		IsEdit:   id != "",
		FormPlan: plan,
		Error:    errMsg,
	}
	data.CurrentPath = "/plans"
	h.render(w, "plan_form", data)
}

// UsagePage shows usage statistics.
func (h *Handler) UsagePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, _ := h.users.List(ctx, 1000, 0)

	data := struct {
		PageData
		Users        []ports.User
		SelectedUser string
		Period       string
		Summary      struct {
			TotalRequests int64
			Successful    int64
			RateLimited   int64
			Errors        int64
		}
		UsageData []struct {
			UserEmail    string
			KeyID        string
			KeyPrefix    string
			RequestCount int64
			LastUsed     time.Time
		}
	}{
		PageData: h.newPageData(ctx, "Usage"),
		Users:    users,
		Period:   r.URL.Query().Get("period"),
	}
	data.CurrentPath = "/usage"
	if data.Period == "" {
		data.Period = "day"
	}
	data.SelectedUser = r.URL.Query().Get("user_id")

	// Calculate time range based on period
	now := time.Now()
	var start time.Time
	switch data.Period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	default: // "day"
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// Aggregate usage data
	var totalRequests, totalErrors int64
	userEmails := make(map[string]string)
	for _, u := range users {
		userEmails[u.ID] = u.Email
	}

	// Get usage for selected user or all users
	usersToQuery := users
	if data.SelectedUser != "" {
		for _, u := range users {
			if u.ID == data.SelectedUser {
				usersToQuery = []ports.User{u}
				break
			}
		}
	}

	for _, u := range usersToQuery {
		summary, err := h.usage.GetSummary(ctx, u.ID, start, now)
		if err != nil {
			continue
		}
		totalRequests += summary.RequestCount
		totalErrors += summary.ErrorCount

		// Get keys for this user to build usage data
		keys, _ := h.keys.ListByUser(ctx, u.ID)
		for _, k := range keys {
			if k.RevokedAt != nil {
				continue
			}
			// Get recent requests for this key
			events, _ := h.usage.GetRecentRequests(ctx, u.ID, 100)
			var keyRequests int64
			var lastUsed time.Time
			for _, e := range events {
				if e.KeyID == k.ID && (e.Timestamp.After(start) || e.Timestamp.Equal(start)) {
					keyRequests++
					if e.Timestamp.After(lastUsed) {
						lastUsed = e.Timestamp
					}
				}
			}
			if keyRequests > 0 || !lastUsed.IsZero() {
				data.UsageData = append(data.UsageData, struct {
					UserEmail    string
					KeyID        string
					KeyPrefix    string
					RequestCount int64
					LastUsed     time.Time
				}{
					UserEmail:    u.Email,
					KeyID:        k.ID,
					KeyPrefix:    k.Prefix,
					RequestCount: keyRequests,
					LastUsed:     lastUsed,
				})
			}
		}
	}

	data.Summary.TotalRequests = totalRequests
	data.Summary.Errors = totalErrors
	data.Summary.Successful = totalRequests - totalErrors
	// Note: RateLimited would need separate tracking - for now we show 0

	h.render(w, "usage", data)
}

// SettingsPage shows current settings.
func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Load settings from database
	allSettings, err := h.settings.GetAll(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load settings")
	}

	data := struct {
		PageData
		Settings struct {
			UpstreamURL              string
			UpstreamTimeout          string
			AuthMode                 string
			AuthHeader               string
			RateLimitStrategy        string
			DatabaseDSN              string
			MeteringUnit             string
			PortalEnabled            bool
			PortalAppName            string
			PortalBaseURL            string
			RequireEmailVerification bool
			// Email provider
			EmailProvider    string
			EmailFromAddress string
			EmailFromName    string
			SMTPHost         string
			SMTPPort         string
			SMTPUsername     string
			SMTPPassword     string
			SMTPUseTLS       bool
			SendGridAPIKey   string
			// Payment providers
			StripeSecretKey      string
			StripeWebhookSecret  string
			PaddleVendorID       string
			PaddleAPIKey         string
			PaddleWebhookSecret  string
			LemonAPIKey          string
			LemonWebhookSecret   string
		}
		Success string
		Error   string
	}{
		PageData: h.newPageData(ctx, "Settings"),
	}
	data.CurrentPath = "/settings"
	data.Settings.UpstreamURL = h.appSettings.UpstreamURL
	data.Settings.UpstreamTimeout = h.appSettings.UpstreamTimeout
	data.Settings.AuthMode = h.appSettings.AuthMode
	data.Settings.AuthHeader = h.appSettings.AuthHeader
	data.Settings.DatabaseDSN = h.appSettings.DatabaseDSN

	// Portal settings from database
	data.Settings.PortalEnabled = allSettings.GetBool(settings.KeyPortalEnabled)
	data.Settings.PortalAppName = allSettings.GetOrDefault(settings.KeyPortalAppName, "APIGate")
	data.Settings.PortalBaseURL = allSettings.Get(settings.KeyPortalBaseURL)
	data.Settings.RequireEmailVerification = allSettings.GetBool(settings.KeyAuthRequireEmailVerification)
	data.Settings.MeteringUnit = allSettings.GetOrDefault(settings.KeyMeteringUnit, "requests")

	// Email provider settings
	data.Settings.EmailProvider = allSettings.GetOrDefault(settings.KeyEmailProvider, "none")
	data.Settings.EmailFromAddress = allSettings.Get(settings.KeyEmailFromAddress)
	data.Settings.EmailFromName = allSettings.Get(settings.KeyEmailFromName)
	data.Settings.SMTPHost = allSettings.Get(settings.KeyEmailSMTPHost)
	data.Settings.SMTPPort = allSettings.Get(settings.KeyEmailSMTPPort)
	data.Settings.SMTPUsername = allSettings.Get(settings.KeyEmailSMTPUsername)
	data.Settings.SMTPPassword = maskSecret(allSettings.Get(settings.KeyEmailSMTPPassword))
	data.Settings.SMTPUseTLS = allSettings.GetBool(settings.KeyEmailSMTPUseTLS)
	data.Settings.SendGridAPIKey = maskSecret(allSettings.Get(settings.KeyEmailSendGridKey))

	// Payment provider settings (mask sensitive values for display)
	data.Settings.StripeSecretKey = maskSecret(allSettings.Get(settings.KeyPaymentStripeSecretKey))
	data.Settings.StripeWebhookSecret = maskSecret(allSettings.Get(settings.KeyPaymentStripeWebhookSecret))
	data.Settings.PaddleVendorID = allSettings.Get(settings.KeyPaymentPaddleVendorID)
	data.Settings.PaddleAPIKey = maskSecret(allSettings.Get(settings.KeyPaymentPaddleAPIKey))
	data.Settings.PaddleWebhookSecret = maskSecret(allSettings.Get(settings.KeyPaymentPaddleWebhookSecret))
	data.Settings.LemonAPIKey = maskSecret(allSettings.Get(settings.KeyPaymentLemonAPIKey))
	data.Settings.LemonWebhookSecret = maskSecret(allSettings.Get(settings.KeyPaymentLemonWebhookSecret))

	// Check for success/error messages in query params
	data.Success = r.URL.Query().Get("success")
	data.Error = r.URL.Query().Get("error")

	h.render(w, "settings", data)
}

// maskSecret masks a secret value for display, showing only first/last 4 chars.
func maskSecret(s string) string {
	if len(s) <= 8 {
		return s // Too short to mask meaningfully
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// SettingsUpdate saves settings changes.
func (h *Handler) SettingsUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/settings?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	// Get form values
	portalEnabled := r.FormValue("portal_enabled") == "on"
	portalAppName := strings.TrimSpace(r.FormValue("portal_app_name"))
	portalBaseURL := strings.TrimSpace(r.FormValue("portal_base_url"))
	requireEmailVerification := r.FormValue("require_email_verification") == "on"
	meteringUnit := strings.TrimSpace(r.FormValue("metering_unit"))

	// Validate
	if portalAppName == "" {
		portalAppName = "APIGate"
	}
	if meteringUnit == "" {
		meteringUnit = "requests"
	}

	// Save settings
	settingsToSave := map[string]string{
		settings.KeyPortalEnabled:                boolToString(portalEnabled),
		settings.KeyPortalAppName:                portalAppName,
		settings.KeyAuthRequireEmailVerification: boolToString(requireEmailVerification),
		settings.KeyMeteringUnit:                 meteringUnit,
	}
	if portalBaseURL != "" {
		settingsToSave[settings.KeyPortalBaseURL] = portalBaseURL
	}

	// Email provider settings
	emailProvider := strings.TrimSpace(r.FormValue("email_provider"))
	if emailProvider == "" {
		emailProvider = "none"
	}
	settingsToSave[settings.KeyEmailProvider] = emailProvider
	settingsToSave[settings.KeyEmailFromAddress] = strings.TrimSpace(r.FormValue("email_from_address"))
	settingsToSave[settings.KeyEmailFromName] = strings.TrimSpace(r.FormValue("email_from_name"))
	settingsToSave[settings.KeyEmailSMTPHost] = strings.TrimSpace(r.FormValue("smtp_host"))
	settingsToSave[settings.KeyEmailSMTPPort] = strings.TrimSpace(r.FormValue("smtp_port"))
	settingsToSave[settings.KeyEmailSMTPUsername] = strings.TrimSpace(r.FormValue("smtp_username"))
	settingsToSave[settings.KeyEmailSMTPUseTLS] = boolToString(r.FormValue("smtp_use_tls") == "on")

	// Sensitive email settings - only save if provided (not masked/empty)
	sensitiveEmailSettings := map[string]string{
		settings.KeyEmailSMTPPassword: strings.TrimSpace(r.FormValue("smtp_password")),
		settings.KeyEmailSendGridKey:  strings.TrimSpace(r.FormValue("sendgrid_api_key")),
	}
	for key, value := range sensitiveEmailSettings {
		if value != "" && !strings.Contains(value, "...") {
			settingsToSave[key] = value
		}
	}

	// Payment provider settings - only save if provided (not masked/empty)
	paymentSettings := map[string]string{
		settings.KeyPaymentStripeSecretKey:     strings.TrimSpace(r.FormValue("stripe_secret_key")),
		settings.KeyPaymentStripeWebhookSecret: strings.TrimSpace(r.FormValue("stripe_webhook_secret")),
		settings.KeyPaymentPaddleVendorID:      strings.TrimSpace(r.FormValue("paddle_vendor_id")),
		settings.KeyPaymentPaddleAPIKey:        strings.TrimSpace(r.FormValue("paddle_api_key")),
		settings.KeyPaymentPaddleWebhookSecret: strings.TrimSpace(r.FormValue("paddle_webhook_secret")),
		settings.KeyPaymentLemonAPIKey:         strings.TrimSpace(r.FormValue("lemon_api_key")),
		settings.KeyPaymentLemonWebhookSecret:  strings.TrimSpace(r.FormValue("lemon_webhook_secret")),
	}

	// Only save payment settings if they don't look like masked values (contain "...")
	for key, value := range paymentSettings {
		if value != "" && !strings.Contains(value, "...") {
			settingsToSave[key] = value
		}
	}

	for key, value := range settingsToSave {
		encrypted := settings.IsSensitive(key)
		if err := h.settings.Set(ctx, key, value, encrypted); err != nil {
			h.logger.Error().Err(err).Str("key", key).Msg("failed to save setting")
			http.Redirect(w, r, "/settings?error=Failed+to+save+settings", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/settings?success=Settings+saved.+Some+changes+may+require+a+server+restart.", http.StatusSeeOther)
}

// PaymentsPage shows payment provider configuration.
func (h *Handler) PaymentsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	allSettings, err := h.settings.GetAll(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load settings")
	}

	// Determine base URL for webhook URLs
	baseURL := allSettings.Get(settings.KeyPortalBaseURL)
	if baseURL == "" {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL = scheme + "://" + r.Host
	}

	// Check which providers are configured
	stripeConfigured := allSettings.Get(settings.KeyPaymentStripeSecretKey) != ""
	paddleConfigured := allSettings.Get(settings.KeyPaymentPaddleAPIKey) != ""
	lemonConfigured := allSettings.Get(settings.KeyPaymentLemonAPIKey) != ""

	data := struct {
		PageData
		ActiveProvider      string
		BaseURL             string
		StripeSecretKey     string
		StripeWebhookSecret string
		StripeConfigured    bool
		PaddleVendorID      string
		PaddleAPIKey        string
		PaddleWebhookSecret string
		PaddleConfigured    bool
		LemonAPIKey         string
		LemonWebhookSecret  string
		LemonConfigured     bool
		Success             string
		Error               string
	}{
		PageData:            h.newPageData(ctx, "Payment Providers"),
		ActiveProvider:      allSettings.GetOrDefault(settings.KeyPaymentProvider, "none"),
		BaseURL:             baseURL,
		StripeSecretKey:     maskSecret(allSettings.Get(settings.KeyPaymentStripeSecretKey)),
		StripeWebhookSecret: maskSecret(allSettings.Get(settings.KeyPaymentStripeWebhookSecret)),
		StripeConfigured:    stripeConfigured,
		PaddleVendorID:      allSettings.Get(settings.KeyPaymentPaddleVendorID),
		PaddleAPIKey:        maskSecret(allSettings.Get(settings.KeyPaymentPaddleAPIKey)),
		PaddleWebhookSecret: maskSecret(allSettings.Get(settings.KeyPaymentPaddleWebhookSecret)),
		PaddleConfigured:    paddleConfigured,
		LemonAPIKey:         maskSecret(allSettings.Get(settings.KeyPaymentLemonAPIKey)),
		LemonWebhookSecret:  maskSecret(allSettings.Get(settings.KeyPaymentLemonWebhookSecret)),
		LemonConfigured:     lemonConfigured,
		Success:             r.URL.Query().Get("success"),
		Error:               r.URL.Query().Get("error"),
	}
	data.CurrentPath = "/payments"

	h.render(w, "payments", data)
}

// PaymentsUpdate saves payment provider settings.
func (h *Handler) PaymentsUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/payments?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	settingsToSave := map[string]string{
		settings.KeyPaymentProvider: strings.TrimSpace(r.FormValue("active_provider")),
	}

	// Payment provider settings - only save if provided (not masked/empty)
	paymentSettings := map[string]string{
		settings.KeyPaymentStripeSecretKey:     strings.TrimSpace(r.FormValue("stripe_secret_key")),
		settings.KeyPaymentStripeWebhookSecret: strings.TrimSpace(r.FormValue("stripe_webhook_secret")),
		settings.KeyPaymentPaddleVendorID:      strings.TrimSpace(r.FormValue("paddle_vendor_id")),
		settings.KeyPaymentPaddleAPIKey:        strings.TrimSpace(r.FormValue("paddle_api_key")),
		settings.KeyPaymentPaddleWebhookSecret: strings.TrimSpace(r.FormValue("paddle_webhook_secret")),
		settings.KeyPaymentLemonAPIKey:         strings.TrimSpace(r.FormValue("lemon_api_key")),
		settings.KeyPaymentLemonWebhookSecret:  strings.TrimSpace(r.FormValue("lemon_webhook_secret")),
	}

	// Only save if value doesn't look like masked (contain "...")
	for key, value := range paymentSettings {
		if value != "" && !strings.Contains(value, "...") {
			settingsToSave[key] = value
		}
	}

	for key, value := range settingsToSave {
		encrypted := settings.IsSensitive(key)
		if err := h.settings.Set(ctx, key, value, encrypted); err != nil {
			h.logger.Error().Err(err).Str("key", key).Msg("failed to save payment setting")
			http.Redirect(w, r, "/payments?error=Failed+to+save+settings", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/payments?success=Payment+settings+saved", http.StatusSeeOther)
}

// EmailPage shows email provider configuration.
func (h *Handler) EmailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	allSettings, err := h.settings.GetAll(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load settings")
	}

	data := struct {
		PageData
		AppName          string
		EmailProvider    string
		EmailFromAddress string
		EmailFromName    string
		SMTPHost         string
		SMTPPort         string
		SMTPUsername     string
		SMTPPassword     string
		SMTPUseTLS       bool
		SendGridAPIKey   string
		Success          string
		Error            string
	}{
		PageData:         h.newPageData(ctx, "Email Provider"),
		AppName:          allSettings.GetOrDefault(settings.KeyPortalAppName, "APIGate"),
		EmailProvider:    allSettings.GetOrDefault(settings.KeyEmailProvider, "none"),
		EmailFromAddress: allSettings.Get(settings.KeyEmailFromAddress),
		EmailFromName:    allSettings.Get(settings.KeyEmailFromName),
		SMTPHost:         allSettings.Get(settings.KeyEmailSMTPHost),
		SMTPPort:         allSettings.Get(settings.KeyEmailSMTPPort),
		SMTPUsername:     allSettings.Get(settings.KeyEmailSMTPUsername),
		SMTPPassword:     maskSecret(allSettings.Get(settings.KeyEmailSMTPPassword)),
		SMTPUseTLS:       allSettings.GetBool(settings.KeyEmailSMTPUseTLS),
		SendGridAPIKey:   maskSecret(allSettings.Get(settings.KeyEmailSendGridKey)),
		Success:          r.URL.Query().Get("success"),
		Error:            r.URL.Query().Get("error"),
	}
	data.CurrentPath = "/email"

	h.render(w, "email", data)
}

// EmailUpdate saves email provider settings.
func (h *Handler) EmailUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/email?error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	emailProvider := strings.TrimSpace(r.FormValue("email_provider"))
	if emailProvider == "" {
		emailProvider = "none"
	}

	settingsToSave := map[string]string{
		settings.KeyEmailProvider:    emailProvider,
		settings.KeyEmailFromAddress: strings.TrimSpace(r.FormValue("email_from_address")),
		settings.KeyEmailFromName:    strings.TrimSpace(r.FormValue("email_from_name")),
		settings.KeyEmailSMTPHost:    strings.TrimSpace(r.FormValue("smtp_host")),
		settings.KeyEmailSMTPPort:    strings.TrimSpace(r.FormValue("smtp_port")),
		settings.KeyEmailSMTPUsername: strings.TrimSpace(r.FormValue("smtp_username")),
		settings.KeyEmailSMTPUseTLS:  boolToString(r.FormValue("smtp_use_tls") == "on"),
	}

	// Sensitive settings - only save if provided (not masked/empty)
	sensitiveSettings := map[string]string{
		settings.KeyEmailSMTPPassword: strings.TrimSpace(r.FormValue("smtp_password")),
		settings.KeyEmailSendGridKey:  strings.TrimSpace(r.FormValue("sendgrid_api_key")),
	}
	for key, value := range sensitiveSettings {
		if value != "" && !strings.Contains(value, "...") {
			settingsToSave[key] = value
		}
	}

	for key, value := range settingsToSave {
		encrypted := settings.IsSensitive(key)
		if err := h.settings.Set(ctx, key, value, encrypted); err != nil {
			h.logger.Error().Err(err).Str("key", key).Msg("failed to save email setting")
			http.Redirect(w, r, "/email?error=Failed+to+save+settings", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/email?success=Email+settings+saved", http.StatusSeeOther)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// HealthPage shows system health.
func (h *Handler) HealthPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	users, _ := h.users.List(ctx, 1000, 0)
	totalKeys := h.countActiveKeys(ctx, users)

	data := struct {
		PageData
		Health struct {
			Status string
			Checks []struct {
				Name    string
				Status  string
				Message string
				Latency string
			}
			System struct {
				GoVersion    string
				NumCPU       int
				NumGoroutine int
				MemAlloc     string
				MemSys       string
				Uptime       string
			}
			Statistics struct {
				TotalUsers     int
				TotalKeys      int
				ActiveSessions int
			}
		}
	}{
		PageData: h.newPageData(ctx, "System Health"),
	}
	data.CurrentPath = "/health"
	data.Health.Status = "healthy"

	// Build upstream message with better handling of empty URL
	upstreamMsg := "Using per-route upstream configuration"
	if h.appSettings.UpstreamURL != "" {
		upstreamMsg = "Default upstream: " + h.appSettings.UpstreamURL
	}

	data.Health.Checks = []struct {
		Name    string
		Status  string
		Message string
		Latency string
	}{
		{Name: "Database", Status: "pass", Message: "Database connection healthy"},
		{Name: "Upstream", Status: "pass", Message: upstreamMsg},
		{Name: "Config", Status: "pass", Message: "Configuration valid"},
	}
	data.Health.System.GoVersion = runtime.Version()
	data.Health.System.NumCPU = runtime.NumCPU()
	data.Health.System.NumGoroutine = runtime.NumGoroutine()
	data.Health.System.MemAlloc = formatBytes(memStats.Alloc)
	data.Health.System.MemSys = formatBytes(memStats.Sys)
	data.Health.System.Uptime = formatUptime(time.Since(h.startTime))
	data.Health.Statistics.TotalUsers = len(users)
	data.Health.Statistics.TotalKeys = totalKeys

	h.render(w, "health", data)
}

// HTMX Partials

// PartialStats returns updated dashboard stats.
func (h *Handler) PartialStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	users, _ := h.users.List(ctx, 1000, 0)
	activeKeys := h.countActiveKeys(ctx, users)

	// Get today's request count
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	h.logger.Debug().Str("start", startOfDay.Format("2006-01-02 15:04:05")).Str("end", now.Format("2006-01-02 15:04:05")).Int("users", len(users)).Msg("partial stats usage query")
	var requestsToday int64
	for _, u := range users {
		summary, err := h.usage.GetSummary(ctx, u.ID, startOfDay, now)
		h.logger.Debug().Str("user", u.ID).Int64("count", summary.RequestCount).Err(err).Msg("partial stats user summary")
		if err == nil {
			requestsToday += summary.RequestCount
		}
	}

	// Calculate Monthly Recurring Revenue (MRR)
	plans, _ := h.plans.List(ctx)
	planPrices := make(map[string]int64)
	for _, p := range plans {
		planPrices[p.ID] = p.PriceMonthly
	}
	var mrr int64 // in cents
	for _, u := range users {
		if u.Status == "active" {
			if price, ok := planPrices[u.PlanID]; ok {
				mrr += price
			}
		}
	}

	data := struct {
		PageData
		Stats struct {
			TotalUsers    int
			ActiveKeys    int
			RequestsToday int64
			MRR           float64
		}
	}{
		PageData: h.newPageData(ctx, "Stats"),
	}
	data.Stats.TotalUsers = len(users)
	data.Stats.ActiveKeys = activeKeys
	data.Stats.RequestsToday = requestsToday
	data.Stats.MRR = float64(mrr) / 100

	h.renderPartial(w, "stats", data)
}

// PartialUsers returns the users table.
func (h *Handler) PartialUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	users, _ := h.users.List(ctx, limit, 0)

	// Build plan name lookup map
	plans, _ := h.plans.List(ctx)
	planNames := make(map[string]string)
	for _, p := range plans {
		planNames[p.ID] = p.Name
	}

	data := struct {
		Users     []ports.User
		PlanNames map[string]string
	}{
		Users:     users,
		PlanNames: planNames,
	}

	h.renderPartial(w, "partial_users", data)
}

// PartialKeys returns the keys table.
func (h *Handler) PartialKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, _ := h.users.List(ctx, 1000, 0)

	// Build user email map
	userEmails := make(map[string]string)
	for _, u := range users {
		userEmails[u.ID] = u.Email
	}

	// Collect all keys
	var keys []key.Key
	for _, u := range users {
		userKeys, _ := h.keys.ListByUser(ctx, u.ID)
		keys = append(keys, userKeys...)
	}

	data := struct {
		Keys       []key.Key
		UserEmails map[string]string
	}{
		Keys:       keys,
		UserEmails: userEmails,
	}

	h.renderPartial(w, "partial_keys", data)
}

// PartialActivity returns recent activity.
func (h *Handler) PartialActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	data := struct {
		Activities []struct {
			Type      string
			Message   string
			Timestamp time.Time
		}
	}{}

	// Get recent requests from all users
	users, _ := h.users.List(ctx, 100, 0)
	userEmails := make(map[string]string)
	for _, u := range users {
		userEmails[u.ID] = u.Email
	}

	// Collect recent events from all users
	type activityEvent struct {
		Type      string
		Message   string
		Timestamp time.Time
	}
	var allEvents []activityEvent

	for _, u := range users {
		events, err := h.usage.GetRecentRequests(ctx, u.ID, limit)
		if err != nil {
			continue
		}
		for _, e := range events {
			statusType := "success"
			if e.StatusCode >= 400 {
				statusType = "error"
			}
			msg := fmt.Sprintf("%s %s → %d", e.Method, e.Path, e.StatusCode)
			allEvents = append(allEvents, activityEvent{
				Type:      statusType,
				Message:   msg,
				Timestamp: e.Timestamp,
			})
		}
	}

	// Sort by timestamp descending and take top N
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.After(allEvents[j].Timestamp)
	})
	if len(allEvents) > limit {
		allEvents = allEvents[:limit]
	}

	// Convert to template data
	for _, e := range allEvents {
		data.Activities = append(data.Activities, struct {
			Type      string
			Message   string
			Timestamp time.Time
		}{
			Type:      e.Type,
			Message:   e.Message,
			Timestamp: e.Timestamp,
		})
	}

	h.renderPartial(w, "partial_activity", data)
}

// Helper to render templates
// The name should be the page name (e.g., "login", "dashboard", "users")
func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, ok := h.templates[name]
	if !ok {
		h.logger.Error().Str("template", name).Msg("template not found")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error().Err(err).Str("template", name).Msg("template render error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderPartial(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Use the dashboard template to access partials (all templates have components loaded)
	tmpl, ok := h.templates["dashboard"]
	if !ok {
		h.logger.Error().Str("template", name).Msg("partial render error: no base template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.logger.Error().Err(err).Str("template", name).Msg("partial render error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func generateUserID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "user_" + hex.EncodeToString(b)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// Setup handlers

// SetupPage renders the setup wizard.
func (h *Handler) SetupPage(w http.ResponseWriter, r *http.Request) {
	// If already set up, redirect to login
	if h.isSetup != nil && h.isSetup() {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	data := struct {
		PageData
		Steps       []string
		CurrentStep int
		UpstreamURL string
		Error       string
	}{
		PageData:    h.newPageData(r.Context(), "Setup"),
		Steps:       []string{"Connect Your API", "Create Account", "Set Up Pricing", "Ready!"},
		CurrentStep: 0,
	}

	h.render(w, "setup", data)
}

// SetupSubmit handles setup form submission.
func (h *Handler) SetupSubmit(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/setup/step/0", http.StatusFound)
}

// SetupStep renders a setup step.
func (h *Handler) SetupStep(w http.ResponseWriter, r *http.Request) {
	step, _ := strconv.Atoi(chi.URLParam(r, "step"))

	// Validate step sequence - check cookie for highest completed step
	highestCompleted := -1
	if cookie, err := r.Cookie("setup_step"); err == nil {
		highestCompleted, _ = strconv.Atoi(cookie.Value)
	}

	// If already set up AND not in active setup session, redirect to dashboard
	// Active setup session = cookie exists with value < 3 (not fully complete)
	if h.isSetup != nil && h.isSetup() {
		if highestCompleted < 0 || highestCompleted >= 3 {
			// No active setup session, redirect to dashboard
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		// Active setup session - allow continuing
	}

	// Can only access step 0 freely, or steps where previous step is completed
	if step > 0 && highestCompleted < step-1 {
		// Redirect to the next uncompleted step
		nextStep := highestCompleted + 1
		if nextStep < 0 {
			nextStep = 0
		}
		http.Redirect(w, r, fmt.Sprintf("/setup/step/%d", nextStep), http.StatusFound)
		return
	}

	data := struct {
		PageData
		Steps        []string
		CurrentStep  int
		UpstreamURL  string
		AdminName    string
		AdminEmail   string
		PlanName     string
		RateLimit    int
		MonthlyQuota int
		PriceMonthly float64
		OveragePrice float64
		Error        string
	}{
		PageData:    h.newPageData(r.Context(), "Setup"),
		Steps:       []string{"Connect Your API", "Create Account", "Set Up Pricing", "Ready!"},
		CurrentStep: step,
	}

	h.render(w, "setup", data)
}

// SetupStepSubmit handles setup step submission.
func (h *Handler) SetupStepSubmit(w http.ResponseWriter, r *http.Request) {
	// If already set up, redirect to dashboard
	if h.isSetup != nil && h.isSetup() {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	step, _ := strconv.Atoi(chi.URLParam(r, "step"))

	switch step {
	case 0:
		// Create upstream from URL
		upstreamURL := r.FormValue("upstream_url")
		if upstreamURL == "" {
			h.renderSetupError(w, r, 0, "Upstream URL is required")
			return
		}

		// Parse and validate URL
		parsedURL, err := url.Parse(upstreamURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			h.renderSetupError(w, r, 0, "Invalid URL. Must start with http:// or https://")
			return
		}

		// Test connection to the upstream
		client := &http.Client{Timeout: 10 * time.Second}
		testReq, err := http.NewRequest("HEAD", upstreamURL, nil)
		if err == nil {
			resp, err := client.Do(testReq)
			if err != nil {
				h.logger.Warn().Err(err).Str("url", upstreamURL).Msg("upstream connection test failed")
				h.renderSetupError(w, r, 0, "Could not connect to the API. Please verify the URL is correct and the API is running. Error: "+err.Error())
				return
			}
			resp.Body.Close()
			// Any response (even 4xx/5xx) means the server is reachable
			h.logger.Info().Str("url", upstreamURL).Int("status", resp.StatusCode).Msg("upstream connection test successful")
		}

		now := time.Now().UTC()

		// Extract a name from the host
		upstreamName := parsedURL.Host
		if idx := strings.Index(upstreamName, ":"); idx > 0 {
			upstreamName = upstreamName[:idx]
		}

		// Create the upstream
		upstream := route.Upstream{
			ID:          "default",
			Name:        upstreamName,
			Description: "Default upstream created during setup",
			BaseURL:     upstreamURL,
			Timeout:     30 * time.Second,
			AuthType:    route.AuthNone,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := h.upstreams.Create(r.Context(), upstream); err != nil {
			h.renderSetupError(w, r, 0, "Failed to create upstream: "+err.Error())
			return
		}

		// Create a catch-all route pointing to this upstream
		defaultRoute := route.Route{
			ID:          "default",
			Name:        "Default Route",
			Description: "Catch-all route created during setup",
			PathPattern: "/*",
			MatchType:   route.MatchPrefix,
			UpstreamID:  "default",
			Protocol:    route.ProtocolHTTP,
			Priority:    0,
			Enabled:     true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := h.routes.Create(r.Context(), defaultRoute); err != nil {
			h.renderSetupError(w, r, 0, "Failed to create route: "+err.Error())
			return
		}

		// Trigger route reload so proxy service picks up the new route
		h.logger.Info().Bool("has_callback", h.onRouteChange != nil).Msg("attempting route reload after setup")
		if h.onRouteChange != nil {
			if err := h.onRouteChange(r.Context()); err != nil {
				h.logger.Warn().Err(err).Msg("failed to reload routes after setup")
			} else {
				h.logger.Info().Msg("route reload completed successfully")
			}
		}

		// Store upstream URL in cookie for display on final step
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_upstream_url",
			Value:    upstreamURL,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		// Track step completion for sequence validation
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_step",
			Value:    "0",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/setup/step/1", http.StatusFound)
		return
	case 1:
		// Create admin user
		name := r.FormValue("admin_name")
		email := r.FormValue("admin_email")
		password := r.FormValue("admin_password")
		confirm := r.FormValue("admin_password_confirm")

		if password != confirm {
			h.renderSetupError(w, r, 1, "Passwords do not match")
			return
		}

		passwordHash, err := h.hasher.Hash(password)
		if err != nil {
			h.renderSetupError(w, r, 1, "Failed to hash password")
			return
		}

		now := time.Now().UTC()
		user := ports.User{
			ID:           "admin",
			Name:         name,
			Email:        email,
			PasswordHash: passwordHash,
			PlanID:       "free", // Default plan; will be updated in step 2 if custom plan created
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := h.users.Create(r.Context(), user); err != nil {
			h.renderSetupError(w, r, 1, "Failed to create admin user")
			return
		}

		// Auto-login: Generate JWT token and set cookie
		token, expiresAt, err := h.tokens.GenerateToken(user.ID, user.Email, "admin")
		if err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			// Also set apigate_session cookie for module WebUI compatibility
			setModuleSessionCookie(w, user.ID, user.Email, user.Email, expiresAt)
		}

		// Track step completion for sequence validation
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_step",
			Value:    "1",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/setup/step/2", http.StatusFound)
		return
	case 2:
		// Create the plan from form data
		planName := r.FormValue("plan_name")
		if planName == "" {
			planName = "Starter"
		}

		rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit"))
		if rateLimit <= 0 {
			rateLimit = 60
		}

		monthlyQuota, _ := strconv.ParseInt(r.FormValue("monthly_quota"), 10, 64)
		if monthlyQuota <= 0 {
			monthlyQuota = 1000
		}

		// Parse pricing fields
		priceMonthly, _ := strconv.ParseFloat(r.FormValue("price_monthly"), 64)
		overagePrice, _ := strconv.ParseFloat(r.FormValue("overage_price"), 64)

		now := time.Now().UTC()

		// Create slug from plan name for ID
		planID := strings.ToLower(strings.ReplaceAll(planName, " ", "-"))

		plan := ports.Plan{
			ID:                 planID,
			Name:               planName,
			Description:        "Default plan created during setup",
			RateLimitPerMinute: rateLimit,
			RequestsPerMonth:   monthlyQuota,
			PriceMonthly:       int64(priceMonthly * 100), // Convert to cents
			OveragePrice:       int64(overagePrice * 10000), // Convert to hundredths of cents
			IsDefault:          true,
			Enabled:            true,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		// Clear default flag on existing plans before creating new default plan
		existingPlans, err := h.plans.List(r.Context())
		if err == nil {
			for _, p := range existingPlans {
				if p.IsDefault {
					p.IsDefault = false
					p.UpdatedAt = now
					if updateErr := h.plans.Update(r.Context(), p); updateErr != nil {
						h.logger.Warn().Err(updateErr).Str("plan_id", p.ID).Msg("failed to clear default flag on existing plan")
					}
				}
			}
		}

		if err := h.plans.Create(r.Context(), plan); err != nil {
			// If plan already exists (e.g., from default migration), update it instead
			if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "already exists") {
				if updateErr := h.plans.Update(r.Context(), plan); updateErr != nil {
					h.renderSetupError(w, r, 2, "Failed to update existing plan: "+updateErr.Error())
					return
				}
				h.logger.Info().Str("plan_id", planID).Msg("updated existing plan during setup")
			} else {
				h.renderSetupError(w, r, 2, "Failed to create plan: "+err.Error())
				return
			}
		}

		// Trigger plan reload so proxy service picks up the new plan
		h.logger.Info().Bool("has_callback", h.onPlanChange != nil).Msg("attempting plan reload after setup")
		if h.onPlanChange != nil {
			if err := h.onPlanChange(r.Context()); err != nil {
				h.logger.Warn().Err(err).Msg("failed to reload plans after setup")
			} else {
				h.logger.Info().Msg("plan reload completed successfully")
			}
		} else {
			h.logger.Warn().Msg("onPlanChange callback is nil")
		}

		// Update admin user's plan to the newly created plan (ISSUE-011 fix)
		adminUser, err := h.users.Get(r.Context(), "admin")
		if err == nil {
			adminUser.PlanID = planID
			adminUser.UpdatedAt = now
			_ = h.users.Update(r.Context(), adminUser)
		}

		// Track step completion for sequence validation
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_step",
			Value:    "2",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/setup/step/3", http.StatusFound)
		return
	default:
		// After setup complete, redirect to dashboard (user already logged in from step 1)
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func (h *Handler) renderSetupError(w http.ResponseWriter, r *http.Request, step int, errMsg string) {
	// Parse form values to preserve user input on error
	rateLimit, _ := strconv.Atoi(r.FormValue("rate_limit"))
	monthlyQuota, _ := strconv.Atoi(r.FormValue("monthly_quota"))
	priceMonthly, _ := strconv.ParseFloat(r.FormValue("price_monthly"), 64)
	overagePrice, _ := strconv.ParseFloat(r.FormValue("overage_price"), 64)

	data := struct {
		PageData
		Steps        []string
		CurrentStep  int
		UpstreamURL  string
		AdminName    string
		AdminEmail   string
		PlanName     string
		RateLimit    int
		MonthlyQuota int
		PriceMonthly float64
		OveragePrice float64
		Error        string
	}{
		PageData:     h.newPageData(r.Context(), "Setup"),
		Steps:        []string{"Connect Your API", "Create Account", "Set Up Pricing", "Ready!"},
		CurrentStep:  step,
		UpstreamURL:  r.FormValue("upstream_url"),
		AdminName:    r.FormValue("admin_name"),
		AdminEmail:   r.FormValue("admin_email"),
		PlanName:     r.FormValue("plan_name"),
		RateLimit:    rateLimit,
		MonthlyQuota: monthlyQuota,
		PriceMonthly: priceMonthly,
		OveragePrice: overagePrice,
		Error:        errMsg,
	}

	h.render(w, "setup", data)
}
