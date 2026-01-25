// Package http provides an HTTP channel that generates REST API endpoints from module definitions.
// It automatically creates list, get, create, update, delete, and custom action endpoints.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/openapi"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/go-chi/chi/v5"
)

// Channel implements the HTTP channel for modules.
type Channel struct {
	router      chi.Router
	runtime     *runtime.Runtime
	modules     map[string]convention.Derived
	addr        string
	server      *http.Server
	authHandler *AuthHandler
}

// New creates a new HTTP channel.
func New(rt *runtime.Runtime, addr string) *Channel {
	c := &Channel{
		router:  chi.NewRouter(),
		runtime: rt,
		modules: make(map[string]convention.Derived),
		addr:    addr,
	}

	// Create auth handler
	c.authHandler = NewAuthHandler(rt)

	// Register auth routes (login, register, logout, me)
	c.router.Mount("/auth", c.authHandler.Routes())

	// Register schema introspection routes
	schemaHandler := NewSchemaHandler(c.modules)
	c.router.Mount("/_schema", schemaHandler.Routes())

	// Register OpenAPI endpoint
	c.router.Get("/_openapi", c.handleOpenAPI)
	c.router.Get("/_openapi.json", c.handleOpenAPI)

	// Mount Swagger UI at /swagger
	c.router.Get("/swagger", c.handleSwaggerUI)
	c.router.Get("/swagger/", c.handleSwaggerUI)

	// Mount Web UI at /ui (and root)
	webHandler := WebUIHandler()
	c.router.Route("/ui", func(r chi.Router) {
		r.Get("/*", webHandler.ServeHTTP)
	})
	// Also serve UI at root for clean URLs
	c.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
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

// AuthRoutes returns the auth router for mounting at additional paths.
// This enables SPA frontends to access auth endpoints at /api/portal/auth/*.
func (c *Channel) AuthRoutes() chi.Router {
	return c.authHandler.Routes()
}

// Register registers a module with the HTTP channel.
func (c *Channel) Register(mod convention.Derived) error {
	// Check if HTTP is enabled for this module
	if !mod.Source.Channels.HTTP.Serve.Enabled {
		return nil
	}

	c.modules[mod.Source.Name] = mod

	// Use configured base_path or derive from plural
	basePath := mod.Source.Channels.HTTP.Serve.BasePath
	if basePath == "" {
		basePath = "/" + mod.Plural
	}

	// Register from explicit endpoints if defined
	if len(mod.Source.Channels.HTTP.Serve.Endpoints) > 0 {
		c.registerExplicitEndpoints(mod, basePath)
	} else {
		// Fall back to implicit CRUD generation
		for _, action := range mod.Actions {
			c.registerActionRoute(mod, action, basePath)
		}
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

// registerExplicitEndpoints registers endpoints exactly as defined in module YAML.
func (c *Channel) registerExplicitEndpoints(mod convention.Derived, basePath string) {
	for _, ep := range mod.Source.Channels.HTTP.Serve.Endpoints {
		path := basePath + ep.Path
		handler := c.makeExplicitHandler(mod, ep.Action, ep.Auth)

		switch strings.ToUpper(ep.Method) {
		case "GET":
			c.router.Get(path, handler)
		case "POST":
			c.router.Post(path, handler)
		case "PUT":
			c.router.Put(path, handler)
		case "PATCH":
			c.router.Patch(path, handler)
		case "DELETE":
			c.router.Delete(path, handler)
		}
	}
}

// makeExplicitHandler creates a handler for an explicitly defined endpoint.
// It routes to the appropriate action based on the action name.
func (c *Channel) makeExplicitHandler(mod convention.Derived, actionName, auth string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// TODO: Check auth if specified (admin, user, public)
		// For now, all module endpoints require authentication to be handled
		// at a higher level (e.g., gateway auth middleware)

		// Find the action definition
		var action *convention.DerivedAction
		for i := range mod.Actions {
			if mod.Actions[i].Name == actionName {
				action = &mod.Actions[i]
				break
			}
		}

		// Route based on action type or name
		switch actionName {
		case "list":
			c.doList(ctx, w, r, mod)
		case "get":
			id := chi.URLParam(r, "id")
			if id == "" {
				id = chi.URLParam(r, "key") // For settings module
			}
			c.doGet(ctx, w, r, mod, id)
		case "create":
			c.doCreate(ctx, w, r, mod)
		case "update":
			id := chi.URLParam(r, "id")
			if id == "" {
				id = chi.URLParam(r, "key") // For settings module
			}
			c.doUpdate(ctx, w, r, mod, id)
		case "delete":
			id := chi.URLParam(r, "id")
			if id == "" {
				id = chi.URLParam(r, "key") // For settings module
			}
			c.doDelete(ctx, w, r, mod, id)
		default:
			// Custom action - handle based on action definition
			if action != nil {
				c.doExplicitAction(ctx, w, r, mod, action)
			} else {
				jsonapi.WriteNotFound(w, "action")
			}
		}
	}
}

// doExplicitAction handles custom actions defined in module YAML.
func (c *Channel) doExplicitAction(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, action *convention.DerivedAction) {
	// Extract input data from path params and query params
	data := make(map[string]any)

	// Extract path parameters
	rctx := chi.RouteContext(r.Context())
	if rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			if key != "" && i < len(rctx.URLParams.Values) {
				data[key] = rctx.URLParams.Values[i]
			}
		}
	}

	// Extract query parameters for action inputs
	for _, input := range action.Source.Input {
		if val := r.URL.Query().Get(input.Name); val != "" {
			data[input.Name] = val
		}
	}

	// Parse body if present
	if r.ContentLength > 0 {
		var bodyData map[string]any
		if err := json.NewDecoder(r.Body).Decode(&bodyData); err != nil {
			jsonapi.WriteBadRequest(w, "Invalid JSON body")
			return
		}
		for k, v := range bodyData {
			data[k] = v
		}
	}

	// Build input - lookup value for record-based actions
	lookup := ""
	if id := chi.URLParam(r, "id"); id != "" {
		lookup = id
	} else if key := chi.URLParam(r, "key"); key != "" {
		lookup = key
	} else if domain := chi.URLParam(r, "domain"); domain != "" {
		lookup = domain
	} else if prefix := chi.URLParam(r, "prefix"); prefix != "" {
		// For list_by_prefix style actions, prefix goes in data
		data["prefix"] = prefix
	}

	input := runtime.ActionInput{
		Data:         data,
		Lookup:       lookup,
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, action.Name, input)
	if err != nil {
		jsonapi.WriteBadRequest(w, err.Error())
		return
	}

	// Handle different result types
	if result.List != nil {
		// List result - return collection
		resources := make([]jsonapi.Resource, 0, len(result.List))
		for _, item := range result.List {
			id := ""
			if idVal, ok := item["id"]; ok {
				id = fmt.Sprintf("%v", idVal)
			}
			rb := jsonapi.NewResource(mod.Plural, id)
			for k, v := range item {
				if k != "id" {
					rb.Attr(k, v)
				}
			}
			resources = append(resources, rb.Build())
		}
		jsonapi.WriteCollection(w, http.StatusOK, resources, nil)
	} else if result.Data != nil {
		// Single record result
		id := result.ID
		if id == "" {
			if idVal, ok := result.Data["id"]; ok {
				id = fmt.Sprintf("%v", idVal)
			}
		}
		rb := jsonapi.NewResource(mod.Plural, id)
		for k, v := range result.Data {
			if k != "id" {
				rb.Attr(k, v)
			}
		}
		jsonapi.WriteResource(w, http.StatusOK, rb.Build())
	} else {
		// No data - return 204
		jsonapi.WriteNoContent(w)
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
		jsonapi.WriteInternalError(w, err.Error())
		return
	}

	// Convert results to JSON:API resources
	resources := make([]jsonapi.Resource, 0, len(result.List))
	for _, item := range result.List {
		id := ""
		if idVal, ok := item["id"]; ok {
			id = fmt.Sprintf("%v", idVal)
		}
		rb := jsonapi.NewResource(mod.Plural, id)
		for k, v := range item {
			if k != "id" {
				rb.Attr(k, v)
			}
		}
		resources = append(resources, rb.Build())
	}

	// Calculate page for pagination
	page := (offset / limit) + 1
	pagination := jsonapi.NewPagination(int64(result.Count), page, limit, r.URL.String())
	jsonapi.WriteCollection(w, http.StatusOK, resources, pagination)
}

// doGet handles get requests.
func (c *Channel) doGet(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	result, err := c.runtime.Execute(ctx, mod.Source.Name, "get", runtime.ActionInput{
		Lookup:   id,
		Channel:  "http",
		RemoteIP: r.RemoteAddr,
	})
	if err != nil {
		jsonapi.WriteNotFound(w, mod.Source.Name)
		return
	}

	// Convert to JSON:API resource
	rb := jsonapi.NewResource(mod.Plural, id)
	for k, v := range result.Data {
		if k != "id" {
			rb.Attr(k, v)
		}
	}
	jsonapi.WriteResource(w, http.StatusOK, rb.Build())
}

// doCreate handles create requests.
func (c *Channel) doCreate(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived) {
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "create", runtime.ActionInput{
		Data:         data,
		Channel:      "http",
		RemoteIP:     r.RemoteAddr,
		RequestBytes: r.ContentLength,
	})
	if err != nil {
		jsonapi.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to JSON:API resource
	rb := jsonapi.NewResource(mod.Plural, result.ID)
	for k, v := range result.Data {
		if k != "id" {
			rb.Attr(k, v)
		}
	}
	// Add meta if present (e.g., raw_key for API keys)
	for k, v := range result.Meta {
		rb.Meta(k, v)
	}
	jsonapi.WriteCreated(w, rb.Build(), "/"+mod.Plural+"/"+result.ID)
}

// doUpdate handles update requests.
func (c *Channel) doUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
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
		jsonapi.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to JSON:API resource
	rb := jsonapi.NewResource(mod.Plural, result.ID)
	for k, v := range result.Data {
		if k != "id" {
			rb.Attr(k, v)
		}
	}
	jsonapi.WriteResource(w, http.StatusOK, rb.Build())
}

// doDelete handles delete requests.
func (c *Channel) doDelete(ctx context.Context, w http.ResponseWriter, r *http.Request, mod convention.Derived, id string) {
	_, err := c.runtime.Execute(ctx, mod.Source.Name, "delete", runtime.ActionInput{
		Lookup:   id,
		Channel:  "http",
		RemoteIP: r.RemoteAddr,
	})
	if err != nil {
		jsonapi.WriteNotFound(w, mod.Source.Name)
		return
	}

	jsonapi.WriteNoContent(w)
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
		jsonapi.WriteNotFound(w, "action")
		return
	}

	var data map[string]any
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			jsonapi.WriteBadRequest(w, "Invalid JSON body")
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
		jsonapi.WriteBadRequest(w, err.Error())
		return
	}

	// Convert to JSON:API resource
	rb := jsonapi.NewResource(mod.Plural, result.ID)
	for k, v := range result.Data {
		if k != "id" {
			rb.Attr(k, v)
		}
	}
	jsonapi.WriteResource(w, http.StatusOK, rb.Build())
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
		jsonapi.WriteInternalError(w, err.Error())
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
