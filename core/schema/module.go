// Package schema defines the core types for declarative module definitions.
// A module is a self-contained unit that owns its data, actions, and channels.
package schema

// Module is the root definition for a declarative module.
// Everything is derived from this minimal definition.
type Module struct {
	// Name is the singular name of the module (e.g., "user", "plan").
	// Plural form is derived by convention.
	Name string `yaml:"module"`

	// Capability indicates this is a capability interface definition.
	// Capability files define the interface that implementing modules must provide.
	Capability string `yaml:"capability,omitempty"`

	// Description for documentation (used in capability definitions).
	Description string `yaml:"description,omitempty"`

	// Schema defines the data fields owned by this module.
	Schema map[string]Field `yaml:"schema"`

	// Actions defines custom actions beyond CRUD.
	// CRUD (list, get, create, update, delete) is implicit.
	Actions map[string]Action `yaml:"actions,omitempty"`

	// Channels defines how this module communicates (serve and consume).
	Channels Channels `yaml:"channels,omitempty"`

	// Hooks defines event handlers for this module.
	Hooks map[string][]Hook `yaml:"hooks,omitempty"`

	// Meta contains optional metadata.
	Meta ModuleMeta `yaml:"meta,omitempty"`
}

// ModuleMeta contains optional module metadata.
type ModuleMeta struct {
	// Version of the module definition.
	Version string `yaml:"version,omitempty"`

	// Description for documentation.
	Description string `yaml:"description,omitempty"`

	// Depends lists other modules this one depends on.
	Depends []string `yaml:"depends,omitempty"`

	// Implements declares which capability interface this module provides.
	// e.g., "payment", "email", "cache", "storage", "auth"
	// A module can implement multiple capabilities.
	Implements []string `yaml:"implements,omitempty"`

	// Requires maps named dependencies to capability requirements.
	// This enables dependency injection where a module can declare it needs
	// an instance of a specific capability type.
	// Example: { "payment": { capability: "payment", required: true } }
	// Example: { "source_db": { capability: "data_source" }, "target_db": { capability: "data_source" } }
	Requires map[string]ModuleRequirement `yaml:"requires,omitempty"`

	// Icon for UI display.
	Icon string `yaml:"icon,omitempty"`

	// DisplayName for UI display.
	DisplayName string `yaml:"display_name,omitempty"`

	// Plural name for the module.
	Plural string `yaml:"plural,omitempty"`
}

// ModuleRequirement defines a required dependency on a capability.
// Used for dependency injection where a module needs an instance of a capability type.
type ModuleRequirement struct {
	// Capability is the capability type required (e.g., "payment", "data_source", "cache").
	Capability string `yaml:"capability"`

	// Required indicates if this dependency is mandatory.
	// If true, the module cannot be enabled without this dependency being satisfied.
	Required bool `yaml:"required,omitempty"`

	// Description explains why this dependency is needed.
	Description string `yaml:"description,omitempty"`

	// Default is the default module instance name to use if not specified.
	Default string `yaml:"default,omitempty"`
}

// Hook defines an event handler.
// Supports shorthand YAML formats:
//   - emit: event.name
//   - call: function_name
// Or explicit format:
//   - type: email
//     template: welcome
//     to: "{{.email}}"
type Hook struct {
	// Shorthand: "- emit: event.name" -> Emit = "event.name"
	Emit string `yaml:"emit,omitempty"`

	// Shorthand: "- call: function_name" -> Call = "function_name"
	Call string `yaml:"call,omitempty"`

	// Type of hook action: email, webhook, emit, log, etc.
	// Used for explicit format: "type: email"
	Type string `yaml:"type,omitempty"`

	// For email hooks
	Template string `yaml:"template,omitempty"`
	To       string `yaml:"to,omitempty"`

	// For emit hooks (explicit format)
	Event string `yaml:"event,omitempty"`

	// For webhook hooks
	URL    string            `yaml:"url,omitempty"`
	Method string            `yaml:"method,omitempty"`
	Body   map[string]string `yaml:"body,omitempty"`

	// Conditional execution
	When string `yaml:"when,omitempty"`
}

// IsCapability returns true if this is a capability interface definition.
func (m Module) IsCapability() bool {
	return m.Capability != ""
}
