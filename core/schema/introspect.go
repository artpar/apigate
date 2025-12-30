// Package schema provides introspection types for exposing module metadata via REST API.
// These types enable clients to dynamically discover available modules, fields, and actions.
package schema

// ModuleListResponse is returned by GET /mod/_schema
type ModuleListResponse struct {
	Modules []ModuleSummary `json:"modules"`
	Count   int             `json:"count"`
}

// ModuleSummary provides a brief overview of a module.
type ModuleSummary struct {
	Name        string `json:"name"`
	Plural      string `json:"plural"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// ModuleSchemaResponse is returned by GET /mod/_schema/{module}
type ModuleSchemaResponse struct {
	Module      string              `json:"module"`
	Plural      string              `json:"plural"`
	Description string              `json:"description,omitempty"`
	Version     string              `json:"version,omitempty"`
	Table       string              `json:"table"`
	Fields      []FieldSchema       `json:"fields"`
	Actions     []ActionSchema      `json:"actions"`
	Lookups     []string            `json:"lookups"`
	Endpoints   []EndpointSchema    `json:"endpoints"`
	Depends     []string            `json:"depends,omitempty"`
}

// FieldSchema describes a module field for introspection.
type FieldSchema struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Required    bool              `json:"required"`
	Unique      bool              `json:"unique,omitempty"`
	Lookup      bool              `json:"lookup,omitempty"`
	Filterable  bool              `json:"filterable,omitempty"`  // can be used in query filters
	Sortable    bool              `json:"sortable,omitempty"`    // can be used in order_by
	Values      []string          `json:"values,omitempty"`      // enum options
	Ref         string            `json:"ref,omitempty"`         // foreign key target module
	Default     any               `json:"default,omitempty"`
	Internal    bool              `json:"internal,omitempty"`    // hidden from API
	Implicit    bool              `json:"implicit,omitempty"`    // auto-generated (id, created_at, etc.)
	SQLType     string            `json:"sql_type,omitempty"`    // for tooling
	Constraints []ConstraintSchema `json:"constraints,omitempty"` // validation rules
	Description string            `json:"description,omitempty"` // human-readable documentation
}

// ConstraintSchema describes a field constraint for introspection.
type ConstraintSchema struct {
	Type    string `json:"type"`
	Value   any    `json:"value,omitempty"`
	Message string `json:"message,omitempty"`
}

// ActionSchema describes a module action for introspection.
type ActionSchema struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"` // list, get, create, update, delete, custom
	Description string        `json:"description,omitempty"`
	Input       []InputSchema `json:"input,omitempty"`
	Output      []string      `json:"output,omitempty"`
	Auth        string        `json:"auth,omitempty"`
	Confirm     bool          `json:"confirm,omitempty"`
	Implicit    bool          `json:"implicit,omitempty"`
	HTTP        *HTTPInfo     `json:"http,omitempty"` // HTTP endpoint info
}

// InputSchema describes an action input parameter.
type InputSchema struct {
	Name       string `json:"name"`
	Field      string `json:"field,omitempty"` // schema field this maps to
	Type       string `json:"type"`
	Required   bool   `json:"required,omitempty"`
	Default    string `json:"default,omitempty"`
	Prompt     bool   `json:"prompt,omitempty"`      // CLI should prompt
	PromptText string `json:"prompt_text,omitempty"` // prompt message
}

// HTTPInfo describes the HTTP endpoint for an action.
type HTTPInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// EndpointSchema describes a full HTTP endpoint.
type EndpointSchema struct {
	Action string `json:"action"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Auth   string `json:"auth,omitempty"`
}
