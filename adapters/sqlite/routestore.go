package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
)

// RouteStore implements ports.RouteStore using SQLite.
type RouteStore struct {
	db *DB
}

// NewRouteStore creates a new SQLite route store.
func NewRouteStore(db *DB) *RouteStore {
	return &RouteStore{db: db}
}

// Get retrieves a route by ID.
func (s *RouteStore) Get(ctx context.Context, id string) (route.Route, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, example_request, example_response,
		       host_pattern, host_match_type,
		       path_pattern, match_type, methods, headers,
		       upstream_id, path_rewrite, method_override,
		       request_transform, response_transform,
		       metering_expr, metering_mode, metering_unit, protocol,
		       auth_required, priority, enabled, created_at, updated_at
		FROM routes
		WHERE id = ?
	`, id)
	return scanRoute(row)
}

// List returns all routes ordered by priority.
func (s *RouteStore) List(ctx context.Context) ([]route.Route, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, example_request, example_response,
		       host_pattern, host_match_type,
		       path_pattern, match_type, methods, headers,
		       upstream_id, path_rewrite, method_override,
		       request_transform, response_transform,
		       metering_expr, metering_mode, metering_unit, protocol,
		       auth_required, priority, enabled, created_at, updated_at
		FROM routes
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []route.Route
	for rows.Next() {
		r, err := scanRouteRows(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

// ListEnabled returns only enabled routes ordered by priority.
func (s *RouteStore) ListEnabled(ctx context.Context) ([]route.Route, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, example_request, example_response,
		       host_pattern, host_match_type,
		       path_pattern, match_type, methods, headers,
		       upstream_id, path_rewrite, method_override,
		       request_transform, response_transform,
		       metering_expr, metering_mode, metering_unit, protocol,
		       auth_required, priority, enabled, created_at, updated_at
		FROM routes
		WHERE enabled = 1
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []route.Route
	for rows.Next() {
		r, err := scanRouteRows(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

// Create stores a new route.
func (s *RouteStore) Create(ctx context.Context, r route.Route) error {
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}

	methodsJSON, err := marshalStringSlice(r.Methods)
	if err != nil {
		return err
	}

	headersJSON, err := marshalHeaderMatches(r.Headers)
	if err != nil {
		return err
	}

	reqTransformJSON, err := marshalTransform(r.RequestTransform)
	if err != nil {
		return err
	}

	respTransformJSON, err := marshalTransform(r.ResponseTransform)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO routes (
			id, name, description, example_request, example_response,
			host_pattern, host_match_type,
			path_pattern, match_type, methods, headers,
			upstream_id, path_rewrite, method_override,
			request_transform, response_transform,
			metering_expr, metering_mode, metering_unit, protocol,
			auth_required, priority, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.ID, r.Name, r.Description, r.ExampleRequest, r.ExampleResponse,
		r.HostPattern, string(r.HostMatchType),
		r.PathPattern, string(r.MatchType),
		methodsJSON, headersJSON,
		r.UpstreamID, nullString(r.PathRewrite), nullString(r.MethodOverride),
		reqTransformJSON, respTransformJSON,
		r.MeteringExpr, r.MeteringMode, r.MeteringUnit, string(r.Protocol),
		boolToInt(r.AuthRequired), r.Priority, boolToInt(r.Enabled), r.CreatedAt, r.UpdatedAt,
	)

	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

// Update modifies an existing route.
func (s *RouteStore) Update(ctx context.Context, r route.Route) error {
	r.UpdatedAt = time.Now().UTC()

	methodsJSON, err := marshalStringSlice(r.Methods)
	if err != nil {
		return err
	}

	headersJSON, err := marshalHeaderMatches(r.Headers)
	if err != nil {
		return err
	}

	reqTransformJSON, err := marshalTransform(r.RequestTransform)
	if err != nil {
		return err
	}

	respTransformJSON, err := marshalTransform(r.ResponseTransform)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE routes
		SET name = ?, description = ?, example_request = ?, example_response = ?,
		    host_pattern = ?, host_match_type = ?,
		    path_pattern = ?, match_type = ?,
		    methods = ?, headers = ?,
		    upstream_id = ?, path_rewrite = ?, method_override = ?,
		    request_transform = ?, response_transform = ?,
		    metering_expr = ?, metering_mode = ?, metering_unit = ?, protocol = ?,
		    auth_required = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`,
		r.Name, r.Description, r.ExampleRequest, r.ExampleResponse,
		r.HostPattern, string(r.HostMatchType),
		r.PathPattern, string(r.MatchType),
		methodsJSON, headersJSON,
		r.UpstreamID, nullString(r.PathRewrite), nullString(r.MethodOverride),
		reqTransformJSON, respTransformJSON,
		r.MeteringExpr, r.MeteringMode, r.MeteringUnit, string(r.Protocol),
		boolToInt(r.AuthRequired), r.Priority, boolToInt(r.Enabled), r.UpdatedAt, r.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a route.
func (s *RouteStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM routes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRoute(row *sql.Row) (route.Route, error) {
	var r route.Route
	var hostMatchType, matchType, protocol string
	var methodsJSON, headersJSON sql.NullString
	var pathRewrite, methodOverride sql.NullString
	var reqTransformJSON, respTransformJSON sql.NullString
	var authRequired, enabled int

	err := row.Scan(
		&r.ID, &r.Name, &r.Description, &r.ExampleRequest, &r.ExampleResponse,
		&r.HostPattern, &hostMatchType,
		&r.PathPattern, &matchType,
		&methodsJSON, &headersJSON,
		&r.UpstreamID, &pathRewrite, &methodOverride,
		&reqTransformJSON, &respTransformJSON,
		&r.MeteringExpr, &r.MeteringMode, &r.MeteringUnit, &protocol,
		&authRequired, &r.Priority, &enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return route.Route{}, ErrNotFound
	}
	if err != nil {
		return route.Route{}, err
	}

	r.HostMatchType = route.HostMatchType(hostMatchType)
	r.MatchType = route.MatchType(matchType)
	r.Protocol = route.Protocol(protocol)
	r.AuthRequired = authRequired == 1
	r.Enabled = enabled == 1

	if pathRewrite.Valid {
		r.PathRewrite = pathRewrite.String
	}
	if methodOverride.Valid {
		r.MethodOverride = methodOverride.String
	}

	if methodsJSON.Valid && methodsJSON.String != "" {
		if err := json.Unmarshal([]byte(methodsJSON.String), &r.Methods); err != nil {
			return route.Route{}, err
		}
	}

	if headersJSON.Valid && headersJSON.String != "" {
		if err := json.Unmarshal([]byte(headersJSON.String), &r.Headers); err != nil {
			return route.Route{}, err
		}
	}

	if reqTransformJSON.Valid && reqTransformJSON.String != "" {
		var t route.Transform
		if err := json.Unmarshal([]byte(reqTransformJSON.String), &t); err != nil {
			return route.Route{}, err
		}
		r.RequestTransform = &t
	}

	if respTransformJSON.Valid && respTransformJSON.String != "" {
		var t route.Transform
		if err := json.Unmarshal([]byte(respTransformJSON.String), &t); err != nil {
			return route.Route{}, err
		}
		r.ResponseTransform = &t
	}

	return r, nil
}

func scanRouteRows(rows *sql.Rows) (route.Route, error) {
	var r route.Route
	var hostMatchType, matchType, protocol string
	var methodsJSON, headersJSON sql.NullString
	var pathRewrite, methodOverride sql.NullString
	var reqTransformJSON, respTransformJSON sql.NullString
	var authRequired, enabled int

	err := rows.Scan(
		&r.ID, &r.Name, &r.Description, &r.ExampleRequest, &r.ExampleResponse,
		&r.HostPattern, &hostMatchType,
		&r.PathPattern, &matchType,
		&methodsJSON, &headersJSON,
		&r.UpstreamID, &pathRewrite, &methodOverride,
		&reqTransformJSON, &respTransformJSON,
		&r.MeteringExpr, &r.MeteringMode, &r.MeteringUnit, &protocol,
		&authRequired, &r.Priority, &enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return route.Route{}, err
	}

	r.HostMatchType = route.HostMatchType(hostMatchType)
	r.MatchType = route.MatchType(matchType)
	r.Protocol = route.Protocol(protocol)
	r.AuthRequired = authRequired == 1
	r.Enabled = enabled == 1

	if pathRewrite.Valid {
		r.PathRewrite = pathRewrite.String
	}
	if methodOverride.Valid {
		r.MethodOverride = methodOverride.String
	}

	if methodsJSON.Valid && methodsJSON.String != "" {
		if err := json.Unmarshal([]byte(methodsJSON.String), &r.Methods); err != nil {
			return route.Route{}, err
		}
	}

	if headersJSON.Valid && headersJSON.String != "" {
		if err := json.Unmarshal([]byte(headersJSON.String), &r.Headers); err != nil {
			return route.Route{}, err
		}
	}

	if reqTransformJSON.Valid && reqTransformJSON.String != "" {
		var t route.Transform
		if err := json.Unmarshal([]byte(reqTransformJSON.String), &t); err != nil {
			return route.Route{}, err
		}
		r.RequestTransform = &t
	}

	if respTransformJSON.Valid && respTransformJSON.String != "" {
		var t route.Transform
		if err := json.Unmarshal([]byte(respTransformJSON.String), &t); err != nil {
			return route.Route{}, err
		}
		r.ResponseTransform = &t
	}

	return r, nil
}

func marshalStringSlice(s []string) (sql.NullString, error) {
	if len(s) == 0 {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

func marshalHeaderMatches(h []route.HeaderMatch) (sql.NullString, error) {
	if len(h) == 0 {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(h)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

func marshalTransform(t *route.Transform) (sql.NullString, error) {
	if t == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

// Ensure interface compliance.
var _ ports.RouteStore = (*RouteStore)(nil)
