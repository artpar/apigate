// Package storage provides a generic storage interface for declarative modules.
// It dynamically creates tables and performs CRUD operations based on module schemas.
package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Store provides generic CRUD operations for any module.
type Store interface {
	// CreateTable creates a table for a module.
	CreateTable(ctx context.Context, mod convention.Derived) error

	// Create inserts a new record.
	Create(ctx context.Context, module string, data map[string]any) (string, error)

	// Get retrieves a record by lookup field.
	Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error)

	// List retrieves multiple records.
	List(ctx context.Context, module string, opts ListOptions) ([]map[string]any, int64, error)

	// Update modifies an existing record.
	Update(ctx context.Context, module string, id string, data map[string]any) error

	// Delete removes a record.
	Delete(ctx context.Context, module string, id string) error

	// Close closes the storage connection.
	Close() error
}

// ListOptions configures list queries.
type ListOptions struct {
	// Limit is the maximum number of records to return.
	Limit int

	// Offset is the number of records to skip.
	Offset int

	// Filters are field-value pairs to filter by.
	Filters map[string]any

	// OrderBy is the field to sort by.
	OrderBy string

	// OrderDesc sorts in descending order.
	OrderDesc bool
}

// ColumnDef defines a database column.
type ColumnDef struct {
	Name       string
	Type       string
	PrimaryKey bool
	NotNull    bool
	Unique     bool
	Default    string
	ForeignKey string
}

// BuildCreateTableSQL generates CREATE TABLE SQL from a derived module.
func BuildCreateTableSQL(mod convention.Derived) string {
	var columns []string
	var constraints []string

	for _, f := range mod.Fields {
		col := buildColumnDef(f)
		columns = append(columns, col)

		if f.Unique && f.Name != "id" {
			constraints = append(constraints, fmt.Sprintf("UNIQUE(%s)", f.Name))
		}

		if f.Ref != "" {
			constraints = append(constraints, fmt.Sprintf(
				"FOREIGN KEY(%s) REFERENCES %s(id)",
				f.Name, convention.Pluralize(f.Ref),
			))
		}

		// Add CHECK constraints from field constraints
		checkConstraints := buildCheckConstraints(f)
		constraints = append(constraints, checkConstraints...)

		// Add enum CHECK constraint
		if f.Type == schema.FieldTypeEnum && len(f.Values) > 0 {
			values := make([]string, len(f.Values))
			for i, v := range f.Values {
				values[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
			}
			constraints = append(constraints, fmt.Sprintf(
				"CHECK(%s IN (%s))",
				f.Name, strings.Join(values, ", "),
			))
		}
	}

	sql := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n  %s",
		mod.Table,
		strings.Join(columns, ",\n  "),
	)

	if len(constraints) > 0 {
		sql += ",\n  " + strings.Join(constraints, ",\n  ")
	}

	sql += "\n)"

	return sql
}

// buildCheckConstraints generates CHECK constraints from field constraints.
func buildCheckConstraints(f convention.DerivedField) []string {
	var checks []string

	for _, c := range f.Constraints {
		switch c.Type {
		case schema.ConstraintMin:
			if v, ok := getNumericValue(c.Value); ok {
				checks = append(checks, fmt.Sprintf("CHECK(%s >= %v)", f.Name, v))
			}
		case schema.ConstraintMax:
			if v, ok := getNumericValue(c.Value); ok {
				checks = append(checks, fmt.Sprintf("CHECK(%s <= %v)", f.Name, v))
			}
		case schema.ConstraintMinLength:
			if v, ok := getNumericValue(c.Value); ok {
				checks = append(checks, fmt.Sprintf("CHECK(LENGTH(%s) >= %v)", f.Name, v))
			}
		case schema.ConstraintMaxLength:
			if v, ok := getNumericValue(c.Value); ok {
				checks = append(checks, fmt.Sprintf("CHECK(LENGTH(%s) <= %v)", f.Name, v))
			}
		case schema.ConstraintNotEmpty:
			checks = append(checks, fmt.Sprintf("CHECK(LENGTH(TRIM(%s)) > 0)", f.Name))
		case schema.ConstraintOneOf:
			if values, ok := c.Value.([]string); ok && len(values) > 0 {
				quotedValues := make([]string, len(values))
				for i, v := range values {
					quotedValues[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
				}
				checks = append(checks, fmt.Sprintf("CHECK(%s IN (%s))", f.Name, strings.Join(quotedValues, ", ")))
			}
		// Note: pattern constraints require regex support which SQLite doesn't have natively
		// We rely on application-level validation for patterns
		}
	}

	return checks
}

// getNumericValue extracts a numeric value from an interface.
func getNumericValue(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

// buildColumnDef builds a column definition from a derived field.
func buildColumnDef(f convention.DerivedField) string {
	var parts []string

	parts = append(parts, f.Name)
	parts = append(parts, f.SQLType)

	if f.Name == "id" {
		parts = append(parts, "PRIMARY KEY")
	}

	if f.Required {
		parts = append(parts, "NOT NULL")
	}

	if f.Default != nil {
		defaultVal := formatDefault(f.Default, f.Type)
		if defaultVal != "" {
			parts = append(parts, "DEFAULT "+defaultVal)
		}
	}

	// Special defaults for timestamps
	if f.Name == "created_at" {
		parts = append(parts, "DEFAULT CURRENT_TIMESTAMP")
	}
	if f.Name == "updated_at" {
		parts = append(parts, "DEFAULT CURRENT_TIMESTAMP")
	}

	return strings.Join(parts, " ")
}

// formatDefault formats a default value for SQL.
func formatDefault(val any, fieldType schema.FieldType) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return ""
	}
}

// BuildIndexSQL generates CREATE INDEX statements for lookup fields.
func BuildIndexSQL(mod convention.Derived) []string {
	var indexes []string

	for _, f := range mod.Fields {
		if f.Lookup && f.Name != "id" {
			idx := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s)",
				mod.Table, f.Name, mod.Table, f.Name,
			)
			indexes = append(indexes, idx)
		}
	}

	return indexes
}
