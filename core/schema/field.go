package schema

// Field defines a data field in a module's schema.
type Field struct {
	// Type is the field type. See FieldType constants.
	Type FieldType `yaml:"type"`

	// Unique indicates this field must have unique values.
	Unique bool `yaml:"unique,omitempty"`

	// Lookup indicates this field can be used to find records (like email).
	// The "id" field is always implicitly a lookup field.
	Lookup bool `yaml:"lookup,omitempty"`

	// Required indicates this field must be provided on create.
	// Defaults to true for fields without a Default value.
	Required *bool `yaml:"required,omitempty"`

	// Default value for this field.
	Default any `yaml:"default,omitempty"`

	// Values lists valid values for enum type fields.
	Values []string `yaml:"values,omitempty"`

	// To specifies the target module for ref type fields.
	To string `yaml:"to,omitempty"`

	// Format specifies additional format constraints (e.g., "email", "url").
	Format string `yaml:"format,omitempty"`

	// Internal marks fields that are never exposed in APIs (like password_hash).
	Internal bool `yaml:"internal,omitempty"`

	// Computed indicates this field is derived, not stored.
	Computed string `yaml:"computed,omitempty"`

	// Index creates a database index on this field.
	Index bool `yaml:"index,omitempty"`

	// Constraints defines validation rules for this field.
	Constraints []Constraint `yaml:"constraints,omitempty"`
}

// FieldType represents the type of a schema field.
type FieldType string

const (
	// Primitive types
	FieldTypeString    FieldType = "string"
	FieldTypeInt       FieldType = "int"
	FieldTypeFloat     FieldType = "float"
	FieldTypeBool      FieldType = "bool"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeDuration  FieldType = "duration"
	FieldTypeJSON      FieldType = "json"
	FieldTypeBytes     FieldType = "bytes"

	// Semantic types (string with validation)
	FieldTypeEmail FieldType = "email"
	FieldTypeURL   FieldType = "url"
	FieldTypeUUID  FieldType = "uuid"

	// Special types
	FieldTypeEnum    FieldType = "enum"    // Requires Values
	FieldTypeRef     FieldType = "ref"     // Requires To (foreign key)
	FieldTypeSecret  FieldType = "secret"  // Hashed, never exposed
	FieldTypeStrings FieldType = "strings" // Array of strings
	FieldTypeInts    FieldType = "ints"    // Array of ints
)

// IsRequired returns whether the field is required.
// Fields are optional by default unless explicitly marked as required.
func (f Field) IsRequired() bool {
	if f.Required != nil {
		return *f.Required
	}
	return false
}

// IsInternal returns whether the field should be hidden from external APIs.
func (f Field) IsInternal() bool {
	return f.Internal || f.Type == FieldTypeSecret
}

// SQLType returns the SQLite column type for this field.
func (f Field) SQLType() string {
	switch f.Type {
	case FieldTypeInt, FieldTypeBool:
		return "INTEGER"
	case FieldTypeFloat:
		return "REAL"
	case FieldTypeBytes, FieldTypeSecret:
		return "BLOB"
	case FieldTypeJSON, FieldTypeStrings, FieldTypeInts:
		return "TEXT" // Stored as JSON
	default:
		return "TEXT"
	}
}
