package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/route"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RoutesPage displays the routes list page.
func (h *Handler) RoutesPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
	}{
		PageData: h.newPageData(r.Context(), "Routes"),
	}
	data.CurrentPath = "/routes"
	h.render(w, "routes", data)
}

// RouteNewPage displays the create route form.
func (h *Handler) RouteNewPage(w http.ResponseWriter, r *http.Request) {
	upstreams, _ := h.upstreams.List(r.Context())
	data := struct {
		PageData
		Route     *route.Route
		Upstreams []route.Upstream
		IsNew     bool
	}{
		PageData: h.newPageData(r.Context(), "New Route"),
		Route: &route.Route{
			MatchType:    "prefix",
			Methods:      []string{},
			Protocol:     "http",
			MeteringMode: "request",
			MeteringExpr: "1",
			Priority:     0,
			Enabled:      true,
		},
		Upstreams: upstreams,
		IsNew:     true,
	}
	data.CurrentPath = "/routes"
	h.render(w, "route_form", data)
}

// RouteCreate creates a new route.
func (h *Handler) RouteCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	rt := route.Route{
		ID:              uuid.New().String(),
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		ExampleRequest:  r.FormValue("example_request"),
		ExampleResponse: r.FormValue("example_response"),
		PathPattern:     r.FormValue("path_pattern"),
		MatchType:       route.MatchType(r.FormValue("match_type")),
		Methods:         parseCSV(r.FormValue("methods")),
		UpstreamID:      r.FormValue("upstream_id"),
		PathRewrite:     r.FormValue("path_rewrite"),
		MethodOverride:  r.FormValue("method_override"),
		MeteringExpr:    r.FormValue("metering_expr"),
		MeteringMode:    r.FormValue("metering_mode"),
		MeteringUnit:    r.FormValue("metering_unit"),
		Protocol:        route.Protocol(r.FormValue("protocol")),
		Priority:        parseInt(r.FormValue("priority")),
		Enabled:         r.FormValue("enabled") == "on",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Default metering unit if not provided
	if rt.MeteringUnit == "" {
		rt.MeteringUnit = "requests"
	}

	// Parse transforms
	rt.RequestTransform = parseTransform(r, "request_")
	rt.ResponseTransform = parseTransform(r, "response_")

	if err := h.routes.Create(r.Context(), rt); err != nil {
		http.Error(w, "Failed to create route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/routes", http.StatusFound)
}

// RouteEditPage displays the edit route form.
func (h *Handler) RouteEditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rt, err := h.routes.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	upstreams, _ := h.upstreams.List(r.Context())
	data := struct {
		PageData
		Route     *route.Route
		Upstreams []route.Upstream
		IsNew     bool
	}{
		PageData:  h.newPageData(r.Context(), "Edit Route"),
		Route:     &rt,
		Upstreams: upstreams,
		IsNew:     false,
	}
	data.CurrentPath = "/routes"
	h.render(w, "route_form", data)
}

// RouteUpdate updates an existing route.
func (h *Handler) RouteUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.routes.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	rt := route.Route{
		ID:              id,
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		ExampleRequest:  r.FormValue("example_request"),
		ExampleResponse: r.FormValue("example_response"),
		PathPattern:     r.FormValue("path_pattern"),
		MatchType:       route.MatchType(r.FormValue("match_type")),
		Methods:         parseCSV(r.FormValue("methods")),
		UpstreamID:      r.FormValue("upstream_id"),
		PathRewrite:     r.FormValue("path_rewrite"),
		MethodOverride:  r.FormValue("method_override"),
		MeteringExpr:    r.FormValue("metering_expr"),
		MeteringMode:    r.FormValue("metering_mode"),
		MeteringUnit:    r.FormValue("metering_unit"),
		Protocol:        route.Protocol(r.FormValue("protocol")),
		Priority:        parseInt(r.FormValue("priority")),
		Enabled:         r.FormValue("enabled") == "on",
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	// Default metering unit if not provided
	if rt.MeteringUnit == "" {
		rt.MeteringUnit = "requests"
	}

	// Parse transforms
	rt.RequestTransform = parseTransform(r, "request_")
	rt.ResponseTransform = parseTransform(r, "response_")

	if err := h.routes.Update(r.Context(), rt); err != nil {
		http.Error(w, "Failed to update route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/routes", http.StatusFound)
}

// RouteDelete deletes a route.
func (h *Handler) RouteDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.routes.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return updated table
	if r.Header.Get("HX-Request") == "true" {
		h.PartialRoutes(w, r)
		return
	}
	http.Redirect(w, r, "/routes", http.StatusFound)
}

// PartialRoutes returns the routes table partial for HTMX.
func (h *Handler) PartialRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.routes.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list routes")
		routes = []route.Route{}
	}

	// Build upstream map for display
	upstreamMap := make(map[string]string)
	upstreams, _ := h.upstreams.List(r.Context())
	for _, u := range upstreams {
		upstreamMap[u.ID] = u.Name
	}

	// Check documentation status for customer-facing docs
	var documentedCount, wildcardCount, totalEnabled int
	for _, rt := range routes {
		if !rt.Enabled {
			continue
		}
		totalEnabled++
		// Check if route is a wildcard (can't be meaningfully documented)
		if strings.Contains(rt.PathPattern, "*") || strings.Contains(rt.PathPattern, "{path}") {
			wildcardCount++
		} else if rt.Description != "" || rt.ExampleRequest != "" || rt.ExampleResponse != "" {
			documentedCount++
		}
	}

	data := struct {
		Routes          []route.Route
		UpstreamMap     map[string]string
		DocumentedCount int
		WildcardCount   int
		TotalEnabled    int
		ShowDocsWarning bool
	}{
		Routes:          routes,
		UpstreamMap:     upstreamMap,
		DocumentedCount: documentedCount,
		WildcardCount:   wildcardCount,
		TotalEnabled:    totalEnabled,
		ShowDocsWarning: totalEnabled > 0 && wildcardCount == totalEnabled,
	}
	h.renderPartial(w, "partial_routes", data)
}

// UpstreamsPage displays the upstreams list page.
func (h *Handler) UpstreamsPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
	}{
		PageData: h.newPageData(r.Context(), "Upstreams"),
	}
	data.CurrentPath = "/upstreams"
	h.render(w, "upstreams", data)
}

// UpstreamNewPage displays the create upstream form.
func (h *Handler) UpstreamNewPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		Upstream *route.Upstream
		IsNew    bool
	}{
		PageData: h.newPageData(r.Context(), "New Upstream"),
		Upstream: &route.Upstream{
			Timeout:         30 * time.Second,
			AuthType:        "none",
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
			Enabled:         true,
		},
		IsNew: true,
	}
	data.CurrentPath = "/upstreams"
	h.render(w, "upstream_form", data)
}

// UpstreamCreate creates a new upstream.
func (h *Handler) UpstreamCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	timeoutMs := parseInt(r.FormValue("timeout_ms"))
	if timeoutMs == 0 {
		timeoutMs = 30000
	}

	idleTimeoutMs := parseInt(r.FormValue("idle_conn_timeout_ms"))
	if idleTimeoutMs == 0 {
		idleTimeoutMs = 90000
	}

	maxIdleConns := parseInt(r.FormValue("max_idle_conns"))
	if maxIdleConns == 0 {
		maxIdleConns = 100
	}

	u := route.Upstream{
		ID:              uuid.New().String(),
		Name:            r.FormValue("name"),
		BaseURL:         r.FormValue("base_url"),
		Timeout:         time.Duration(timeoutMs) * time.Millisecond,
		AuthType:        route.AuthType(r.FormValue("auth_type")),
		AuthHeader:      r.FormValue("auth_header"),
		AuthValue:       r.FormValue("auth_value"),
		MaxIdleConns:    maxIdleConns,
		IdleConnTimeout: time.Duration(idleTimeoutMs) * time.Millisecond,
		Enabled:         r.FormValue("enabled") == "on",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := h.upstreams.Create(r.Context(), u); err != nil {
		http.Error(w, "Failed to create upstream", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/upstreams", http.StatusFound)
}

// UpstreamEditPage displays the edit upstream form.
func (h *Handler) UpstreamEditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Upstream not found", http.StatusNotFound)
		return
	}

	data := struct {
		PageData
		Upstream *route.Upstream
		IsNew    bool
	}{
		PageData: h.newPageData(r.Context(), "Edit Upstream"),
		Upstream: &u,
		IsNew:    false,
	}
	data.CurrentPath = "/upstreams"
	h.render(w, "upstream_form", data)
}

// UpstreamUpdate updates an existing upstream.
func (h *Handler) UpstreamUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Upstream not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	timeoutMs := parseInt(r.FormValue("timeout_ms"))
	if timeoutMs == 0 {
		timeoutMs = 30000
	}

	idleTimeoutMs := parseInt(r.FormValue("idle_conn_timeout_ms"))
	if idleTimeoutMs == 0 {
		idleTimeoutMs = 90000
	}

	maxIdleConns := parseInt(r.FormValue("max_idle_conns"))
	if maxIdleConns == 0 {
		maxIdleConns = 100
	}

	u := route.Upstream{
		ID:              id,
		Name:            r.FormValue("name"),
		BaseURL:         r.FormValue("base_url"),
		Timeout:         time.Duration(timeoutMs) * time.Millisecond,
		AuthType:        route.AuthType(r.FormValue("auth_type")),
		AuthHeader:      r.FormValue("auth_header"),
		AuthValue:       r.FormValue("auth_value"),
		MaxIdleConns:    maxIdleConns,
		IdleConnTimeout: time.Duration(idleTimeoutMs) * time.Millisecond,
		Enabled:         r.FormValue("enabled") == "on",
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       time.Now(),
	}

	if err := h.upstreams.Update(r.Context(), u); err != nil {
		http.Error(w, "Failed to update upstream", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/upstreams", http.StatusFound)
}

// UpstreamDelete deletes an upstream.
func (h *Handler) UpstreamDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.upstreams.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete upstream", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.PartialUpstreams(w, r)
		return
	}
	http.Redirect(w, r, "/upstreams", http.StatusFound)
}

// PartialUpstreams returns the upstreams table partial for HTMX.
func (h *Handler) PartialUpstreams(w http.ResponseWriter, r *http.Request) {
	upstreams, err := h.upstreams.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list upstreams")
		upstreams = []route.Upstream{}
	}

	data := struct {
		Upstreams []route.Upstream
	}{
		Upstreams: upstreams,
	}
	h.renderPartial(w, "partial_upstreams", data)
}

// Helper functions

func parseCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func parseTransform(r *http.Request, prefix string) *route.Transform {
	setHeaders := r.FormValue(prefix + "set_headers")
	deleteHeaders := r.FormValue(prefix + "delete_headers")
	bodyExpr := r.FormValue(prefix + "body_expr")
	setQuery := r.FormValue(prefix + "set_query")
	deleteQuery := r.FormValue(prefix + "delete_query")

	// If all empty, return nil
	if setHeaders == "" && deleteHeaders == "" && bodyExpr == "" && setQuery == "" && deleteQuery == "" {
		return nil
	}

	t := &route.Transform{
		DeleteHeaders: parseCSV(deleteHeaders),
		BodyExpr:      bodyExpr,
		DeleteQuery:   parseCSV(deleteQuery),
	}

	// Parse key=value pairs for SetHeaders
	if setHeaders != "" {
		t.SetHeaders = parseKeyValue(setHeaders)
	}

	// Parse key=value pairs for SetQuery
	if setQuery != "" {
		t.SetQuery = parseKeyValue(setQuery)
	}

	return t
}

func parseKeyValue(s string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}

// ValidateExpr validates an Expr expression via JSON API.
// POST /api/expr/validate
// Request: {"expression": "...", "context": "request|response|streaming"}
// Response: {"valid": true/false, "error": "...", "message": "..."}
func (h *Handler) ValidateExpr(w http.ResponseWriter, r *http.Request) {
	if h.exprValidator == nil {
		http.Error(w, "Expression validation not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Expression string `json:"expression"`
		Context    string `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"error": "Invalid JSON request",
		})
		return
	}

	// Default context to "request" if not specified
	if req.Context == "" {
		req.Context = "request"
	}

	result := h.exprValidator.ValidateExpr(req.Expression, req.Context)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// TestRoute tests route matching and transformation.
// POST /api/routes/test
// Request: {"method": "POST", "path": "/v1/chat", "headers": {...}, "body": "...", "route_id": "..."}
// Response: RouteTestResult JSON
func (h *Handler) TestRoute(w http.ResponseWriter, r *http.Request) {
	if h.routeTester == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(app.RouteTestResult{
			Error: "Route testing not available",
		})
		return
	}

	var req app.RouteTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(app.RouteTestResult{
			Error: "Invalid JSON request: " + err.Error(),
		})
		return
	}

	// Default method if not specified
	if req.Method == "" {
		req.Method = "GET"
	}

	result := h.routeTester.TestRoute(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
