package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// JSON:API resource type constants for routes
const (
	TypeRoute    = "routes"
	TypeUpstream = "upstreams"
)

// RoutesHandler handles route and upstream admin endpoints.
type RoutesHandler struct {
	routes        ports.RouteStore
	upstreams     ports.UpstreamStore
	logger        zerolog.Logger
	onRouteChange func() // Called when routes or upstreams change
}

// RoutesHandlerConfig holds configuration for the routes handler.
type RoutesHandlerConfig struct {
	Routes        ports.RouteStore
	Upstreams     ports.UpstreamStore
	Logger        zerolog.Logger
	OnRouteChange func() // Optional callback for cache invalidation
}

// NewRoutesHandler creates a new routes admin handler.
func NewRoutesHandler(routes ports.RouteStore, upstreams ports.UpstreamStore, logger zerolog.Logger) *RoutesHandler {
	return &RoutesHandler{
		routes:    routes,
		upstreams: upstreams,
		logger:    logger,
	}
}

// NewRoutesHandlerWithConfig creates a new routes admin handler with full configuration.
func NewRoutesHandlerWithConfig(cfg RoutesHandlerConfig) *RoutesHandler {
	return &RoutesHandler{
		routes:        cfg.Routes,
		upstreams:     cfg.Upstreams,
		logger:        cfg.Logger,
		onRouteChange: cfg.OnRouteChange,
	}
}

// notifyChange calls the route change callback if set.
func (h *RoutesHandler) notifyChange() {
	if h.onRouteChange != nil {
		h.onRouteChange()
	}
}

// RegisterRoutes adds route and upstream endpoints to the router.
func (h *RoutesHandler) RegisterRoutes(r chi.Router) {
	// Routes
	r.Get("/routes", h.ListRoutes)
	r.Post("/routes", h.CreateRoute)
	r.Get("/routes/{id}", h.GetRoute)
	r.Put("/routes/{id}", h.UpdateRoute)
	r.Patch("/routes/{id}", h.UpdateRoute)
	r.Delete("/routes/{id}", h.DeleteRoute)

	// Upstreams
	r.Get("/upstreams", h.ListUpstreams)
	r.Post("/upstreams", h.CreateUpstream)
	r.Get("/upstreams/{id}", h.GetUpstream)
	r.Put("/upstreams/{id}", h.UpdateUpstream)
	r.Patch("/upstreams/{id}", h.UpdateUpstream)
	r.Delete("/upstreams/{id}", h.DeleteUpstream)
}

// -----------------------------------------------------------------------------
// Route Types
// -----------------------------------------------------------------------------

// RouteResponse represents a route in API responses.
type RouteResponse struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	HostPattern       string           `json:"host_pattern,omitempty"`
	HostMatchType     string           `json:"host_match_type,omitempty"`
	PathPattern       string           `json:"path_pattern"`
	MatchType         string           `json:"match_type"`
	Methods           []string         `json:"methods,omitempty"`
	Headers           []HeaderMatchDTO `json:"headers,omitempty"`
	UpstreamID        string           `json:"upstream_id,omitempty"`
	PathRewrite       string           `json:"path_rewrite,omitempty"`
	MethodOverride    string           `json:"method_override,omitempty"`
	RequestTransform  *TransformDTO    `json:"request_transform,omitempty"`
	ResponseTransform *TransformDTO    `json:"response_transform,omitempty"`
	MeteringExpr      string           `json:"metering_expr,omitempty"`
	MeteringMode      string           `json:"metering_mode,omitempty"`
	Protocol          string           `json:"protocol"`
	AuthRequired      bool             `json:"auth_required"`
	Priority          int              `json:"priority"`
	Enabled           bool             `json:"enabled"`
	CreatedAt         string           `json:"created_at"`
	UpdatedAt         string           `json:"updated_at"`
}

// HeaderMatchDTO represents a header match condition.
type HeaderMatchDTO struct {
	Name     string `json:"name"`
	Value    string `json:"value,omitempty"`
	IsRegex  bool   `json:"is_regex,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// TransformDTO represents request/response transformation.
type TransformDTO struct {
	SetHeaders    map[string]string `json:"set_headers,omitempty"`
	DeleteHeaders []string          `json:"delete_headers,omitempty"`
	BodyExpr      string            `json:"body_expr,omitempty"`
	SetQuery      map[string]string `json:"set_query,omitempty"`
	DeleteQuery   []string          `json:"delete_query,omitempty"`
}

// CreateRouteRequest represents a request to create a route.
type CreateRouteRequest struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	HostPattern       string           `json:"host_pattern,omitempty"`
	HostMatchType     string           `json:"host_match_type,omitempty"`
	PathPattern       string           `json:"path_pattern"`
	MatchType         string           `json:"match_type,omitempty"`
	Methods           []string         `json:"methods,omitempty"`
	Headers           []HeaderMatchDTO `json:"headers,omitempty"`
	UpstreamID        string           `json:"upstream_id,omitempty"`
	PathRewrite       string           `json:"path_rewrite,omitempty"`
	MethodOverride    string           `json:"method_override,omitempty"`
	RequestTransform  *TransformDTO    `json:"request_transform,omitempty"`
	ResponseTransform *TransformDTO    `json:"response_transform,omitempty"`
	MeteringExpr      string           `json:"metering_expr,omitempty"`
	MeteringMode      string           `json:"metering_mode,omitempty"`
	Protocol          string           `json:"protocol,omitempty"`
	AuthRequired      *bool            `json:"auth_required,omitempty"`
	Priority          int              `json:"priority,omitempty"`
	Enabled           *bool            `json:"enabled,omitempty"`
}

// UpdateRouteRequest represents a request to update a route.
type UpdateRouteRequest struct {
	Name              *string          `json:"name,omitempty"`
	Description       *string          `json:"description,omitempty"`
	HostPattern       *string          `json:"host_pattern,omitempty"`
	HostMatchType     *string          `json:"host_match_type,omitempty"`
	PathPattern       *string          `json:"path_pattern,omitempty"`
	MatchType         *string          `json:"match_type,omitempty"`
	Methods           []string         `json:"methods,omitempty"`
	Headers           []HeaderMatchDTO `json:"headers,omitempty"`
	UpstreamID        *string          `json:"upstream_id,omitempty"`
	PathRewrite       *string          `json:"path_rewrite,omitempty"`
	MethodOverride    *string          `json:"method_override,omitempty"`
	RequestTransform  *TransformDTO    `json:"request_transform,omitempty"`
	ResponseTransform *TransformDTO    `json:"response_transform,omitempty"`
	MeteringExpr      *string          `json:"metering_expr,omitempty"`
	MeteringMode      *string          `json:"metering_mode,omitempty"`
	Protocol          *string          `json:"protocol,omitempty"`
	AuthRequired      *bool            `json:"auth_required,omitempty"`
	Priority          *int             `json:"priority,omitempty"`
	Enabled           *bool            `json:"enabled,omitempty"`
}

// -----------------------------------------------------------------------------
// Route Handlers
// -----------------------------------------------------------------------------

// ListRoutes returns all routes.
//
//	@Summary		List all routes
//	@Description	Returns a list of all configured proxy routes
//	@Tags			Routes
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]RouteResponse	"List of routes"
//	@Failure		500	{object}	ErrorResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/routes [get]
func (h *RoutesHandler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.routes.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list routes")
		jsonapi.WriteInternalError(w, "Failed to list routes")
		return
	}

	resources := make([]jsonapi.Resource, len(routes))
	for i, rt := range routes {
		resources[i] = routeToResource(rt)
	}

	jsonapi.WriteCollection(w, http.StatusOK, resources, nil)
}

// CreateRoute creates a new route.
//
//	@Summary		Create a new route
//	@Description	Creates a new proxy route configuration
//	@Tags			Routes
//	@Accept			json
//	@Produce		json
//	@Param			route	body		CreateRouteRequest	true	"Route configuration"
//	@Success		201		{object}	RouteResponse		"Created route"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Failure		500		{object}	ErrorResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/routes [post]
func (h *RoutesHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Name == "" {
		jsonapi.WriteValidationError(w, "name", "name is required")
		return
	}
	if req.PathPattern == "" {
		jsonapi.WriteValidationError(w, "path_pattern", "path_pattern is required")
		return
	}

	now := time.Now().UTC()
	rt := route.Route{
		ID:             generateRouteID(),
		Name:           req.Name,
		Description:    req.Description,
		HostPattern:    req.HostPattern,
		HostMatchType:  route.HostMatchType(req.HostMatchType),
		PathPattern:    req.PathPattern,
		MatchType:      route.MatchType(req.MatchType),
		Methods:        req.Methods,
		Headers:        dtoToHeaderMatches(req.Headers),
		UpstreamID:     req.UpstreamID,
		PathRewrite:    req.PathRewrite,
		MethodOverride: req.MethodOverride,
		MeteringExpr:   req.MeteringExpr,
		MeteringMode:   req.MeteringMode,
		Protocol:       route.Protocol(req.Protocol),
		AuthRequired:   true, // Default to requiring authentication
		Priority:       req.Priority,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.AuthRequired != nil {
		rt.AuthRequired = *req.AuthRequired
	}
	if req.Enabled != nil {
		rt.Enabled = *req.Enabled
	}
	if rt.MatchType == "" {
		rt.MatchType = route.MatchPrefix
	}
	if rt.Protocol == "" {
		rt.Protocol = route.ProtocolHTTP
	}
	if rt.MeteringExpr == "" {
		rt.MeteringExpr = "1"
	}

	if req.RequestTransform != nil {
		rt.RequestTransform = dtoToTransform(req.RequestTransform)
	}
	if req.ResponseTransform != nil {
		rt.ResponseTransform = dtoToTransform(req.ResponseTransform)
	}

	if err := h.routes.Create(r.Context(), rt); err != nil {
		h.logger.Error().Err(err).Msg("failed to create route")
		jsonapi.WriteInternalError(w, "Failed to create route")
		return
	}

	h.logger.Info().Str("route_id", rt.ID).Str("name", rt.Name).Msg("route created via admin api")
	h.notifyChange()
	jsonapi.WriteCreated(w, routeToResource(rt), "/admin/routes/"+rt.ID)
}

// GetRoute returns a single route.
//
//	@Summary		Get a route by ID
//	@Description	Returns a single route by its ID
//	@Tags			Routes
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string			true	"Route ID"
//	@Success		200	{object}	RouteResponse	"Route details"
//	@Failure		404	{object}	ErrorResponse	"Route not found"
//	@Security		BearerAuth
//	@Router			/admin/routes/{id} [get]
func (h *RoutesHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rt, err := h.routes.Get(r.Context(), id)
	if err != nil {
		jsonapi.WriteNotFound(w, "route")
		return
	}

	jsonapi.WriteResource(w, http.StatusOK, routeToResource(rt))
}

// UpdateRoute updates a route.
//
//	@Summary		Update a route
//	@Description	Updates an existing proxy route configuration
//	@Tags			Routes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Route ID"
//	@Param			route	body		UpdateRouteRequest	true	"Route update data"
//	@Success		200		{object}	RouteResponse		"Updated route"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Failure		404		{object}	ErrorResponse		"Route not found"
//	@Failure		500		{object}	ErrorResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/routes/{id} [put]
//	@Router			/admin/routes/{id} [patch]
func (h *RoutesHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rt, err := h.routes.Get(r.Context(), id)
	if err != nil {
		jsonapi.WriteNotFound(w, "route")
		return
	}

	var req UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Name != nil {
		rt.Name = *req.Name
	}
	if req.Description != nil {
		rt.Description = *req.Description
	}
	if req.HostPattern != nil {
		rt.HostPattern = *req.HostPattern
	}
	if req.HostMatchType != nil {
		rt.HostMatchType = route.HostMatchType(*req.HostMatchType)
	}
	if req.PathPattern != nil {
		rt.PathPattern = *req.PathPattern
	}
	if req.MatchType != nil {
		rt.MatchType = route.MatchType(*req.MatchType)
	}
	if req.Methods != nil {
		rt.Methods = req.Methods
	}
	if req.Headers != nil {
		rt.Headers = dtoToHeaderMatches(req.Headers)
	}
	if req.UpstreamID != nil {
		rt.UpstreamID = *req.UpstreamID
	}
	if req.PathRewrite != nil {
		rt.PathRewrite = *req.PathRewrite
	}
	if req.MethodOverride != nil {
		rt.MethodOverride = *req.MethodOverride
	}
	if req.RequestTransform != nil {
		rt.RequestTransform = dtoToTransform(req.RequestTransform)
	}
	if req.ResponseTransform != nil {
		rt.ResponseTransform = dtoToTransform(req.ResponseTransform)
	}
	if req.MeteringExpr != nil {
		rt.MeteringExpr = *req.MeteringExpr
	}
	if req.MeteringMode != nil {
		rt.MeteringMode = *req.MeteringMode
	}
	if req.Protocol != nil {
		rt.Protocol = route.Protocol(*req.Protocol)
	}
	if req.AuthRequired != nil {
		rt.AuthRequired = *req.AuthRequired
	}
	if req.Priority != nil {
		rt.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rt.Enabled = *req.Enabled
	}

	rt.UpdatedAt = time.Now().UTC()

	if err := h.routes.Update(r.Context(), rt); err != nil {
		h.logger.Error().Err(err).Msg("failed to update route")
		jsonapi.WriteInternalError(w, "Failed to update route")
		return
	}

	h.logger.Info().Str("route_id", rt.ID).Msg("route updated via admin api")
	h.notifyChange()
	jsonapi.WriteResource(w, http.StatusOK, routeToResource(rt))
}

// DeleteRoute deletes a route.
//
//	@Summary		Delete a route
//	@Description	Deletes a proxy route configuration
//	@Tags			Routes
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string			true	"Route ID"
//	@Success		200	{object}	map[string]string	"Deletion confirmation"
//	@Failure		404	{object}	ErrorResponse	"Route not found"
//	@Security		BearerAuth
//	@Router			/admin/routes/{id} [delete]
func (h *RoutesHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.routes.Delete(r.Context(), id); err != nil {
		jsonapi.WriteNotFound(w, "route")
		return
	}

	h.logger.Info().Str("route_id", id).Msg("route deleted via admin api")
	h.notifyChange()
	jsonapi.WriteNoContent(w)
}

// -----------------------------------------------------------------------------
// Upstream Types
// -----------------------------------------------------------------------------

// UpstreamResponse represents an upstream in API responses.
type UpstreamResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	BaseURL         string `json:"base_url"`
	TimeoutMs       int64  `json:"timeout_ms"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	IdleConnTimeout int64  `json:"idle_conn_timeout_ms"`
	AuthType        string `json:"auth_type"`
	AuthHeader      string `json:"auth_header,omitempty"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateUpstreamRequest represents a request to create an upstream.
type CreateUpstreamRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	BaseURL         string `json:"base_url"`
	TimeoutMs       int64  `json:"timeout_ms,omitempty"`
	MaxIdleConns    int    `json:"max_idle_conns,omitempty"`
	IdleConnTimeout int64  `json:"idle_conn_timeout_ms,omitempty"`
	AuthType        string `json:"auth_type,omitempty"`
	AuthHeader      string `json:"auth_header,omitempty"`
	AuthValue       string `json:"auth_value,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

// UpdateUpstreamRequest represents a request to update an upstream.
type UpdateUpstreamRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	TimeoutMs       *int64  `json:"timeout_ms,omitempty"`
	MaxIdleConns    *int    `json:"max_idle_conns,omitempty"`
	IdleConnTimeout *int64  `json:"idle_conn_timeout_ms,omitempty"`
	AuthType        *string `json:"auth_type,omitempty"`
	AuthHeader      *string `json:"auth_header,omitempty"`
	AuthValue       *string `json:"auth_value,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

// -----------------------------------------------------------------------------
// Upstream Handlers
// -----------------------------------------------------------------------------

// ListUpstreams returns all upstreams.
//
//	@Summary		List all upstreams
//	@Description	Returns a list of all configured upstream servers
//	@Tags			Upstreams
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]UpstreamResponse	"List of upstreams"
//	@Failure		500	{object}	ErrorResponse					"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/upstreams [get]
func (h *RoutesHandler) ListUpstreams(w http.ResponseWriter, r *http.Request) {
	upstreams, err := h.upstreams.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list upstreams")
		jsonapi.WriteInternalError(w, "Failed to list upstreams")
		return
	}

	resources := make([]jsonapi.Resource, len(upstreams))
	for i, u := range upstreams {
		resources[i] = upstreamToResource(u)
	}

	jsonapi.WriteCollection(w, http.StatusOK, resources, nil)
}

// CreateUpstream creates a new upstream.
//
//	@Summary		Create an upstream
//	@Description	Creates a new upstream server configuration
//	@Tags			Upstreams
//	@Accept			json
//	@Produce		json
//	@Param			upstream	body		CreateUpstreamRequest	true	"Upstream configuration"
//	@Success		201			{object}	UpstreamResponse		"Created upstream"
//	@Failure		400			{object}	ErrorResponse			"Invalid request"
//	@Failure		500			{object}	ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/upstreams [post]
func (h *RoutesHandler) CreateUpstream(w http.ResponseWriter, r *http.Request) {
	var req CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Name == "" {
		jsonapi.WriteValidationError(w, "name", "name is required")
		return
	}
	if req.BaseURL == "" {
		jsonapi.WriteValidationError(w, "base_url", "base_url is required")
		return
	}

	now := time.Now().UTC()
	u := route.Upstream{
		ID:              generateUpstreamID(),
		Name:            req.Name,
		Description:     req.Description,
		BaseURL:         req.BaseURL,
		Timeout:         time.Duration(req.TimeoutMs) * time.Millisecond,
		MaxIdleConns:    req.MaxIdleConns,
		IdleConnTimeout: time.Duration(req.IdleConnTimeout) * time.Millisecond,
		AuthType:        route.AuthType(req.AuthType),
		AuthHeader:      req.AuthHeader,
		AuthValue:       req.AuthValue,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if req.Enabled != nil {
		u.Enabled = *req.Enabled
	}
	if u.Timeout == 0 {
		u.Timeout = 30 * time.Second
	}
	if u.MaxIdleConns == 0 {
		u.MaxIdleConns = 100
	}
	if u.IdleConnTimeout == 0 {
		u.IdleConnTimeout = 90 * time.Second
	}
	if u.AuthType == "" {
		u.AuthType = route.AuthNone
	}

	if err := h.upstreams.Create(r.Context(), u); err != nil {
		h.logger.Error().Err(err).Msg("failed to create upstream")
		jsonapi.WriteInternalError(w, "Failed to create upstream")
		return
	}

	h.logger.Info().Str("upstream_id", u.ID).Str("name", u.Name).Msg("upstream created via admin api")
	h.notifyChange()
	jsonapi.WriteCreated(w, upstreamToResource(u), "/admin/upstreams/"+u.ID)
}

// GetUpstream returns a single upstream.
//
//	@Summary		Get an upstream
//	@Description	Returns a single upstream server configuration by ID
//	@Tags			Upstreams
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"Upstream ID"
//	@Success		200	{object}	UpstreamResponse	"Upstream details"
//	@Failure		404	{object}	ErrorResponse		"Upstream not found"
//	@Security		BearerAuth
//	@Router			/admin/upstreams/{id} [get]
func (h *RoutesHandler) GetUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	u, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		jsonapi.WriteNotFound(w, "upstream")
		return
	}

	jsonapi.WriteResource(w, http.StatusOK, upstreamToResource(u))
}

// UpdateUpstream updates an upstream.
//
//	@Summary		Update an upstream
//	@Description	Updates an existing upstream server configuration
//	@Tags			Upstreams
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Upstream ID"
//	@Param			upstream	body		UpdateUpstreamRequest	true	"Upstream update data"
//	@Success		200			{object}	UpstreamResponse		"Updated upstream"
//	@Failure		400			{object}	ErrorResponse			"Invalid request"
//	@Failure		404			{object}	ErrorResponse			"Upstream not found"
//	@Failure		500			{object}	ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/admin/upstreams/{id} [put]
//	@Router			/admin/upstreams/{id} [patch]
func (h *RoutesHandler) UpdateUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	u, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		jsonapi.WriteNotFound(w, "upstream")
		return
	}

	var req UpdateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body")
		return
	}

	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Description != nil {
		u.Description = *req.Description
	}
	if req.BaseURL != nil {
		u.BaseURL = *req.BaseURL
	}
	if req.TimeoutMs != nil {
		u.Timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
	}
	if req.MaxIdleConns != nil {
		u.MaxIdleConns = *req.MaxIdleConns
	}
	if req.IdleConnTimeout != nil {
		u.IdleConnTimeout = time.Duration(*req.IdleConnTimeout) * time.Millisecond
	}
	if req.AuthType != nil {
		u.AuthType = route.AuthType(*req.AuthType)
	}
	if req.AuthHeader != nil {
		u.AuthHeader = *req.AuthHeader
	}
	if req.AuthValue != nil {
		u.AuthValue = *req.AuthValue
	}
	if req.Enabled != nil {
		u.Enabled = *req.Enabled
	}

	u.UpdatedAt = time.Now().UTC()

	if err := h.upstreams.Update(r.Context(), u); err != nil {
		h.logger.Error().Err(err).Msg("failed to update upstream")
		jsonapi.WriteInternalError(w, "Failed to update upstream")
		return
	}

	h.logger.Info().Str("upstream_id", u.ID).Msg("upstream updated via admin api")
	h.notifyChange()
	jsonapi.WriteResource(w, http.StatusOK, upstreamToResource(u))
}

// DeleteUpstream deletes an upstream.
//
//	@Summary		Delete an upstream
//	@Description	Deletes an upstream server configuration
//	@Tags			Upstreams
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"Upstream ID"
//	@Success		200	{object}	map[string]string	"Deletion confirmation"
//	@Failure		404	{object}	ErrorResponse		"Upstream not found"
//	@Security		BearerAuth
//	@Router			/admin/upstreams/{id} [delete]
func (h *RoutesHandler) DeleteUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.upstreams.Delete(r.Context(), id); err != nil {
		jsonapi.WriteNotFound(w, "upstream")
		return
	}

	h.logger.Info().Str("upstream_id", id).Msg("upstream deleted via admin api")
	h.notifyChange()
	jsonapi.WriteNoContent(w)
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// routeToResource converts a route to a JSON:API Resource.
func routeToResource(rt route.Route) jsonapi.Resource {
	rb := jsonapi.NewResource(TypeRoute, rt.ID).
		Attr("name", rt.Name).
		Attr("description", rt.Description).
		Attr("host_pattern", rt.HostPattern).
		Attr("host_match_type", string(rt.HostMatchType)).
		Attr("path_pattern", rt.PathPattern).
		Attr("match_type", string(rt.MatchType)).
		Attr("methods", rt.Methods).
		Attr("headers", headerMatchesToDTO(rt.Headers)).
		Attr("path_rewrite", rt.PathRewrite).
		Attr("method_override", rt.MethodOverride).
		Attr("metering_expr", rt.MeteringExpr).
		Attr("metering_mode", rt.MeteringMode).
		Attr("protocol", string(rt.Protocol)).
		Attr("auth_required", rt.AuthRequired).
		Attr("priority", rt.Priority).
		Attr("enabled", rt.Enabled).
		Attr("created_at", rt.CreatedAt.Format(time.RFC3339)).
		Attr("updated_at", rt.UpdatedAt.Format(time.RFC3339))

	if rt.UpstreamID != "" {
		rb.BelongsTo("upstream", TypeUpstream, rt.UpstreamID)
	}
	if rt.RequestTransform != nil {
		rb.Attr("request_transform", transformToDTO(rt.RequestTransform))
	}
	if rt.ResponseTransform != nil {
		rb.Attr("response_transform", transformToDTO(rt.ResponseTransform))
	}

	return rb.Build()
}

// upstreamToResource converts an upstream to a JSON:API Resource.
func upstreamToResource(u route.Upstream) jsonapi.Resource {
	return jsonapi.NewResource(TypeUpstream, u.ID).
		Attr("name", u.Name).
		Attr("description", u.Description).
		Attr("base_url", u.BaseURL).
		Attr("timeout_ms", u.Timeout.Milliseconds()).
		Attr("max_idle_conns", u.MaxIdleConns).
		Attr("idle_conn_timeout_ms", u.IdleConnTimeout.Milliseconds()).
		Attr("auth_type", string(u.AuthType)).
		Attr("auth_header", u.AuthHeader).
		Attr("enabled", u.Enabled).
		Attr("created_at", u.CreatedAt.Format(time.RFC3339)).
		Attr("updated_at", u.UpdatedAt.Format(time.RFC3339)).
		Build()
}

func routeToResponse(rt route.Route) RouteResponse {
	resp := RouteResponse{
		ID:             rt.ID,
		Name:           rt.Name,
		Description:    rt.Description,
		HostPattern:    rt.HostPattern,
		HostMatchType:  string(rt.HostMatchType),
		PathPattern:    rt.PathPattern,
		MatchType:      string(rt.MatchType),
		Methods:        rt.Methods,
		Headers:        headerMatchesToDTO(rt.Headers),
		UpstreamID:     rt.UpstreamID,
		PathRewrite:    rt.PathRewrite,
		MethodOverride: rt.MethodOverride,
		MeteringExpr:   rt.MeteringExpr,
		MeteringMode:   rt.MeteringMode,
		Protocol:       string(rt.Protocol),
		Priority:       rt.Priority,
		Enabled:        rt.Enabled,
		CreatedAt:      rt.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      rt.UpdatedAt.Format(time.RFC3339),
	}

	if rt.RequestTransform != nil {
		resp.RequestTransform = transformToDTO(rt.RequestTransform)
	}
	if rt.ResponseTransform != nil {
		resp.ResponseTransform = transformToDTO(rt.ResponseTransform)
	}

	return resp
}

func upstreamToResponse(u route.Upstream) UpstreamResponse {
	return UpstreamResponse{
		ID:              u.ID,
		Name:            u.Name,
		Description:     u.Description,
		BaseURL:         u.BaseURL,
		TimeoutMs:       u.Timeout.Milliseconds(),
		MaxIdleConns:    u.MaxIdleConns,
		IdleConnTimeout: u.IdleConnTimeout.Milliseconds(),
		AuthType:        string(u.AuthType),
		AuthHeader:      u.AuthHeader,
		Enabled:         u.Enabled,
		CreatedAt:       u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       u.UpdatedAt.Format(time.RFC3339),
	}
}

func headerMatchesToDTO(headers []route.HeaderMatch) []HeaderMatchDTO {
	if headers == nil {
		return nil
	}
	result := make([]HeaderMatchDTO, len(headers))
	for i, h := range headers {
		result[i] = HeaderMatchDTO{
			Name:     h.Name,
			Value:    h.Value,
			IsRegex:  h.IsRegex,
			Required: h.Required,
		}
	}
	return result
}

func dtoToHeaderMatches(dto []HeaderMatchDTO) []route.HeaderMatch {
	if dto == nil {
		return nil
	}
	result := make([]route.HeaderMatch, len(dto))
	for i, h := range dto {
		result[i] = route.HeaderMatch{
			Name:     h.Name,
			Value:    h.Value,
			IsRegex:  h.IsRegex,
			Required: h.Required,
		}
	}
	return result
}

func transformToDTO(t *route.Transform) *TransformDTO {
	if t == nil {
		return nil
	}
	return &TransformDTO{
		SetHeaders:    t.SetHeaders,
		DeleteHeaders: t.DeleteHeaders,
		BodyExpr:      t.BodyExpr,
		SetQuery:      t.SetQuery,
		DeleteQuery:   t.DeleteQuery,
	}
}

func dtoToTransform(dto *TransformDTO) *route.Transform {
	if dto == nil {
		return nil
	}
	return &route.Transform{
		SetHeaders:    dto.SetHeaders,
		DeleteHeaders: dto.DeleteHeaders,
		BodyExpr:      dto.BodyExpr,
		SetQuery:      dto.SetQuery,
		DeleteQuery:   dto.DeleteQuery,
	}
}

func generateRouteID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "rt_" + hex.EncodeToString(b)
}

func generateUpstreamID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "up_" + hex.EncodeToString(b)
}
