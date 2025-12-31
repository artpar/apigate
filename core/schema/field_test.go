package schema

import (
	"testing"
)

func TestFieldIsRequired(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected bool
	}{
		{
			name:     "nil Required (default false)",
			field:    Field{Type: FieldTypeString},
			expected: false,
		},
		{
			name:     "Required set to true",
			field:    Field{Type: FieldTypeString, Required: boolPtr(true)},
			expected: true,
		},
		{
			name:     "Required set to false",
			field:    Field{Type: FieldTypeString, Required: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.IsRequired(); got != tt.expected {
				t.Errorf("Field.IsRequired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFieldIsInternal(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected bool
	}{
		{
			name:     "not internal",
			field:    Field{Type: FieldTypeString},
			expected: false,
		},
		{
			name:     "internal flag set",
			field:    Field{Type: FieldTypeString, Internal: true},
			expected: true,
		},
		{
			name:     "secret type (implicitly internal)",
			field:    Field{Type: FieldTypeSecret},
			expected: true,
		},
		{
			name:     "secret type with internal flag",
			field:    Field{Type: FieldTypeSecret, Internal: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.IsInternal(); got != tt.expected {
				t.Errorf("Field.IsInternal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFieldSQLType(t *testing.T) {
	tests := []struct {
		fieldType FieldType
		expected  string
	}{
		{FieldTypeString, "TEXT"},
		{FieldTypeInt, "INTEGER"},
		{FieldTypeFloat, "REAL"},
		{FieldTypeBool, "INTEGER"},
		{FieldTypeTimestamp, "TEXT"},
		{FieldTypeDuration, "TEXT"},
		{FieldTypeJSON, "TEXT"},
		{FieldTypeBytes, "BLOB"},
		{FieldTypeEmail, "TEXT"},
		{FieldTypeURL, "TEXT"},
		{FieldTypeUUID, "TEXT"},
		{FieldTypeEnum, "TEXT"},
		{FieldTypeRef, "TEXT"},
		{FieldTypeSecret, "BLOB"},
		{FieldTypeStrings, "TEXT"},
		{FieldTypeInts, "TEXT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.fieldType), func(t *testing.T) {
			f := Field{Type: tt.fieldType}
			if got := f.SQLType(); got != tt.expected {
				t.Errorf("Field{Type: %s}.SQLType() = %q, want %q", tt.fieldType, got, tt.expected)
			}
		})
	}
}

func TestFieldTypeConstants(t *testing.T) {
	// Verify all field type constants are defined correctly
	tests := []struct {
		fieldType FieldType
		expected  string
	}{
		{FieldTypeString, "string"},
		{FieldTypeInt, "int"},
		{FieldTypeFloat, "float"},
		{FieldTypeBool, "bool"},
		{FieldTypeTimestamp, "timestamp"},
		{FieldTypeDuration, "duration"},
		{FieldTypeJSON, "json"},
		{FieldTypeBytes, "bytes"},
		{FieldTypeEmail, "email"},
		{FieldTypeURL, "url"},
		{FieldTypeUUID, "uuid"},
		{FieldTypeEnum, "enum"},
		{FieldTypeRef, "ref"},
		{FieldTypeSecret, "secret"},
		{FieldTypeStrings, "strings"},
		{FieldTypeInts, "ints"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.fieldType) != tt.expected {
				t.Errorf("FieldType constant = %q, want %q", tt.fieldType, tt.expected)
			}
		})
	}
}

func TestFieldStruct(t *testing.T) {
	// Test that Field struct can hold all expected fields
	f := Field{
		Type:        FieldTypeEnum,
		Unique:      true,
		Lookup:      true,
		Required:    boolPtr(true),
		Default:     "active",
		Values:      []string{"active", "inactive"},
		To:          "user",
		Format:      "email",
		Internal:    false,
		Computed:    "now()",
		Index:       true,
		Constraints: []Constraint{{Type: ConstraintMinLength, Value: 3}},
		Description: "Test field",
	}

	if f.Type != FieldTypeEnum {
		t.Error("Field.Type not set correctly")
	}
	if !f.Unique {
		t.Error("Field.Unique not set correctly")
	}
	if !f.Lookup {
		t.Error("Field.Lookup not set correctly")
	}
	if !f.IsRequired() {
		t.Error("Field.Required not set correctly")
	}
	if f.Default != "active" {
		t.Error("Field.Default not set correctly")
	}
	if len(f.Values) != 2 || f.Values[0] != "active" {
		t.Error("Field.Values not set correctly")
	}
	if f.To != "user" {
		t.Error("Field.To not set correctly")
	}
	if f.Format != "email" {
		t.Error("Field.Format not set correctly")
	}
	if f.Internal {
		t.Error("Field.Internal not set correctly")
	}
	if f.Computed != "now()" {
		t.Error("Field.Computed not set correctly")
	}
	if !f.Index {
		t.Error("Field.Index not set correctly")
	}
	if len(f.Constraints) != 1 || f.Constraints[0].Type != ConstraintMinLength {
		t.Error("Field.Constraints not set correctly")
	}
	if f.Description != "Test field" {
		t.Error("Field.Description not set correctly")
	}
}

func TestFieldDefaultValues(t *testing.T) {
	// Test field with various default value types
	tests := []struct {
		name         string
		defaultValue any
	}{
		{"string default", "hello"},
		{"int default", 42},
		{"float default", 3.14},
		{"bool default", true},
		{"nil default", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Field{Type: FieldTypeString, Default: tt.defaultValue}
			if f.Default != tt.defaultValue {
				t.Errorf("Field.Default = %v, want %v", f.Default, tt.defaultValue)
			}
		})
	}
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}
