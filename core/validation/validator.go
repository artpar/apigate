// Package validation provides field and action validation using schema constraints.
// Validation is enforced at runtime before storage operations.
package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Validator validates input data against module schemas.
type Validator struct {
	modules map[string]convention.Derived
}

// New creates a new validator with the given modules.
func New(modules map[string]convention.Derived) *Validator {
	return &Validator{
		modules: modules,
	}
}

// UpdateModules updates the validator's module registry.
func (v *Validator) UpdateModules(modules map[string]convention.Derived) {
	v.modules = modules
}

// ValidateCreate validates input data for a create action.
// Returns a ValidationResult with all validation errors.
func (v *Validator) ValidateCreate(moduleName string, data map[string]any) schema.ValidationResult {
	result := schema.ValidationResult{Valid: true}

	mod, ok := v.modules[moduleName]
	if !ok {
		result.AddError("_module", "unknown", moduleName, fmt.Sprintf("unknown module: %s", moduleName))
		return result
	}

	// Build field lookup for unknown field detection
	knownFields := make(map[string]bool)
	for _, f := range mod.Fields {
		knownFields[f.Name] = true
	}

	// Check for unknown fields (strict mode - fail loud)
	for fieldName := range data {
		if !knownFields[fieldName] {
			result.AddError(fieldName, "unknown_field", fieldName,
				fmt.Sprintf("unknown field '%s' - not defined in schema", fieldName))
		}
	}

	for _, field := range mod.Fields {
		value, hasValue := data[field.Name]

		// Skip implicit fields (id, created_at, updated_at)
		if field.Implicit {
			continue
		}

		// Check required fields
		if field.Required && !hasValue {
			// Check if there's a default value
			if field.Default == nil {
				result.AddError(field.Name, "required", nil, "field is required")
			}
			continue
		}

		// If no value provided, skip further validation
		if !hasValue || value == nil {
			continue
		}

		// Validate field type
		v.validateFieldType(&result, field, value)

		// Validate constraints
		v.validateConstraints(&result, field, value)
	}

	return result
}

// ValidateUpdate validates input data for an update action.
// Unlike create, update doesn't require all fields.
func (v *Validator) ValidateUpdate(moduleName string, data map[string]any) schema.ValidationResult {
	result := schema.ValidationResult{Valid: true}

	mod, ok := v.modules[moduleName]
	if !ok {
		result.AddError("_module", "unknown", moduleName, fmt.Sprintf("unknown module: %s", moduleName))
		return result
	}

	// Build field lookup
	fieldMap := make(map[string]convention.DerivedField)
	for _, f := range mod.Fields {
		fieldMap[f.Name] = f
	}

	// Only validate provided fields
	for fieldName, value := range data {
		field, ok := fieldMap[fieldName]
		if !ok {
			// Unknown field - strict mode: reject unknown fields
			result.AddError(fieldName, "unknown_field", fieldName,
				fmt.Sprintf("unknown field '%s' - not defined in schema", fieldName))
			continue
		}

		// Skip implicit fields
		if field.Implicit {
			continue
		}

		// If nil value, skip validation (explicit null to clear field)
		if value == nil {
			continue
		}

		// Validate field type
		v.validateFieldType(&result, field, value)

		// Validate constraints
		v.validateConstraints(&result, field, value)
	}

	return result
}

// validateFieldType validates the value matches the expected field type.
func (v *Validator) validateFieldType(result *schema.ValidationResult, field convention.DerivedField, value any) {
	switch field.Type {
	case schema.FieldTypeEmail:
		if str, ok := value.(string); ok {
			if _, err := mail.ParseAddress(str); err != nil {
				result.AddError(field.Name, "type", value, "invalid email address")
			}
		}

	case schema.FieldTypeURL:
		if str, ok := value.(string); ok {
			if _, err := url.ParseRequestURI(str); err != nil {
				result.AddError(field.Name, "type", value, "invalid URL")
			}
		}

	case schema.FieldTypeUUID:
		if str, ok := value.(string); ok {
			if !isValidUUID(str) {
				result.AddError(field.Name, "type", value, "invalid UUID format")
			}
		}

	case schema.FieldTypeEnum:
		if str, ok := value.(string); ok {
			if !containsString(field.Values, str) {
				result.AddError(field.Name, "enum", value,
					fmt.Sprintf("must be one of: %s", strings.Join(field.Values, ", ")))
			}
		}

	case schema.FieldTypeInt:
		switch value.(type) {
		case int, int32, int64, float64:
			// Valid numeric types
		default:
			result.AddError(field.Name, "type", value, "must be an integer")
		}

	case schema.FieldTypeFloat:
		switch value.(type) {
		case float32, float64, int, int32, int64:
			// Valid numeric types
		default:
			result.AddError(field.Name, "type", value, "must be a number")
		}

	case schema.FieldTypeBool:
		if _, ok := value.(bool); !ok {
			result.AddError(field.Name, "type", value, "must be a boolean")
		}

	case schema.FieldTypeRef:
		// Reference validation happens at storage level (foreign key check)
		// We just validate format here
		if str, ok := value.(string); ok {
			if strings.TrimSpace(str) == "" {
				result.AddError(field.Name, "type", value, "reference cannot be empty")
			}
		}
	}
}

// validateConstraints validates the value against field constraints.
func (v *Validator) validateConstraints(result *schema.ValidationResult, field convention.DerivedField, value any) {
	for _, c := range field.Constraints {
		if err := schema.ValidateConstraint(field.Name, value, c); err != nil {
			result.Errors = append(result.Errors, *err)
			result.Valid = false
		}
	}
}

// isValidUUID checks if a string is a valid UUID format.
func isValidUUID(s string) bool {
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidPattern.MatchString(s)
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ValidateField validates a single field value against its schema.
// This is useful for client-side or partial validation.
func ValidateField(field convention.DerivedField, value any) schema.ValidationResult {
	result := schema.ValidationResult{Valid: true}

	if value == nil {
		if field.Required && field.Default == nil {
			result.AddError(field.Name, "required", nil, "field is required")
		}
		return result
	}

	v := &Validator{}
	v.validateFieldType(&result, field, value)
	v.validateConstraints(&result, field, value)

	return result
}
