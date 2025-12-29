package schema

// Action defines an operation that can be performed on a module.
// CRUD actions (list, get, create, update, delete) are implicit.
// Custom actions are defined here.
type Action struct {
	// Set defines field values to set when this action is executed.
	// Used for simple state transitions like activate: {set: {status: active}}
	Set map[string]string `yaml:"set,omitempty"`

	// Input defines the fields accepted by this action.
	// If nil, defaults are derived from the action type.
	Input []ActionInput `yaml:"input,omitempty"`

	// Output defines fields returned by this action.
	// If nil, returns the full record.
	Output []string `yaml:"output,omitempty"`

	// Auth defines who can execute this action.
	// Values: "public", "user", "admin", "owner", or combinations like "admin|owner"
	Auth string `yaml:"auth,omitempty"`

	// Confirm requires confirmation before execution (for destructive actions).
	Confirm bool `yaml:"confirm,omitempty"`

	// Hooks to run before/after this action.
	Before []string `yaml:"before,omitempty"`
	After  []string `yaml:"after,omitempty"`

	// Description for documentation and help text.
	Description string `yaml:"description,omitempty"`
}

// ActionInput defines an input parameter for an action.
type ActionInput struct {
	// Field is the schema field name this input maps to.
	Field string `yaml:"field,omitempty"`

	// Name overrides the input name (defaults to Field).
	Name string `yaml:"name,omitempty"`

	// Type overrides the field type for this input.
	// Useful for special handling like "password" → hash before storing.
	Type string `yaml:"type,omitempty"`

	// Required indicates this input must be provided.
	Required bool `yaml:"required,omitempty"`

	// Default value if not provided.
	Default string `yaml:"default,omitempty"`

	// Prompt indicates CLI should prompt for this value if not provided.
	Prompt bool `yaml:"prompt,omitempty"`

	// PromptText is the prompt message (defaults to field name).
	PromptText string `yaml:"prompt_text,omitempty"`
}

// ActionType represents the type of action.
type ActionType int

const (
	// ActionTypeList retrieves multiple records.
	ActionTypeList ActionType = iota

	// ActionTypeGet retrieves a single record by lookup field.
	ActionTypeGet

	// ActionTypeCreate creates a new record.
	ActionTypeCreate

	// ActionTypeUpdate modifies an existing record.
	ActionTypeUpdate

	// ActionTypeDelete removes a record.
	ActionTypeDelete

	// ActionTypeCustom is a user-defined action.
	ActionTypeCustom
)

// String returns the action type name.
func (t ActionType) String() string {
	switch t {
	case ActionTypeList:
		return "list"
	case ActionTypeGet:
		return "get"
	case ActionTypeCreate:
		return "create"
	case ActionTypeUpdate:
		return "update"
	case ActionTypeDelete:
		return "delete"
	case ActionTypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// ImplicitActions returns the standard CRUD actions that every module has.
func ImplicitActions() []string {
	return []string{"list", "get", "create", "update", "delete"}
}

// IsImplicit returns true if the action name is an implicit CRUD action.
func IsImplicit(name string) bool {
	switch name {
	case "list", "get", "create", "update", "delete":
		return true
	default:
		return false
	}
}

// ActionTypeFromName returns the ActionType for a given action name.
func ActionTypeFromName(name string) ActionType {
	switch name {
	case "list":
		return ActionTypeList
	case "get":
		return ActionTypeGet
	case "create":
		return ActionTypeCreate
	case "update":
		return ActionTypeUpdate
	case "delete":
		return ActionTypeDelete
	default:
		return ActionTypeCustom
	}
}
