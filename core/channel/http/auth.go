// Package http provides authentication endpoints for the web UI.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/core/runtime"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// contextKey is a private type for context keys in this package.
type contextKey string

const authUserIDKey contextKey = "auth_user_id"

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	runtime *runtime.Runtime
	tokens  *auth.TokenService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(rt *runtime.Runtime) *AuthHandler {
	return &AuthHandler{
		runtime: rt,
	}
}

// SetTokenService injects the shared JWT token service.
func (h *AuthHandler) SetTokenService(ts *auth.TokenService) {
	h.tokens = ts
}

// Routes returns the auth routes.
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.handleRegister)
	r.Post("/login", h.handleLogin)
	r.Post("/logout", h.handleLogout)
	r.Get("/me", h.handleMe)
	r.Get("/setup-required", h.handleSetupRequired)
	r.Post("/setup", h.handleSetup)

	return r
}

// handleSetupRequired checks if first-time setup is needed.
func (h *AuthHandler) handleSetupRequired(w http.ResponseWriter, r *http.Request) {
	// Check if any users exist
	result, err := h.runtime.Execute(r.Context(), "user", "list", runtime.ActionInput{
		Data:    map[string]any{"limit": 1},
		Channel: "http",
	})

	setupRequired := err != nil || len(result.List) == 0

	authWriteJSON(w, map[string]any{
		"setup_required": setupRequired,
	})
}

// handleSetup handles first-time setup - creates admin user.
func (h *AuthHandler) handleSetup(w http.ResponseWriter, r *http.Request) {
	// Check if setup is still needed
	result, err := h.runtime.Execute(r.Context(), "user", "list", runtime.ActionInput{
		Data:    map[string]any{"limit": 1},
		Channel: "http",
	})

	if err == nil && len(result.List) > 0 {
		authWriteError(w, fmt.Errorf("setup already completed"), http.StatusBadRequest)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		authWriteError(w, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	// Validate
	if req.Email == "" {
		authWriteError(w, fmt.Errorf("email is required"), http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		authWriteError(w, fmt.Errorf("password is required"), http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		authWriteError(w, fmt.Errorf("password must be at least 8 characters"), http.StatusBadRequest)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		authWriteError(w, fmt.Errorf("failed to hash password"), http.StatusInternalServerError)
		return
	}

	// Create admin user
	createResult, err := h.runtime.Execute(r.Context(), "user", "create", runtime.ActionInput{
		Data: map[string]any{
			"email":         req.Email,
			"password_hash": string(hash),
			"name":          req.Name,
			"status":        "active",
		},
		Channel: "http",
	})

	if err != nil {
		authWriteError(w, fmt.Errorf("failed to create user: %w", err), http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    createResult.ID,
			"email": req.Email,
			"name":  req.Name,
		},
	}

	// Generate JWT if token service is available
	if h.tokens != nil {
		planID, _ := createResult.Data["plan_id"].(string)
		role, _ := createResult.Data["role"].(string)
		token, expiresAt, err := h.tokens.GenerateToken(createResult.ID, req.Email, role, planID)
		if err == nil {
			resp["token"] = token
			resp["expires_at"] = expiresAt.Format(time.RFC3339)
		}
	}

	authWriteJSON(w, resp)
}

// handleRegister handles user registration.
func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		authWriteError(w, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	// Validate
	if req.Email == "" {
		authWriteError(w, fmt.Errorf("email is required"), http.StatusBadRequest)
		return
	}
	if !strings.Contains(req.Email, "@") {
		authWriteError(w, fmt.Errorf("invalid email format"), http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		authWriteError(w, fmt.Errorf("password is required"), http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		authWriteError(w, fmt.Errorf("password must be at least 8 characters"), http.StatusBadRequest)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		authWriteError(w, fmt.Errorf("failed to hash password"), http.StatusInternalServerError)
		return
	}

	// Create user
	result, err := h.runtime.Execute(r.Context(), "user", "create", runtime.ActionInput{
		Data: map[string]any{
			"email":         req.Email,
			"password_hash": string(hash),
			"name":          req.Name,
			"status":        "active",
		},
		Channel: "http",
	})

	if err != nil {
		// Check for duplicate email
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "UNIQUE") {
			authWriteError(w, fmt.Errorf("email already registered"), http.StatusConflict)
			return
		}
		authWriteError(w, fmt.Errorf("failed to create user: %w", err), http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    result.ID,
			"email": req.Email,
			"name":  req.Name,
		},
	}

	// Generate JWT if token service is available
	if h.tokens != nil {
		planID, _ := result.Data["plan_id"].(string)
		role, _ := result.Data["role"].(string)
		token, expiresAt, err := h.tokens.GenerateToken(result.ID, req.Email, role, planID)
		if err == nil {
			resp["token"] = token
			resp["expires_at"] = expiresAt.Format(time.RFC3339)
		}
	}

	w.WriteHeader(http.StatusCreated)
	authWriteJSON(w, resp)
}

// handleLogin handles user login.
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		authWriteError(w, fmt.Errorf("invalid request: %w", err), http.StatusBadRequest)
		return
	}

	// Validate
	if req.Email == "" {
		authWriteError(w, fmt.Errorf("email is required"), http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		authWriteError(w, fmt.Errorf("password is required"), http.StatusBadRequest)
		return
	}

	// Find user by email
	result, err := h.runtime.Execute(r.Context(), "user", "get", runtime.ActionInput{
		Lookup:  req.Email,
		Channel: "http",
	})

	if err != nil {
		authWriteError(w, fmt.Errorf("invalid email or password"), http.StatusUnauthorized)
		return
	}

	// Check password - handle both string and []byte for password_hash
	var passwordHashBytes []byte
	switch v := result.Data["password_hash"].(type) {
	case string:
		passwordHashBytes = []byte(v)
	case []byte:
		passwordHashBytes = v
	default:
		authWriteError(w, fmt.Errorf("invalid email or password"), http.StatusUnauthorized)
		return
	}

	if len(passwordHashBytes) == 0 {
		authWriteError(w, fmt.Errorf("invalid email or password"), http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword(passwordHashBytes, []byte(req.Password)); err != nil {
		authWriteError(w, fmt.Errorf("invalid email or password"), http.StatusUnauthorized)
		return
	}

	// Check user status
	status, _ := result.Data["status"].(string)
	if status != "active" {
		authWriteError(w, fmt.Errorf("account is %s", status), http.StatusForbidden)
		return
	}

	userID, _ := result.Data["id"].(string)
	email, _ := result.Data["email"].(string)
	name, _ := result.Data["name"].(string)

	resp := map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    userID,
			"email": email,
			"name":  name,
		},
	}

	// Generate JWT if token service is available
	if h.tokens != nil {
		planID, _ := result.Data["plan_id"].(string)
		role, _ := result.Data["role"].(string)
		token, expiresAt, err := h.tokens.GenerateToken(userID, email, role, planID)
		if err == nil {
			resp["token"] = token
			resp["expires_at"] = expiresAt.Format(time.RFC3339)
		}
	}

	authWriteJSON(w, resp)
}

// handleLogout handles user logout. Stateless — just return success.
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	authWriteJSON(w, map[string]any{
		"success": true,
	})
}

// handleMe returns the current user.
func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID, ok := r.Context().Value(authUserIDKey).(string)
	if !ok || userID == "" {
		// Fall back to extracting from Bearer token directly
		userID = h.extractUserID(r)
		if userID == "" {
			authWriteError(w, fmt.Errorf("not authenticated"), http.StatusUnauthorized)
			return
		}
	}

	// Fetch fresh user data
	result, err := h.runtime.Execute(r.Context(), "user", "get", runtime.ActionInput{
		Lookup:  userID,
		Channel: "http",
	})

	if err != nil {
		authWriteError(w, fmt.Errorf("user not found"), http.StatusUnauthorized)
		return
	}

	// Remove sensitive fields
	delete(result.Data, "password_hash")

	authWriteJSON(w, map[string]any{
		"user": result.Data,
	})
}

// extractUserID extracts the user ID from a Bearer token in the Authorization header.
func (h *AuthHandler) extractUserID(r *http.Request) string {
	if h.tokens == nil {
		return ""
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "" // No "Bearer " prefix
	}

	claims, err := h.tokens.ValidateToken(token)
	if err != nil {
		return ""
	}

	return claims.UserID
}

// AuthMiddleware checks if user is authenticated via Bearer JWT.
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.tokens == nil {
			authWriteError(w, fmt.Errorf("authentication not configured"), http.StatusUnauthorized)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			authWriteError(w, fmt.Errorf("authentication required"), http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			authWriteError(w, fmt.Errorf("invalid authorization format"), http.StatusUnauthorized)
			return
		}

		claims, err := h.tokens.ValidateToken(token)
		if err != nil {
			authWriteError(w, fmt.Errorf("invalid or expired token"), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), authUserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper functions (use local names to avoid conflicts)
func authWriteJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func authWriteError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}
