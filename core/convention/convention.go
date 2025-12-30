// Package convention derives defaults from minimal module definitions.
// It applies naming conventions, implicit fields, and default behaviors.
package convention

import (
	"strconv"
	"strings"

	"github.com/artpar/apigate/core/schema"
)

// Derived contains all derived information from a module definition.
// This is the fully-expanded form used by the runtime.
type Derived struct {
	// Source is the original module definition.
	Source schema.Module

	// Plural is the plural form of the module name.
	Plural string

	// Table is the database table name.
	Table string

	// Fields contains all fields including implicit ones (id, created_at, updated_at).
	Fields []DerivedField

	// Actions contains all actions including implicit CRUD.
	Actions []DerivedAction

	// Lookups are field names that can be used to find records.
	Lookups []string

	// Paths contains all path claims for this module.
	Paths []schema.PathClaim
}

// DerivedField is a fully-derived field with all defaults applied.
type DerivedField struct {
	// Name of the field.
	Name string

	// Source is the original field definition (nil for implicit fields).
	Source *schema.Field

	// Type is the resolved field type.
	Type schema.FieldType

	// SQLType is the SQL column type.
	SQLType string

	// Unique indicates this field must have unique values.
	Unique bool

	// Required indicates this field must be provided.
	Required bool

	// Lookup indicates this field can be used to find records.
	Lookup bool

	// Internal indicates this field is never exposed.
	Internal bool

	// Default value.
	Default any

	// Values for enum fields.
	Values []string

	// Ref target for reference fields.
	Ref string

	// Implicit indicates this is an auto-generated field.
	Implicit bool

	// Constraints are validation rules for this field.
	Constraints []schema.Constraint

	// Description provides human-readable documentation for this field.
	Description string
}

// DerivedAction is a fully-derived action with all defaults applied.
type DerivedAction struct {
	// Name of the action.
	Name string

	// Type of the action.
	Type schema.ActionType

	// Source is the original action definition (nil for implicit actions).
	Source *schema.Action

	// Set contains fields to set (for custom actions).
	Set map[string]string

	// Input defines the input parameters.
	Input []ActionInput

	// Output defines the output fields.
	Output []string

	// Auth defines who can execute this action.
	Auth string

	// Confirm indicates this action requires confirmation.
	Confirm bool

	// Description for documentation.
	Description string

	// Implicit indicates this is an auto-generated action.
	Implicit bool
}

// ActionInput is a fully-derived action input.
type ActionInput struct {
	// Name of the input parameter.
	Name string

	// Field this input maps to (if any).
	Field string

	// Type of the input.
	Type schema.FieldType

	// Required indicates this input must be provided.
	Required bool

	// Default value.
	Default string

	// Prompt indicates CLI should prompt for this value.
	Prompt bool

	// PromptText is the prompt message.
	PromptText string
}

// Derive expands a minimal module definition into a fully-derived form.
func Derive(mod schema.Module) Derived {
	d := Derived{
		Source: mod,
		Plural: Pluralize(mod.Name),
		Table:  Pluralize(mod.Name),
	}

	d.Fields = deriveFields(mod)
	d.Actions = deriveActions(mod, d.Fields)
	d.Lookups = deriveLookups(d.Fields)
	d.Paths = schema.ExtractPaths(mod, d.Plural)

	return d
}

// deriveFields creates the full list of fields including implicit ones.
func deriveFields(mod schema.Module) []DerivedField {
	fields := make([]DerivedField, 0, len(mod.Schema)+3)

	// Implicit ID field
	fields = append(fields, DerivedField{
		Name:     "id",
		Type:     schema.FieldTypeUUID,
		SQLType:  "TEXT",
		Unique:   true,
		Required: false, // Auto-generated
		Lookup:   true,
		Implicit: true,
	})

	// User-defined fields
	for name, f := range mod.Schema {
		field := DerivedField{
			Name:        name,
			Source:      &f,
			Type:        f.Type,
			SQLType:     f.SQLType(),
			Unique:      f.Unique,
			Required:    f.IsRequired(),
			Lookup:      f.Lookup,
			Internal:    f.IsInternal(),
			Default:     f.Default,
			Values:      f.Values,
			Ref:         f.To,
			Implicit:    false,
			Constraints: f.Constraints,
			Description: f.Description,
		}
		fields = append(fields, field)
	}

	// Implicit timestamp fields
	fields = append(fields, DerivedField{
		Name:     "created_at",
		Type:     schema.FieldTypeTimestamp,
		SQLType:  "TEXT",
		Required: false,
		Implicit: true,
	})

	fields = append(fields, DerivedField{
		Name:     "updated_at",
		Type:     schema.FieldTypeTimestamp,
		SQLType:  "TEXT",
		Required: false,
		Implicit: true,
	})

	return fields
}

// deriveActions creates the full list of actions including implicit CRUD.
func deriveActions(mod schema.Module, fields []DerivedField) []DerivedAction {
	actions := make([]DerivedAction, 0, 5+len(mod.Actions))

	// Collect editable and listable fields
	var editableFields, listableFields, outputFields []string
	for _, f := range fields {
		if f.Implicit || f.Internal {
			continue
		}
		if f.Type != schema.FieldTypeRef { // Refs handled separately
			editableFields = append(editableFields, f.Name)
		}
		listableFields = append(listableFields, f.Name)
		outputFields = append(outputFields, f.Name)
	}

	// Add implicit id to output
	outputFields = append([]string{"id"}, outputFields...)

	// Implicit CRUD actions
	actions = append(actions, DerivedAction{
		Name:        "list",
		Type:        schema.ActionTypeList,
		Output:      listableFields,
		Auth:        "admin",
		Description: "List all " + mod.Name + "s",
		Implicit:    true,
	})

	actions = append(actions, DerivedAction{
		Name:        "get",
		Type:        schema.ActionTypeGet,
		Output:      outputFields,
		Auth:        "admin",
		Description: "Get " + mod.Name + " details",
		Implicit:    true,
	})

	actions = append(actions, DerivedAction{
		Name:        "create",
		Type:        schema.ActionTypeCreate,
		Input:       deriveCreateInputs(fields),
		Output:      outputFields,
		Auth:        "admin",
		Description: "Create a new " + mod.Name,
		Implicit:    true,
	})

	actions = append(actions, DerivedAction{
		Name:        "update",
		Type:        schema.ActionTypeUpdate,
		Input:       deriveUpdateInputs(fields),
		Output:      outputFields,
		Auth:        "admin",
		Description: "Update " + mod.Name,
		Implicit:    true,
	})

	actions = append(actions, DerivedAction{
		Name:        "delete",
		Type:        schema.ActionTypeDelete,
		Auth:        "admin",
		Confirm:     true,
		Description: "Delete " + mod.Name,
		Implicit:    true,
	})

	// Custom actions
	for name, action := range mod.Actions {
		a := DerivedAction{
			Name:        name,
			Type:        schema.ActionTypeCustom,
			Source:      &action,
			Set:         action.Set,
			Input:       deriveCustomActionInputs(action.Input),
			Auth:        action.Auth,
			Confirm:     action.Confirm,
			Description: action.Description,
			Implicit:    false,
		}

		if a.Auth == "" {
			a.Auth = "admin"
		}

		if a.Description == "" {
			a.Description = strings.Title(name) + " " + mod.Name
		}

		actions = append(actions, a)
	}

	return actions
}

// deriveCustomActionInputs converts schema.ActionInput to convention.ActionInput for custom actions.
func deriveCustomActionInputs(inputs []schema.ActionInput) []ActionInput {
	if len(inputs) == 0 {
		return nil
	}

	result := make([]ActionInput, len(inputs))
	for i, input := range inputs {
		name := input.Name
		if name == "" {
			name = input.Field
		}

		result[i] = ActionInput{
			Name:       name,
			Field:      input.Field,
			Type:       schema.FieldType(input.Type),
			Required:   input.Required,
			Default:    input.Default,
			Prompt:     input.Prompt,
			PromptText: input.PromptText,
		}
	}
	return result
}

// deriveCreateInputs derives input parameters for create action.
func deriveCreateInputs(fields []DerivedField) []ActionInput {
	var inputs []ActionInput

	for _, f := range fields {
		if f.Implicit || f.Name == "id" {
			continue
		}

		input := ActionInput{
			Name:     f.Name,
			Field:    f.Name,
			Type:     f.Type,
			Required: f.Required,
		}

		if f.Default != nil {
			input.Default = toString(f.Default)
			input.Required = false
		}

		if f.Type == schema.FieldTypeSecret {
			input.Prompt = true
			input.PromptText = "Enter " + f.Name
		}

		inputs = append(inputs, input)
	}

	return inputs
}

// deriveUpdateInputs derives input parameters for update action.
func deriveUpdateInputs(fields []DerivedField) []ActionInput {
	var inputs []ActionInput

	for _, f := range fields {
		if f.Implicit || f.Name == "id" {
			continue
		}

		input := ActionInput{
			Name:     f.Name,
			Field:    f.Name,
			Type:     f.Type,
			Required: false, // Updates don't require all fields
		}

		if f.Type == schema.FieldTypeSecret {
			input.Prompt = true
			input.PromptText = "Enter new " + f.Name
		}

		inputs = append(inputs, input)
	}

	return inputs
}

// deriveLookups extracts all lookup field names.
func deriveLookups(fields []DerivedField) []string {
	lookups := make([]string, 0)

	for _, f := range fields {
		if f.Lookup {
			lookups = append(lookups, f.Name)
		}
	}

	return lookups
}

// toString converts a value to string.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}
