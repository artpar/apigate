// Package admin provides HTTP handlers for the Admin API.
package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// Handler provides admin API endpoints.
type Handler struct {
	users         ports.UserStore
	keys          ports.KeyStore
	usage         ports.UsageStore
	routes        ports.RouteStore
	upstreams     ports.UpstreamStore
	config        *config.Config
	logger        zerolog.Logger
	hasher        ports.Hasher
	sessions      *SessionStore
	routesHandler *RoutesHandler
}

// Deps contains dependencies for the admin handler.
type Deps struct {
	Users     ports.UserStore
	Keys      ports.KeyStore
	Usage     ports.UsageStore
	Routes    ports.RouteStore
	Upstreams ports.UpstreamStore
	Config    *config.Config
	Logger    zerolog.Logger
	Hasher    ports.Hasher
}

// NewHandler creates a new admin API handler.
func NewHandler(deps Deps) *Handler {
	h := &Handler{
		users:     deps.Users,
		keys:      deps.Keys,
		usage:     deps.Usage,
		routes:    deps.Routes,
		upstreams: deps.Upstreams,
		config:    deps.Config,
		logger:    deps.Logger,
		hasher:    deps.Hasher,
		sessions:  NewSessionStore(),
	}

	// Create routes handler if stores are provided
	if deps.Routes != nil && deps.Upstreams != nil {
		h.routesHandler = NewRoutesHandler(deps.Routes, deps.Upstreams, deps.Logger)
	}

	return h
}

// Router returns the admin API router.
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Public endpoints (no auth required)
	r.Post("/login", h.Login)

	// Protected endpoints (require auth)
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Post("/logout", h.Logout)

		// Users
		r.Get("/users", h.ListUsers)
		r.Post("/users", h.CreateUser)
		r.Get("/users/{id}", h.GetUser)
		r.Put("/users/{id}", h.UpdateUser)
		r.Delete("/users/{id}", h.DeleteUser)

		// Keys
		r.Get("/keys", h.ListKeys)
		r.Post("/keys", h.CreateKey)
		r.Delete("/keys/{id}", h.RevokeKey)

		// Plans
		r.Get("/plans", h.ListPlans)
		r.Post("/plans", h.CreatePlan)
		r.Put("/plans/{id}", h.UpdatePlan)

		// Usage
		r.Get("/usage", h.GetUsage)

		// Settings
		r.Get("/settings", h.GetSettings)
		r.Put("/settings", h.UpdateSettings)

		// Doctor (system health)
		r.Get("/doctor", h.Doctor)

		// Routes and Upstreams (if configured)
		if h.routesHandler != nil {
			h.routesHandler.RegisterRoutes(r)
		}
	})

	return r
}

// -----------------------------------------------------------------------------
// Authentication
// -----------------------------------------------------------------------------

// Session represents an admin session.
type Session struct {
	ID        string
	UserID    string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore manages admin sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session.
func (s *SessionStore) Create(userID, email string, duration time.Duration) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateSessionID()
	session := &Session{
		ID:        id,
		UserID:    userID,
		Email:     email,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(duration),
	}
	s.sessions[id] = session
	return session
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil
	}
	return session
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
	User      struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

// Login authenticates an admin user.
//
//	@Summary		Admin login
//	@Description	Authenticate with email/password or API key
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login credentials"
//	@Success		200		{object}	LoginResponse	"Login successful"
//	@Failure		401		{object}	ErrorResponse	"Invalid credentials"
//	@Router			/admin/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Try API key authentication first
	if req.APIKey != "" {
		if err := h.authenticateByAPIKey(r.Context(), req.APIKey); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid API key")
			return
		}

		// Get user by email if provided, otherwise use first admin
		user, err := h.users.GetByEmail(r.Context(), req.Email)
		if err != nil {
			// For API key auth without email, create a session for "admin"
			session := h.sessions.Create("admin", "admin@apigate", 24*time.Hour)
			writeJSON(w, http.StatusOK, LoginResponse{
				SessionID: session.ID,
				ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
				User: struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				}{ID: "admin", Email: "admin@apigate"},
			})
			return
		}

		session := h.sessions.Create(user.ID, user.Email, 24*time.Hour)
		writeJSON(w, http.StatusOK, LoginResponse{
			SessionID: session.ID,
			ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
			User: struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			}{ID: user.ID, Email: user.Email},
		})
		return
	}

	// Email/password authentication
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "missing_credentials", "Email or API key required")
		return
	}

	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "missing_password", "Password is required for email login")
		return
	}

	// Look up user by email
	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Check if user has password hash set
	if len(user.PasswordHash) == 0 {
		writeError(w, http.StatusUnauthorized, "no_password", "User has no password set. Use API key auth.")
		return
	}

	// Verify password
	if !h.hasher.Compare(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Check user status
	if user.Status != "active" {
		writeError(w, http.StatusForbidden, "account_inactive", "Account is not active")
		return
	}

	// Create session
	session := h.sessions.Create(user.ID, user.Email, 24*time.Hour)
	writeJSON(w, http.StatusOK, LoginResponse{
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{ID: user.ID, Email: user.Email},
	})
}

func (h *Handler) authenticateByAPIKey(ctx context.Context, apiKey string) error {
	// Extract prefix for lookup
	if len(apiKey) < 12 {
		return ErrInvalidCredentials
	}
	prefix := apiKey[:12]

	keys, err := h.keys.Get(ctx, prefix)
	if err != nil || len(keys) == 0 {
		return ErrInvalidCredentials
	}

	// Verify the key matches
	for _, k := range keys {
		if h.hasher.Compare(k.Hash, apiKey) {
			if k.RevokedAt != nil {
				return ErrInvalidCredentials
			}
			return nil
		}
	}

	return ErrInvalidCredentials
}

// Logout ends an admin session.
//
//	@Summary		Admin logout
//	@Description	End the current session
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Logged out"
//	@Security		AdminAuth
//	@Router			/admin/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Context().Value(ctxSessionKey).(string)
	h.sessions.Delete(sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// AuthMiddleware validates admin authentication.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try session cookie first
		if cookie, err := r.Cookie("session_id"); err == nil {
			if session := h.sessions.Get(cookie.Value); session != nil {
				ctx := context.WithValue(r.Context(), ctxSessionKey, session.ID)
				ctx = context.WithValue(ctx, ctxUserIDKey, session.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try Authorization header (Bearer session_id or API key)
		auth := r.Header.Get("Authorization")
		if auth != "" {
			token := strings.TrimPrefix(auth, "Bearer ")

			// Check if it's a session ID
			if session := h.sessions.Get(token); session != nil {
				ctx := context.WithValue(r.Context(), ctxSessionKey, session.ID)
				ctx = context.WithValue(ctx, ctxUserIDKey, session.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Check if it's an API key
			if err := h.authenticateByAPIKey(r.Context(), token); err == nil {
				ctx := context.WithValue(r.Context(), ctxSessionKey, "api_key")
				ctx = context.WithValue(ctx, ctxUserIDKey, "admin")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try X-API-Key header
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if err := h.authenticateByAPIKey(r.Context(), apiKey); err == nil {
				ctx := context.WithValue(r.Context(), ctxSessionKey, "api_key")
				ctx = context.WithValue(ctx, ctxUserIDKey, "admin")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		writeError(w, http.StatusUnauthorized, "unauthorized", "Valid session or API key required")
	})
}

// Context keys
type ctxKey string

const (
	ctxSessionKey ctxKey = "session_id"
	ctxUserIDKey  ctxKey = "user_id"
)

// -----------------------------------------------------------------------------
// Users API
// -----------------------------------------------------------------------------

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	PlanID    string `json:"plan_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
	PlanID   string `json:"plan_id,omitempty"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
	PlanID   string `json:"plan_id,omitempty"`
	Status   string `json:"status,omitempty"`
}

// ListUsers returns all users.
//
//	@Summary		List users
//	@Description	Get all users with pagination
//	@Tags			Admin - Users
//	@Produce		json
//	@Param			limit	query		int					false	"Max results"	default(100)
//	@Param			offset	query		int					false	"Offset"		default(0)
//	@Success		200		{object}	map[string]interface{}	"Users list"
//	@Security		AdminAuth
//	@Router			/admin/users [get]
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 100)
	offset := parseIntQuery(r, "offset", 0)

	users, err := h.users.List(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list users")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list users")
		return
	}

	total, _ := h.users.Count(r.Context())

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = userToResponse(u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": response,
		"total": total,
	})
}

// CreateUser creates a new user.
//
//	@Summary		Create user
//	@Description	Create a new user
//	@Tags			Admin - Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateUserRequest	true	"User data"
//	@Success		201		{object}	UserResponse		"Created user"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Security		AdminAuth
//	@Router			/admin/users [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "missing_email", "Email is required")
		return
	}

	// Check if email already exists
	if _, err := h.users.GetByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "email_exists", "User with this email already exists")
		return
	}

	planID := req.PlanID
	if planID == "" {
		planID = "free"
	}

	now := time.Now().UTC()
	user := ports.User{
		ID:        generateUserID(),
		Email:     req.Email,
		Name:      req.Name,
		PlanID:    planID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Hash password if provided
	if req.Password != "" {
		hash, err := h.hasher.Hash(req.Password)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to hash password")
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create user")
			return
		}
		user.PasswordHash = hash
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to create user")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create user")
		return
	}

	h.logger.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("user created via admin api")
	writeJSON(w, http.StatusCreated, userToResponse(user))
}

// GetUser returns a single user.
//
//	@Summary		Get user
//	@Description	Get user by ID
//	@Tags			Admin - Users
//	@Produce		json
//	@Param			id	path		string			true	"User ID"
//	@Success		200	{object}	UserResponse	"User data"
//	@Failure		404	{object}	ErrorResponse	"User not found"
//	@Security		AdminAuth
//	@Router			/admin/users/{id} [get]
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.users.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

// UpdateUser updates a user.
//
//	@Summary		Update user
//	@Description	Update user by ID
//	@Tags			Admin - Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"User ID"
//	@Param			request	body		UpdateUserRequest	true	"Update data"
//	@Success		200		{object}	UserResponse		"Updated user"
//	@Failure		404		{object}	ErrorResponse		"User not found"
//	@Security		AdminAuth
//	@Router			/admin/users/{id} [put]
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.users.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.PlanID != "" {
		user.PlanID = req.PlanID
	}
	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Password != "" {
		hash, err := h.hasher.Hash(req.Password)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to hash password")
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update user")
			return
		}
		user.PasswordHash = hash
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update user")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update user")
		return
	}

	h.logger.Info().Str("user_id", user.ID).Msg("user updated via admin api")
	writeJSON(w, http.StatusOK, userToResponse(user))
}

// DeleteUser deletes a user.
//
//	@Summary		Delete user
//	@Description	Delete user by ID
//	@Tags			Admin - Users
//	@Produce		json
//	@Param			id	path		string				true	"User ID"
//	@Success		200	{object}	map[string]string	"Deleted"
//	@Failure		404	{object}	ErrorResponse		"User not found"
//	@Security		AdminAuth
//	@Router			/admin/users/{id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := h.users.Get(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Note: UserStore doesn't have Delete method yet, we'll update status to "deleted"
	user, _ := h.users.Get(r.Context(), id)
	user.Status = "deleted"
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete user")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete user")
		return
	}

	h.logger.Info().Str("user_id", id).Msg("user deleted via admin api")
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func userToResponse(u ports.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		PlanID:    u.PlanID,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func generateUserID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "user_" + hex.EncodeToString(b)
}

// -----------------------------------------------------------------------------
// Keys API
// -----------------------------------------------------------------------------

// KeyResponse represents a key in API responses.
type KeyResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Prefix    string  `json:"prefix"`
	Name      string  `json:"name,omitempty"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	RevokedAt *string `json:"revoked_at,omitempty"`
	LastUsed  *string `json:"last_used,omitempty"`
}

// CreateKeyRequest represents a request to create a key.
type CreateKeyRequest struct {
	UserID    string     `json:"user_id"`
	Name      string     `json:"name,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateKeyResponse includes the raw key (only shown once).
type CreateKeyResponse struct {
	Key    string      `json:"key"`
	KeyID  string      `json:"key_id"`
	Prefix string      `json:"prefix"`
	UserID string      `json:"user_id"`
	Name   string      `json:"name,omitempty"`
	Note   string      `json:"note"`
}

// ListKeys returns all keys.
//
//	@Summary		List keys
//	@Description	Get all API keys
//	@Tags			Admin - Keys
//	@Produce		json
//	@Param			user_id	query		string					false	"Filter by user ID"
//	@Success		200		{object}	map[string]interface{}	"Keys list"
//	@Security		AdminAuth
//	@Router			/admin/keys [get]
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	var keys []key.Key
	var err error

	if userID != "" {
		keys, err = h.keys.ListByUser(r.Context(), userID)
	} else {
		// List all keys - we need to get all users and their keys
		users, err := h.users.List(r.Context(), 1000, 0)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to list users")
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list keys")
			return
		}

		for _, u := range users {
			userKeys, _ := h.keys.ListByUser(r.Context(), u.ID)
			keys = append(keys, userKeys...)
		}
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list keys")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list keys")
		return
	}

	response := make([]KeyResponse, len(keys))
	for i, k := range keys {
		response[i] = keyToResponse(k)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keys":  response,
		"total": len(response),
	})
}

// CreateKey creates a new API key.
//
//	@Summary		Create key
//	@Description	Create a new API key for a user
//	@Tags			Admin - Keys
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateKeyRequest	true	"Key data"
//	@Success		201		{object}	CreateKeyResponse	"Created key (save the key, shown once)"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Security		AdminAuth
//	@Router			/admin/keys [post]
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "missing_user_id", "user_id is required")
		return
	}

	// Verify user exists
	if _, err := h.users.Get(r.Context(), req.UserID); err != nil {
		writeError(w, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	// Generate key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(req.UserID).WithName(req.Name)
	if req.ExpiresAt != nil {
		keyData.ExpiresAt = req.ExpiresAt
	}

	if err := h.keys.Create(r.Context(), keyData); err != nil {
		h.logger.Error().Err(err).Msg("failed to create key")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create key")
		return
	}

	h.logger.Info().Str("key_id", keyData.ID).Str("user_id", req.UserID).Msg("key created via admin api")

	writeJSON(w, http.StatusCreated, CreateKeyResponse{
		Key:    rawKey,
		KeyID:  keyData.ID,
		Prefix: keyData.Prefix,
		UserID: req.UserID,
		Name:   req.Name,
		Note:   "Save this key securely. It will not be shown again.",
	})
}

// RevokeKey revokes an API key.
//
//	@Summary		Revoke key
//	@Description	Revoke an API key
//	@Tags			Admin - Keys
//	@Produce		json
//	@Param			id	path		string				true	"Key ID"
//	@Success		200	{object}	map[string]string	"Revoked"
//	@Failure		404	{object}	ErrorResponse		"Key not found"
//	@Security		AdminAuth
//	@Router			/admin/keys/{id} [delete]
func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.keys.Revoke(r.Context(), id, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Str("key_id", id).Msg("failed to revoke key")
		writeError(w, http.StatusNotFound, "not_found", "Key not found")
		return
	}

	h.logger.Info().Str("key_id", id).Msg("key revoked via admin api")
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func keyToResponse(k key.Key) KeyResponse {
	resp := KeyResponse{
		ID:        k.ID,
		UserID:    k.UserID,
		Prefix:    k.Prefix,
		Name:      k.Name,
		CreatedAt: k.CreatedAt.Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		s := k.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	if k.RevokedAt != nil {
		s := k.RevokedAt.Format(time.RFC3339)
		resp.RevokedAt = &s
	}
	if k.LastUsed != nil {
		s := k.LastUsed.Format(time.RFC3339)
		resp.LastUsed = &s
	}
	return resp
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var ErrInvalidCredentials = errorType{"invalid_credentials", "Invalid credentials"}

type errorType struct {
	code    string
	message string
}

func (e errorType) Error() string {
	return e.message
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func parseIntQuery(r *http.Request, name string, defaultVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	var v int
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return defaultVal
	}
	return v
}
