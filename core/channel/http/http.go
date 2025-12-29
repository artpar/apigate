// Package http provides an HTTP channel that generates REST API endpoints from module definitions.
// It automatically creates list, get, create, update, delete, and custom action endpoints.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/openapi"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/go-chi/chi/v5"
)

// Channel implements the HTTP channel for modules.
type Channel struct {
	router  chi.Router
	runtime *runtime.Runtime
	modules map[string]convention.Derived
	addr    string
	server  *http.Server
}

// New creates a new HTTP channel.
func New(rt *runtime.Runtime, addr string) *Channel {
	c := &Channel{
		router:  chi.NewRouter(),
		runtime: rt,
		modules: make(map[string]convention.Derived),
		addr:    addr,
	}

	// Register schema introspection routes
	schemaHandler := NewSchemaHandler(c.modules)
	c.router.Mount("/_schema", schemaHandler.Routes())

	// Register OpenAPI endpoint
	c.router.Get("/_openapi", c.handleOpenAPI)
	c.router.Get("/_openapi.json", c.handleOpenAPI)

	// Mount Swagger UI at /swagger
	c.router.Get("/swagger", c.handleSwaggerUI)
	c.router.Get("/swagger/", c.handleSwaggerUI)

	// Mount Web UI at /ui
	// Create a subrouter for the UI that handles all subpaths
	c.router.Route("/ui", func(r chi.Router) {
		webHandler := WebUIHandler()
		r.Get("/*", webHandler.ServeHTTP)
	})

	return c
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "http"
}

// Handler returns the HTTP handler.
func (c *Channel) Handler() http.Handler {
	return c.router
}

// Register registers a module with the HTTP channel.
func (c *Channel) Register(mod convention.Derived) error {
	// Check if HTTP is enabled for this module
	if !mod.Source.Channels.HTTP.Serve.Enabled {
		return nil
	}

	c.modules[mod.Source.Name] = mod

	// Build base path
	basePath := mod.Source.Channels.HTTP.Serve.BasePath
	if basePath == "" {
		basePath = "/api/" + mod.Plural
	}

	// Register routes for each action
	for _, action := range mod.Actions {
		c.registerActionRoute(mod, action, basePath)
	}

	return nil
}

// Start starts the HTTP server.
func (c *Channel) Start(ctx context.Context) error {
	// Only start if addr is set (standalone mode)
	if c.addr == "" {
		return nil
	}

	c.server = &http.Server{
		Addr:    c.addr,
		Handler: c.router,
	}

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server.
func (c *Channel) Stop(ctx context.Context) error {
	if c.server != nil {
		return c.server.Shutdown(ctx)
	}
	return nil
}

// registerActionRoute registers an HTTP route for an action.
func (c *Channel) registerActionRoute(mod convention.Derived, action convention.DerivedAction, basePath string) {
	switch action.Type {
	case schema.ActionTypeList:
		// GET /plural - list all
		c.router.Get(basePath, c.handleList(mod))
		// POST /plural - create new
		c.router.Post(basePath, c.handleCreate(mod))

	case schema.ActionTypeGet:
		// GET /plural/{id} - get by id
		c.router.Get(basePath+"/{id}", c.handleGet(mod))

	case schema.ActionTypeUpdate:
		// PUT /plural/{id} - update
		c.router.Put(basePath+"/{id}", c.handleUpdate(mod))
		c.router.Patch(basePath+"/{id}", c.handleUpdate(mod))

	case schema.ActionTypeDelete:
		// DELETE /plural/{id} - delete
		c.router.Delete(basePath+"/{id}", c.handleDelete(mod))

	case schema.ActionTypeCustom:
		// POST /plural/{id}/{action} - custom action
		c.router.Post(basePath+"/{id}/"+action.Name, c.handleCustomAction(mod, action.Name))
	}
}

// handleList handles GET requests for listing records.
func (c *Channel) handleList(mod convention.Derived) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.doList(r.Context(), w, r, mod)
	}
}

// handleCreate handles POST requests for creating records.
func (c *Channel) handleCreate(mod convention.Derived) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.doCreate(r.Context(), w, r, mod)
	}
}

// handleGet handles GET requests for a single record.
func (c *Channel) handleGet(mod convention.Derived) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		c.doGet(r.Context(), w, r, mod, id)
	}
}

// handleUpdate handles PUT/PATCH requests for updating records.
func (c *Channel) handleUpdate(mod convention.Derived) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		c.doUpdate(r.Context(), w, r, mod, id)
	}
}

// handleDelete handles DELETE requests.
func (c *Channel) handleDelete(mod convention.Derived) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		c.doDelete(r.Context(), w, r, mod, id)
	}
}

// handleCustomAction handles POST requests for custom actions.
func (c *Channel) handleCustomAction(mod convention.Derived, actionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		c.doCustomAction(r.Context(), w, r, mod, id, actionName)
	}
}

// doList handles list requests.
func (c *Channel) doList(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived) {
	// Parse query parameters
	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	// Build filters from query params
	filters := make(map[string]any)
	for _, f := range mod.Fields {
		if val := r.URL.Query().Get(f.Name); val != "" {
			filters[f.Name] = val
		}
	}

	input := runtime.ActionInput{
		Data: map[string]any{
			"limit":   limit,
			"offset":  offset,
			"filters": filters,
		},
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "list", input)
	if err != nil {
		c.writeError(w, err, http.StatusInternalServerError)
		return
	}

	c.writeJSON(w, map[string]any{
		"data":  result.List,
		"count": result.Count,
	})
}

// doGet handles get requests.
func (c *Channel) doGet(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	result, err := c.runtime.Execute(ctx, mod.Source.Name, "get", runtime.ActionInput{
		Lookup:   id,
		Channel:  "http",
		RemoteIP: r.RemoteAddr,
	})
	if err != nil {
		c.writeError(w, err, http.StatusNotFound)
		return
	}

	c.writeJSON(w, map[string]any{
		"data": result.Data,
	})
}

// doCreate handles create requests.
func (c *Channel) doCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived) {
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		c.writeError(w, fmt.Errorf("invalid JSON: %w", err), http.StatusBadRequest)
		return
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "create", runtime.ActionInput{
		Data:         data,
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	})
	if err != nil {
		c.writeError(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	c.writeJSON(w, map[string]any{
		"id":   result.ID,
		"data": result.Data,
	})
}

// doUpdate handles update requests.
func (c *Channel) doUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		c.writeError(w, fmt.Errorf("invalid JSON: %w", err), http.StatusBadRequest)
		return
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "update", runtime.ActionInput{
		Lookup:       id,
		Data:         data,
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	})
	if err != nil {
		c.writeError(w, err, http.StatusBadRequest)
		return
	}

	c.writeJSON(w, map[string]any{
		"id":   result.ID,
		"data": result.Data,
	})
}

// doDelete handles delete requests.
func (c *Channel) doDelete(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	_, err := c.runtime.Execute(ctx, mod.Source.Name, "delete", runtime.ActionInput{
		Lookup:   id,
		Channel:  "http",
		RemoteIP: r.RemoteAddr,
	})
	if err != nil {
		c.writeError(w, err, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// doCustomAction handles custom action requests.
func (c *Channel) doCustomAction(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string, actionName string) {
	// Find the action
	var action *convention.DerivedAction
	for i := range mod.Actions {
		if mod.Actions[i].Name == actionName {
			action = &mod.Actions[i]
			break
		}
	}

	if action == nil {
		c.writeError(w, fmt.Errorf("action %q not found", actionName), http.StatusNotFound)
		return
	}

	var data map[string]any
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			c.writeError(w, fmt.Errorf("invalid JSON: %w", err), http.StatusBadRequest)
			return
		}
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, actionName, runtime.ActionInput{
		Lookup:       id,
		Data:         data,
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	})
	if err != nil {
		c.writeError(w, err, http.StatusBadRequest)
		return
	}

	c.writeJSON(w, map[string]any{
		"id":   result.ID,
		"data": result.Data,
	})
}

// writeJSON writes a JSON response.
func (c *Channel) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (c *Channel) writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

// handleOpenAPI returns the OpenAPI specification.
func (c *Channel) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	gen := openapi.NewGenerator(c.modules)

	// Set API info
	gen.SetInfo(openapi.Info{
		Title:       "APIGate API",
		Description: "Auto-generated REST API from module schemas. All endpoints support JSON request/response bodies.",
		Version:     "1.0.0",
	})

	// Add server from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	gen.AddServer(fmt.Sprintf("%s://%s/mod", scheme, r.Host), "Current server")

	spec := gen.Generate()

	data, err := spec.ToJSON()
	if err != nil {
		c.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

// handleSwaggerUI serves the Swagger UI page.
func (c *Channel) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>APIGate - API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 30px 0; }
        .swagger-ui .info .title { font-size: 2em; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: window.location.origin + "/mod/_openapi",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 2,
                docExpansion: "list",
                filter: true,
                showExtensions: true,
                showCommonExtensions: true
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
