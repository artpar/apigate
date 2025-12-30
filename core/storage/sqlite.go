package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/artpar/apigate/core/convention"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex

	// modules maps module names to their derived definitions
	modules map[string]convention.Derived
}

// NewSQLiteStore creates a new SQLite storage.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA foreign_keys = ON",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("set pragma: %w", err)
		}
	}

	return &SQLiteStore{
		db:      db,
		modules: make(map[string]convention.Derived),
	}, nil
}

// NewSQLiteStoreFromDB creates a SQLite storage from an existing connection.
func NewSQLiteStoreFromDB(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		db:      db,
		modules: make(map[string]convention.Derived),
	}
}

// CreateTable creates a table for a module.
func (s *SQLiteStore) CreateTable(ctx context.Context, mod convention.Derived) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store module definition
	s.modules[mod.Source.Name] = mod

	// Create table
	createSQL := BuildCreateTableSQL(mod)
	if _, err := s.db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("create table %s: %w", mod.Table, err)
	}

	// Create indexes
	for _, indexSQL := range BuildIndexSQL(mod) {
		if _, err := s.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// Create inserts a new record.
func (s *SQLiteStore) Create(ctx context.Context, module string, data map[string]any) (string, error) {
	s.mu.RLock()
	mod, ok := s.modules[module]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("module %q not registered", module)
	}

	// Validate references before insert
	if err := s.validateReferences(ctx, mod, data); err != nil {
		return "", err
	}

	// Generate ID if not provided
	id, ok := data["id"].(string)
	if !ok || id == "" {
		id = uuid.New().String()
		data["id"] = id
	}

	// Build INSERT statement
	var columns []string
	var placeholders []string
	var values []any

	for _, f := range mod.Fields {
		if f.Name == "created_at" || f.Name == "updated_at" {
			continue // Let DB handle these
		}

		val, exists := data[f.Name]
		if !exists {
			if f.Default != nil {
				val = f.Default
			} else if f.Required {
				return "", fmt.Errorf("required field %q not provided", f.Name)
			} else {
				continue
			}
		}

		columns = append(columns, f.Name)
		placeholders = append(placeholders, "?")
		values = append(values, convertValue(val, f))
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		mod.Table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	if _, err := s.db.ExecContext(ctx, insertSQL, values...); err != nil {
		return "", fmt.Errorf("insert: %w", err)
	}

	return id, nil
}

// Get retrieves a record by lookup field.
func (s *SQLiteStore) Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error) {
	s.mu.RLock()
	mod, ok := s.modules[module]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("module %q not registered", module)
	}

	// Build column list
	var columns []string
	for _, f := range mod.Fields {
		columns = append(columns, f.Name)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = ?",
		strings.Join(columns, ", "),
		mod.Table,
		lookup,
	)

	row := s.db.QueryRowContext(ctx, query, value)

	// Scan into interface values
	values := make([]any, len(columns))
	scanDest := make([]any, len(columns))
	for i := range values {
		scanDest[i] = &values[i]
	}

	if err := row.Scan(scanDest...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Build result map
	result := make(map[string]any)
	for i, col := range columns {
		result[col] = convertFromDB(values[i], mod.Fields[i])
	}

	return result, nil
}

// List retrieves multiple records.
func (s *SQLiteStore) List(ctx context.Context, module string, opts ListOptions) ([]map[string]any, int64, error) {
	s.mu.RLock()
	mod, ok := s.modules[module]
	s.mu.RUnlock()

	if !ok {
		return nil, 0, fmt.Errorf("module %q not registered", module)
	}

	// Build column list
	var columns []string
	for _, f := range mod.Fields {
		columns = append(columns, f.Name)
	}

	// Build query
	var whereClause string
	var args []any

	if len(opts.Filters) > 0 {
		var conditions []string
		for k, v := range opts.Filters {
			conditions = append(conditions, k+" = ?")
			args = append(args, v)
		}
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get count
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", mod.Table, whereClause)
	var count int64
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	// Build main query
	querySQL := fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(columns, ", "), mod.Table, whereClause)

	// Add ordering - validate orderBy against actual field names to prevent SQL injection
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	} else {
		// Validate that orderBy is an actual field name
		validField := false
		for _, f := range mod.Fields {
			if f.Name == orderBy {
				validField = true
				break
			}
		}
		if !validField {
			orderBy = "created_at" // Fall back to safe default
		}
	}
	if opts.OrderDesc {
		querySQL += fmt.Sprintf(" ORDER BY %s DESC", orderBy)
	} else {
		querySQL += fmt.Sprintf(" ORDER BY %s ASC", orderBy)
	}

	// Add pagination
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	querySQL += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		scanDest := make([]any, len(columns))
		for i := range values {
			scanDest[i] = &values[i]
		}

		if err := rows.Scan(scanDest...); err != nil {
			continue
		}

		record := make(map[string]any)
		for i, col := range columns {
			record[col] = convertFromDB(values[i], mod.Fields[i])
		}
		results = append(results, record)
	}

	return results, count, nil
}

// Update modifies an existing record.
func (s *SQLiteStore) Update(ctx context.Context, module string, id string, data map[string]any) error {
	s.mu.RLock()
	mod, ok := s.modules[module]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("module %q not registered", module)
	}

	// Validate references before update
	if err := s.validateReferences(ctx, mod, data); err != nil {
		return err
	}

	// Build UPDATE statement
	var sets []string
	var values []any

	for k, v := range data {
		if k == "id" || k == "created_at" {
			continue
		}

		// Find field definition
		var field *convention.DerivedField
		for i := range mod.Fields {
			if mod.Fields[i].Name == k {
				field = &mod.Fields[i]
				break
			}
		}

		if field == nil {
			continue // Skip unknown fields
		}

		sets = append(sets, k+" = ?")
		values = append(values, convertValue(v, *field))
	}

	if len(sets) == 0 {
		return nil // Nothing to update
	}

	// Always update updated_at
	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	values = append(values, id)

	updateSQL := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = ?",
		mod.Table,
		strings.Join(sets, ", "),
	)

	result, err := s.db.ExecContext(ctx, updateSQL, values...)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("record not found: %s", id)
	}

	return nil
}

// Delete removes a record.
func (s *SQLiteStore) Delete(ctx context.Context, module string, id string) error {
	s.mu.RLock()
	mod, ok := s.modules[module]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("module %q not registered", module)
	}

	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE id = ?", mod.Table)

	result, err := s.db.ExecContext(ctx, deleteSQL, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("record not found: %s", id)
	}

	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// convertValue converts a Go value to a database value.
func convertValue(val any, f convention.DerivedField) any {
	if val == nil {
		return nil
	}

	switch f.Type {
	case "bool":
		switch v := val.(type) {
		case bool:
			if v {
				return 1
			}
			return 0
		case string:
			if v == "true" || v == "1" {
				return 1
			}
			return 0
		default:
			return 0
		}
	case "secret", "bytes":
		// Keep binary data as []byte for BLOB storage
		// If passed as string (legacy), convert to bytes
		if s, ok := val.(string); ok {
			return []byte(s)
		}
		return val
	default:
		return val
	}
}

// convertFromDB converts a database value to a Go value.
func convertFromDB(val any, f convention.DerivedField) any {
	if val == nil {
		return nil
	}

	switch f.Type {
	case "bool":
		switch v := val.(type) {
		case int64:
			return v != 0
		case int:
			return v != 0
		default:
			return false
		}
	case "secret", "bytes":
		// Keep binary data as []byte - don't convert to string
		// This is important for bcrypt hashes and other binary data
		if b, ok := val.([]byte); ok {
			return b
		}
		// If stored as string (legacy), convert back to bytes
		if s, ok := val.(string); ok {
			return []byte(s)
		}
		return val
	default:
		// Handle byte slices as strings for text fields
		if b, ok := val.([]byte); ok {
			return string(b)
		}
		return val
	}
}

// validateReferences checks that all referenced records exist.
func (s *SQLiteStore) validateReferences(ctx context.Context, mod convention.Derived, data map[string]any) error {
	for _, field := range mod.Fields {
		// Skip fields without references
		if field.Ref == "" {
			continue
		}

		// Get the reference value from data
		refValue, exists := data[field.Name]
		if !exists || refValue == nil {
			continue // No value provided, skip validation
		}

		refID, ok := refValue.(string)
		if !ok || refID == "" {
			continue // Empty or invalid reference, skip
		}

		// Get the referenced module
		refMod, ok := s.modules[field.Ref]
		if !ok {
			return fmt.Errorf("referenced module %q not registered for field %q", field.Ref, field.Name)
		}

		// Check if the referenced record exists
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", refMod.Table)
		if err := s.db.QueryRowContext(ctx, query, refID).Scan(&count); err != nil {
			return fmt.Errorf("check reference for field %q: %w", field.Name, err)
		}

		if count == 0 {
			return fmt.Errorf("referenced %s with id %q does not exist (field: %s)", field.Ref, refID, field.Name)
		}
	}

	return nil
}
