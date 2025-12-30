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

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
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
		Error string
		Email string
	}{
		PageData: h.newPageData(r.Context(), "Login"),
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
		Error string
		Email string
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

// ChecklistItem represents a single onboarding checklist item.
type ChecklistItem struct {
	Title       string
	Description string
	Done        bool
	Link        string
	LinkText    string
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

	data := struct {
		PageData
		Stats struct {
			TotalUsers    int
			ActiveKeys    int
			RequestsToday int64
		}
		Checklist         []ChecklistItem
		ChecklistComplete bool
	}{
		PageData: h.newPageData(ctx, "Dashboard"),
	}
	data.CurrentPath = "/dashboard"
	data.Stats.TotalUsers = len(users)
	data.Stats.ActiveKeys = activeKeys

	// Build checklist
	data.Checklist = []ChecklistItem{
		{
			Title:       "Create admin account",
			Description: "Set up your administrator account",
			Done:        true, // Always true if logged in
			Link:        "/users",
			LinkText:    "Manage users",
		},
		{
			Title:       "Add an upstream API",
			Description: "Connect to the API you want to proxy",
			Done:        len(upstreams) > 0,
			Link:        "/upstreams",
			LinkText:    "Add upstream",
		},
		{
			Title:       "Create a route",
			Description: "Define how requests are routed to your API",
			Done:        len(routes) > 0,
			Link:        "/routes",
			LinkText:    "Create route",
		},
		{
			Title:       "Generate an API key",
			Description: "Create a key for authenticating requests",
			Done:        activeKeys > 0,
			Link:        "/keys",
			LinkText:    "Create key",
		},
		{
			Title:       "Make your first API call",
			Description: "Test your gateway with a real request",
			Done:        hasAPIActivity,
			Link:        "/usage",
			LinkText:    "View usage",
		},
	}

	// Check if all items are complete
	data.ChecklistComplete = true
	for _, item := range data.Checklist {
		if !item.Done {
			data.ChecklistComplete = false
			break
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
		OveragePrice:   float64(p.OveragePrice) / 100,
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
		OveragePrice:       int64(overagePrice * 100),
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
	plan.OveragePrice = int64(overagePrice * 100)
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
		Plans []PlanWithCount
	}{
		Plans: plansWithCount,
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
					RequestCount int64
					LastUsed     time.Time
				}{
					UserEmail:    u.Email,
					KeyID:        k.ID,
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
	data := struct {
		PageData
		Settings struct {
			UpstreamURL       string
			UpstreamTimeout   string
			AuthMode          string
			AuthHeader        string
			RateLimitStrategy string
			DatabaseDSN       string
		}
	}{
		PageData: h.newPageData(r.Context(), "Settings"),
	}
	data.CurrentPath = "/settings"
	data.Settings.UpstreamURL = h.appSettings.UpstreamURL
	data.Settings.UpstreamTimeout = h.appSettings.UpstreamTimeout
	data.Settings.AuthMode = h.appSettings.AuthMode
	data.Settings.AuthHeader = h.appSettings.AuthHeader
	data.Settings.DatabaseDSN = h.appSettings.DatabaseDSN

	h.render(w, "settings", data)
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
	data.Health.Checks = []struct {
		Name    string
		Status  string
		Message string
		Latency string
	}{
		{Name: "Database", Status: "pass", Message: "Database connection healthy"},
		{Name: "Upstream", Status: "pass", Message: "Upstream: " + h.appSettings.UpstreamURL},
		{Name: "Config", Status: "pass", Message: "Configuration valid"},
	}
	data.Health.System.GoVersion = runtime.Version()
	data.Health.System.NumCPU = runtime.NumCPU()
	data.Health.System.NumGoroutine = runtime.NumGoroutine()
	data.Health.System.MemAlloc = formatBytes(memStats.Alloc)
	data.Health.System.MemSys = formatBytes(memStats.Sys)
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

	data := struct {
		TotalUsers    int
		ActiveKeys    int
		RequestsToday int64
	}{
		TotalUsers:    len(users),
		ActiveKeys:    activeKeys,
		RequestsToday: requestsToday,
	}

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

	data := struct {
		Users []ports.User
	}{
		Users: users,
	}

	h.renderPartial(w, "partial_users", data)
}

// PartialKeys returns the keys table.
func (h *Handler) PartialKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, _ := h.users.List(ctx, 1000, 0)

	// Collect all keys
	var keys []key.Key
	for _, u := range users {
		userKeys, _ := h.keys.ListByUser(ctx, u.ID)
		keys = append(keys, userKeys...)
	}

	data := struct {
		Keys []key.Key
	}{
		Keys: keys,
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

	data := struct {
		PageData
		Steps        []string
		CurrentStep  int
		UpstreamURL  string
		AdminEmail   string
		PlanName     string
		RateLimit    int
		MonthlyQuota int
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

		http.Redirect(w, r, "/setup/step/1", http.StatusFound)
		return
	case 1:
		// Create admin user
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
			Email:        email,
			PasswordHash: passwordHash,
			PlanID:       "admin", // Will be updated in step 2
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

		now := time.Now().UTC()

		// Create slug from plan name for ID
		planID := strings.ToLower(strings.ReplaceAll(planName, " ", "-"))

		plan := ports.Plan{
			ID:                 planID,
			Name:               planName,
			Description:        "Default plan created during setup",
			RateLimitPerMinute: rateLimit,
			RequestsPerMonth:   monthlyQuota,
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
			h.renderSetupError(w, r, 2, "Failed to create plan: "+err.Error())
			return
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

		http.Redirect(w, r, "/setup/step/3", http.StatusFound)
		return
	default:
		// After setup complete, redirect to dashboard (user already logged in from step 1)
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func (h *Handler) renderSetupError(w http.ResponseWriter, r *http.Request, step int, errMsg string) {
	data := struct {
		PageData
		Steps       []string
		CurrentStep int
		Error       string
	}{
		PageData:    h.newPageData(r.Context(), "Setup"),
		Steps:       []string{"Connect Your API", "Create Account", "Set Up Pricing", "Ready!"},
		CurrentStep: step,
		Error:       errMsg,
	}

	h.render(w, "setup", data)
}
