package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/domain/entitlement"
)

// EntitlementStore implements ports.EntitlementStore with SQLite.
type EntitlementStore struct {
	db *DB
}

// NewEntitlementStore creates a new SQLite entitlement store.
func NewEntitlementStore(db *DB) *EntitlementStore {
	return &EntitlementStore{db: db}
}

// List returns all entitlements.
func (s *EntitlementStore) List(ctx context.Context) ([]entitlement.Entitlement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(category, 'feature'), COALESCE(value_type, 'boolean'),
		       COALESCE(default_value, 'true'), COALESCE(header_name, ''),
		       enabled, created_at, updated_at
		FROM entitlements
		ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntitlements(rows)
}

// ListEnabled returns only enabled entitlements.
func (s *EntitlementStore) ListEnabled(ctx context.Context) ([]entitlement.Entitlement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(category, 'feature'), COALESCE(value_type, 'boolean'),
		       COALESCE(default_value, 'true'), COALESCE(header_name, ''),
		       enabled, created_at, updated_at
		FROM entitlements
		WHERE enabled = 1
		ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntitlements(rows)
}

// Get retrieves an entitlement by ID.
func (s *EntitlementStore) Get(ctx context.Context, id string) (entitlement.Entitlement, error) {
	var e entitlement.Entitlement
	var category, valueType string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(category, 'feature'), COALESCE(value_type, 'boolean'),
		       COALESCE(default_value, 'true'), COALESCE(header_name, ''),
		       enabled, created_at, updated_at
		FROM entitlements WHERE id = ?
	`, id).Scan(
		&e.ID, &e.Name, &e.DisplayName, &e.Description,
		&category, &valueType,
		&e.DefaultValue, &e.HeaderName,
		&e.Enabled, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return e, sql.ErrNoRows
	}
	e.Category = entitlement.Category(category)
	e.ValueType = entitlement.ValueType(valueType)
	return e, err
}

// GetByName retrieves an entitlement by name.
func (s *EntitlementStore) GetByName(ctx context.Context, name string) (entitlement.Entitlement, error) {
	var e entitlement.Entitlement
	var category, valueType string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(category, 'feature'), COALESCE(value_type, 'boolean'),
		       COALESCE(default_value, 'true'), COALESCE(header_name, ''),
		       enabled, created_at, updated_at
		FROM entitlements WHERE name = ?
	`, name).Scan(
		&e.ID, &e.Name, &e.DisplayName, &e.Description,
		&category, &valueType,
		&e.DefaultValue, &e.HeaderName,
		&e.Enabled, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return e, sql.ErrNoRows
	}
	e.Category = entitlement.Category(category)
	e.ValueType = entitlement.ValueType(valueType)
	return e, err
}

// Create stores a new entitlement.
func (s *EntitlementStore) Create(ctx context.Context, e entitlement.Entitlement) error {
	now := time.Now()
	category := string(e.Category)
	if category == "" {
		category = "feature"
	}
	valueType := string(e.ValueType)
	if valueType == "" {
		valueType = "boolean"
	}
	defaultValue := e.DefaultValue
	if defaultValue == "" {
		defaultValue = "true"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entitlements (id, name, display_name, description, category,
		                         value_type, default_value, header_name, enabled,
		                         created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.Name, e.DisplayName, e.Description, category,
		valueType, defaultValue, e.HeaderName, e.Enabled,
		now, now)
	return err
}

// Update modifies an entitlement.
func (s *EntitlementStore) Update(ctx context.Context, e entitlement.Entitlement) error {
	category := string(e.Category)
	if category == "" {
		category = "feature"
	}
	valueType := string(e.ValueType)
	if valueType == "" {
		valueType = "boolean"
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE entitlements SET
			name = ?, display_name = ?, description = ?, category = ?,
			value_type = ?, default_value = ?, header_name = ?, enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, e.Name, e.DisplayName, e.Description, category,
		valueType, e.DefaultValue, e.HeaderName, e.Enabled, e.ID)
	return err
}

// Delete removes an entitlement.
func (s *EntitlementStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM entitlements WHERE id = ?", id)
	return err
}

// scanEntitlements scans rows into entitlement slice.
func scanEntitlements(rows *sql.Rows) ([]entitlement.Entitlement, error) {
	var entitlements []entitlement.Entitlement
	for rows.Next() {
		var e entitlement.Entitlement
		var category, valueType string
		if err := rows.Scan(
			&e.ID, &e.Name, &e.DisplayName, &e.Description,
			&category, &valueType,
			&e.DefaultValue, &e.HeaderName,
			&e.Enabled, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			continue
		}
		e.Category = entitlement.Category(category)
		e.ValueType = entitlement.ValueType(valueType)
		entitlements = append(entitlements, e)
	}
	return entitlements, rows.Err()
}
