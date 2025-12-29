// Package web provides the user self-service portal.
package web

import (
	"context"
	"net/http"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	domainAuth "github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// PortalHandler provides the user portal endpoints.
type PortalHandler struct {
	tokens       *auth.TokenService
	users        ports.UserStore
	keys         ports.KeyStore
	usage        ports.UsageStore
	sessions     ports.SessionStore
	authTokens   ports.TokenStore
	emailSender  ports.EmailSender
	logger       zerolog.Logger
	hasher       ports.Hasher
	idGen        ports.IDGenerator

	// Portal-specific settings
	baseURL      string
	appName      string
}

// PortalDeps contains dependencies for the portal handler.
type PortalDeps struct {
	Users       ports.UserStore
	Keys        ports.KeyStore
	Usage       ports.UsageStore
	Sessions    ports.SessionStore
	AuthTokens  ports.TokenStore
	EmailSender ports.EmailSender
	Logger      zerolog.Logger
	Hasher      ports.Hasher
	IDGen       ports.IDGenerator
	JWTSecret   string
	BaseURL     string
	AppName     string
}

// NewPortalHandler creates a new user portal handler.
func NewPortalHandler(deps PortalDeps) (*PortalHandler, error) {
	appName := deps.AppName
	if appName == "" {
		appName = "APIGate"
	}

	return &PortalHandler{
		tokens:      auth.NewTokenService(deps.JWTSecret, 7*24*time.Hour), // 7 day sessions
		users:       deps.Users,
		keys:        deps.Keys,
		usage:       deps.Usage,
		sessions:    deps.Sessions,
		authTokens:  deps.AuthTokens,
		emailSender: deps.EmailSender,
		logger:      deps.Logger,
		hasher:      deps.Hasher,
		idGen:       deps.IDGen,
		baseURL:     deps.BaseURL,
		appName:     appName,
	}, nil
}

// Router returns the portal router.
func (h *PortalHandler) Router() chi.Router {
	r := chi.NewRouter()

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
		r.Get("/", h.PortalDashboard)
		r.Get("/dashboard", h.PortalDashboard)

		// API Keys
		r.Get("/api-keys", h.APIKeysPage)
		r.Post("/api-keys", h.CreateAPIKey)
		r.Delete("/api-keys/{id}", h.RevokeAPIKey)

		// Usage
		r.Get("/usage", h.PortalUsagePage)

		// Account settings
		r.Get("/settings", h.AccountSettingsPage)
		r.Post("/settings", h.UpdateAccountSettings)
		r.Post("/settings/password", h.ChangePassword)
		r.Post("/settings/close-account", h.CloseAccount)

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
// Signup
// -----------------------------------------------------------------------------

func (h *PortalHandler) SignupPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderSignupPage("", nil)))
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

	// Validate
	result := domainAuth.ValidateSignup(req)
	if !result.Valid {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderSignupPage(req.Email, result.Errors)))
		return
	}

	// Check if email already exists
	if _, err := h.users.GetByEmail(ctx, req.Email); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(h.renderSignupPage(req.Email, map[string]string{"email": "Email already registered"})))
		return
	}

	// Hash password
	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to hash password")
		h.renderError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	// Create user
	userID := h.idGen.New()
	user := ports.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		PlanID:       "free",
		Status:       "pending", // Not active until email verified
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.users.Create(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("failed to create user")
		h.renderError(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

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

	// Redirect to success page
	http.Redirect(w, r, "/portal/login?signup=success", http.StatusFound)
}

// -----------------------------------------------------------------------------
// Login
// -----------------------------------------------------------------------------

func (h *PortalHandler) PortalLoginPage(w http.ResponseWriter, r *http.Request) {
	message := ""
	messageType := ""

	if r.URL.Query().Get("signup") == "success" {
		message = "Account created! Please check your email to verify your account."
		messageType = "success"
	} else if r.URL.Query().Get("verified") == "true" {
		message = "Email verified! You can now log in."
		messageType = "success"
	} else if r.URL.Query().Get("reset") == "success" {
		message = "Password reset successful! You can now log in."
		messageType = "success"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderLoginPage("", message, messageType, nil)))
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderDashboardPage(user, len(keys), summary.RequestCount)))
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAPIKeysPage(user, keys)))
}

func (h *PortalHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Placeholder - implement key creation
	http.Redirect(w, r, "/portal/api-keys", http.StatusFound)
}

func (h *PortalHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// Placeholder - implement key revocation
	http.Redirect(w, r, "/portal/api-keys", http.StatusFound)
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
	w.Write([]byte(h.renderUsagePage(user, summary)))
}

// -----------------------------------------------------------------------------
// Account Settings
// -----------------------------------------------------------------------------

func (h *PortalHandler) AccountSettingsPage(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAccountSettingsPage(user, nil)))
}

func (h *PortalHandler) UpdateAccountSettings(w http.ResponseWriter, r *http.Request) {
	// Placeholder - implement account update
	http.Redirect(w, r, "/portal/settings", http.StatusFound)
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
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, result.Errors)))
		return
	}

	if newPassword != confirmPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, map[string]string{"confirm_password": "Passwords do not match"})))
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
		w.Write([]byte(h.renderAccountSettingsPage(portalUser, map[string]string{"current_password": "Current password is incorrect"})))
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
		w.Write([]byte(h.renderAccountSettingsPage(user, map[string]string{"password": "Incorrect password"})))
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
// Error Handling
// -----------------------------------------------------------------------------

func (h *PortalHandler) renderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(h.renderErrorPage(message)))
}
