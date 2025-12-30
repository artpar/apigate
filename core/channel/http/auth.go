// Package http provides authentication endpoints for the web UI.
package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/core/runtime"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	runtime   *runtime.Runtime
	jwtSecret []byte
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(rt *runtime.Runtime) *AuthHandler {
	// Generate a random JWT secret if not configured
	secret := make([]byte, 32)
	rand.Read(secret)
	return &AuthHandler{
		runtime:   rt,
		jwtSecret: secret,
	}
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

// SessionCookie is the name of the session cookie.
const SessionCookie = "apigate_session"

// Session represents a user session stored in cookie.
type Session struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at"`
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

	// Create session
	session := Session{
		UserID:    createResult.ID,
		Email:     req.Email,
		Name:      req.Name,
		ExpiresAt: time.Now().Add(24 * time.Hour * 7), // 7 days
	}

	h.setSessionCookie(w, session)

	authWriteJSON(w, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    createResult.ID,
			"email": req.Email,
			"name":  req.Name,
		},
	})
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

	// Create session
	session := Session{
		UserID:    result.ID,
		Email:     req.Email,
		Name:      req.Name,
		ExpiresAt: time.Now().Add(24 * time.Hour * 7), // 7 days
	}

	h.setSessionCookie(w, session)

	w.WriteHeader(http.StatusCreated)
	authWriteJSON(w, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    result.ID,
			"email": req.Email,
			"name":  req.Name,
		},
	})
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

	// Create session
	userID, _ := result.Data["id"].(string)
	email, _ := result.Data["email"].(string)
	name, _ := result.Data["name"].(string)

	session := Session{
		UserID:    userID,
		Email:     email,
		Name:      name,
		ExpiresAt: time.Now().Add(24 * time.Hour * 7), // 7 days
	}

	h.setSessionCookie(w, session)

	authWriteJSON(w, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    userID,
			"email": email,
			"name":  name,
		},
	})
}

// handleLogout handles user logout.
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authWriteJSON(w, map[string]any{
		"success": true,
	})
}

// handleMe returns the current user.
func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	session, err := h.getSession(r)
	if err != nil {
		authWriteError(w, fmt.Errorf("not authenticated"), http.StatusUnauthorized)
		return
	}

	// Fetch fresh user data
	result, err := h.runtime.Execute(r.Context(), "user", "get", runtime.ActionInput{
		Lookup:  session.UserID,
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

// setSessionCookie sets the session cookie.
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, session Session) {
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    encoded,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true in production with HTTPS
	})
}

// getSession retrieves the session from cookie.
func (h *AuthHandler) getSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// AuthMiddleware checks if user is authenticated.
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := h.getSession(r)
		if err != nil {
			authWriteError(w, fmt.Errorf("authentication required"), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
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
