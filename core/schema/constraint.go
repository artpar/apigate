// Package schema provides constraint types for field validation.
// Constraints are defined in module schema and enforced at runtime.
package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Constraint defines a validation rule for a field.
type Constraint struct {
	// Type is the constraint type (min, max, min_length, max_length, pattern, etc.)
	Type ConstraintType `yaml:"type" json:"type"`

	// Value is the constraint parameter (number, regex pattern, etc.)
	Value any `yaml:"value" json:"value"`

	// Message is the custom error message (optional).
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
}

// ConstraintType identifies the type of constraint.
type ConstraintType string

const (
	// Numeric constraints
	ConstraintMin ConstraintType = "min" // Minimum numeric value
	ConstraintMax ConstraintType = "max" // Maximum numeric value

	// String constraints
	ConstraintMinLength ConstraintType = "min_length" // Minimum string length
	ConstraintMaxLength ConstraintType = "max_length" // Maximum string length
	ConstraintPattern   ConstraintType = "pattern"    // Regex pattern match

	// Reference constraints
	ConstraintRefExists ConstraintType = "ref_exists" // Referenced record must exist

	// Custom constraints
	ConstraintNotEmpty ConstraintType = "not_empty" // String must not be empty/whitespace
	ConstraintOneOf    ConstraintType = "one_of"    // Value must be one of list (for non-enum validation)
)

// ConstraintError represents a validation failure.
type ConstraintError struct {
	Field      string `json:"field"`
	Constraint string `json:"constraint"`
	Value      any    `json:"value,omitempty"`
	Message    string `json:"message"`
}

func (e ConstraintError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds all validation errors for a request.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ConstraintError `json:"errors,omitempty"`
}

// AddError adds a validation error.
func (r *ValidationResult) AddError(field, constraint string, value any, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ConstraintError{
		Field:      field,
		Constraint: constraint,
		Value:      value,
		Message:    message,
	})
}

// Error returns a combined error message.
func (r ValidationResult) Error() string {
	if r.Valid {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// ValidateConstraint validates a value against a single constraint.
// This is a PURE function.
func ValidateConstraint(fieldName string, value any, c Constraint) *ConstraintError {
	switch c.Type {
	case ConstraintMin:
		return validateMin(fieldName, value, c)
	case ConstraintMax:
		return validateMax(fieldName, value, c)
	case ConstraintMinLength:
		return validateMinLength(fieldName, value, c)
	case ConstraintMaxLength:
		return validateMaxLength(fieldName, value, c)
	case ConstraintPattern:
		return validatePattern(fieldName, value, c)
	case ConstraintNotEmpty:
		return validateNotEmpty(fieldName, value, c)
	case ConstraintOneOf:
		return validateOneOf(fieldName, value, c)
	case ConstraintRefExists:
		// Reference existence is checked at runtime, not here
		return nil
	default:
		return nil
	}
}

func validateMin(field string, value any, c Constraint) *ConstraintError {
	min, err := toFloat64(c.Value)
	if err != nil {
		return nil // Invalid constraint config, skip
	}

	val, err := toFloat64(value)
	if err != nil {
		return nil // Can't validate non-numeric, skip
	}

	if val < min {
		msg := c.Message
		if msg == "" {
			msg = fmt.Sprintf("must be at least %v", min)
		}
		return &ConstraintError{Field: field, Constraint: "min", Value: value, Message: msg}
	}
	return nil
}

func validateMax(field string, value any, c Constraint) *ConstraintError {
	max, err := toFloat64(c.Value)
	if err != nil {
		return nil
	}

	val, err := toFloat64(value)
	if err != nil {
		return nil
	}

	if val > max {
		msg := c.Message
		if msg == "" {
			msg = fmt.Sprintf("must be at most %v", max)
		}
		return &ConstraintError{Field: field, Constraint: "max", Value: value, Message: msg}
	}
	return nil
}

func validateMinLength(field string, value any, c Constraint) *ConstraintError {
	minLen, err := toInt(c.Value)
	if err != nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	if len(str) < minLen {
		msg := c.Message
		if msg == "" {
			msg = fmt.Sprintf("must be at least %d characters", minLen)
		}
		return &ConstraintError{Field: field, Constraint: "min_length", Value: len(str), Message: msg}
	}
	return nil
}

func validateMaxLength(field string, value any, c Constraint) *ConstraintError {
	maxLen, err := toInt(c.Value)
	if err != nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	if len(str) > maxLen {
		msg := c.Message
		if msg == "" {
			msg = fmt.Sprintf("must be at most %d characters", maxLen)
		}
		return &ConstraintError{Field: field, Constraint: "max_length", Value: len(str), Message: msg}
	}
	return nil
}

func validatePattern(field string, value any, c Constraint) *ConstraintError {
	pattern, ok := c.Value.(string)
	if !ok {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil // Invalid regex, skip
	}

	if !re.MatchString(str) {
		msg := c.Message
		if msg == "" {
			msg = "does not match required pattern"
		}
		return &ConstraintError{Field: field, Constraint: "pattern", Value: value, Message: msg}
	}
	return nil
}

func validateNotEmpty(field string, value any, c Constraint) *ConstraintError {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	if strings.TrimSpace(str) == "" {
		msg := c.Message
		if msg == "" {
			msg = "must not be empty"
		}
		return &ConstraintError{Field: field, Constraint: "not_empty", Value: value, Message: msg}
	}
	return nil
}

func validateOneOf(field string, value any, c Constraint) *ConstraintError {
	allowedVals, ok := c.Value.([]any)
	if !ok {
		// Try string slice
		if strVals, ok := c.Value.([]string); ok {
			allowedVals = make([]any, len(strVals))
			for i, v := range strVals {
				allowedVals[i] = v
			}
		} else {
			return nil
		}
	}

	strVal := fmt.Sprintf("%v", value)
	for _, allowed := range allowedVals {
		if fmt.Sprintf("%v", allowed) == strVal {
			return nil
		}
	}

	msg := c.Message
	if msg == "" {
		var options []string
		for _, v := range allowedVals {
			options = append(options, fmt.Sprintf("%v", v))
		}
		msg = fmt.Sprintf("must be one of: %s", strings.Join(options, ", "))
	}
	return &ConstraintError{Field: field, Constraint: "one_of", Value: value, Message: msg}
}

// toFloat64 converts various numeric types to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// toInt converts various types to int.
func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case int32:
		return int(n), nil
	case float64:
		return int(n), nil
	case string:
		return strconv.Atoi(n)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}
