package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFile parses a module definition from a YAML file.
func ParseFile(path string) (Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Module{}, fmt.Errorf("read file %s: %w", path, err)
	}

	return Parse(data)
}

// Parse parses a module definition from YAML bytes.
func Parse(data []byte) (Module, error) {
	var mod Module
	if err := yaml.Unmarshal(data, &mod); err != nil {
		return Module{}, fmt.Errorf("parse yaml: %w", err)
	}

	if err := Validate(mod); err != nil {
		return Module{}, fmt.Errorf("validate module %q: %w", mod.Name, err)
	}

	return mod, nil
}

// ParseDir parses all module definitions from a directory, including subdirectories.
func ParseDir(dir string) ([]Module, error) {
	var modules []Module

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			// Recursively parse subdirectories (e.g., providers/, capabilities/)
			subModules, err := ParseDir(path)
			if err != nil {
				return nil, err
			}
			modules = append(modules, subModules...)
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		mod, err := ParseFile(path)
		if err != nil {
			return nil, err
		}

		modules = append(modules, mod)
	}

	return modules, nil
}

// Validate validates a module definition.
func Validate(mod Module) error {
	// Skip validation for capability interface definitions
	if mod.IsCapability() {
		return validateCapability(mod)
	}

	var errs []string

	if mod.Name == "" {
		errs = append(errs, "module name is required")
	}

	if !isValidIdentifier(mod.Name) {
		errs = append(errs, fmt.Sprintf("module name %q is not a valid identifier", mod.Name))
	}

	if len(mod.Schema) == 0 {
		errs = append(errs, "schema must have at least one field")
	}

	// Validate fields
	for name, field := range mod.Schema {
		if !isValidIdentifier(name) {
			errs = append(errs, fmt.Sprintf("field name %q is not a valid identifier", name))
		}

		if err := validateField(name, field); err != nil {
			errs = append(errs, err.Error())
		}
	}

	// Validate actions
	for name, action := range mod.Actions {
		if !isValidIdentifier(name) {
			errs = append(errs, fmt.Sprintf("action name %q is not a valid identifier", name))
		}

		if err := validateAction(name, action, mod.Schema); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// validateCapability validates a capability interface definition.
// Capability definitions have different rules - they define interfaces, not concrete modules.
func validateCapability(mod Module) error {
	var errs []string

	if mod.Capability == "" {
		errs = append(errs, "capability name is required")
	}

	if !isValidIdentifier(mod.Capability) {
		errs = append(errs, fmt.Sprintf("capability name %q is not a valid identifier", mod.Capability))
	}

	// Validate action names (but don't validate output fields against schema)
	for name := range mod.Actions {
		if !isValidIdentifier(name) {
			errs = append(errs, fmt.Sprintf("action name %q is not a valid identifier", name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// validateField validates a single field definition.
func validateField(name string, field Field) error {
	// Check type is valid
	if !isValidFieldType(field.Type) {
		return fmt.Errorf("field %q: unknown type %q", name, field.Type)
	}

	// Enum requires values
	if field.Type == FieldTypeEnum && len(field.Values) == 0 {
		return fmt.Errorf("field %q: enum type requires values", name)
	}

	// Ref requires target
	if field.Type == FieldTypeRef && field.To == "" {
		return fmt.Errorf("field %q: ref type requires 'to' target", name)
	}

	// Default must match type (basic validation)
	if field.Default != nil {
		if err := validateDefault(name, field); err != nil {
			return err
		}
	}

	return nil
}

// validateAction validates a single action definition.
func validateAction(name string, action Action, schema map[string]Field) error {
	// Validate set fields exist in schema
	for fieldName := range action.Set {
		if _, ok := schema[fieldName]; !ok {
			return fmt.Errorf("action %q: set field %q not in schema", name, fieldName)
		}
	}

	// Validate input fields
	for _, input := range action.Input {
		if input.Field != "" {
			if _, ok := schema[input.Field]; !ok {
				return fmt.Errorf("action %q: input field %q not in schema", name, input.Field)
			}
		}
	}

	// Note: Output fields don't need to be in schema - actions can define custom return structures

	return nil
}

// validateDefault validates that a default value matches the field type.
func validateDefault(name string, field Field) error {
	switch field.Type {
	case FieldTypeInt:
		switch field.Default.(type) {
		case int, int64, float64:
			return nil
		default:
			return fmt.Errorf("field %q: default must be an integer", name)
		}
	case FieldTypeBool:
		if _, ok := field.Default.(bool); !ok {
			return fmt.Errorf("field %q: default must be a boolean", name)
		}
	case FieldTypeEnum:
		s, ok := field.Default.(string)
		if !ok {
			return fmt.Errorf("field %q: default must be a string", name)
		}
		found := false
		for _, v := range field.Values {
			if v == s {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %q: default %q is not a valid enum value", name, s)
		}
	}
	return nil
}

// isValidIdentifier checks if a string is a valid identifier.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	for i, c := range s {
		if i == 0 {
			if !isLetter(c) && c != '_' {
				return false
			}
		} else {
			if !isLetter(c) && !isDigit(c) && c != '_' {
				return false
			}
		}
	}

	return true
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

// isValidFieldType checks if a field type is valid.
func isValidFieldType(t FieldType) bool {
	switch t {
	case FieldTypeString, FieldTypeInt, FieldTypeFloat, FieldTypeBool,
		FieldTypeTimestamp, FieldTypeDuration, FieldTypeJSON, FieldTypeBytes,
		FieldTypeEmail, FieldTypeURL, FieldTypeUUID,
		FieldTypeEnum, FieldTypeRef, FieldTypeSecret,
		FieldTypeStrings, FieldTypeInts:
		return true
	default:
		return false
	}
}
