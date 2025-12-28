package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// RoutesHandler handles route and upstream admin endpoints.
type RoutesHandler struct {
	routes    ports.RouteStore
	upstreams ports.UpstreamStore
	logger    zerolog.Logger
}

// NewRoutesHandler creates a new routes admin handler.
func NewRoutesHandler(routes ports.RouteStore, upstreams ports.UpstreamStore, logger zerolog.Logger) *RoutesHandler {
	return &RoutesHandler{
		routes:    routes,
		upstreams: upstreams,
		logger:    logger,
	}
}

// RegisterRoutes adds route and upstream endpoints to the router.
func (h *RoutesHandler) RegisterRoutes(r chi.Router) {
	// Routes
	r.Get("/routes", h.ListRoutes)
	r.Post("/routes", h.CreateRoute)
	r.Get("/routes/{id}", h.GetRoute)
	r.Put("/routes/{id}", h.UpdateRoute)
	r.Delete("/routes/{id}", h.DeleteRoute)

	// Upstreams
	r.Get("/upstreams", h.ListUpstreams)
	r.Post("/upstreams", h.CreateUpstream)
	r.Get("/upstreams/{id}", h.GetUpstream)
	r.Put("/upstreams/{id}", h.UpdateUpstream)
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
	Priority          int              `json:"priority,omitempty"`
	Enabled           *bool            `json:"enabled,omitempty"`
}

// UpdateRouteRequest represents a request to update a route.
type UpdateRouteRequest struct {
	Name              *string          `json:"name,omitempty"`
	Description       *string          `json:"description,omitempty"`
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
	Priority          *int             `json:"priority,omitempty"`
	Enabled           *bool            `json:"enabled,omitempty"`
}

// -----------------------------------------------------------------------------
// Route Handlers
// -----------------------------------------------------------------------------

// ListRoutes returns all routes.
func (h *RoutesHandler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.routes.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list routes")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list routes")
		return
	}

	response := make([]RouteResponse, len(routes))
	for i, rt := range routes {
		response[i] = routeToResponse(rt)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"routes": response,
		"total":  len(response),
	})
}

// CreateRoute creates a new route.
func (h *RoutesHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_name", "name is required")
		return
	}
	if req.PathPattern == "" {
		writeError(w, http.StatusBadRequest, "missing_path_pattern", "path_pattern is required")
		return
	}

	now := time.Now().UTC()
	rt := route.Route{
		ID:             generateRouteID(),
		Name:           req.Name,
		Description:    req.Description,
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
		Priority:       req.Priority,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
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
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create route")
		return
	}

	h.logger.Info().Str("route_id", rt.ID).Str("name", rt.Name).Msg("route created via admin api")
	writeJSON(w, http.StatusCreated, routeToResponse(rt))
}

// GetRoute returns a single route.
func (h *RoutesHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rt, err := h.routes.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Route not found")
		return
	}

	writeJSON(w, http.StatusOK, routeToResponse(rt))
}

// UpdateRoute updates a route.
func (h *RoutesHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rt, err := h.routes.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Route not found")
		return
	}

	var req UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name != nil {
		rt.Name = *req.Name
	}
	if req.Description != nil {
		rt.Description = *req.Description
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
	if req.Priority != nil {
		rt.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rt.Enabled = *req.Enabled
	}

	rt.UpdatedAt = time.Now().UTC()

	if err := h.routes.Update(r.Context(), rt); err != nil {
		h.logger.Error().Err(err).Msg("failed to update route")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update route")
		return
	}

	h.logger.Info().Str("route_id", rt.ID).Msg("route updated via admin api")
	writeJSON(w, http.StatusOK, routeToResponse(rt))
}

// DeleteRoute deletes a route.
func (h *RoutesHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.routes.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Route not found")
		return
	}

	h.logger.Info().Str("route_id", id).Msg("route deleted via admin api")
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
func (h *RoutesHandler) ListUpstreams(w http.ResponseWriter, r *http.Request) {
	upstreams, err := h.upstreams.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list upstreams")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list upstreams")
		return
	}

	response := make([]UpstreamResponse, len(upstreams))
	for i, u := range upstreams {
		response[i] = upstreamToResponse(u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"upstreams": response,
		"total":     len(response),
	})
}

// CreateUpstream creates a new upstream.
func (h *RoutesHandler) CreateUpstream(w http.ResponseWriter, r *http.Request) {
	var req CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_name", "name is required")
		return
	}
	if req.BaseURL == "" {
		writeError(w, http.StatusBadRequest, "missing_base_url", "base_url is required")
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
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create upstream")
		return
	}

	h.logger.Info().Str("upstream_id", u.ID).Str("name", u.Name).Msg("upstream created via admin api")
	writeJSON(w, http.StatusCreated, upstreamToResponse(u))
}

// GetUpstream returns a single upstream.
func (h *RoutesHandler) GetUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	u, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
		return
	}

	writeJSON(w, http.StatusOK, upstreamToResponse(u))
}

// UpdateUpstream updates an upstream.
func (h *RoutesHandler) UpdateUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	u, err := h.upstreams.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
		return
	}

	var req UpdateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
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
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update upstream")
		return
	}

	h.logger.Info().Str("upstream_id", u.ID).Msg("upstream updated via admin api")
	writeJSON(w, http.StatusOK, upstreamToResponse(u))
}

// DeleteUpstream deletes an upstream.
func (h *RoutesHandler) DeleteUpstream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.upstreams.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
		return
	}

	h.logger.Info().Str("upstream_id", id).Msg("upstream deleted via admin api")
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func routeToResponse(rt route.Route) RouteResponse {
	resp := RouteResponse{
		ID:             rt.ID,
		Name:           rt.Name,
		Description:    rt.Description,
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
