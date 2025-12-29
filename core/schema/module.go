// Package schema defines the core types for declarative module definitions.
// A module is a self-contained unit that owns its data, actions, and channels.
package schema

// Module is the root definition for a declarative module.
// Everything is derived from this minimal definition.
type Module struct {
	// Name is the singular name of the module (e.g., "user", "plan").
	// Plural form is derived by convention.
	Name string `yaml:"module"`

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
}

// Hook defines an event handler.
type Hook struct {
	// Type of hook action: email, webhook, emit, log, etc.
	Type string `yaml:"type,omitempty"`

	// For email hooks
	Template string `yaml:"template,omitempty"`
	To       string `yaml:"to,omitempty"`

	// For emit hooks
	Event string `yaml:"event,omitempty"`

	// For webhook hooks
	URL    string            `yaml:"url,omitempty"`
	Method string            `yaml:"method,omitempty"`
	Body   map[string]string `yaml:"body,omitempty"`

	// Conditional execution
	When string `yaml:"when,omitempty"`
}
