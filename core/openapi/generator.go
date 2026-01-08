// Package openapi generates OpenAPI 3.0 specifications from module schemas.
// It auto-generates paths, schemas, and validation rules from the module definitions.
package openapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Spec represents an OpenAPI 3.0 specification.
type Spec struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info provides API metadata.
type Info struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version"`
	Contact     *Contact `json:"contact,omitempty"`
	License     *License `json:"license,omitempty"`
}

// Contact provides contact information.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License provides license information.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server URL.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem contains operations for a path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation represents an API operation.
type Operation struct {
	Tags        []string            `json:"tags,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"`
}

// Parameter represents an API parameter.
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // path, query, header
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      *Schema     `json:"schema,omitempty"`
}

// RequestBody represents a request body.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

// Response represents an API response.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType represents a media type.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Schema represents a JSON Schema.
type Schema struct {
	Type        string            `json:"type,omitempty"`
	Format      string            `json:"format,omitempty"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string          `json:"required,omitempty"`
	Items       *Schema           `json:"items,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
	Ref         string            `json:"$ref,omitempty"`
	MinLength   *int              `json:"minLength,omitempty"`
	MaxLength   *int              `json:"maxLength,omitempty"`
	Minimum     *float64          `json:"minimum,omitempty"`
	Maximum     *float64          `json:"maximum,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	Default     any               `json:"default,omitempty"`
	Example     any               `json:"example,omitempty"`
	AllOf       []*Schema         `json:"allOf,omitempty"`
	OneOf       []*Schema         `json:"oneOf,omitempty"`
}

// Components contains reusable schemas.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme defines an authentication method.
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
}

// SecurityRequirement specifies required security schemes.
type SecurityRequirement map[string][]string

// Tag provides metadata for a group of operations.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Generator generates OpenAPI specs from modules.
type Generator struct {
	modules map[string]convention.Derived
	info    Info
	servers []Server
}

// NewGenerator creates a new OpenAPI generator.
func NewGenerator(modules map[string]convention.Derived) *Generator {
	return &Generator{
		modules: modules,
		info: Info{
			Title:   "APIGate API",
			Version: "1.0.0",
			Description: "Auto-generated API documentation from module schemas",
		},
	}
}

// SetInfo sets the API info.
func (g *Generator) SetInfo(info Info) {
	g.info = info
}

// AddServer adds a server URL.
func (g *Generator) AddServer(url, description string) {
	g.servers = append(g.servers, Server{
		URL:         url,
		Description: description,
	})
}

// Generate creates the OpenAPI specification.
func (g *Generator) Generate() *Spec {
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info:    g.info,
		Servers: g.servers,
		Paths:   make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]*Schema),
			SecuritySchemes: map[string]SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
					Description:  "JWT authentication",
				},
				"apiKey": {
					Type:        "apiKey",
					In:          "header",
					Name:        "X-API-Key",
					Description: "API key authentication",
				},
			},
		},
		Tags: make([]Tag, 0),
	}

	// Sort modules for consistent output
	var moduleNames []string
	for name := range g.modules {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	// Generate for each module
	for _, name := range moduleNames {
		mod := g.modules[name]
		g.generateModule(spec, mod)
	}

	return spec
}

// generateModule adds a module to the spec.
func (g *Generator) generateModule(spec *Spec, mod convention.Derived) {
	moduleName := mod.Source.Name
	plural := mod.Plural
	description := mod.Source.Meta.Description

	// Add tag
	spec.Tags = append(spec.Tags, Tag{
		Name:        moduleName,
		Description: description,
	})

	// Generate schemas
	g.generateSchemas(spec, mod)

	// Get base path
	basePath := mod.Source.Channels.HTTP.Serve.BasePath
	if basePath == "" {
		basePath = "/mod/" + plural
	}

	// Generate paths for each action
	for _, action := range mod.Actions {
		g.generateActionPath(spec, mod, action, basePath)
	}
}

// generateSchemas creates component schemas for a module.
func (g *Generator) generateSchemas(spec *Spec, mod convention.Derived) {
	moduleName := mod.Source.Name
	title := strings.Title(moduleName)

	// Create schema
	createSchema := g.buildCreateSchema(mod)
	spec.Components.Schemas[title+"Create"] = createSchema

	// Update schema
	updateSchema := g.buildUpdateSchema(mod)
	spec.Components.Schemas[title+"Update"] = updateSchema

	// Response schema (full record)
	responseSchema := g.buildResponseSchema(mod)
	spec.Components.Schemas[title] = responseSchema

	// List response schema
	listSchema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"data": {
				Type:  "array",
				Items: &Schema{Ref: "#/components/schemas/" + title},
			},
			"count": {
				Type:        "integer",
				Description: "Total count of records",
			},
		},
	}
	spec.Components.Schemas[title+"List"] = listSchema
}

// buildCreateSchema builds schema for create requests.
func (g *Generator) buildCreateSchema(mod convention.Derived) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for _, field := range mod.Fields {
		// Skip internal and implicit fields
		if field.Internal || field.Implicit {
			continue
		}

		fieldSchema := g.fieldToSchema(field)
		schema.Properties[field.Name] = fieldSchema

		if field.Required {
			schema.Required = append(schema.Required, field.Name)
		}
	}

	return schema
}

// buildUpdateSchema builds schema for update requests.
func (g *Generator) buildUpdateSchema(mod convention.Derived) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range mod.Fields {
		// Skip internal and implicit fields
		if field.Internal || field.Implicit {
			continue
		}

		fieldSchema := g.fieldToSchema(field)
		schema.Properties[field.Name] = fieldSchema
	}

	return schema
}

// buildResponseSchema builds schema for response records.
func (g *Generator) buildResponseSchema(mod convention.Derived) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for _, field := range mod.Fields {
		// Include all non-internal fields in response
		if field.Internal {
			continue
		}

		fieldSchema := g.fieldToSchema(field)
		schema.Properties[field.Name] = fieldSchema
	}

	return schema
}

// fieldToSchema converts a field to OpenAPI schema.
func (g *Generator) fieldToSchema(field convention.DerivedField) *Schema {
	s := &Schema{
		Description: field.Description,
	}

	// Map field type to OpenAPI type
	switch field.Type {
	case schema.FieldTypeString:
		s.Type = "string"
		s.Example = g.generateExample(field, "example-value")
	case schema.FieldTypeInt:
		s.Type = "integer"
		s.Example = g.generateExample(field, 100)
	case schema.FieldTypeFloat:
		s.Type = "number"
		s.Format = "float"
		s.Example = g.generateExample(field, 99.99)
	case schema.FieldTypeBool:
		s.Type = "boolean"
		s.Example = true
	case schema.FieldTypeEmail:
		s.Type = "string"
		s.Format = "email"
		s.Example = "user@example.com"
	case schema.FieldTypeURL:
		s.Type = "string"
		s.Format = "uri"
		s.Example = "https://api.example.com/v1"
	case schema.FieldTypeTimestamp:
		s.Type = "string"
		s.Format = "date-time"
		s.Example = "2024-01-15T10:30:00Z"
	case schema.FieldTypeDuration:
		s.Type = "string"
		s.Description = "Duration in Go format (e.g., 1h30m, 24h)"
		s.Example = "1h30m"
	case schema.FieldTypeUUID:
		s.Type = "string"
		s.Format = "uuid"
		s.Example = "550e8400-e29b-41d4-a716-446655440000"
	case schema.FieldTypeJSON:
		s.Type = "object"
		s.Example = map[string]any{"key": "value"}
	case schema.FieldTypeBytes:
		s.Type = "string"
		s.Format = "byte"
		s.Example = "YmFzZTY0LWVuY29kZWQ="
	case schema.FieldTypeSecret:
		s.Type = "string"
		s.Format = "password"
		s.Example = "********"
	case schema.FieldTypeEnum:
		s.Type = "string"
		s.Enum = field.Values
		if len(field.Values) > 0 {
			s.Example = field.Values[0]
		}
	case schema.FieldTypeRef:
		s.Type = "string"
		if s.Description == "" {
			s.Description = fmt.Sprintf("Reference to %s", field.Ref)
		}
		s.Example = fmt.Sprintf("%s-id-or-lookup", field.Ref)
	case schema.FieldTypeStrings:
		s.Type = "array"
		s.Items = &Schema{Type: "string"}
		s.Example = []string{"value1", "value2"}
	case schema.FieldTypeInts:
		s.Type = "array"
		s.Items = &Schema{Type: "integer"}
		s.Example = []int{1, 2, 3}
	default:
		s.Type = "string"
		s.Example = "example"
	}

	// Add default value
	if field.Default != nil {
		s.Default = field.Default
		// Use default as example if available
		s.Example = field.Default
	}

	// Add constraints and collect descriptions
	var constraintDescs []string
	for _, c := range field.Constraints {
		if desc := g.applyConstraint(s, c); desc != "" {
			constraintDescs = append(constraintDescs, desc)
		}
	}

	// Append constraint descriptions to field description
	if len(constraintDescs) > 0 {
		constraintInfo := strings.Join(constraintDescs, ". ")
		if s.Description != "" {
			s.Description = fmt.Sprintf("%s. Constraints: %s.", s.Description, constraintInfo)
		} else {
			s.Description = fmt.Sprintf("Constraints: %s.", constraintInfo)
		}
	}

	return s
}

// generateExample creates a contextual example based on field name and type.
func (g *Generator) generateExample(field convention.DerivedField, defaultExample any) any {
	// Generate contextual examples based on field name
	name := strings.ToLower(field.Name)

	switch {
	case strings.Contains(name, "name"):
		if strings.Contains(name, "user") {
			return "John Doe"
		}
		return "Example Name"
	case strings.Contains(name, "email"):
		return "user@example.com"
	case strings.Contains(name, "url") || strings.Contains(name, "endpoint"):
		return "https://api.example.com"
	case strings.Contains(name, "path"):
		return "/api/v1/resource"
	case strings.Contains(name, "method"):
		return "GET"
	case strings.Contains(name, "port"):
		return 8080
	case strings.Contains(name, "host"):
		return "localhost"
	case strings.Contains(name, "timeout"):
		return "30s"
	case strings.Contains(name, "limit"):
		return 100
	case strings.Contains(name, "count"):
		return 10
	case strings.Contains(name, "rate"):
		return 60
	case strings.Contains(name, "price"):
		return 999
	case strings.Contains(name, "description"):
		return "A brief description of this item"
	case strings.Contains(name, "key"):
		return "ak_example_key_prefix"
	case strings.Contains(name, "secret"):
		return "********"
	case strings.Contains(name, "id") && field.Type == schema.FieldTypeUUID:
		return "550e8400-e29b-41d4-a716-446655440000"
	}

	return defaultExample
}

// applyConstraint applies a constraint to a schema and returns a human-readable description.
func (g *Generator) applyConstraint(s *Schema, c schema.Constraint) string {
	switch c.Type {
	case schema.ConstraintMinLength:
		if v, ok := c.Value.(int); ok {
			s.MinLength = &v
			return fmt.Sprintf("Minimum length: %d", v)
		}
	case schema.ConstraintMaxLength:
		if v, ok := c.Value.(int); ok {
			s.MaxLength = &v
			return fmt.Sprintf("Maximum length: %d", v)
		}
	case schema.ConstraintMin:
		if v, ok := c.Value.(float64); ok {
			s.Minimum = &v
			return fmt.Sprintf("Minimum value: %g", v)
		} else if v, ok := c.Value.(int); ok {
			f := float64(v)
			s.Minimum = &f
			return fmt.Sprintf("Minimum value: %d", v)
		}
	case schema.ConstraintMax:
		if v, ok := c.Value.(float64); ok {
			s.Maximum = &v
			return fmt.Sprintf("Maximum value: %g", v)
		} else if v, ok := c.Value.(int); ok {
			f := float64(v)
			s.Maximum = &f
			return fmt.Sprintf("Maximum value: %d", v)
		}
	case schema.ConstraintPattern:
		if v, ok := c.Value.(string); ok {
			s.Pattern = v
			// Use custom message if available, otherwise show pattern
			if c.Message != "" {
				return c.Message
			}
			return fmt.Sprintf("Must match pattern: %s", v)
		}
	case schema.ConstraintNotEmpty:
		return "Must not be empty"
	case schema.ConstraintOneOf:
		if values, ok := c.Value.([]string); ok && len(values) > 0 {
			return fmt.Sprintf("Must be one of: %s", strings.Join(values, ", "))
		}
	case schema.ConstraintRefExists:
		return "Referenced record must exist"
	}
	return ""
}

// generateActionPath creates path items for an action.
func (g *Generator) generateActionPath(spec *Spec, mod convention.Derived, action convention.DerivedAction, basePath string) {
	moduleName := mod.Source.Name
	title := strings.Title(moduleName)

	switch action.Type {
	case schema.ActionTypeList:
		g.addListPath(spec, mod, basePath, title)

	case schema.ActionTypeGet:
		g.addGetPath(spec, mod, basePath, title)

	case schema.ActionTypeCreate:
		g.addCreatePath(spec, mod, basePath, title)

	case schema.ActionTypeUpdate:
		g.addUpdatePath(spec, mod, basePath, title)

	case schema.ActionTypeDelete:
		g.addDeletePath(spec, mod, basePath, title)

	case schema.ActionTypeCustom:
		g.addCustomActionPath(spec, mod, action, basePath, title)
	}
}

// addListPath adds list operation.
func (g *Generator) addListPath(spec *Spec, mod convention.Derived, basePath, title string) {
	path := spec.Paths[basePath]

	// Collect filterable and sortable field names
	var filterableFields, sortableFields []string
	for _, field := range mod.Fields {
		if field.Internal {
			continue
		}
		// Filterable: lookup, unique, or id fields
		if field.Lookup || field.Unique || field.Name == "id" {
			filterableFields = append(filterableFields, field.Name)
		}
		// Sortable: non-internal basic types
		if isBasicType(field.Type) {
			sortableFields = append(sortableFields, field.Name)
		}
	}

	// Query parameters for list
	params := []Parameter{
		{Name: "limit", In: "query", Description: "Maximum number of records", Schema: &Schema{Type: "integer", Default: 100}},
		{Name: "offset", In: "query", Description: "Number of records to skip", Schema: &Schema{Type: "integer", Default: 0}},
	}

	// Add sort parameter with sortable field documentation
	if len(sortableFields) > 0 {
		params = append(params, Parameter{
			Name:        "order_by",
			In:          "query",
			Description: fmt.Sprintf("Field to sort by. Sortable fields: %s", strings.Join(sortableFields, ", ")),
			Schema:      &Schema{Type: "string", Enum: sortableFields},
		})
		params = append(params, Parameter{
			Name:        "order_desc",
			In:          "query",
			Description: "Sort in descending order (default: false)",
			Schema:      &Schema{Type: "boolean", Default: false},
		})
	}

	// Add filter parameters for filterable fields
	for _, field := range mod.Fields {
		if field.Internal {
			continue
		}
		if field.Lookup || field.Unique || field.Name == "id" {
			params = append(params, Parameter{
				Name:        field.Name,
				In:          "query",
				Description: fmt.Sprintf("Filter by %s (filterable)", field.Name),
				Schema:      g.fieldToSchema(field),
			})
		}
	}

	// Build description with filterable/sortable info
	description := fmt.Sprintf("Retrieve a list of %s records.", mod.Source.Name)
	if len(filterableFields) > 0 {
		description += fmt.Sprintf("\n\n**Filterable fields:** %s", strings.Join(filterableFields, ", "))
	}
	if len(sortableFields) > 0 {
		description += fmt.Sprintf("\n\n**Sortable fields:** %s", strings.Join(sortableFields, ", "))
	}

	path.Get = &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("List %s", mod.Plural),
		Description: description,
		OperationID: fmt.Sprintf("list%s", title),
		Parameters:  params,
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{Ref: "#/components/schemas/" + title + "List"}},
				},
			},
			"401": {Description: "Unauthorized"},
			"500": {Description: "Internal server error"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	spec.Paths[basePath] = path
}

// isBasicType returns true if the field type is a basic sortable type.
func isBasicType(t schema.FieldType) bool {
	switch t {
	case schema.FieldTypeString, schema.FieldTypeInt, schema.FieldTypeFloat,
		schema.FieldTypeBool, schema.FieldTypeEmail, schema.FieldTypeTimestamp,
		schema.FieldTypeEnum, schema.FieldTypeRef:
		return true
	default:
		return false
	}
}

// addGetPath adds get operation.
func (g *Generator) addGetPath(spec *Spec, mod convention.Derived, basePath, title string) {
	pathWithID := basePath + "/{id}"
	path := spec.Paths[pathWithID]

	path.Get = &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("Get %s by ID", mod.Source.Name),
		Description: fmt.Sprintf("Retrieve a single %s record by ID or lookup field", mod.Source.Name),
		OperationID: fmt.Sprintf("get%s", title),
		Parameters: []Parameter{
			{Name: "id", In: "path", Required: true, Description: "Record ID or lookup value", Schema: &Schema{Type: "string"}},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"data": {Ref: "#/components/schemas/" + title},
						},
					}},
				},
			},
			"404": {Description: "Record not found"},
			"401": {Description: "Unauthorized"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	spec.Paths[pathWithID] = path
}

// addCreatePath adds create operation.
func (g *Generator) addCreatePath(spec *Spec, mod convention.Derived, basePath, title string) {
	path := spec.Paths[basePath]

	path.Post = &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("Create %s", mod.Source.Name),
		Description: fmt.Sprintf("Create a new %s record", mod.Source.Name),
		OperationID: fmt.Sprintf("create%s", title),
		RequestBody: &RequestBody{
			Required:    true,
			Description: fmt.Sprintf("%s data to create", mod.Source.Name),
			Content: map[string]MediaType{
				"application/json": {Schema: &Schema{Ref: "#/components/schemas/" + title + "Create"}},
			},
		},
		Responses: map[string]Response{
			"201": {
				Description: "Record created successfully",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"id":   {Type: "string"},
							"data": {Ref: "#/components/schemas/" + title},
						},
					}},
				},
			},
			"400": {Description: "Invalid request data"},
			"401": {Description: "Unauthorized"},
			"409": {Description: "Conflict (duplicate unique field)"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	spec.Paths[basePath] = path
}

// addUpdatePath adds update operation.
func (g *Generator) addUpdatePath(spec *Spec, mod convention.Derived, basePath, title string) {
	pathWithID := basePath + "/{id}"
	path := spec.Paths[pathWithID]

	params := []Parameter{
		{Name: "id", In: "path", Required: true, Description: "Record ID", Schema: &Schema{Type: "string"}},
	}

	updateOp := &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("Update %s", mod.Source.Name),
		Description: fmt.Sprintf("Update an existing %s record", mod.Source.Name),
		OperationID: fmt.Sprintf("update%s", title),
		Parameters:  params,
		RequestBody: &RequestBody{
			Required:    true,
			Description: fmt.Sprintf("%s data to update", mod.Source.Name),
			Content: map[string]MediaType{
				"application/json": {Schema: &Schema{Ref: "#/components/schemas/" + title + "Update"}},
			},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Record updated successfully",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"id":   {Type: "string"},
							"data": {Ref: "#/components/schemas/" + title},
						},
					}},
				},
			},
			"400": {Description: "Invalid request data"},
			"401": {Description: "Unauthorized"},
			"404": {Description: "Record not found"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	path.Put = updateOp
	path.Patch = updateOp

	spec.Paths[pathWithID] = path
}

// addDeletePath adds delete operation.
func (g *Generator) addDeletePath(spec *Spec, mod convention.Derived, basePath, title string) {
	pathWithID := basePath + "/{id}"
	path := spec.Paths[pathWithID]

	path.Delete = &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("Delete %s", mod.Source.Name),
		Description: fmt.Sprintf("Delete a %s record", mod.Source.Name),
		OperationID: fmt.Sprintf("delete%s", title),
		Parameters: []Parameter{
			{Name: "id", In: "path", Required: true, Description: "Record ID", Schema: &Schema{Type: "string"}},
		},
		Responses: map[string]Response{
			"204": {Description: "Record deleted successfully"},
			"401": {Description: "Unauthorized"},
			"404": {Description: "Record not found"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	spec.Paths[pathWithID] = path
}

// addCustomActionPath adds custom action operation.
func (g *Generator) addCustomActionPath(spec *Spec, mod convention.Derived, action convention.DerivedAction, basePath, title string) {
	actionPath := basePath + "/{id}/" + action.Name
	path := spec.Paths[actionPath]

	// Build request body schema from action inputs
	var requestBody *RequestBody
	if len(action.Input) > 0 {
		inputSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}

		for _, input := range action.Input {
			inputSchema.Properties[input.Name] = &Schema{
				Type: string(input.Type),
			}
			if input.Required {
				inputSchema.Required = append(inputSchema.Required, input.Name)
			}
		}

		requestBody = &RequestBody{
			Description: action.Description,
			Content: map[string]MediaType{
				"application/json": {Schema: inputSchema},
			},
		}
	}

	path.Post = &Operation{
		Tags:        []string{mod.Source.Name},
		Summary:     fmt.Sprintf("%s - %s", action.Name, action.Description),
		Description: action.Description,
		OperationID: fmt.Sprintf("%s%s", action.Name, title),
		Parameters: []Parameter{
			{Name: "id", In: "path", Required: true, Description: "Record ID", Schema: &Schema{Type: "string"}},
		},
		RequestBody: requestBody,
		Responses: map[string]Response{
			"200": {
				Description: "Action executed successfully",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"id":   {Type: "string"},
							"data": {Ref: "#/components/schemas/" + title},
						},
					}},
				},
			},
			"400": {Description: "Invalid request"},
			"401": {Description: "Unauthorized"},
			"404": {Description: "Record not found"},
		},
		Security: []SecurityRequirement{{"bearerAuth": {}}, {"apiKey": {}}},
	}

	spec.Paths[actionPath] = path
}

// ToJSON converts the spec to JSON.
func (spec *Spec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

// ToJSONCompact converts the spec to compact JSON.
func (spec *Spec) ToJSONCompact() ([]byte, error) {
	return json.Marshal(spec)
}
