package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
)

// UpstreamStore implements ports.UpstreamStore using SQLite.
type UpstreamStore struct {
	db *DB
}

// NewUpstreamStore creates a new SQLite upstream store.
func NewUpstreamStore(db *DB) *UpstreamStore {
	return &UpstreamStore{db: db}
}

// Get retrieves an upstream by ID.
func (s *UpstreamStore) Get(ctx context.Context, id string) (route.Upstream, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, base_url, timeout_ms, max_idle_conns, idle_conn_timeout_ms,
		       auth_type, auth_header, auth_value_encrypted, enabled, created_at, updated_at
		FROM upstreams
		WHERE id = ?
	`, id)
	return scanUpstream(row)
}

// List returns all upstreams.
func (s *UpstreamStore) List(ctx context.Context) ([]route.Upstream, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, base_url, timeout_ms, max_idle_conns, idle_conn_timeout_ms,
		       auth_type, auth_header, auth_value_encrypted, enabled, created_at, updated_at
		FROM upstreams
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var upstreams []route.Upstream
	for rows.Next() {
		u, err := scanUpstreamRows(rows)
		if err != nil {
			return nil, err
		}
		upstreams = append(upstreams, u)
	}
	return upstreams, rows.Err()
}

// ListEnabled returns only enabled upstreams.
func (s *UpstreamStore) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, base_url, timeout_ms, max_idle_conns, idle_conn_timeout_ms,
		       auth_type, auth_header, auth_value_encrypted, enabled, created_at, updated_at
		FROM upstreams
		WHERE enabled = 1
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var upstreams []route.Upstream
	for rows.Next() {
		u, err := scanUpstreamRows(rows)
		if err != nil {
			return nil, err
		}
		upstreams = append(upstreams, u)
	}
	return upstreams, rows.Err()
}

// Create stores a new upstream.
func (s *UpstreamStore) Create(ctx context.Context, u route.Upstream) error {
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO upstreams (
			id, name, description, base_url, timeout_ms, max_idle_conns, idle_conn_timeout_ms,
			auth_type, auth_header, auth_value_encrypted, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		u.ID, u.Name, u.Description, u.BaseURL,
		u.Timeout.Milliseconds(), u.MaxIdleConns, u.IdleConnTimeout.Milliseconds(),
		string(u.AuthType), nullString(u.AuthHeader), nullBytes([]byte(u.AuthValue)),
		boolToInt(u.Enabled), u.CreatedAt, u.UpdatedAt,
	)

	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

// Update modifies an existing upstream.
func (s *UpstreamStore) Update(ctx context.Context, u route.Upstream) error {
	u.UpdatedAt = time.Now().UTC()

	result, err := s.db.ExecContext(ctx, `
		UPDATE upstreams
		SET name = ?, description = ?, base_url = ?, timeout_ms = ?,
		    max_idle_conns = ?, idle_conn_timeout_ms = ?,
		    auth_type = ?, auth_header = ?, auth_value_encrypted = ?,
		    enabled = ?, updated_at = ?
		WHERE id = ?
	`,
		u.Name, u.Description, u.BaseURL,
		u.Timeout.Milliseconds(), u.MaxIdleConns, u.IdleConnTimeout.Milliseconds(),
		string(u.AuthType), nullString(u.AuthHeader), nullBytes([]byte(u.AuthValue)),
		boolToInt(u.Enabled), u.UpdatedAt, u.ID,
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

// Delete removes an upstream.
func (s *UpstreamStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM upstreams WHERE id = ?`, id)
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

func scanUpstream(row *sql.Row) (route.Upstream, error) {
	var u route.Upstream
	var timeoutMs, idleConnTimeoutMs int64
	var authType string
	var authHeader sql.NullString
	var authValue []byte
	var enabled int

	err := row.Scan(
		&u.ID, &u.Name, &u.Description, &u.BaseURL,
		&timeoutMs, &u.MaxIdleConns, &idleConnTimeoutMs,
		&authType, &authHeader, &authValue,
		&enabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return route.Upstream{}, ErrNotFound
	}
	if err != nil {
		return route.Upstream{}, err
	}

	u.Timeout = time.Duration(timeoutMs) * time.Millisecond
	u.IdleConnTimeout = time.Duration(idleConnTimeoutMs) * time.Millisecond
	u.AuthType = route.AuthType(authType)
	if authHeader.Valid {
		u.AuthHeader = authHeader.String
	}
	u.AuthValue = string(authValue)
	u.Enabled = enabled == 1

	return u, nil
}

func scanUpstreamRows(rows *sql.Rows) (route.Upstream, error) {
	var u route.Upstream
	var timeoutMs, idleConnTimeoutMs int64
	var authType string
	var authHeader sql.NullString
	var authValue []byte
	var enabled int

	err := rows.Scan(
		&u.ID, &u.Name, &u.Description, &u.BaseURL,
		&timeoutMs, &u.MaxIdleConns, &idleConnTimeoutMs,
		&authType, &authHeader, &authValue,
		&enabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return route.Upstream{}, err
	}

	u.Timeout = time.Duration(timeoutMs) * time.Millisecond
	u.IdleConnTimeout = time.Duration(idleConnTimeoutMs) * time.Millisecond
	u.AuthType = route.AuthType(authType)
	if authHeader.Valid {
		u.AuthHeader = authHeader.String
	}
	u.AuthValue = string(authValue)
	u.Enabled = enabled == 1

	return u, nil
}

func nullBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Ensure interface compliance.
var _ ports.UpstreamStore = (*UpstreamStore)(nil)
