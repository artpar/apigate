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

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// JSON:API resource type constants
const (
	TypeUser    = "users"
	TypeKey     = "api_keys"
	TypeSession = "sessions"
)

// Handler provides admin API endpoints.
type Handler struct {
	users          ports.UserStore
	keys           ports.KeyStore
	usage          ports.UsageStore
	routes         ports.RouteStore
	upstreams      ports.UpstreamStore
	plans          ports.PlanStore
	logger         zerolog.Logger
	hasher         ports.Hasher
	sessions       *SessionStore
	tokens         *auth.TokenService // JWT token service for Web UI session validation
	routesHandler  *RoutesHandler
	meterHandler   *MeterHandler
	reloadCallback func(context.Context) error // Called when explicit reload is requested
}

// Deps contains dependencies for the admin handler.
type Deps struct {
	Users          ports.UserStore
	Keys           ports.KeyStore
	Usage          ports.UsageStore
	Routes         ports.RouteStore
	Upstreams      ports.UpstreamStore
	Plans          ports.PlanStore
	Logger         zerolog.Logger
	Hasher         ports.Hasher
	JWTSecret      string                       // Optional JWT secret for Web UI session validation
	OnRouteChange  func()                       // Optional callback when routes/upstreams change (for cache invalidation)
	ReloadCallback func(context.Context) error  // Optional callback for explicit reload (POST /admin/reload)
}

// NewHandler creates a new admin API handler.
func NewHandler(deps Deps) *Handler {
	h := &Handler{
		users:          deps.Users,
		keys:           deps.Keys,
		usage:          deps.Usage,
		routes:         deps.Routes,
		upstreams:      deps.Upstreams,
		plans:          deps.Plans,
		logger:         deps.Logger,
		hasher:         deps.Hasher,
		sessions:       NewSessionStore(),
		reloadCallback: deps.ReloadCallback,
	}

	// Create token service for Web UI session validation (if JWT secret provided)
	if deps.JWTSecret != "" {
		h.tokens = auth.NewTokenService(deps.JWTSecret, 24*time.Hour)
	}

	// Create routes handler if stores are provided
	if deps.Routes != nil && deps.Upstreams != nil {
		h.routesHandler = NewRoutesHandlerWithConfig(RoutesHandlerConfig{
			Routes:        deps.Routes,
			Upstreams:     deps.Upstreams,
			Logger:        deps.Logger,
			OnRouteChange: deps.OnRouteChange,
		})
	}

	// Create meter handler if usage store is provided
	if deps.Usage != nil {
		h.meterHandler = NewMeterHandler(MeterHandlerConfig{
			Usage:  deps.Usage,
			Users:  deps.Users,
			Logger: deps.Logger,
		})
	}

	return h
}

// Router returns the admin API router.
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Root endpoint - API info
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})

	// Public endpoints (no auth required)
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)

	// Protected endpoints (require auth)
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Get("/me", h.Me)
		r.Post("/logout", h.Logout)

		// Users
		r.Get("/users", h.ListUsers)
		r.Post("/users", h.CreateUser)
		r.Get("/users/{id}", h.GetUser)
		r.Put("/users/{id}", h.UpdateUser)
		r.Patch("/users/{id}", h.UpdateUser)
		r.Delete("/users/{id}", h.DeleteUser)

		// Keys
		r.Get("/keys", h.ListKeys)
		r.Post("/keys", h.CreateKey)
		r.Delete("/keys/{id}", h.RevokeKey)

		// Plans
		r.Get("/plans", h.ListPlans)
		r.Post("/plans", h.CreatePlan)
		r.Get("/plans/{id}", h.GetPlan)
		r.Put("/plans/{id}", h.UpdatePlan)
		r.Patch("/plans/{id}", h.UpdatePlan)
		r.Delete("/plans/{id}", h.DeletePlan)

		// Usage
		r.Get("/usage", h.GetUsage)

		// Doctor (system health)
		r.Get("/doctor", h.Doctor)

		// Reload (hot-reload routes, upstreams, and config)
		r.Post("/reload", h.Reload)

		// Routes and Upstreams (if configured)
		if h.routesHandler != nil {
			h.routesHandler.RegisterRoutes(r)
		}

		// Metering API (if configured)
		if h.meterHandler != nil {
			r.Mount("/meter", h.meterHandler.Router())
		}
	})

	return r
}

// MeterRouter returns the meter handler's router for external mounting.
// This allows the metering API to be mounted at /api/v1/meter for service account access.
func (h *Handler) MeterRouter() chi.Router {
	if h.meterHandler == nil {
		return nil
	}
	return h.meterHandler.Router()
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
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	// Try API key authentication first
	if req.APIKey != "" {
		if err := h.authenticateByAPIKey(r.Context(), req.APIKey); err != nil {
			jsonapi.WriteUnauthorized(w, "Invalid API key")
			return
		}

		// Get user by email if provided, otherwise use first admin
		user, err := h.users.GetByEmail(r.Context(), req.Email)
		if err != nil {
			// For API key auth without email, create a session for "admin"
			session := h.sessions.Create("admin", "admin@apigate", 24*time.Hour)
			token := h.generateTokenIfAvailable("admin", "admin@apigate")
			jsonapi.WriteResource(w, http.StatusOK, sessionToResource(session, "admin", "admin@apigate", token))
			return
		}

		session := h.sessions.Create(user.ID, user.Email, 24*time.Hour)
		token := h.generateTokenIfAvailable(user.ID, user.Email)
		jsonapi.WriteResource(w, http.StatusOK, sessionToResource(session, user.ID, user.Email, token))
		return
	}

	// Email/password authentication
	if req.Email == "" {
		jsonapi.WriteValidationError(w, "email", "Email or API key required")
		return
	}

	if req.Password == "" {
		jsonapi.WriteValidationError(w, "password", "Password is required for email login")
		return
	}

	// Look up user by email
	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		jsonapi.WriteUnauthorized(w, "Invalid email or password")
		return
	}

	// Check if user has password hash set
	if len(user.PasswordHash) == 0 {
		jsonapi.WriteUnauthorized(w, "User has no password set. Use API key auth.")
		return
	}

	// Verify password
	if !h.hasher.Compare(user.PasswordHash, req.Password) {
		jsonapi.WriteUnauthorized(w, "Invalid email or password")
		return
	}

	// Check user status
	if user.Status != "active" {
		jsonapi.WriteForbidden(w, "Account is not active")
		return
	}

	// Create session and generate JWT token
	session := h.sessions.Create(user.ID, user.Email, 24*time.Hour)
	token := h.generateTokenIfAvailable(user.ID, user.Email)
	jsonapi.WriteResource(w, http.StatusOK, sessionToResource(session, user.ID, user.Email, token))
}

// generateTokenIfAvailable generates a JWT token if the token service is configured.
func (h *Handler) generateTokenIfAvailable(userID, email string) string {
	if h.tokens == nil {
		return ""
	}
	token, _, err := h.tokens.GenerateToken(userID, email, "admin")
	if err != nil {
		return ""
	}
	return token
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
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{"status": "logged_out"})
}

// Me returns the current authenticated user's information.
//
//	@Summary		Get current user
//	@Description	Returns information about the currently authenticated user
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	jsonapi.Document	"Current user"
//	@Failure		401	{object}	ErrorResponse		"Not authenticated"
//	@Security		AdminAuth
//	@Router			/admin/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserIDKey).(string)

	// If it's an API key auth, return minimal admin info
	if userID == "admin" {
		jsonapi.WriteResource(w, http.StatusOK, jsonapi.NewResource(TypeUser, "admin").
			Attr("email", "admin@apigate").
			Attr("role", "admin").
			Build())
		return
	}

	// Get user from store
	user, err := h.users.Get(r.Context(), userID)
	if err != nil {
		jsonapi.WriteNotFound(w, "User not found")
		return
	}

	jsonapi.WriteResource(w, http.StatusOK, userToResource(user))
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// Register creates a new user account.
//
//	@Summary		Register new user
//	@Description	Create a new user account (if registration is enabled)
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Registration details"
//	@Success		201		{object}	jsonapi.Document	"User created"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Failure		409		{object}	ErrorResponse		"Email already exists"
//	@Router			/admin/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Email == "" {
		jsonapi.WriteValidationError(w, "email", "Email is required")
		return
	}

	if req.Password == "" {
		jsonapi.WriteValidationError(w, "password", "Password is required")
		return
	}

	// Check if user already exists
	if _, err := h.users.GetByEmail(r.Context(), req.Email); err == nil {
		jsonapi.WriteConflict(w, "Email already registered")
		return
	}

	// Hash password
	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		jsonapi.WriteInternalError(w, "Failed to hash password")
		return
	}

	// Create user
	user := ports.User{
		ID:           generateUserID(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passwordHash,
		Status:       "active",
		PlanID:       "free", // Default plan
		CreatedAt:    time.Now(),
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		jsonapi.WriteInternalError(w, "Failed to create user")
		return
	}

	// Generate JWT token for immediate login
	token := h.generateTokenIfAvailable(user.ID, user.Email)

	// Return user with token
	rb := jsonapi.NewResource(TypeUser, user.ID).
		Attr("email", user.Email).
		Attr("name", user.Name).
		Attr("status", user.Status).
		Attr("plan_id", user.PlanID).
		Attr("created_at", user.CreatedAt.Format(time.RFC3339))
	if token != "" {
		rb.Attr("token", token)
	}
	jsonapi.WriteResource(w, http.StatusCreated, rb.Build())
}

// AuthRouter returns a router with only auth-related endpoints.
// This is mounted at /auth/* as an alias for /admin/* auth endpoints.
func (h *Handler) AuthRouter() chi.Router {
	r := chi.NewRouter()

	// Public auth endpoints
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)

	// Protected auth endpoints
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/me", h.Me)
		r.Post("/logout", h.Logout)
	})

	return r
}

// AuthMiddleware validates admin authentication.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try Admin API session cookie first
		if cookie, err := r.Cookie("session_id"); err == nil {
			if session := h.sessions.Get(cookie.Value); session != nil {
				ctx := context.WithValue(r.Context(), ctxSessionKey, session.ID)
				ctx = context.WithValue(ctx, ctxUserIDKey, session.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try Web UI JWT token cookie (enables Admin UI to make Admin API calls)
		if h.tokens != nil {
			if cookie, err := r.Cookie("token"); err == nil {
				if claims, err := h.tokens.ValidateToken(cookie.Value); err == nil {
					// JWT token is valid - allow access
					ctx := context.WithValue(r.Context(), ctxSessionKey, "jwt_token")
					ctx = context.WithValue(ctx, ctxUserIDKey, claims.UserID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// Try Authorization header (Bearer JWT, session_id, or API key)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Try JWT validation first (if token service configured)
			if h.tokens != nil {
				if claims, err := h.tokens.ValidateToken(token); err == nil {
					ctx := context.WithValue(r.Context(), ctxSessionKey, "jwt_token")
					ctx = context.WithValue(ctx, ctxUserIDKey, claims.UserID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

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

		jsonapi.WriteUnauthorized(w, "Valid session or API key required")
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
	page, perPage := jsonapi.ParsePaginationParams(r.URL.Query(), 20)
	offset := (page - 1) * perPage

	users, err := h.users.List(r.Context(), perPage, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list users")
		jsonapi.WriteInternalError(w, "Failed to list users")
		return
	}

	total, _ := h.users.Count(r.Context())

	resources := make([]jsonapi.Resource, len(users))
	for i, u := range users {
		resources[i] = userToResource(u)
	}

	pagination := jsonapi.NewPagination(int64(total), page, perPage, r.URL.String())
	jsonapi.WriteCollection(w, http.StatusOK, resources, pagination)
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
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Email == "" {
		jsonapi.WriteValidationError(w, "email", "Email is required")
		return
	}

	// Check if email already exists
	if _, err := h.users.GetByEmail(r.Context(), req.Email); err == nil {
		jsonapi.WriteConflict(w, "User with this email already exists")
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
			jsonapi.WriteInternalError(w, "Failed to create user")
			return
		}
		user.PasswordHash = hash
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to create user")
		jsonapi.WriteInternalError(w, "Failed to create user")
		return
	}

	h.logger.Info().Str("user_id", user.ID).Str("email", user.Email).Msg("user created via admin api")
	jsonapi.WriteCreated(w, userToResource(user), "/admin/users/"+user.ID)
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
		jsonapi.WriteNotFound(w, "user")
		return
	}

	jsonapi.WriteResource(w, http.StatusOK, userToResource(user))
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
//	@Router			/admin/users/{id} [patch]
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.users.Get(r.Context(), id)
	if err != nil {
		jsonapi.WriteNotFound(w, "user")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
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
			jsonapi.WriteInternalError(w, "Failed to update user")
			return
		}
		user.PasswordHash = hash
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to update user")
		jsonapi.WriteInternalError(w, "Failed to update user")
		return
	}

	h.logger.Info().Str("user_id", user.ID).Msg("user updated via admin api")
	jsonapi.WriteResource(w, http.StatusOK, userToResource(user))
}

// DeleteUser deletes a user.
//
//	@Summary		Delete user
//	@Description	Delete user by ID
//	@Tags			Admin - Users
//	@Produce		json
//	@Param			id	path		string				true	"User ID"
//	@Success		204	"No content - successfully deleted"
//	@Failure		404	{object}	ErrorResponse		"User not found"
//	@Security		AdminAuth
//	@Router			/admin/users/{id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := h.users.Get(r.Context(), id); err != nil {
		jsonapi.WriteNotFound(w, "user")
		return
	}

	// Note: UserStore doesn't have Delete method yet, we'll update status to "deleted"
	user, _ := h.users.Get(r.Context(), id)
	user.Status = "deleted"
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete user")
		jsonapi.WriteInternalError(w, "Failed to delete user")
		return
	}

	h.logger.Info().Str("user_id", id).Msg("user deleted via admin api")
	jsonapi.WriteNoContent(w)
}

// userToResource converts a User to a JSON:API Resource.
func userToResource(u ports.User) jsonapi.Resource {
	return jsonapi.NewResource(TypeUser, u.ID).
		Attr("email", u.Email).
		Attr("name", u.Name).
		Attr("status", u.Status).
		Attr("created_at", u.CreatedAt.Format(time.RFC3339)).
		Attr("updated_at", u.UpdatedAt.Format(time.RFC3339)).
		BelongsTo("plan", "plans", u.PlanID).
		Build()
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
			jsonapi.WriteInternalError(w, "Failed to list keys")
			return
		}

		for _, u := range users {
			userKeys, _ := h.keys.ListByUser(r.Context(), u.ID)
			keys = append(keys, userKeys...)
		}
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list keys")
		jsonapi.WriteInternalError(w, "Failed to list keys")
		return
	}

	resources := make([]jsonapi.Resource, len(keys))
	for i, k := range keys {
		resources[i] = keyToResource(k)
	}

	// Keys don't have pagination, just return collection with total in meta
	jsonapi.WriteCollection(w, http.StatusOK, resources, nil)
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
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.UserID == "" {
		jsonapi.WriteValidationError(w, "user_id", "user_id is required")
		return
	}

	// Verify user exists
	if _, err := h.users.Get(r.Context(), req.UserID); err != nil {
		jsonapi.WriteNotFound(w, "user")
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
		jsonapi.WriteInternalError(w, "Failed to create key")
		return
	}

	h.logger.Info().Str("key_id", keyData.ID).Str("user_id", req.UserID).Msg("key created via admin api")

	// Return key resource with the raw key in meta (only shown once)
	resource := jsonapi.NewResource(TypeKey, keyData.ID).
		Attr("prefix", keyData.Prefix).
		Attr("name", req.Name).
		Attr("created_at", keyData.CreatedAt.Format(time.RFC3339)).
		BelongsTo("user", TypeUser, req.UserID).
		Meta("key", rawKey).
		Meta("note", "Save this key securely. It will not be shown again.").
		Build()

	jsonapi.WriteCreated(w, resource, "/admin/keys/"+keyData.ID)
}

// RevokeKey revokes an API key.
//
//	@Summary		Revoke key
//	@Description	Revoke an API key
//	@Tags			Admin - Keys
//	@Produce		json
//	@Param			id	path		string				true	"Key ID"
//	@Success		204	"No content - successfully revoked"
//	@Failure		404	{object}	ErrorResponse		"Key not found"
//	@Security		AdminAuth
//	@Router			/admin/keys/{id} [delete]
func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.keys.Revoke(r.Context(), id, time.Now().UTC()); err != nil {
		h.logger.Error().Err(err).Str("key_id", id).Msg("failed to revoke key")
		jsonapi.WriteNotFound(w, "key")
		return
	}

	h.logger.Info().Str("key_id", id).Msg("key revoked via admin api")
	jsonapi.WriteNoContent(w)
}

// keyToResource converts a Key to a JSON:API Resource.
func keyToResource(k key.Key) jsonapi.Resource {
	rb := jsonapi.NewResource(TypeKey, k.ID).
		Attr("prefix", k.Prefix).
		Attr("name", k.Name).
		Attr("created_at", k.CreatedAt.Format(time.RFC3339)).
		BelongsTo("user", TypeUser, k.UserID)

	if k.ExpiresAt != nil {
		rb.Attr("expires_at", k.ExpiresAt.Format(time.RFC3339))
	}
	if k.RevokedAt != nil {
		rb.Attr("revoked_at", k.RevokedAt.Format(time.RFC3339))
	}
	if k.LastUsed != nil {
		rb.Attr("last_used", k.LastUsed.Format(time.RFC3339))
	}
	return rb.Build()
}

// sessionToResource converts a Session to a JSON:API Resource.
// If token is provided, it's included in the response for API authentication.
func sessionToResource(s *Session, userID, userEmail, token string) jsonapi.Resource {
	rb := jsonapi.NewResource(TypeSession, s.ID).
		Attr("expires_at", s.ExpiresAt.Format(time.RFC3339)).
		Attr("created_at", s.CreatedAt.Format(time.RFC3339)).
		BelongsTo("user", TypeUser, userID).
		Meta("user_email", userEmail)
	if token != "" {
		rb.Attr("token", token)
	}
	return rb.Build()
}

// -----------------------------------------------------------------------------
// System Operations
// -----------------------------------------------------------------------------

// ReloadResponse represents the response from a reload operation.
type ReloadResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// Reload triggers a hot-reload of routes, upstreams, and configuration.
//
//	@Summary		Reload configuration
//	@Description	Hot-reload routes, upstreams, and other configuration from database
//	@Tags			Admin - System
//	@Produce		json
//	@Success		200	{object}	ReloadResponse	"Reload successful"
//	@Failure		500	{object}	ErrorResponse	"Reload failed"
//	@Security		AdminAuth
//	@Router			/admin/reload [post]
func (h *Handler) Reload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if reload callback is configured
	if h.reloadCallback == nil {
		jsonapi.WriteInternalError(w, "Reload not configured")
		return
	}

	// Execute reload
	if err := h.reloadCallback(ctx); err != nil {
		h.logger.Error().Err(err).Msg("reload failed")
		jsonapi.WriteInternalError(w, "Reload failed: "+err.Error())
		return
	}

	h.logger.Info().Msg("configuration reloaded via admin API")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ReloadResponse{
		Status:    "success",
		Message:   "Routes, upstreams, and configuration reloaded",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// ErrorResponse represents an API error (kept for OpenAPI documentation).
type ErrorResponse struct {
	Errors []struct {
		Status string `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail,omitempty"`
	} `json:"errors"`
}

var ErrInvalidCredentials = errorType{"invalid_credentials", "Invalid credentials"}

type errorType struct {
	code    string
	message string
}

func (e errorType) Error() string {
	return e.message
}
