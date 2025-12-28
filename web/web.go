// Package web provides the SSR admin web interface.
// All templates and static files are embedded in the binary.
// Stateless design - no server-side session storage.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

//go:embed templates/* static/*
var assets embed.FS

// Handler provides the web UI endpoints.
type Handler struct {
	templates map[string]*template.Template // One template per page
	tokens    *auth.TokenService
	users     ports.UserStore
	keys      ports.KeyStore
	usage     ports.UsageStore
	routes    ports.RouteStore
	upstreams ports.UpstreamStore
	config    *config.Config
	logger    zerolog.Logger
	hasher    ports.Hasher
	isSetup   func() bool // Returns true if initial setup is complete
}

// Deps contains dependencies for the web handler.
type Deps struct {
	Users     ports.UserStore
	Keys      ports.KeyStore
	Usage     ports.UsageStore
	Routes    ports.RouteStore
	Upstreams ports.UpstreamStore
	Config    *config.Config
	Logger    zerolog.Logger
	Hasher    ports.Hasher
	JWTSecret string
	IsSetup   func() bool
}

// NewHandler creates a new web UI handler.
func NewHandler(deps Deps) (*Handler, error) {
	// Parse all templates
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	return &Handler{
		templates: tmpl,
		tokens:    auth.NewTokenService(deps.JWTSecret, 24*time.Hour),
		users:     deps.Users,
		keys:      deps.Keys,
		usage:     deps.Usage,
		routes:    deps.Routes,
		upstreams: deps.Upstreams,
		config:    deps.Config,
		logger:    deps.Logger,
		hasher:    deps.Hasher,
		isSetup:   deps.IsSetup,
	}, nil
}

// Router returns the web UI router.
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Static files (CSS, JS) - no auth required
	staticFS, _ := fs.Sub(assets, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// First-run setup wizard (no auth required)
	r.Get("/setup", h.SetupPage)
	r.Post("/setup", h.SetupSubmit)
	r.Get("/setup/step/{step}", h.SetupStep)
	r.Post("/setup/step/{step}", h.SetupStepSubmit)

	// Login page (no auth required)
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.LoginSubmit)
	r.Post("/logout", h.Logout)

	// Protected pages (require auth)
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		// Dashboard
		r.Get("/", h.Dashboard)
		r.Get("/dashboard", h.Dashboard)

		// Users
		r.Get("/users", h.UsersPage)
		r.Get("/users/new", h.UserNewPage)
		r.Post("/users", h.UserCreate)
		r.Get("/users/{id}", h.UserEditPage)
		r.Post("/users/{id}", h.UserUpdate)
		r.Delete("/users/{id}", h.UserDelete)

		// Keys
		r.Get("/keys", h.KeysPage)
		r.Post("/keys", h.KeyCreate)
		r.Delete("/keys/{id}", h.KeyRevoke)

		// Plans
		r.Get("/plans", h.PlansPage)

		// Routes
		r.Get("/routes", h.RoutesPage)
		r.Get("/routes/new", h.RouteNewPage)
		r.Post("/routes", h.RouteCreate)
		r.Get("/routes/{id}", h.RouteEditPage)
		r.Post("/routes/{id}", h.RouteUpdate)
		r.Delete("/routes/{id}", h.RouteDelete)

		// Upstreams
		r.Get("/upstreams", h.UpstreamsPage)
		r.Get("/upstreams/new", h.UpstreamNewPage)
		r.Post("/upstreams", h.UpstreamCreate)
		r.Get("/upstreams/{id}", h.UpstreamEditPage)
		r.Post("/upstreams/{id}", h.UpstreamUpdate)
		r.Delete("/upstreams/{id}", h.UpstreamDelete)

		// Usage
		r.Get("/usage", h.UsagePage)

		// Settings
		r.Get("/settings", h.SettingsPage)

		// System Status
		r.Get("/system", h.HealthPage)

		// HTMX partial endpoints (for dynamic updates)
		r.Get("/partials/stats", h.PartialStats)
		r.Get("/partials/users", h.PartialUsers)
		r.Get("/partials/keys", h.PartialKeys)
		r.Get("/partials/activity", h.PartialActivity)
		r.Get("/partials/routes", h.PartialRoutes)
		r.Get("/partials/upstreams", h.PartialUpstreams)
	})

	return r
}

// AuthMiddleware validates JWT token from cookie.
// Stateless - no server-side session lookup.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for JWT token in cookie
		cookie, err := r.Cookie("token")
		if err != nil {
			h.redirectToLogin(w, r)
			return
		}

		claims, err := h.tokens.ValidateToken(cookie.Value)
		if err != nil {
			h.redirectToLogin(w, r)
			return
		}

		// Add claims to request context
		ctx := withClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// For HTMX requests, return 401 with redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// SetupRequired middleware redirects to setup if not configured.
func (h *Handler) SetupRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip for setup pages and static files
		if strings.HasPrefix(r.URL.Path, "/setup") || strings.HasPrefix(r.URL.Path, "/static") {
			next.ServeHTTP(w, r)
			return
		}

		// Check if setup is complete
		if h.isSetup != nil && !h.isSetup() {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper to parse all templates with layouts
func parseTemplates() (map[string]*template.Template, error) {
	// Define template functions
	funcs := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			switch {
			case d < time.Minute:
				return "just now"
			case d < time.Hour:
				return formatDuration(d.Minutes(), "minute")
			case d < 24*time.Hour:
				return formatDuration(d.Hours(), "hour")
			default:
				return formatDuration(d.Hours()/24, "day")
			}
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"maskKey": func(s string) string {
			if len(s) <= 8 {
				return "****"
			}
			return s[:8] + "****" + s[len(s)-4:]
		},
		"add": func(a, b int) int {
			return a + b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []string:
				return len(val)
			case string:
				return len(val)
			default:
				return 0
			}
		},
	}

	templates := make(map[string]*template.Template)

	// Read layout content
	layoutContent, err := fs.ReadFile(assets, "templates/layouts/base.html")
	if err != nil {
		return nil, err
	}

	// Read component content
	var componentContent []byte
	components, err := fs.Glob(assets, "templates/components/*.html")
	if err != nil {
		return nil, err
	}
	for _, comp := range components {
		content, err := fs.ReadFile(assets, comp)
		if err != nil {
			return nil, err
		}
		componentContent = append(componentContent, content...)
	}

	// Parse each page as its own template (layout + components + page)
	pages, err := fs.Glob(assets, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		// Extract page name (e.g., "login" from "templates/pages/login.html")
		name := strings.TrimPrefix(page, "templates/pages/")
		name = strings.TrimSuffix(name, ".html")

		pageContent, err := fs.ReadFile(assets, page)
		if err != nil {
			return nil, err
		}

		// Create template with layout + components + this page
		tmpl := template.New(name).Funcs(funcs)
		_, err = tmpl.Parse(string(layoutContent))
		if err != nil {
			return nil, fmt.Errorf("parse layout for %s: %w", name, err)
		}
		if len(componentContent) > 0 {
			_, err = tmpl.Parse(string(componentContent))
			if err != nil {
				return nil, fmt.Errorf("parse components for %s: %w", name, err)
			}
		}
		_, err = tmpl.Parse(string(pageContent))
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}

		templates[name] = tmpl
	}

	return templates, nil
}

func formatDuration(n float64, unit string) string {
	i := int(n)
	if i == 1 {
		return "1 " + unit + " ago"
	}
	return string(rune('0'+i%10)) + " " + unit + "s ago"
}
