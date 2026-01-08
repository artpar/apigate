// Package web provides the user self-service portal.
package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/core/terminology"
	domainAuth "github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/entitlement"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// PortalHandler provides the user portal endpoints.
type PortalHandler struct {
	tokens           *auth.TokenService
	users            ports.UserStore
	keys             ports.KeyStore
	usage            ports.UsageStore
	plans            ports.PlanStore
	sessions         ports.SessionStore
	authTokens       ports.TokenStore
	emailSender      ports.EmailSender
	settings         ports.SettingsStore
	subscriptions    ports.SubscriptionStore
	invoices         ports.InvoiceStore
	entitlements     ports.EntitlementStore
	planEntitlements ports.PlanEntitlementStore
	webhooks         ports.WebhookStore
	deliveries       ports.DeliveryStore
	logger           zerolog.Logger
	hasher           ports.Hasher
	idGen            ports.IDGenerator
	payment          ports.PaymentProvider

	// Portal-specific settings
	baseURL string
	appName string
}

// PortalDeps contains dependencies for the portal handler.
type PortalDeps struct {
	Users            ports.UserStore
	Keys             ports.KeyStore
	Usage            ports.UsageStore
	Plans            ports.PlanStore
	Sessions         ports.SessionStore
	AuthTokens       ports.TokenStore
	EmailSender      ports.EmailSender
	Settings         ports.SettingsStore
	Subscriptions    ports.SubscriptionStore
	Invoices         ports.InvoiceStore
	Entitlements     ports.EntitlementStore
	PlanEntitlements ports.PlanEntitlementStore
	Webhooks         ports.WebhookStore
	Deliveries       ports.DeliveryStore
	Logger           zerolog.Logger
	Hasher           ports.Hasher
	IDGen            ports.IDGenerator
	Payment          ports.PaymentProvider
	JWTSecret        string
	BaseURL          string
	AppName          string
}

// NewPortalHandler creates a new user portal handler.
func NewPortalHandler(deps PortalDeps) (*PortalHandler, error) {
	appName := deps.AppName
	if appName == "" {
		appName = "APIGate"
	}

	return &PortalHandler{
		tokens:           auth.NewTokenService(deps.JWTSecret, 7*24*time.Hour), // 7 day sessions
		users:            deps.Users,
		keys:             deps.Keys,
		usage:            deps.Usage,
		plans:            deps.Plans,
		sessions:         deps.Sessions,
		authTokens:       deps.AuthTokens,
		emailSender:      deps.EmailSender,
		settings:         deps.Settings,
		subscriptions:    deps.Subscriptions,
		invoices:         deps.Invoices,
		entitlements:     deps.Entitlements,
		planEntitlements: deps.PlanEntitlements,
		webhooks:         deps.Webhooks,
		deliveries:       deps.Deliveries,
		logger:           deps.Logger,
		hasher:           deps.Hasher,
		idGen:            deps.IDGen,
		payment:          deps.Payment,
		baseURL:          deps.BaseURL,
		appName:          appName,
	}, nil
}

// getLabels returns the terminology labels based on settings.
func (h *PortalHandler) getLabels(ctx context.Context) terminology.Labels {
	if h.settings == nil {
		return terminology.Default()
	}
	setting, err := h.settings.Get(ctx, "metering_unit")
	if err != nil || setting.Value == "" {
		return terminology.Default()
	}
	return terminology.ForUnit(setting.Value)
}

// Router returns the portal router.
func (h *PortalHandler) Router() chi.Router {
	r := chi.NewRouter()

	// Landing page (public, redirects to dashboard if logged in)
	r.Get("/", h.LandingPage)

	// Public routes (no auth required)
	r.Get("/signup", h.SignupPage)
	r.Post("/signup", h.SignupSubmit)
	r.Get("/login", h.PortalLoginPage)
	r.Post("/login", h.PortalLoginSubmit)
	r.Get("/forgot-password", h.ForgotPasswordPage)
	r.Post("/forgot-password", h.ForgotPasswordSubmit)
	r.Get("/reset-password", h.ResetPasswordPage)
	r.Post("/reset-password", h.ResetPasswordSubmit)
	r.Get("/verify-email", h.VerifyEmail)
	r.Post("/resend-verification", h.ResendVerification)

	// Protected routes (require auth)
	r.Group(func(r chi.Router) {
		r.Use(h.PortalAuthMiddleware)

		// Dashboard
		r.Get("/dashboard", h.PortalDashboard)

		// API Keys
		r.Get("/api-keys", h.APIKeysPage)
		r.Post("/api-keys", h.CreateAPIKey)
		r.Post("/api-keys/{id}/revoke", h.RevokeAPIKey)

		// Usage
		r.Get("/usage", h.PortalUsagePage)

		// Billing
		r.Get("/billing", h.BillingPage)

		// Plans (upgrade/downgrade)
		r.Get("/plans", h.PlansPage)
		r.Post("/plans/change", h.ChangePlan)

		// Subscription management
		r.Get("/subscription/checkout-success", h.CheckoutSuccess)
		r.Get("/subscription/checkout-cancel", h.CheckoutCancel)
		r.Get("/subscription/manage", h.ManageSubscription)
		r.Get("/subscription/cancel", h.CancelSubscriptionPage)
		r.Post("/subscription/cancel", h.CancelSubscription)

		// Account settings
		r.Get("/settings", h.AccountSettingsPage)
		r.Post("/settings", h.UpdateAccountSettings)
		r.Post("/settings/password", h.ChangePassword)
		r.Post("/settings/close-account", h.CloseAccount)

		// Webhooks
		r.Get("/webhooks", h.PortalWebhooksPage)
		r.Get("/webhooks/new", h.PortalWebhookNewPage)
		r.Post("/webhooks", h.PortalWebhookCreate)
		r.Get("/webhooks/{id}", h.PortalWebhookEditPage)
		r.Post("/webhooks/{id}", h.PortalWebhookUpdate)
		r.Delete("/webhooks/{id}", h.PortalWebhookDelete)

		// Logout
		r.Post("/logout", h.PortalLogout)
	})

	return r
}

// PortalAuthMiddleware validates JWT token for portal users.
func (h *PortalHandler) PortalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("portal_token")
		if err != nil {
			http.Redirect(w, r, "/portal/login", http.StatusFound)
			return
		}

		claims, err := h.tokens.ValidateToken(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/portal/login", http.StatusFound)
			return
		}

		// Verify user still exists and is active
		user, err := h.users.Get(r.Context(), claims.UserID)
		if err != nil || user.Status != "active" {
			h.clearPortalCookie(w)
			http.Redirect(w, r, "/portal/login", http.StatusFound)
			return
		}

		ctx := withPortalUser(r.Context(), &PortalUser{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Portal page data
type PortalPageData struct {
	Title   string
	User    *PortalUser
	Flash   *FlashMessage
	AppName string
	Data    interface{}
}

// PortalUser represents a logged-in portal user.
type PortalUser struct {
	ID    string
	Email string
	Name  string
}

// Portal context key
type portalCtxKey string

const portalUserKey portalCtxKey = "portalUser"

func withPortalUser(ctx context.Context, user *PortalUser) context.Context {
	return context.WithValue(ctx, portalUserKey, user)
}

func getPortalUser(ctx context.Context) *PortalUser {
	user, _ := ctx.Value(portalUserKey).(*PortalUser)
	return user
}

func (h *PortalHandler) newPortalPageData(ctx context.Context, title string) PortalPageData {
	return PortalPageData{
		Title:   title,
		User:    getPortalUser(ctx),
		AppName: h.appName,
	}
}

// setPortalCookie sets the JWT cookie for portal session.
func (h *PortalHandler) setPortalCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "portal_token",
		Value:    token,
		Path:     "/portal",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

func (h *PortalHandler) clearPortalCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "portal_token",
		Value:    "",
		Path:     "/portal",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	})
}

// -----------------------------------------------------------------------------
// Landing Page
// -----------------------------------------------------------------------------

// LandingPage shows a public landing page or redirects to dashboard if logged in
func (h *PortalHandler) LandingPage(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in via JWT cookie
	cookie, err := r.Cookie("portal_token")
	if err == nil && cookie.Value != "" {
		// Validate JWT token
		claims, err := h.tokens.ValidateToken(cookie.Value)
		if err == nil {
			// Verify user still exists and is active
			user, err := h.users.Get(r.Context(), claims.UserID)
			if err == nil && user.Status == "active" {
				// User is logged in, redirect to dashboard
				http.Redirect(w, r, "/portal/dashboard", http.StatusFound)
				return
			}
		}
	}

	// Show landing page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderLandingPage()))
}

// -----------------------------------------------------------------------------
// Signup
// -----------------------------------------------------------------------------

func (h *PortalHandler) SignupPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Find default plan to show user what they're signing up for
	var defaultPlan *ports.Plan
	if plans, err := h.plans.List(ctx); err == nil {
		for _, p := range plans {
			if p.IsDefault && p.Enabled {
				defaultPlan = &p
				break
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderSignupPageWithPlan("", "", defaultPlan, h.getLabels(r.Context()), nil)))
}

func (h *PortalHandler) SignupSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	req := domainAuth.SignupRequest{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
		Name:     r.FormValue("name"),
	}

	// Helper to get default plan for error pages
	getDefaultPlan := func() *ports.Plan {
		if plans, err := h.plans.List(ctx); err == nil {
			for _, p := range plans {
				if p.IsDefault && p.Enabled {
					return &p
				}
			}
		}
		return nil
	}

	// Validate
	result := domainAuth.ValidateSignup(req)
	if !result.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderSignupPageWithPlan(req.Name, req.Email, getDefaultPlan(), h.getLabels(ctx), result.Errors)))
		return
	}

	// Check if email already exists
	if _, err := h.users.GetByEmail(ctx, req.Email); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(h.renderSignupPageWithPlan(req.Name, req.Email, getDefaultPlan(), h.getLabels(ctx), map[string]string{"email": "Email already registered"})))
		return
	}

	// Hash password
	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to hash password")
		h.renderError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// Check if email verification is required
	requireVerification := false
	if h.settings != nil {
		allSettings, err := h.settings.GetAll(ctx)
		if err == nil {
			requireVerification = allSettings.GetBool(settings.KeyAuthRequireEmailVerification)
		}
	}

	// Create user
	userID := h.idGen.New()
	userStatus := "active" // Default to active when verification not required
	if requireVerification {
		userStatus = "pending" // Not active until email verified
	}

	// Find default plan for new users
	defaultPlanID := "free" // fallback if no default plan configured
	if plans, err := h.plans.List(ctx); err == nil {
		for _, p := range plans {
			if p.IsDefault {
				defaultPlanID = p.ID
				break
			}
		}
	}

	user := ports.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		PlanID:       defaultPlanID,
		Status:       userStatus,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.users.Create(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to create user")
		h.renderError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// Only send verification email if required
	if requireVerification {
		// Generate verification token
		tokenResult := domainAuth.GenerateToken(userID, req.Email, domainAuth.TokenTypeEmailVerification, 24*time.Hour)
		// Hash the raw token for storage
		tokenWithHash := tokenResult.Token.WithHash(domainAuth.HashToken(tokenResult.RawToken))

		// Store token
		if err := h.authTokens.Create(ctx, tokenWithHash); err != nil {
			h.logger.Error().Err(err).Msg("failed to store verification token")
		} else {
			// Send verification email
			if err := h.emailSender.SendVerification(ctx, req.Email, req.Name, tokenResult.RawToken); err != nil {
				h.logger.Error().Err(err).Str("email", req.Email).Msg("failed to send verification email")
			}
		}
		// Redirect to login with verification message, pre-fill email
		http.Redirect(w, r, "/portal/login?signup=success&email="+url.QueryEscape(req.Email), http.StatusFound)
	} else {
		// Auto-login: generate JWT and set cookie, then redirect to dashboard
		token, _, err := h.tokens.GenerateToken(userID, req.Email, "user")
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to generate token after signup")
			// Fall back to login redirect
			http.Redirect(w, r, "/portal/login?signup=ready&email="+url.QueryEscape(req.Email), http.StatusFound)
			return
		}

		h.setPortalCookie(w, token)
		h.logger.Info().Str("user_id", userID).Str("email", req.Email).Msg("user signed up and auto-logged in")

		// Redirect to dashboard
		http.Redirect(w, r, "/portal/dashboard", http.StatusFound)
	}
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

func (h *PortalHandler) PortalLoginPage(w http.ResponseWriter, r *http.Request) {
	message := ""
	messageType := ""
	email := r.URL.Query().Get("email") // Pre-fill email from signup redirect

	switch r.URL.Query().Get("signup") {
	case "success":
		message = "Account created! Please check your email to verify your account."
		messageType = "success"
	case "ready":
		message = "Account created! You can now log in."
		messageType = "success"
	}

	if r.URL.Query().Get("verified") == "true" {
		message = "Email verified! You can now log in."
		messageType = "success"
	} else if r.URL.Query().Get("reset") == "success" {
		message = "Password reset successful! You can now log in."
		messageType = "success"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderLoginPage(email, message, messageType, nil)))
}

func (h *PortalHandler) PortalLoginSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	req := domainAuth.LoginRequest{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	// Validate input
	result := domainAuth.ValidateLogin(req)
	if !result.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderLoginPage(req.Email, "", "", result.Errors)))
		return
	}

	// Get user
	user, err := h.users.GetByEmail(ctx, req.Email)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(h.renderLoginPage(req.Email, "Invalid email or password", "error", nil)))
		return
	}

	// Check password
	if !h.hasher.Compare(user.PasswordHash, req.Password) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(h.renderLoginPage(req.Email, "Invalid email or password", "error", nil)))
		return
	}

	// Check status
	if user.Status == "pending" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(h.renderLoginPage(req.Email, "Please verify your email before logging in", "warning", nil)))
		return
	}
	if user.Status != "active" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(h.renderLoginPage(req.Email, "Your account is not active", "error", nil)))
		return
	}

	// Generate JWT
	token, _, err := h.tokens.GenerateToken(user.ID, user.Email, "user")
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate token")
		h.renderError(w, http.StatusInternalServerError, "Failed to log in")
		return
	}

	h.setPortalCookie(w, token)
	http.Redirect(w, r, "/portal/dashboard", http.StatusFound)
}

func (h *PortalHandler) PortalLogout(w http.ResponseWriter, r *http.Request) {
	h.clearPortalCookie(w)
	http.Redirect(w, r, "/portal/login", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Email Verification
// -----------------------------------------------------------------------------

func (h *PortalHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawToken := r.URL.Query().Get("token")

	if rawToken == "" {
		h.renderError(w, http.StatusBadRequest, "Missing verification token")
		return
	}

	// Get token by hash
	hash := domainAuth.HashToken(rawToken)
	token, err := h.authTokens.GetByHash(ctx, hash)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid or expired verification link")
		return
	}

	// Check token type
	if token.Type != domainAuth.TokenTypeEmailVerification {
		h.renderError(w, http.StatusBadRequest, "Invalid token type")
		return
	}

	// Check expiry
	if token.ExpiresAt.Before(time.Now().UTC()) {
		h.renderError(w, http.StatusBadRequest, "Verification link has expired. Please request a new one.")
		return
	}

	// Check if already used
	if token.UsedAt != nil {
		http.Redirect(w, r, "/portal/login?verified=true", http.StatusFound)
		return
	}

	// Get user and update status
	user, err := h.users.Get(ctx, token.UserID)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "User not found")
		return
	}

	user.Status = "active"
	user.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update user status")
		h.renderError(w, http.StatusInternalServerError, "Failed to verify email")
		return
	}

	// Mark token as used
	if err := h.authTokens.MarkUsed(ctx, token.ID, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark token as used")
	}

	// Send welcome email
	if err := h.emailSender.SendWelcome(ctx, user.Email, user.Name); err != nil {
		h.logger.Error().Err(err).Msg("failed to send welcome email")
	}

	http.Redirect(w, r, "/portal/login?verified=true", http.StatusFound)
}

func (h *PortalHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	email := r.FormValue("email")
	if email == "" {
		h.renderError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Get user
	user, err := h.users.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		http.Redirect(w, r, "/portal/login?resent=true", http.StatusFound)
		return
	}

	if user.Status != "pending" {
		// Already verified
		http.Redirect(w, r, "/portal/login", http.StatusFound)
		return
	}

	// Generate new verification token
	tokenResult := domainAuth.GenerateToken(user.ID, user.Email, domainAuth.TokenTypeEmailVerification, 24*time.Hour)
	tokenWithHash := tokenResult.Token.WithHash(domainAuth.HashToken(tokenResult.RawToken))

	if err := h.authTokens.Create(ctx, tokenWithHash); err != nil {
		h.logger.Error().Err(err).Msg("failed to store verification token")
		h.renderError(w, http.StatusInternalServerError, "Failed to send verification email")
		return
	}

	if err := h.emailSender.SendVerification(ctx, user.Email, user.Name, tokenResult.RawToken); err != nil {
		h.logger.Error().Err(err).Msg("failed to send verification email")
	}

	http.Redirect(w, r, "/portal/login?resent=true", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Password Reset
// -----------------------------------------------------------------------------

func (h *PortalHandler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderForgotPasswordPage("", "", "")))
}

func (h *PortalHandler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	email := r.FormValue("email")
	req := domainAuth.PasswordResetRequest{Email: email}
	valid, errMsg := domainAuth.ValidatePasswordResetRequest(req)
	if !valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderForgotPasswordPage(email, errMsg, "error")))
		return
	}

	// Always show success to prevent email enumeration
	successMsg := "If an account exists with this email, you will receive a password reset link."

	// Get user (if exists)
	user, err := h.users.GetByEmail(ctx, email)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(h.renderForgotPasswordPage("", successMsg, "success")))
		return
	}

	// Generate reset token
	tokenResult := domainAuth.GenerateToken(user.ID, user.Email, domainAuth.TokenTypePasswordReset, 1*time.Hour)
	tokenWithHash := tokenResult.Token.WithHash(domainAuth.HashToken(tokenResult.RawToken))

	if err := h.authTokens.Create(ctx, tokenWithHash); err != nil {
		h.logger.Error().Err(err).Msg("failed to store reset token")
	} else if err := h.emailSender.SendPasswordReset(ctx, user.Email, user.Name, tokenResult.RawToken); err != nil {
		h.logger.Error().Err(err).Msg("failed to send reset email")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderForgotPasswordPage("", successMsg, "success")))
}

func (h *PortalHandler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderError(w, http.StatusBadRequest, "Missing reset token")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderResetPasswordPage(token, nil)))
}

func (h *PortalHandler) ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	rawToken := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate passwords match
	if password != confirmPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderResetPasswordPage(rawToken, map[string]string{"confirm_password": "Passwords do not match"})))
		return
	}

	// Validate password strength
	req := domainAuth.PasswordResetConfirm{Token: rawToken, NewPassword: password}
	result := domainAuth.ValidatePasswordResetConfirm(req)
	if !result.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderResetPasswordPage(rawToken, result.Errors)))
		return
	}

	// Verify token
	hash := domainAuth.HashToken(rawToken)
	token, err := h.authTokens.GetByHash(ctx, hash)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid or expired reset link")
		return
	}

	if token.Type != domainAuth.TokenTypePasswordReset {
		h.renderError(w, http.StatusBadRequest, "Invalid token type")
		return
	}

	if token.ExpiresAt.Before(time.Now().UTC()) {
		h.renderError(w, http.StatusBadRequest, "Reset link has expired. Please request a new one.")
		return
	}

	if token.UsedAt != nil {
		h.renderError(w, http.StatusBadRequest, "This reset link has already been used")
		return
	}

	// Get user and update password
	user, err := h.users.Get(ctx, token.UserID)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "User not found")
		return
	}

	passwordHash, err := h.hasher.Hash(password)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to hash password")
		h.renderError(w, http.StatusInternalServerError, "Failed to reset password")
		return
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update password")
		h.renderError(w, http.StatusInternalServerError, "Failed to reset password")
		return
	}

	// Mark token as used
	if err := h.authTokens.MarkUsed(ctx, token.ID, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark token as used")
	}

	// Invalidate all sessions
	if err := h.sessions.DeleteByUser(ctx, user.ID); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete sessions")
	}

	http.Redirect(w, r, "/portal/login?reset=success", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Dashboard
// -----------------------------------------------------------------------------

func (h *PortalHandler) PortalDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	// Get user's API keys
	keys, err := h.keys.ListByUser(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get keys")
	}

	// Get usage summary
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	summary, err := h.usage.GetSummary(ctx, user.ID, start, now)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get usage")
	}

	// Get user's plan info
	var planName string
	var planID string
	var requestsPerMonth int64
	var rateLimitPerMinute int
	if h.plans != nil {
		dbUser, err := h.users.Get(ctx, user.ID)
		if err == nil && dbUser.PlanID != "" {
			planID = dbUser.PlanID
			plan, err := h.plans.Get(ctx, dbUser.PlanID)
			if err == nil {
				planName = plan.Name
				requestsPerMonth = plan.RequestsPerMonth
				rateLimitPerMinute = plan.RateLimitPerMinute
			}
		}
	}

	// Get user's entitlements based on their plan
	var userEntitlements []entitlement.UserEntitlement
	if h.entitlements != nil && h.planEntitlements != nil && planID != "" {
		ents, err := h.entitlements.ListEnabled(ctx)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to get entitlements")
		} else {
			planEnts, err := h.planEntitlements.ListByPlan(ctx, planID)
			if err != nil {
				h.logger.Error().Err(err).Msg("failed to get plan entitlements")
			} else {
				userEntitlements = entitlement.ResolveForPlan(planID, ents, planEnts)
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderDashboardPage(user, len(keys), summary.RequestCount, planName, requestsPerMonth, rateLimitPerMinute, userEntitlements, h.getLabels(ctx))))
}

// -----------------------------------------------------------------------------
// API Keys
// -----------------------------------------------------------------------------

func (h *PortalHandler) APIKeysPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	keys, err := h.keys.ListByUser(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get keys")
		keys = nil
	}

	revokedMsg := r.URL.Query().Get("revoked") == "true"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAPIKeysPage(user, keys, revokedMsg)))
}

func (h *PortalHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	keyName := r.FormValue("name")

	// Generate API key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(user.ID)
	if keyName != "" {
		keyData.Name = keyName
	}

	// Store the key
	if err := h.keys.Create(ctx, keyData); err != nil {
		h.logger.Error().Err(err).Msg("failed to create API key")
		h.renderError(w, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	// Show the key to the user (only shown once)
	h.renderKeyCreatedPage(w, r, user, rawKey, keyName)
}

func (h *PortalHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)
	keyID := chi.URLParam(r, "id")

	if keyID == "" {
		http.Error(w, "Key ID required", http.StatusBadRequest)
		return
	}

	// Verify the key belongs to this user (security check)
	keys, err := h.keys.ListByUser(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list user keys")
		http.Error(w, "Failed to verify key ownership", http.StatusInternalServerError)
		return
	}

	keyBelongsToUser := false
	for _, k := range keys {
		if k.ID == keyID {
			keyBelongsToUser = true
			break
		}
	}

	if !keyBelongsToUser {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	// Revoke the key
	if err := h.keys.Revoke(ctx, keyID, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Str("key_id", keyID).Msg("failed to revoke key")
		http.Error(w, "Failed to revoke key", http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("key_id", keyID).Str("user_id", user.ID).Msg("API key revoked")
	http.Redirect(w, r, "/portal/api-keys?revoked=true", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Usage
// -----------------------------------------------------------------------------

func (h *PortalHandler) PortalUsagePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	summary, err := h.usage.GetSummary(ctx, user.ID, start, now)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get usage")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderUsagePage(user, summary, h.getLabels(ctx))))
}

// -----------------------------------------------------------------------------
// Account Settings
// -----------------------------------------------------------------------------

func (h *PortalHandler) AccountSettingsPage(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	success := ""
	if r.URL.Query().Get("password") == "changed" {
		success = "Password changed successfully"
	} else if r.URL.Query().Get("profile") == "updated" {
		success = "Profile updated successfully"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAccountSettingsPage(user, nil, success)))
}

func (h *PortalHandler) UpdateAccountSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	portalUser := getPortalUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	// Validate name
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Name is required"
	} else if len(name) < 2 {
		errors["name"] = "Name must be at least 2 characters"
	} else if len(name) > 100 {
		errors["name"] = "Name must be less than 100 characters"
	}

	if len(errors) > 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, errors, "")))
		return
	}

	// Get user and update name
	user, err := h.users.Get(ctx, portalUser.ID)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	user.Name = name
	user.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update profile")
		h.renderError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	http.Redirect(w, r, "/portal/settings?profile=updated", http.StatusSeeOther)
}

func (h *PortalHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	portalUser := getPortalUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate
	req := domainAuth.ChangePasswordRequest{
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	}
	result := domainAuth.ValidateChangePassword(req)
	if !result.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, result.Errors, "")))
		return
	}

	if newPassword != confirmPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, map[string]string{"confirm_password": "Passwords do not match"}, "")))
		return
	}

	// Get user and verify current password
	user, err := h.users.Get(ctx, portalUser.ID)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	if !h.hasher.Compare(user.PasswordHash, currentPassword) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, map[string]string{"current_password": "Current password is incorrect"}, "")))
		return
	}

	// Update password
	passwordHash, err := h.hasher.Hash(newPassword)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to hash password")
		h.renderError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update password")
		h.renderError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	http.Redirect(w, r, "/portal/settings?password=changed", http.StatusFound)
}

func (h *PortalHandler) CloseAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Require password confirmation
	password := r.FormValue("password")
	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	if !h.hasher.Compare(dbUser.PasswordHash, password) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(h.renderAccountSettingsPage(user, map[string]string{"password": "Incorrect password"}, "")))
		return
	}

	// Soft delete - mark as cancelled
	dbUser.Status = "cancelled"
	dbUser.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, dbUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to close account")
		h.renderError(w, http.StatusInternalServerError, "Failed to close account")
		return
	}

	// Delete sessions
	if err := h.sessions.DeleteByUser(ctx, user.ID); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete sessions")
	}

	h.clearPortalCookie(w)
	http.Redirect(w, r, "/portal/login?closed=true", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Billing
// -----------------------------------------------------------------------------

func (h *PortalHandler) BillingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	// Get user's subscription
	var subscription *billing.Subscription
	var currentPlan *ports.Plan
	if h.subscriptions != nil {
		sub, err := h.subscriptions.GetByUser(ctx, user.ID)
		if err == nil {
			subscription = &sub
			// Get the plan for this subscription
			if h.plans != nil {
				plan, err := h.plans.Get(ctx, sub.PlanID)
				if err == nil {
					currentPlan = &plan
				}
			}
		}
	}

	// If no subscription found, get user's plan directly
	if currentPlan == nil && h.plans != nil {
		dbUser, err := h.users.Get(ctx, user.ID)
		if err == nil && dbUser.PlanID != "" {
			plan, err := h.plans.Get(ctx, dbUser.PlanID)
			if err == nil {
				currentPlan = &plan
			}
		}
	}

	// Get user's invoices
	var invoices []billing.Invoice
	if h.invoices != nil {
		inv, err := h.invoices.ListByUser(ctx, user.ID, 10)
		if err == nil {
			invoices = inv
		}
	}

	// Check for confirmation messages
	var successMsg, errorMsg string
	cancelled := r.URL.Query().Get("cancelled")
	if cancelled == "now" {
		successMsg = "Your subscription has been cancelled. You've been downgraded to the free plan."
	} else if cancelled == "end_of_period" {
		successMsg = "Your subscription has been set to cancel at the end of the current billing period."
	}

	if errType := r.URL.Query().Get("error"); errType != "" {
		switch errType {
		case "payment":
			errorMsg = "There was an error processing your request with the payment provider."
		case "internal":
			errorMsg = "An internal error occurred. Please try again."
		case "no_subscription":
			errorMsg = "No active subscription found."
		default:
			errorMsg = "An error occurred. Please try again."
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderBillingPage(user, subscription, currentPlan, invoices, successMsg, errorMsg)))
}

// -----------------------------------------------------------------------------
// Plans (Upgrade/Downgrade)
// -----------------------------------------------------------------------------

func (h *PortalHandler) PlansPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	// Get current user details
	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user")
		h.renderError(w, http.StatusInternalServerError, "Failed to load account")
		return
	}

	// Get all enabled plans
	allPlans, err := h.plans.List(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list plans")
		h.renderError(w, http.StatusInternalServerError, "Failed to load plans")
		return
	}

	// Filter to enabled plans only
	var plans []ports.Plan
	var currentPlan *ports.Plan
	for _, p := range allPlans {
		if p.Enabled {
			planCopy := p
			plans = append(plans, planCopy)
			if p.ID == dbUser.PlanID {
				currentPlan = &planCopy
			}
		}
	}

	// Check for success/error messages
	success := ""
	errorMsg := ""
	if r.URL.Query().Get("changed") == "true" {
		success = "Your plan has been changed successfully."
	}
	if r.URL.Query().Get("error") == "payment" {
		errorMsg = "Payment is required for this plan. Please contact support."
	}
	if r.URL.Query().Get("error") == "cancelled" {
		errorMsg = "Payment was cancelled. You can try again when ready."
	}
	if r.URL.Query().Get("error") == "no_subscription" {
		errorMsg = "No active subscription found. Please upgrade to a paid plan first."
	}

	// Check if user has a Stripe subscription
	hasStripeSubscription := dbUser.StripeID != ""

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderPlansPage(user, plans, currentPlan, success, errorMsg, hasStripeSubscription, h.getLabels(ctx))))
}

func (h *PortalHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/portal/plans?error=invalid", http.StatusFound)
		return
	}

	newPlanID := r.FormValue("plan_id")
	if newPlanID == "" {
		http.Redirect(w, r, "/portal/plans?error=invalid", http.StatusFound)
		return
	}

	// Get the new plan
	newPlan, err := h.plans.Get(ctx, newPlanID)
	if err != nil || !newPlan.Enabled {
		http.Redirect(w, r, "/portal/plans?error=invalid", http.StatusFound)
		return
	}

	// Get current user
	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user for plan change")
		http.Redirect(w, r, "/portal/plans?error=internal", http.StatusFound)
		return
	}

	// If the new plan has a price > 0, redirect to payment checkout
	if newPlan.PriceMonthly > 0 {
		// Check if payment provider is configured (NoopProvider returns "none")
		if h.payment == nil || h.payment.Name() == "none" {
			h.logger.Error().Msg("no payment provider configured for paid plan change")
			http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
			return
		}

		// Get or create Stripe customer ID for user
		customerID := dbUser.StripeID
		if customerID == "" {
			// Create new customer in Stripe
			var err error
			customerID, err = h.payment.CreateCustomer(ctx, dbUser.Email, dbUser.Name, dbUser.ID)
			if err != nil {
				h.logger.Error().Err(err).Msg("failed to create Stripe customer")
				http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
				return
			}
			// Store customer ID
			dbUser.StripeID = customerID
			dbUser.UpdatedAt = time.Now().UTC()
			if err := h.users.Update(ctx, dbUser); err != nil {
				h.logger.Error().Err(err).Msg("failed to store Stripe customer ID")
			}
		}

		// Get the price ID for this plan (not required for dummy/demo mode)
		priceID := newPlan.StripePriceID
		if priceID == "" && h.payment.Name() != "dummy" {
			h.logger.Error().Str("plan_id", newPlanID).Msg("plan has no price ID configured")
			http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
			return
		}

		// Build success/cancel URLs
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL := scheme + "://" + r.Host
		successURL := baseURL + "/portal/subscription/checkout-success?plan=" + url.QueryEscape(newPlanID)
		cancelURL := baseURL + "/portal/subscription/checkout-cancel"

		// Create checkout session with trial period if configured
		checkoutURL, err := h.payment.CreateCheckoutSession(ctx, customerID, priceID, successURL, cancelURL, newPlan.TrialDays)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to create checkout session")
			http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
			return
		}

		h.logger.Info().
			Str("user_id", user.ID).
			Str("new_plan", newPlanID).
			Int64("price", newPlan.PriceMonthly).
			Msg("redirecting to payment checkout")

		http.Redirect(w, r, checkoutURL, http.StatusFound)
		return
	}

	// Free plan change - update directly
	dbUser.PlanID = newPlanID
	dbUser.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, dbUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to update user plan")
		http.Redirect(w, r, "/portal/plans?error=internal", http.StatusFound)
		return
	}

	h.logger.Info().Str("user_id", user.ID).Str("old_plan", dbUser.PlanID).Str("new_plan", newPlanID).Msg("user changed to free plan")
	http.Redirect(w, r, "/portal/plans?changed=true", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Subscription Management
// -----------------------------------------------------------------------------

// CheckoutSuccess handles the return from a successful Stripe checkout
func (h *PortalHandler) CheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	planID := r.URL.Query().Get("plan")
	if planID == "" {
		http.Redirect(w, r, "/portal/plans?error=invalid", http.StatusFound)
		return
	}

	// Verify plan exists and is enabled
	plan, err := h.plans.Get(ctx, planID)
	if err != nil || !plan.Enabled {
		h.logger.Error().Err(err).Str("plan_id", planID).Msg("invalid plan in checkout success")
		http.Redirect(w, r, "/portal/plans?error=invalid", http.StatusFound)
		return
	}

	// Update user's plan
	// Note: In production, this should be handled by Stripe webhooks
	// This is a simplified flow for immediate feedback
	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user in checkout success")
		http.Redirect(w, r, "/portal/plans?error=internal", http.StatusFound)
		return
	}

	oldPlan := dbUser.PlanID
	dbUser.PlanID = planID
	dbUser.UpdatedAt = time.Now().UTC()
	if err := h.users.Update(ctx, dbUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to update user plan after checkout")
		http.Redirect(w, r, "/portal/plans?error=internal", http.StatusFound)
		return
	}

	h.logger.Info().
		Str("user_id", user.ID).
		Str("old_plan", oldPlan).
		Str("new_plan", planID).
		Msg("user upgraded plan after checkout")

	http.Redirect(w, r, "/portal/plans?changed=true", http.StatusFound)
}

// CheckoutCancel handles the return from a cancelled Stripe checkout
func (h *PortalHandler) CheckoutCancel(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/portal/plans?error=cancelled", http.StatusFound)
}

// ManageSubscription redirects to Stripe Customer Portal for subscription management
func (h *PortalHandler) ManageSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	if h.payment == nil {
		http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
		return
	}

	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user for subscription management")
		http.Redirect(w, r, "/portal/plans?error=internal", http.StatusFound)
		return
	}

	if dbUser.StripeID == "" {
		http.Redirect(w, r, "/portal/plans?error=no_subscription", http.StatusFound)
		return
	}

	// Build return URL
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	returnURL := scheme + "://" + r.Host + "/portal/plans"

	// Create Stripe Customer Portal session
	portalURL, err := h.payment.CreatePortalSession(ctx, dbUser.StripeID, returnURL)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create portal session")
		http.Redirect(w, r, "/portal/plans?error=payment", http.StatusFound)
		return
	}

	http.Redirect(w, r, portalURL, http.StatusFound)
}

// CancelSubscriptionPage shows the cancel confirmation page
func (h *PortalHandler) CancelSubscriptionPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	// Get user's subscription
	subscription, err := h.subscriptions.GetByUser(ctx, user.ID)
	if err != nil {
		// No active subscription - redirect to plans
		http.Redirect(w, r, "/portal/plans", http.StatusFound)
		return
	}

	// Get current plan
	plan, err := h.plans.Get(ctx, subscription.PlanID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get plan for cancel page")
		http.Redirect(w, r, "/portal/billing?error=internal", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderCancelSubscriptionPage(user, &subscription, &plan)))
}

// CancelSubscription handles subscription cancellation
func (h *PortalHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getPortalUser(ctx)

	// Check if subscription store is configured
	if h.subscriptions == nil {
		http.Redirect(w, r, "/portal/billing?error=not_configured", http.StatusFound)
		return
	}

	// Parse form for cancel mode
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/portal/billing?error=invalid", http.StatusFound)
		return
	}
	cancelImmediately := r.FormValue("cancel_mode") == "immediately"

	// Get user's subscription
	subscription, err := h.subscriptions.GetByUser(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get subscription for cancellation")
		http.Redirect(w, r, "/portal/billing?error=no_subscription", http.StatusFound)
		return
	}

	// Get user for Stripe info
	dbUser, err := h.users.Get(ctx, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user for cancellation")
		http.Redirect(w, r, "/portal/billing?error=internal", http.StatusFound)
		return
	}

	now := time.Now().UTC()

	// If there's a payment provider and subscription has provider ID, cancel via provider
	if h.payment != nil && subscription.ProviderID != "" {
		if err := h.payment.CancelSubscription(ctx, subscription.ProviderID, cancelImmediately); err != nil {
			h.logger.Error().Err(err).Msg("failed to cancel subscription via payment provider")
			http.Redirect(w, r, "/portal/billing?error=payment", http.StatusFound)
			return
		}
	}

	// Update local subscription record
	if cancelImmediately {
		subscription.Status = billing.SubscriptionStatusCancelled
		subscription.CancelledAt = &now
	} else {
		subscription.CancelAtPeriodEnd = true
		subscription.CancelledAt = &now
	}

	if err := h.subscriptions.Update(ctx, subscription); err != nil {
		h.logger.Error().Err(err).Msg("failed to update subscription status")
		// Don't fail - provider cancellation already happened
	}

	// If cancelled immediately, downgrade user to free plan
	if cancelImmediately {
		dbUser.PlanID = "free"
		dbUser.UpdatedAt = now
		if err := h.users.Update(ctx, dbUser); err != nil {
			h.logger.Error().Err(err).Msg("failed to downgrade user to free plan")
		}
	}

	// Redirect to billing with confirmation
	if cancelImmediately {
		http.Redirect(w, r, "/portal/billing?cancelled=now", http.StatusFound)
	} else {
		http.Redirect(w, r, "/portal/billing?cancelled=end_of_period", http.StatusFound)
	}
}

// -----------------------------------------------------------------------------
// Error Handling
// -----------------------------------------------------------------------------

func (h *PortalHandler) renderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(h.renderErrorPage(message)))
}
