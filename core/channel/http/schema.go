// Package http provides schema introspection endpoints.
// These endpoints enable clients to discover available modules, fields, and actions at runtime.
package http

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/go-chi/chi/v5"
)

// SchemaHandler handles schema introspection requests.
type SchemaHandler struct {
	modules map[string]convention.Derived
}

// NewSchemaHandler creates a new schema handler.
func NewSchemaHandler(modules map[string]convention.Derived) *SchemaHandler {
	return &SchemaHandler{
		modules: modules,
	}
}

// Routes returns a router with all schema routes.
func (h *SchemaHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.listModules)
	r.Get("/{module}", h.getModuleSchema)
	return r
}

// listModules handles GET /mod/_schema
func (h *SchemaHandler) listModules(w http.ResponseWriter, r *http.Request) {
	var summaries []schema.ModuleSummary

	for name, mod := range h.modules {
		summary := schema.ModuleSummary{
			Name:        name,
			Plural:      mod.Plural,
			Description: mod.Source.Meta.Description,
			Version:     mod.Source.Meta.Version,
		}
		summaries = append(summaries, summary)
	}

	// Sort by name for consistent output
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	// Return as JSON:API meta response
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
		"modules": summaries,
		"count":   len(summaries),
	})
}

// getModuleSchema handles GET /mod/_schema/{module}
func (h *SchemaHandler) getModuleSchema(w http.ResponseWriter, r *http.Request) {
	moduleName := chi.URLParam(r, "module")

	mod, ok := h.modules[moduleName]
	if !ok {
		jsonapi.WriteNotFound(w, "module")
		return
	}

	resp := h.buildModuleSchema(mod)
	// Return as JSON:API meta response (schema is metadata, not a resource)
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
		"module":      resp.Module,
		"plural":      resp.Plural,
		"description": resp.Description,
		"version":     resp.Version,
		"table":       resp.Table,
		"fields":      resp.Fields,
		"actions":     resp.Actions,
		"lookups":     resp.Lookups,
		"endpoints":   resp.Endpoints,
		"depends":     resp.Depends,
	})
}

// buildModuleSchema converts convention.Derived to schema.ModuleSchemaResponse
func (h *SchemaHandler) buildModuleSchema(mod convention.Derived) schema.ModuleSchemaResponse {
	// Build base path from plural (single source of truth)
	basePath := "/" + mod.Plural

	resp := schema.ModuleSchemaResponse{
		Module:      mod.Source.Name,
		Plural:      mod.Plural,
		Description: mod.Source.Meta.Description,
		Version:     mod.Source.Meta.Version,
		Table:       mod.Table,
		Fields:      h.buildFields(mod.Fields),
		Actions:     h.buildActions(mod.Actions, basePath),
		Lookups:     mod.Lookups,
		Endpoints:   h.buildEndpoints(mod.Actions, basePath),
		Depends:     mod.Source.Meta.Depends,
	}

	return resp
}

// buildFields converts convention.DerivedField to schema.FieldSchema
func (h *SchemaHandler) buildFields(fields []convention.DerivedField) []schema.FieldSchema {
	result := make([]schema.FieldSchema, 0, len(fields))

	for _, f := range fields {
		// Determine if field is filterable (indexed fields)
		// Fields are filterable if they are: lookup, unique, or id
		filterable := f.Lookup || f.Unique || f.Name == "id"

		// Determine if field is sortable (non-internal, basic types)
		sortable := !f.Internal && h.isSortableType(f.Type)

		fs := schema.FieldSchema{
			Name:        f.Name,
			Type:        string(f.Type),
			Required:    f.Required,
			Unique:      f.Unique,
			Lookup:      f.Lookup,
			Filterable:  filterable,
			Sortable:    sortable,
			Values:      f.Values,
			Ref:         f.Ref,
			Default:     f.Default,
			Internal:    f.Internal,
			Implicit:    f.Implicit,
			SQLType:     f.SQLType,
			Constraints: h.buildConstraints(f.Constraints),
			Description: f.Description,
		}
		result = append(result, fs)
	}

	return result
}

// isSortableType returns true if the field type supports sorting
func (h *SchemaHandler) isSortableType(t schema.FieldType) bool {
	switch t {
	case schema.FieldTypeString,
		schema.FieldTypeInt,
		schema.FieldTypeFloat,
		schema.FieldTypeBool,
		schema.FieldTypeEmail,
		schema.FieldTypeTimestamp,
		schema.FieldTypeUUID,
		schema.FieldTypeEnum,
		schema.FieldTypeRef:
		return true
	default:
		return false
	}
}

// buildConstraints converts schema.Constraint to schema.ConstraintSchema
func (h *SchemaHandler) buildConstraints(constraints []schema.Constraint) []schema.ConstraintSchema {
	if len(constraints) == 0 {
		return nil
	}

	result := make([]schema.ConstraintSchema, 0, len(constraints))
	for _, c := range constraints {
		// Generate human-readable description if not provided
		message := c.Message
		if message == "" {
			message = h.generateConstraintDescription(c)
		}

		cs := schema.ConstraintSchema{
			Type:    string(c.Type),
			Value:   c.Value,
			Message: message,
		}
		result = append(result, cs)
	}
	return result
}

// generateConstraintDescription creates a human-readable description for a constraint.
func (h *SchemaHandler) generateConstraintDescription(c schema.Constraint) string {
	switch c.Type {
	case schema.ConstraintMin:
		return fmt.Sprintf("Value must be at least %v", c.Value)
	case schema.ConstraintMax:
		return fmt.Sprintf("Value must be at most %v", c.Value)
	case schema.ConstraintMinLength:
		return fmt.Sprintf("Must be at least %v characters", c.Value)
	case schema.ConstraintMaxLength:
		return fmt.Sprintf("Must be at most %v characters", c.Value)
	case schema.ConstraintPattern:
		return fmt.Sprintf("Must match pattern: %v", c.Value)
	case schema.ConstraintRefExists:
		return fmt.Sprintf("Referenced %v must exist", c.Value)
	case schema.ConstraintNotEmpty:
		return "Must not be empty"
	case schema.ConstraintOneOf:
		if vals, ok := c.Value.([]string); ok {
			return fmt.Sprintf("Must be one of: %v", vals)
		}
		return fmt.Sprintf("Must be one of: %v", c.Value)
	default:
		return ""
	}
}

// buildActions converts convention.DerivedAction to schema.ActionSchema
func (h *SchemaHandler) buildActions(actions []convention.DerivedAction, basePath string) []schema.ActionSchema {
	result := make([]schema.ActionSchema, 0, len(actions))

	for _, a := range actions {
		as := schema.ActionSchema{
			Name:        a.Name,
			Type:        a.Type.String(),
			Description: a.Description,
			Input:       h.buildInputs(a.Input),
			Output:      a.Output,
			Auth:        a.Auth,
			Confirm:     a.Confirm,
			Implicit:    a.Implicit,
			HTTP:        h.buildHTTPInfo(a, basePath),
		}
		result = append(result, as)
	}

	return result
}

// buildInputs converts convention.ActionInput to schema.InputSchema
func (h *SchemaHandler) buildInputs(inputs []convention.ActionInput) []schema.InputSchema {
	if len(inputs) == 0 {
		return nil
	}

	result := make([]schema.InputSchema, 0, len(inputs))

	for _, i := range inputs {
		defaultStr, _ := i.Default.(string)
		is := schema.InputSchema{
			Name:       i.Name,
			Field:      i.Field,
			Type:       string(i.Type),
			Required:   i.Required,
			Default:    defaultStr,
			Prompt:     i.Prompt,
			PromptText: i.PromptText,
		}
		result = append(result, is)
	}

	return result
}

// buildHTTPInfo derives HTTP method and path for an action
func (h *SchemaHandler) buildHTTPInfo(action convention.DerivedAction, basePath string) *schema.HTTPInfo {
	switch action.Type {
	case schema.ActionTypeList:
		return &schema.HTTPInfo{
			Method: "GET",
			Path:   basePath,
		}
	case schema.ActionTypeGet:
		return &schema.HTTPInfo{
			Method: "GET",
			Path:   basePath + "/{id}",
		}
	case schema.ActionTypeCreate:
		return &schema.HTTPInfo{
			Method: "POST",
			Path:   basePath,
		}
	case schema.ActionTypeUpdate:
		return &schema.HTTPInfo{
			Method: "PUT",
			Path:   basePath + "/{id}",
		}
	case schema.ActionTypeDelete:
		return &schema.HTTPInfo{
			Method: "DELETE",
			Path:   basePath + "/{id}",
		}
	case schema.ActionTypeCustom:
		return &schema.HTTPInfo{
			Method: "POST",
			Path:   basePath + "/{id}/" + action.Name,
		}
	default:
		return nil
	}
}

// buildEndpoints creates the full endpoint list for a module
func (h *SchemaHandler) buildEndpoints(actions []convention.DerivedAction, basePath string) []schema.EndpointSchema {
	var endpoints []schema.EndpointSchema

	for _, a := range actions {
		httpInfo := h.buildHTTPInfo(a, basePath)
		if httpInfo == nil {
			continue
		}

		endpoint := schema.EndpointSchema{
			Action: a.Name,
			Method: httpInfo.Method,
			Path:   httpInfo.Path,
			Auth:   a.Auth,
		}
		endpoints = append(endpoints, endpoint)

		// Update action also supports PATCH
		if a.Type == schema.ActionTypeUpdate {
			endpoints = append(endpoints, schema.EndpointSchema{
				Action: a.Name,
				Method: "PATCH",
				Path:   httpInfo.Path,
				Auth:   a.Auth,
			})
		}
	}

	return endpoints
}

