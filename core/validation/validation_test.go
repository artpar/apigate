// Package validation provides comprehensive tests for field and action validation.
package validation

import (
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Helper function to create a boolean pointer.
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to create test modules.
func createTestModules() map[string]convention.Derived {
	return map[string]convention.Derived{
		"user": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "email", Type: schema.FieldTypeEmail, Required: true},
				{Name: "name", Type: schema.FieldTypeString, Required: true},
				{Name: "website", Type: schema.FieldTypeURL, Required: false},
				{Name: "age", Type: schema.FieldTypeInt, Required: false},
				{Name: "score", Type: schema.FieldTypeFloat, Required: false},
				{Name: "active", Type: schema.FieldTypeBool, Required: false},
				{Name: "status", Type: schema.FieldTypeEnum, Values: []string{"active", "inactive", "pending"}, Required: false},
				{Name: "uuid_field", Type: schema.FieldTypeUUID, Required: false},
				{Name: "created_at", Type: schema.FieldTypeTimestamp, Implicit: true},
				{Name: "updated_at", Type: schema.FieldTypeTimestamp, Implicit: true},
			},
		},
		"post": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "title", Type: schema.FieldTypeString, Required: true},
				{Name: "content", Type: schema.FieldTypeString, Required: false},
				{Name: "author_id", Type: schema.FieldTypeRef, Ref: "user", Required: true},
				{Name: "created_at", Type: schema.FieldTypeTimestamp, Implicit: true},
				{Name: "updated_at", Type: schema.FieldTypeTimestamp, Implicit: true},
			},
		},
		"product": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "name", Type: schema.FieldTypeString, Required: true},
				{Name: "price", Type: schema.FieldTypeFloat, Required: true, Constraints: []schema.Constraint{
					{Type: schema.ConstraintMin, Value: 0},
					{Type: schema.ConstraintMax, Value: 10000},
				}},
				{Name: "description", Type: schema.FieldTypeString, Required: false, Constraints: []schema.Constraint{
					{Type: schema.ConstraintMinLength, Value: 10},
					{Type: schema.ConstraintMaxLength, Value: 1000},
				}},
				{Name: "sku", Type: schema.FieldTypeString, Required: true, Constraints: []schema.Constraint{
					{Type: schema.ConstraintPattern, Value: "^[A-Z]{3}-[0-9]{4}$"},
				}},
				{Name: "category", Type: schema.FieldTypeString, Required: false, Constraints: []schema.Constraint{
					{Type: schema.ConstraintOneOf, Value: []any{"electronics", "clothing", "food"}},
				}},
				{Name: "notes", Type: schema.FieldTypeString, Required: false, Constraints: []schema.Constraint{
					{Type: schema.ConstraintNotEmpty},
				}},
				{Name: "created_at", Type: schema.FieldTypeTimestamp, Implicit: true},
				{Name: "updated_at", Type: schema.FieldTypeTimestamp, Implicit: true},
			},
		},
		"settings": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "theme", Type: schema.FieldTypeString, Required: false, Default: "light"},
				{Name: "language", Type: schema.FieldTypeString, Required: true, Default: "en"},
				{Name: "created_at", Type: schema.FieldTypeTimestamp, Implicit: true},
				{Name: "updated_at", Type: schema.FieldTypeTimestamp, Implicit: true},
			},
		},
	}
}

// TestNew tests the Validator constructor.
func TestNew(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	if v == nil {
		t.Fatal("expected non-nil validator")
	}

	if v.modules == nil {
		t.Fatal("expected modules to be set")
	}

	if len(v.modules) != len(modules) {
		t.Errorf("expected %d modules, got %d", len(modules), len(v.modules))
	}
}

// TestUpdateModules tests updating the module registry.
func TestUpdateModules(t *testing.T) {
	v := New(nil)

	if v.modules != nil {
		t.Error("expected nil modules initially")
	}

	modules := createTestModules()
	v.UpdateModules(modules)

	if len(v.modules) != len(modules) {
		t.Errorf("expected %d modules after update, got %d", len(modules), len(v.modules))
	}

	// Test updating with new modules
	newModules := map[string]convention.Derived{
		"new_module": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "field", Type: schema.FieldTypeString, Required: true},
			},
		},
	}
	v.UpdateModules(newModules)

	if len(v.modules) != 1 {
		t.Errorf("expected 1 module after second update, got %d", len(v.modules))
	}
}

// TestValidateCreate_UnknownModule tests validation with unknown module.
func TestValidateCreate_UnknownModule(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateCreate("unknown_module", map[string]any{
		"field": "value",
	})

	if result.Valid {
		t.Error("expected validation to fail for unknown module")
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Field != "_module" {
		t.Errorf("expected error field '_module', got '%s'", result.Errors[0].Field)
	}

	if result.Errors[0].Constraint != "unknown" {
		t.Errorf("expected constraint 'unknown', got '%s'", result.Errors[0].Constraint)
	}
}

// TestValidateCreate_UnknownFields tests detection of unknown fields.
func TestValidateCreate_UnknownFields(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateCreate("user", map[string]any{
		"email":         "test@example.com",
		"name":          "Test User",
		"unknown_field": "value",
		"another_bad":   123,
	})

	if result.Valid {
		t.Error("expected validation to fail for unknown fields")
	}

	unknownFieldErrors := 0
	for _, err := range result.Errors {
		if err.Constraint == "unknown_field" {
			unknownFieldErrors++
		}
	}

	if unknownFieldErrors != 2 {
		t.Errorf("expected 2 unknown field errors, got %d", unknownFieldErrors)
	}
}

// TestValidateCreate_RequiredFields tests required field validation.
func TestValidateCreate_RequiredFields(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	// Missing required fields
	result := v.ValidateCreate("user", map[string]any{})

	if result.Valid {
		t.Error("expected validation to fail for missing required fields")
	}

	requiredErrors := 0
	for _, err := range result.Errors {
		if err.Constraint == "required" {
			requiredErrors++
		}
	}

	if requiredErrors != 2 {
		t.Errorf("expected 2 required field errors (email, name), got %d", requiredErrors)
	}
}

// TestValidateCreate_RequiredFieldWithDefault tests that required fields with defaults pass.
func TestValidateCreate_RequiredFieldWithDefault(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	// 'language' is required but has a default
	result := v.ValidateCreate("settings", map[string]any{})

	// Should not have error for 'language' since it has default
	for _, err := range result.Errors {
		if err.Field == "language" && err.Constraint == "required" {
			t.Error("expected no required error for field with default")
		}
	}
}

// TestValidateCreate_ValidData tests successful validation.
func TestValidateCreate_ValidData(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateCreate("user", map[string]any{
		"email":      "test@example.com",
		"name":       "Test User",
		"website":    "https://example.com",
		"age":        30,
		"score":      95.5,
		"active":     true,
		"status":     "active",
		"uuid_field": "550e8400-e29b-41d4-a716-446655440000",
	})

	if !result.Valid {
		t.Errorf("expected validation to pass, got errors: %v", result.Errors)
	}
}

// TestValidateCreate_InvalidEmail tests email validation.
func TestValidateCreate_InvalidEmail(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with name", "Test User <test@example.com>", true},
		{"invalid - no @", "testexample.com", false},
		{"invalid - no domain", "test@", false},
		{"invalid - spaces", "test @example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email": tc.email,
				"name":  "Test User",
			})

			hasEmailError := false
			for _, err := range result.Errors {
				if err.Field == "email" && err.Constraint == "type" {
					hasEmailError = true
					break
				}
			}

			if tc.valid && hasEmailError {
				t.Errorf("expected valid email '%s' to pass", tc.email)
			}

			if !tc.valid && !hasEmailError {
				t.Errorf("expected invalid email '%s' to fail", tc.email)
			}
		})
	}
}

// TestValidateCreate_InvalidURL tests URL validation.
func TestValidateCreate_InvalidURL(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		url   string
		valid bool
	}{
		{"valid https", "https://example.com", true},
		{"valid http", "http://example.com/path", true},
		{"valid with port", "https://example.com:8080/path", true},
		{"valid - absolute path (request URI)", "/path/to/resource", true}, // ParseRequestURI accepts absolute paths
		{"invalid - no scheme", "example.com", false},
		{"invalid - empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email":   "test@example.com",
				"name":    "Test User",
				"website": tc.url,
			})

			hasURLError := false
			for _, err := range result.Errors {
				if err.Field == "website" && err.Constraint == "type" {
					hasURLError = true
					break
				}
			}

			if tc.valid && hasURLError {
				t.Errorf("expected valid URL '%s' to pass", tc.url)
			}

			if !tc.valid && !hasURLError {
				t.Errorf("expected invalid URL '%s' to fail", tc.url)
			}
		})
	}
}

// TestValidateCreate_InvalidUUID tests UUID validation.
func TestValidateCreate_InvalidUUID(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		uuid  string
		valid bool
	}{
		{"valid UUID lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid UUID mixed case", "550e8400-E29B-41d4-A716-446655440000", true},
		{"invalid - no hyphens", "550e8400e29b41d4a716446655440000", false},
		{"invalid - too short", "550e8400-e29b-41d4-a716", false},
		{"invalid - invalid chars", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"invalid - wrong format", "550e8400-e29b-41d4-a716446655440000", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email":      "test@example.com",
				"name":       "Test User",
				"uuid_field": tc.uuid,
			})

			hasUUIDError := false
			for _, err := range result.Errors {
				if err.Field == "uuid_field" && err.Constraint == "type" {
					hasUUIDError = true
					break
				}
			}

			if tc.valid && hasUUIDError {
				t.Errorf("expected valid UUID '%s' to pass", tc.uuid)
			}

			if !tc.valid && !hasUUIDError {
				t.Errorf("expected invalid UUID '%s' to fail", tc.uuid)
			}
		})
	}
}

// TestValidateCreate_InvalidEnum tests enum validation.
func TestValidateCreate_InvalidEnum(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name   string
		status string
		valid  bool
	}{
		{"valid - active", "active", true},
		{"valid - inactive", "inactive", true},
		{"valid - pending", "pending", true},
		{"invalid - unknown value", "unknown", false},
		{"invalid - case sensitive", "Active", false},
		{"invalid - empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email":  "test@example.com",
				"name":   "Test User",
				"status": tc.status,
			})

			hasEnumError := false
			for _, err := range result.Errors {
				if err.Field == "status" && err.Constraint == "enum" {
					hasEnumError = true
					break
				}
			}

			if tc.valid && hasEnumError {
				t.Errorf("expected valid enum value '%s' to pass", tc.status)
			}

			if !tc.valid && !hasEnumError {
				t.Errorf("expected invalid enum value '%s' to fail", tc.status)
			}
		})
	}
}

// TestValidateCreate_InvalidInt tests integer type validation.
func TestValidateCreate_InvalidInt(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		value any
		valid bool
	}{
		{"valid int", 30, true},
		{"valid int32", int32(30), true},
		{"valid int64", int64(30), true},
		{"valid float64 (JSON number)", float64(30), true},
		{"invalid - string", "thirty", false},
		{"invalid - bool", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
				"age":   tc.value,
			})

			hasIntError := false
			for _, err := range result.Errors {
				if err.Field == "age" && err.Constraint == "type" {
					hasIntError = true
					break
				}
			}

			if tc.valid && hasIntError {
				t.Errorf("expected valid int value '%v' to pass", tc.value)
			}

			if !tc.valid && !hasIntError {
				t.Errorf("expected invalid int value '%v' to fail", tc.value)
			}
		})
	}
}

// TestValidateCreate_InvalidFloat tests float type validation.
func TestValidateCreate_InvalidFloat(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		value any
		valid bool
	}{
		{"valid float64", 95.5, true},
		{"valid float32", float32(95.5), true},
		{"valid int", 95, true},
		{"valid int32", int32(95), true},
		{"valid int64", int64(95), true},
		{"invalid - string", "ninety-five", false},
		{"invalid - bool", true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
				"score": tc.value,
			})

			hasFloatError := false
			for _, err := range result.Errors {
				if err.Field == "score" && err.Constraint == "type" {
					hasFloatError = true
					break
				}
			}

			if tc.valid && hasFloatError {
				t.Errorf("expected valid float value '%v' to pass", tc.value)
			}

			if !tc.valid && !hasFloatError {
				t.Errorf("expected invalid float value '%v' to fail", tc.value)
			}
		})
	}
}

// TestValidateCreate_InvalidBool tests boolean type validation.
func TestValidateCreate_InvalidBool(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		value any
		valid bool
	}{
		{"valid true", true, true},
		{"valid false", false, true},
		{"invalid - string", "true", false},
		{"invalid - int", 1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("user", map[string]any{
				"email":  "test@example.com",
				"name":   "Test User",
				"active": tc.value,
			})

			hasBoolError := false
			for _, err := range result.Errors {
				if err.Field == "active" && err.Constraint == "type" {
					hasBoolError = true
					break
				}
			}

			if tc.valid && hasBoolError {
				t.Errorf("expected valid bool value '%v' to pass", tc.value)
			}

			if !tc.valid && !hasBoolError {
				t.Errorf("expected invalid bool value '%v' to fail", tc.value)
			}
		})
	}
}

// TestValidateCreate_RefField tests reference field validation.
func TestValidateCreate_RefField(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	testCases := []struct {
		name  string
		value any
		valid bool
	}{
		{"valid ref", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid ref - any string", "user-123", true},
		{"invalid - empty string", "", false},
		{"invalid - whitespace only", "   ", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.ValidateCreate("post", map[string]any{
				"title":     "Test Post",
				"author_id": tc.value,
			})

			hasRefError := false
			for _, err := range result.Errors {
				if err.Field == "author_id" && err.Constraint == "type" {
					hasRefError = true
					break
				}
			}

			if tc.valid && hasRefError {
				t.Errorf("expected valid ref value '%v' to pass", tc.value)
			}

			if !tc.valid && !hasRefError {
				t.Errorf("expected invalid ref value '%v' to fail", tc.value)
			}
		})
	}
}

// TestValidateCreate_NilValue tests that nil values skip validation.
func TestValidateCreate_NilValue(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateCreate("user", map[string]any{
		"email":   "test@example.com",
		"name":    "Test User",
		"website": nil,
		"age":     nil,
	})

	// nil values for optional fields should not cause errors
	if !result.Valid {
		for _, err := range result.Errors {
			if err.Field == "website" || err.Field == "age" {
				t.Errorf("nil value should not cause error for field '%s'", err.Field)
			}
		}
	}
}

// TestValidateCreate_Constraints tests constraint validation.
func TestValidateCreate_Constraints(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	t.Run("min constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":  "Test Product",
			"price": -5.0,
			"sku":   "ABC-1234",
		})

		hasMinError := false
		for _, err := range result.Errors {
			if err.Field == "price" && err.Constraint == "min" {
				hasMinError = true
				break
			}
		}

		if !hasMinError {
			t.Error("expected min constraint error for negative price")
		}
	})

	t.Run("max constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":  "Test Product",
			"price": 50000.0,
			"sku":   "ABC-1234",
		})

		hasMaxError := false
		for _, err := range result.Errors {
			if err.Field == "price" && err.Constraint == "max" {
				hasMaxError = true
				break
			}
		}

		if !hasMaxError {
			t.Error("expected max constraint error for price > 10000")
		}
	})

	t.Run("min_length constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":        "Test Product",
			"price":       50.0,
			"sku":         "ABC-1234",
			"description": "short",
		})

		hasMinLengthError := false
		for _, err := range result.Errors {
			if err.Field == "description" && err.Constraint == "min_length" {
				hasMinLengthError = true
				break
			}
		}

		if !hasMinLengthError {
			t.Error("expected min_length constraint error for short description")
		}
	})

	t.Run("max_length constraint", func(t *testing.T) {
		longDesc := ""
		for i := 0; i < 1100; i++ {
			longDesc += "a"
		}

		result := v.ValidateCreate("product", map[string]any{
			"name":        "Test Product",
			"price":       50.0,
			"sku":         "ABC-1234",
			"description": longDesc,
		})

		hasMaxLengthError := false
		for _, err := range result.Errors {
			if err.Field == "description" && err.Constraint == "max_length" {
				hasMaxLengthError = true
				break
			}
		}

		if !hasMaxLengthError {
			t.Error("expected max_length constraint error for long description")
		}
	})

	t.Run("pattern constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":  "Test Product",
			"price": 50.0,
			"sku":   "invalid-sku",
		})

		hasPatternError := false
		for _, err := range result.Errors {
			if err.Field == "sku" && err.Constraint == "pattern" {
				hasPatternError = true
				break
			}
		}

		if !hasPatternError {
			t.Error("expected pattern constraint error for invalid SKU")
		}
	})

	t.Run("one_of constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":     "Test Product",
			"price":    50.0,
			"sku":      "ABC-1234",
			"category": "toys",
		})

		hasOneOfError := false
		for _, err := range result.Errors {
			if err.Field == "category" && err.Constraint == "one_of" {
				hasOneOfError = true
				break
			}
		}

		if !hasOneOfError {
			t.Error("expected one_of constraint error for invalid category")
		}
	})

	t.Run("not_empty constraint", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":  "Test Product",
			"price": 50.0,
			"sku":   "ABC-1234",
			"notes": "   ",
		})

		hasNotEmptyError := false
		for _, err := range result.Errors {
			if err.Field == "notes" && err.Constraint == "not_empty" {
				hasNotEmptyError = true
				break
			}
		}

		if !hasNotEmptyError {
			t.Error("expected not_empty constraint error for whitespace-only notes")
		}
	})

	t.Run("valid constraints pass", func(t *testing.T) {
		result := v.ValidateCreate("product", map[string]any{
			"name":        "Test Product",
			"price":       99.99,
			"sku":         "ABC-1234",
			"description": "This is a valid product description with more than 10 characters.",
			"category":    "electronics",
			"notes":       "Some notes",
		})

		if !result.Valid {
			t.Errorf("expected validation to pass, got errors: %v", result.Errors)
		}
	})
}

// TestValidateUpdate_UnknownModule tests update validation with unknown module.
func TestValidateUpdate_UnknownModule(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("unknown_module", map[string]any{
		"field": "value",
	})

	if result.Valid {
		t.Error("expected validation to fail for unknown module")
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Field != "_module" {
		t.Errorf("expected error field '_module', got '%s'", result.Errors[0].Field)
	}
}

// TestValidateUpdate_UnknownFields tests update with unknown fields.
func TestValidateUpdate_UnknownFields(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{
		"name":          "Updated Name",
		"unknown_field": "value",
	})

	if result.Valid {
		t.Error("expected validation to fail for unknown fields")
	}

	hasUnknownFieldError := false
	for _, err := range result.Errors {
		if err.Constraint == "unknown_field" {
			hasUnknownFieldError = true
			break
		}
	}

	if !hasUnknownFieldError {
		t.Error("expected unknown_field error")
	}
}

// TestValidateUpdate_NoRequiredFields tests that update doesn't require all fields.
func TestValidateUpdate_NoRequiredFields(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	// Only update name, don't provide email
	result := v.ValidateUpdate("user", map[string]any{
		"name": "Updated Name",
	})

	if !result.Valid {
		t.Errorf("expected update validation to pass without all required fields, got errors: %v", result.Errors)
	}
}

// TestValidateUpdate_ValidData tests successful update validation.
func TestValidateUpdate_ValidData(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{
		"email":   "newemail@example.com",
		"name":    "New Name",
		"website": "https://newsite.com",
		"age":     35,
	})

	if !result.Valid {
		t.Errorf("expected validation to pass, got errors: %v", result.Errors)
	}
}

// TestValidateUpdate_InvalidFieldTypes tests update with invalid field types.
func TestValidateUpdate_InvalidFieldTypes(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{
		"email":  "invalid-email",
		"age":    "not a number",
		"active": "not a bool",
	})

	if result.Valid {
		t.Error("expected validation to fail for invalid types")
	}

	expectedErrors := map[string]bool{
		"email":  false,
		"age":    false,
		"active": false,
	}

	for _, err := range result.Errors {
		if _, ok := expectedErrors[err.Field]; ok {
			expectedErrors[err.Field] = true
		}
	}

	for field, found := range expectedErrors {
		if !found {
			t.Errorf("expected error for field '%s'", field)
		}
	}
}

// TestValidateUpdate_NilValues tests that nil values are skipped in update.
func TestValidateUpdate_NilValues(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{
		"email":   nil,
		"name":    nil,
		"website": nil,
	})

	// nil values should be skipped (used to clear fields)
	if !result.Valid {
		t.Errorf("expected validation to pass for nil values, got errors: %v", result.Errors)
	}
}

// TestValidateUpdate_ImplicitFieldsSkipped tests that implicit fields are skipped.
func TestValidateUpdate_ImplicitFieldsSkipped(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{
		"id":         "some-id",
		"created_at": "2024-01-01",
		"updated_at": "2024-01-02",
	})

	// Implicit fields should be skipped, not cause errors
	if !result.Valid {
		t.Errorf("expected validation to pass for implicit fields, got errors: %v", result.Errors)
	}
}

// TestValidateUpdate_Constraints tests constraint validation on update.
func TestValidateUpdate_Constraints(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("product", map[string]any{
		"price": -10.0,
	})

	if result.Valid {
		t.Error("expected validation to fail for constraint violation")
	}

	hasMinError := false
	for _, err := range result.Errors {
		if err.Field == "price" && err.Constraint == "min" {
			hasMinError = true
			break
		}
	}

	if !hasMinError {
		t.Error("expected min constraint error")
	}
}

// TestValidateField tests the standalone field validation function.
func TestValidateField(t *testing.T) {
	t.Run("nil value for required field without default", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "test_field",
			Type:     schema.FieldTypeString,
			Required: true,
			Default:  nil,
		}

		result := ValidateField(field, nil)

		if result.Valid {
			t.Error("expected validation to fail for nil required field without default")
		}

		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(result.Errors))
		}

		if result.Errors[0].Constraint != "required" {
			t.Errorf("expected 'required' constraint, got '%s'", result.Errors[0].Constraint)
		}
	})

	t.Run("nil value for required field with default", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "test_field",
			Type:     schema.FieldTypeString,
			Required: true,
			Default:  "default_value",
		}

		result := ValidateField(field, nil)

		if !result.Valid {
			t.Error("expected validation to pass for nil required field with default")
		}
	})

	t.Run("nil value for optional field", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "test_field",
			Type:     schema.FieldTypeString,
			Required: false,
		}

		result := ValidateField(field, nil)

		if !result.Valid {
			t.Error("expected validation to pass for nil optional field")
		}
	})

	t.Run("valid email", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "email",
			Type:     schema.FieldTypeEmail,
			Required: true,
		}

		result := ValidateField(field, "test@example.com")

		if !result.Valid {
			t.Errorf("expected validation to pass for valid email, got errors: %v", result.Errors)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "email",
			Type:     schema.FieldTypeEmail,
			Required: true,
		}

		result := ValidateField(field, "invalid-email")

		if result.Valid {
			t.Error("expected validation to fail for invalid email")
		}
	})

	t.Run("valid with constraints", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "score",
			Type:     schema.FieldTypeInt,
			Required: true,
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
				{Type: schema.ConstraintMax, Value: 100},
			},
		}

		result := ValidateField(field, 50)

		if !result.Valid {
			t.Errorf("expected validation to pass for valid score, got errors: %v", result.Errors)
		}
	})

	t.Run("invalid with constraints", func(t *testing.T) {
		field := convention.DerivedField{
			Name:     "score",
			Type:     schema.FieldTypeInt,
			Required: true,
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
				{Type: schema.ConstraintMax, Value: 100},
			},
		}

		result := ValidateField(field, 150)

		if result.Valid {
			t.Error("expected validation to fail for score > 100")
		}
	})
}

// TestIsValidUUID tests the UUID validation helper.
func TestIsValidUUID(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"ffffffff-ffff-ffff-ffff-ffffffffffff", true},
		{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", true},
		{"", false},
		{"not-a-uuid", false},
		{"550e8400e29b41d4a716446655440000", false},
		{"550e8400-e29b-41d4-a716-44665544000", false},
		{"550e8400-e29b-41d4-a716-4466554400000", false},
		{"550e8400-e29b-41d4-a716-44665544000g", false},
		{"g50e8400-e29b-41d4-a716-446655440000", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := isValidUUID(tc.input)
			if result != tc.expected {
				t.Errorf("isValidUUID(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestContainsString tests the string slice contains helper.
func TestContainsString(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{"found at start", []string{"a", "b", "c"}, "a", true},
		{"found in middle", []string{"a", "b", "c"}, "b", true},
		{"found at end", []string{"a", "b", "c"}, "c", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"empty string found", []string{"", "a"}, "", true},
		{"empty string not found", []string{"a", "b"}, "", false},
		{"case sensitive", []string{"a", "b", "c"}, "A", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsString(tc.slice, tc.s)
			if result != tc.expected {
				t.Errorf("containsString(%v, %q) = %v, expected %v", tc.slice, tc.s, result, tc.expected)
			}
		})
	}
}

// TestValidationResult_Error tests the Error method of ValidationResult.
func TestValidationResult_Error(t *testing.T) {
	t.Run("valid result returns empty string", func(t *testing.T) {
		result := schema.ValidationResult{Valid: true}
		if result.Error() != "" {
			t.Errorf("expected empty string for valid result, got '%s'", result.Error())
		}
	})

	t.Run("invalid result returns combined errors", func(t *testing.T) {
		result := schema.ValidationResult{Valid: false}
		result.AddError("email", "required", nil, "field is required")
		result.AddError("name", "min_length", "ab", "must be at least 3 characters")

		errorStr := result.Error()
		if errorStr == "" {
			t.Error("expected non-empty error string")
		}

		// Should contain both error messages
		if !contains(errorStr, "email") {
			t.Error("expected error string to contain 'email'")
		}

		if !contains(errorStr, "name") {
			t.Error("expected error string to contain 'name'")
		}
	})
}

// TestMultipleConstraintErrors tests that multiple constraint errors are collected.
func TestMultipleConstraintErrors(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateCreate("product", map[string]any{
		"name":        "Test",
		"price":       -100.0,
		"sku":         "invalid",
		"description": "short",
		"category":    "invalid",
		"notes":       "   ",
	})

	if result.Valid {
		t.Error("expected validation to fail with multiple errors")
	}

	// Should have errors for: price (min), sku (pattern), description (min_length),
	// category (one_of), notes (not_empty)
	if len(result.Errors) < 5 {
		t.Errorf("expected at least 5 errors, got %d", len(result.Errors))
	}
}

// TestNonStringValueForStringTypeField tests non-string values for email/url/uuid fields.
func TestNonStringValueForStringTypeField(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	t.Run("non-string for email field", func(t *testing.T) {
		result := v.ValidateCreate("user", map[string]any{
			"email": 123,
			"name":  "Test User",
		})

		// Non-string value for email should pass type check (no error added)
		// because the validateFieldType only validates if value is string
		hasEmailTypeError := false
		for _, err := range result.Errors {
			if err.Field == "email" && err.Constraint == "type" {
				hasEmailTypeError = true
				break
			}
		}

		if hasEmailTypeError {
			t.Error("non-string value for email should not produce type error (skipped)")
		}
	})

	t.Run("non-string for URL field", func(t *testing.T) {
		result := v.ValidateCreate("user", map[string]any{
			"email":   "test@example.com",
			"name":    "Test User",
			"website": 123,
		})

		hasURLTypeError := false
		for _, err := range result.Errors {
			if err.Field == "website" && err.Constraint == "type" {
				hasURLTypeError = true
				break
			}
		}

		if hasURLTypeError {
			t.Error("non-string value for URL should not produce type error (skipped)")
		}
	})

	t.Run("non-string for UUID field", func(t *testing.T) {
		result := v.ValidateCreate("user", map[string]any{
			"email":      "test@example.com",
			"name":       "Test User",
			"uuid_field": 123,
		})

		hasUUIDTypeError := false
		for _, err := range result.Errors {
			if err.Field == "uuid_field" && err.Constraint == "type" {
				hasUUIDTypeError = true
				break
			}
		}

		if hasUUIDTypeError {
			t.Error("non-string value for UUID should not produce type error (skipped)")
		}
	})

	t.Run("non-string for enum field", func(t *testing.T) {
		result := v.ValidateCreate("user", map[string]any{
			"email":  "test@example.com",
			"name":   "Test User",
			"status": 123,
		})

		hasEnumError := false
		for _, err := range result.Errors {
			if err.Field == "status" && err.Constraint == "enum" {
				hasEnumError = true
				break
			}
		}

		if hasEnumError {
			t.Error("non-string value for enum should not produce enum error (skipped)")
		}
	})

	t.Run("non-string for ref field", func(t *testing.T) {
		result := v.ValidateCreate("post", map[string]any{
			"title":     "Test Post",
			"author_id": 123,
		})

		hasRefTypeError := false
		for _, err := range result.Errors {
			if err.Field == "author_id" && err.Constraint == "type" {
				hasRefTypeError = true
				break
			}
		}

		if hasRefTypeError {
			t.Error("non-string value for ref should not produce type error (skipped)")
		}
	})
}

// TestEmptyModules tests validator with empty modules map.
func TestEmptyModules(t *testing.T) {
	v := New(map[string]convention.Derived{})

	result := v.ValidateCreate("any_module", map[string]any{
		"field": "value",
	})

	if result.Valid {
		t.Error("expected validation to fail for empty modules")
	}

	if len(result.Errors) != 1 || result.Errors[0].Field != "_module" {
		t.Error("expected unknown module error")
	}
}

// TestValidateCreateWithEmptyData tests validation with empty data map.
func TestValidateCreateWithEmptyData(t *testing.T) {
	modules := map[string]convention.Derived{
		"optional_module": {
			Fields: []convention.DerivedField{
				{Name: "id", Type: schema.FieldTypeUUID, Implicit: true},
				{Name: "optional_field", Type: schema.FieldTypeString, Required: false},
				{Name: "created_at", Type: schema.FieldTypeTimestamp, Implicit: true},
				{Name: "updated_at", Type: schema.FieldTypeTimestamp, Implicit: true},
			},
		},
	}
	v := New(modules)

	result := v.ValidateCreate("optional_module", map[string]any{})

	if !result.Valid {
		t.Errorf("expected validation to pass for empty data with all optional fields, got errors: %v", result.Errors)
	}
}

// TestValidateUpdateWithEmptyData tests update validation with empty data map.
func TestValidateUpdateWithEmptyData(t *testing.T) {
	modules := createTestModules()
	v := New(modules)

	result := v.ValidateUpdate("user", map[string]any{})

	if !result.Valid {
		t.Errorf("expected update validation to pass for empty data, got errors: %v", result.Errors)
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
