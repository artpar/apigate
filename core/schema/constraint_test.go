package schema

import (
	"strings"
	"testing"
)

func TestConstraintError(t *testing.T) {
	err := ConstraintError{
		Field:      "email",
		Constraint: "pattern",
		Value:      "invalid-email",
		Message:    "must be a valid email",
	}

	expected := "email: must be a valid email"
	if got := err.Error(); got != expected {
		t.Errorf("ConstraintError.Error() = %q, want %q", got, expected)
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := ValidationResult{Valid: true}

	result.AddError("name", "min_length", "ab", "must be at least 3 characters")

	if result.Valid {
		t.Error("ValidationResult.Valid should be false after AddError")
	}
	if len(result.Errors) != 1 {
		t.Errorf("ValidationResult.Errors length = %d, want 1", len(result.Errors))
	}

	err := result.Errors[0]
	if err.Field != "name" {
		t.Errorf("Error.Field = %q, want %q", err.Field, "name")
	}
	if err.Constraint != "min_length" {
		t.Errorf("Error.Constraint = %q, want %q", err.Constraint, "min_length")
	}
	if err.Value != "ab" {
		t.Errorf("Error.Value = %v, want %q", err.Value, "ab")
	}
	if err.Message != "must be at least 3 characters" {
		t.Errorf("Error.Message = %q, want %q", err.Message, "must be at least 3 characters")
	}
}

func TestValidationResult_Error(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := ValidationResult{Valid: true}
		if got := result.Error(); got != "" {
			t.Errorf("ValidationResult.Error() = %q, want empty string", got)
		}
	})

	t.Run("invalid result with one error", func(t *testing.T) {
		result := ValidationResult{Valid: false}
		result.Errors = append(result.Errors, ConstraintError{
			Field:   "name",
			Message: "is required",
		})

		expected := "name: is required"
		if got := result.Error(); got != expected {
			t.Errorf("ValidationResult.Error() = %q, want %q", got, expected)
		}
	})

	t.Run("invalid result with multiple errors", func(t *testing.T) {
		result := ValidationResult{Valid: false}
		result.Errors = append(result.Errors,
			ConstraintError{Field: "name", Message: "is required"},
			ConstraintError{Field: "email", Message: "is invalid"},
		)

		got := result.Error()
		if !strings.Contains(got, "name: is required") {
			t.Errorf("ValidationResult.Error() missing 'name: is required', got %q", got)
		}
		if !strings.Contains(got, "email: is invalid") {
			t.Errorf("ValidationResult.Error() missing 'email: is invalid', got %q", got)
		}
		if !strings.Contains(got, "; ") {
			t.Errorf("ValidationResult.Error() should use '; ' separator, got %q", got)
		}
	})
}

func TestValidateConstraint_Min(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "int above min",
			value:      10,
			constraint: Constraint{Type: ConstraintMin, Value: 5},
			wantErr:    false,
		},
		{
			name:       "int equal to min",
			value:      5,
			constraint: Constraint{Type: ConstraintMin, Value: 5},
			wantErr:    false,
		},
		{
			name:       "int below min",
			value:      3,
			constraint: Constraint{Type: ConstraintMin, Value: 5},
			wantErr:    true,
		},
		{
			name:       "float64 above min",
			value:      10.5,
			constraint: Constraint{Type: ConstraintMin, Value: 5.0},
			wantErr:    false,
		},
		{
			name:       "float64 below min",
			value:      3.5,
			constraint: Constraint{Type: ConstraintMin, Value: 5.0},
			wantErr:    true,
		},
		{
			name:       "string value (non-numeric)",
			value:      "hello",
			constraint: Constraint{Type: ConstraintMin, Value: 5},
			wantErr:    false, // non-numeric values are skipped
		},
		{
			name:       "invalid constraint value",
			value:      10,
			constraint: Constraint{Type: ConstraintMin, Value: "invalid"},
			wantErr:    false, // invalid constraint config is skipped
		},
		{
			name:       "custom message",
			value:      3,
			constraint: Constraint{Type: ConstraintMin, Value: 5, Message: "must be at least 5"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.constraint.Message != "" && err.Message != tt.constraint.Message {
				t.Errorf("ValidateConstraint() message = %q, want %q", err.Message, tt.constraint.Message)
			}
		})
	}
}

func TestValidateConstraint_Max(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "int below max",
			value:      3,
			constraint: Constraint{Type: ConstraintMax, Value: 10},
			wantErr:    false,
		},
		{
			name:       "int equal to max",
			value:      10,
			constraint: Constraint{Type: ConstraintMax, Value: 10},
			wantErr:    false,
		},
		{
			name:       "int above max",
			value:      15,
			constraint: Constraint{Type: ConstraintMax, Value: 10},
			wantErr:    true,
		},
		{
			name:       "float64 below max",
			value:      5.5,
			constraint: Constraint{Type: ConstraintMax, Value: 10.0},
			wantErr:    false,
		},
		{
			name:       "float64 above max",
			value:      15.5,
			constraint: Constraint{Type: ConstraintMax, Value: 10.0},
			wantErr:    true,
		},
		{
			name:       "custom message",
			value:      15,
			constraint: Constraint{Type: ConstraintMax, Value: 10, Message: "must be at most 10"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_MinLength(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "string longer than min",
			value:      "hello",
			constraint: Constraint{Type: ConstraintMinLength, Value: 3},
			wantErr:    false,
		},
		{
			name:       "string equal to min",
			value:      "hey",
			constraint: Constraint{Type: ConstraintMinLength, Value: 3},
			wantErr:    false,
		},
		{
			name:       "string shorter than min",
			value:      "hi",
			constraint: Constraint{Type: ConstraintMinLength, Value: 3},
			wantErr:    true,
		},
		{
			name:       "empty string",
			value:      "",
			constraint: Constraint{Type: ConstraintMinLength, Value: 1},
			wantErr:    true,
		},
		{
			name:       "non-string value",
			value:      123,
			constraint: Constraint{Type: ConstraintMinLength, Value: 3},
			wantErr:    false, // non-string values are skipped
		},
		{
			name:       "custom message",
			value:      "hi",
			constraint: Constraint{Type: ConstraintMinLength, Value: 3, Message: "too short"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_MaxLength(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "string shorter than max",
			value:      "hi",
			constraint: Constraint{Type: ConstraintMaxLength, Value: 5},
			wantErr:    false,
		},
		{
			name:       "string equal to max",
			value:      "hello",
			constraint: Constraint{Type: ConstraintMaxLength, Value: 5},
			wantErr:    false,
		},
		{
			name:       "string longer than max",
			value:      "hello world",
			constraint: Constraint{Type: ConstraintMaxLength, Value: 5},
			wantErr:    true,
		},
		{
			name:       "custom message",
			value:      "hello world",
			constraint: Constraint{Type: ConstraintMaxLength, Value: 5, Message: "too long"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_Pattern(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "matches pattern",
			value:      "hello123",
			constraint: Constraint{Type: ConstraintPattern, Value: "^[a-z]+[0-9]+$"},
			wantErr:    false,
		},
		{
			name:       "does not match pattern",
			value:      "123hello",
			constraint: Constraint{Type: ConstraintPattern, Value: "^[a-z]+[0-9]+$"},
			wantErr:    true,
		},
		{
			name:       "email pattern",
			value:      "test@example.com",
			constraint: Constraint{Type: ConstraintPattern, Value: `^[^@]+@[^@]+\.[^@]+$`},
			wantErr:    false,
		},
		{
			name:       "invalid email pattern",
			value:      "invalid-email",
			constraint: Constraint{Type: ConstraintPattern, Value: `^[^@]+@[^@]+\.[^@]+$`},
			wantErr:    true,
		},
		{
			name:       "non-string value",
			value:      123,
			constraint: Constraint{Type: ConstraintPattern, Value: "^[0-9]+$"},
			wantErr:    false, // non-string values are skipped
		},
		{
			name:       "invalid regex",
			value:      "test",
			constraint: Constraint{Type: ConstraintPattern, Value: "[invalid(regex"},
			wantErr:    false, // invalid regex is skipped
		},
		{
			name:       "non-string pattern",
			value:      "test",
			constraint: Constraint{Type: ConstraintPattern, Value: 123},
			wantErr:    false, // non-string pattern is skipped
		},
		{
			name:       "custom message",
			value:      "123",
			constraint: Constraint{Type: ConstraintPattern, Value: "^[a-z]+$", Message: "letters only"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_NotEmpty(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "non-empty string",
			value:      "hello",
			constraint: Constraint{Type: ConstraintNotEmpty},
			wantErr:    false,
		},
		{
			name:       "empty string",
			value:      "",
			constraint: Constraint{Type: ConstraintNotEmpty},
			wantErr:    true,
		},
		{
			name:       "whitespace only",
			value:      "   ",
			constraint: Constraint{Type: ConstraintNotEmpty},
			wantErr:    true,
		},
		{
			name:       "tabs and newlines",
			value:      "\t\n  ",
			constraint: Constraint{Type: ConstraintNotEmpty},
			wantErr:    true,
		},
		{
			name:       "non-string value",
			value:      123,
			constraint: Constraint{Type: ConstraintNotEmpty},
			wantErr:    false, // non-string values are skipped
		},
		{
			name:       "custom message",
			value:      "",
			constraint: Constraint{Type: ConstraintNotEmpty, Message: "required field"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_OneOf(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		constraint Constraint
		wantErr    bool
	}{
		{
			name:       "value in list ([]any)",
			value:      "active",
			constraint: Constraint{Type: ConstraintOneOf, Value: []any{"active", "inactive", "pending"}},
			wantErr:    false,
		},
		{
			name:       "value not in list ([]any)",
			value:      "unknown",
			constraint: Constraint{Type: ConstraintOneOf, Value: []any{"active", "inactive", "pending"}},
			wantErr:    true,
		},
		{
			name:       "value in list ([]string)",
			value:      "active",
			constraint: Constraint{Type: ConstraintOneOf, Value: []string{"active", "inactive", "pending"}},
			wantErr:    false,
		},
		{
			name:       "value not in list ([]string)",
			value:      "unknown",
			constraint: Constraint{Type: ConstraintOneOf, Value: []string{"active", "inactive", "pending"}},
			wantErr:    true,
		},
		{
			name:       "integer value in list",
			value:      1,
			constraint: Constraint{Type: ConstraintOneOf, Value: []any{1, 2, 3}},
			wantErr:    false,
		},
		{
			name:       "integer value not in list",
			value:      5,
			constraint: Constraint{Type: ConstraintOneOf, Value: []any{1, 2, 3}},
			wantErr:    true,
		},
		{
			name:       "invalid constraint value",
			value:      "test",
			constraint: Constraint{Type: ConstraintOneOf, Value: "not a slice"},
			wantErr:    false, // invalid constraint config is skipped
		},
		{
			name:       "custom message",
			value:      "unknown",
			constraint: Constraint{Type: ConstraintOneOf, Value: []any{"a", "b"}, Message: "must be a or b"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConstraint("field", tt.value, tt.constraint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConstraint_RefExists(t *testing.T) {
	// RefExists is validated at runtime, not by ValidateConstraint
	err := ValidateConstraint("field", "some-id", Constraint{Type: ConstraintRefExists})
	if err != nil {
		t.Errorf("ValidateConstraint(RefExists) should return nil, got %v", err)
	}
}

func TestValidateConstraint_Unknown(t *testing.T) {
	// Unknown constraint types should be skipped
	err := ValidateConstraint("field", "value", Constraint{Type: "unknown_type"})
	if err != nil {
		t.Errorf("ValidateConstraint(unknown) should return nil, got %v", err)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    float64
		wantErr bool
	}{
		{"float64", float64(3.14), 3.14, false},
		{"float32", float32(3.14), float64(float32(3.14)), false},
		{"int", int(42), 42.0, false},
		{"int64", int64(42), 42.0, false},
		{"int32", int32(42), 42.0, false},
		{"string valid", "3.14", 3.14, false},
		{"string invalid", "not a number", 0, true},
		{"bool", true, 0, true},
		{"nil", nil, 0, true},
		{"slice", []int{1, 2, 3}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("toFloat64(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    int
		wantErr bool
	}{
		{"int", int(42), 42, false},
		{"int64", int64(42), 42, false},
		{"int32", int32(42), 42, false},
		{"float64", float64(42.9), 42, false}, // truncates
		{"string valid", "42", 42, false},
		{"string invalid", "not a number", 0, true},
		{"bool", true, 0, true},
		{"nil", nil, 0, true},
		{"slice", []int{1, 2, 3}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toInt(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("toInt(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("toInt(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestConstraintTypes(t *testing.T) {
	// Test that constraint type constants are defined correctly
	tests := []struct {
		constVal ConstraintType
		want     string
	}{
		{ConstraintMin, "min"},
		{ConstraintMax, "max"},
		{ConstraintMinLength, "min_length"},
		{ConstraintMaxLength, "max_length"},
		{ConstraintPattern, "pattern"},
		{ConstraintRefExists, "ref_exists"},
		{ConstraintNotEmpty, "not_empty"},
		{ConstraintOneOf, "one_of"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.constVal) != tt.want {
				t.Errorf("ConstraintType constant = %q, want %q", tt.constVal, tt.want)
			}
		})
	}
}

func TestConstraintStruct(t *testing.T) {
	// Test that Constraint struct can hold all expected fields
	c := Constraint{
		Type:    ConstraintMin,
		Value:   10,
		Message: "custom error message",
	}

	if c.Type != ConstraintMin {
		t.Error("Constraint.Type not set correctly")
	}
	if c.Value != 10 {
		t.Error("Constraint.Value not set correctly")
	}
	if c.Message != "custom error message" {
		t.Error("Constraint.Message not set correctly")
	}
}
