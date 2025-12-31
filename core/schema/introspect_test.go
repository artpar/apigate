package schema

import (
	"testing"
)

func TestModuleListResponseStruct(t *testing.T) {
	resp := ModuleListResponse{
		Modules: []ModuleSummary{
			{Name: "user", Plural: "users", Description: "User module"},
			{Name: "plan", Plural: "plans", Version: "1.0"},
		},
		Count: 2,
	}

	if len(resp.Modules) != 2 {
		t.Error("ModuleListResponse.Modules not set correctly")
	}
	if resp.Count != 2 {
		t.Error("ModuleListResponse.Count not set correctly")
	}
}

func TestModuleSummaryStruct(t *testing.T) {
	summary := ModuleSummary{
		Name:        "user",
		Plural:      "users",
		Description: "User management module",
		Version:     "2.0.0",
	}

	if summary.Name != "user" {
		t.Error("ModuleSummary.Name not set correctly")
	}
	if summary.Plural != "users" {
		t.Error("ModuleSummary.Plural not set correctly")
	}
	if summary.Description != "User management module" {
		t.Error("ModuleSummary.Description not set correctly")
	}
	if summary.Version != "2.0.0" {
		t.Error("ModuleSummary.Version not set correctly")
	}
}

func TestModuleSchemaResponseStruct(t *testing.T) {
	resp := ModuleSchemaResponse{
		Module:      "user",
		Plural:      "users",
		Description: "User management",
		Version:     "1.0.0",
		Table:       "users",
		Fields: []FieldSchema{
			{Name: "id", Type: "string", Implicit: true},
			{Name: "email", Type: "email", Required: true, Unique: true},
		},
		Actions: []ActionSchema{
			{Name: "list", Type: "list", Implicit: true},
			{Name: "activate", Type: "custom"},
		},
		Lookups:   []string{"id", "email"},
		Endpoints: []EndpointSchema{{Action: "list", Method: "GET", Path: "/users"}},
		Depends:   []string{"auth"},
	}

	if resp.Module != "user" {
		t.Error("ModuleSchemaResponse.Module not set correctly")
	}
	if resp.Plural != "users" {
		t.Error("ModuleSchemaResponse.Plural not set correctly")
	}
	if resp.Table != "users" {
		t.Error("ModuleSchemaResponse.Table not set correctly")
	}
	if len(resp.Fields) != 2 {
		t.Error("ModuleSchemaResponse.Fields not set correctly")
	}
	if len(resp.Actions) != 2 {
		t.Error("ModuleSchemaResponse.Actions not set correctly")
	}
	if len(resp.Lookups) != 2 {
		t.Error("ModuleSchemaResponse.Lookups not set correctly")
	}
	if len(resp.Endpoints) != 1 {
		t.Error("ModuleSchemaResponse.Endpoints not set correctly")
	}
	if len(resp.Depends) != 1 {
		t.Error("ModuleSchemaResponse.Depends not set correctly")
	}
}

func TestFieldSchemaStruct(t *testing.T) {
	field := FieldSchema{
		Name:       "status",
		Type:       "enum",
		Required:   true,
		Unique:     false,
		Lookup:     true,
		Filterable: true,
		Sortable:   true,
		Values:     []string{"active", "inactive", "pending"},
		Ref:        "",
		Default:    "pending",
		Internal:   false,
		Implicit:   false,
		SQLType:    "TEXT",
		Constraints: []ConstraintSchema{
			{Type: "one_of", Value: []string{"active", "inactive", "pending"}},
		},
		Description: "User status",
	}

	if field.Name != "status" {
		t.Error("FieldSchema.Name not set correctly")
	}
	if field.Type != "enum" {
		t.Error("FieldSchema.Type not set correctly")
	}
	if !field.Required {
		t.Error("FieldSchema.Required not set correctly")
	}
	if !field.Lookup {
		t.Error("FieldSchema.Lookup not set correctly")
	}
	if !field.Filterable {
		t.Error("FieldSchema.Filterable not set correctly")
	}
	if !field.Sortable {
		t.Error("FieldSchema.Sortable not set correctly")
	}
	if len(field.Values) != 3 {
		t.Error("FieldSchema.Values not set correctly")
	}
	if field.Default != "pending" {
		t.Error("FieldSchema.Default not set correctly")
	}
	if field.SQLType != "TEXT" {
		t.Error("FieldSchema.SQLType not set correctly")
	}
	if len(field.Constraints) != 1 {
		t.Error("FieldSchema.Constraints not set correctly")
	}
	if field.Description != "User status" {
		t.Error("FieldSchema.Description not set correctly")
	}
}

func TestFieldSchemaWithRef(t *testing.T) {
	field := FieldSchema{
		Name: "plan_id",
		Type: "ref",
		Ref:  "plan",
	}

	if field.Ref != "plan" {
		t.Error("FieldSchema.Ref not set correctly")
	}
}

func TestFieldSchemaImplicit(t *testing.T) {
	fields := []FieldSchema{
		{Name: "id", Type: "uuid", Implicit: true},
		{Name: "created_at", Type: "timestamp", Implicit: true},
		{Name: "updated_at", Type: "timestamp", Implicit: true},
	}

	for _, f := range fields {
		if !f.Implicit {
			t.Errorf("Field %s should be implicit", f.Name)
		}
	}
}

func TestFieldSchemaInternal(t *testing.T) {
	field := FieldSchema{
		Name:     "password_hash",
		Type:     "string",
		Internal: true,
	}

	if !field.Internal {
		t.Error("FieldSchema.Internal not set correctly")
	}
}

func TestConstraintSchemaStruct(t *testing.T) {
	tests := []struct {
		name       string
		constraint ConstraintSchema
	}{
		{
			name: "min constraint",
			constraint: ConstraintSchema{
				Type:    "min",
				Value:   0,
				Message: "must be non-negative",
			},
		},
		{
			name: "max constraint",
			constraint: ConstraintSchema{
				Type:  "max",
				Value: 100,
			},
		},
		{
			name: "min_length constraint",
			constraint: ConstraintSchema{
				Type:  "min_length",
				Value: 3,
			},
		},
		{
			name: "max_length constraint",
			constraint: ConstraintSchema{
				Type:  "max_length",
				Value: 255,
			},
		},
		{
			name: "pattern constraint",
			constraint: ConstraintSchema{
				Type:    "pattern",
				Value:   "^[a-z]+$",
				Message: "lowercase letters only",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.constraint
			if c.Type == "" {
				t.Error("ConstraintSchema.Type should be set")
			}
		})
	}
}

func TestActionSchemaStruct(t *testing.T) {
	action := ActionSchema{
		Name:        "activate",
		Type:        "custom",
		Description: "Activate the user account",
		Input: []InputSchema{
			{Name: "reason", Type: "string", Required: false},
		},
		Output:   []string{"id", "status", "activated_at"},
		Auth:     "admin",
		Confirm:  true,
		Implicit: false,
		HTTP: &HTTPInfo{
			Method: "POST",
			Path:   "/users/{id}/activate",
		},
	}

	if action.Name != "activate" {
		t.Error("ActionSchema.Name not set correctly")
	}
	if action.Type != "custom" {
		t.Error("ActionSchema.Type not set correctly")
	}
	if action.Description != "Activate the user account" {
		t.Error("ActionSchema.Description not set correctly")
	}
	if len(action.Input) != 1 {
		t.Error("ActionSchema.Input not set correctly")
	}
	if len(action.Output) != 3 {
		t.Error("ActionSchema.Output not set correctly")
	}
	if action.Auth != "admin" {
		t.Error("ActionSchema.Auth not set correctly")
	}
	if !action.Confirm {
		t.Error("ActionSchema.Confirm not set correctly")
	}
	if action.HTTP == nil {
		t.Error("ActionSchema.HTTP not set correctly")
	}
	if action.HTTP.Method != "POST" {
		t.Error("ActionSchema.HTTP.Method not set correctly")
	}
}

func TestActionSchemaImplicit(t *testing.T) {
	actions := []ActionSchema{
		{Name: "list", Type: "list", Implicit: true},
		{Name: "get", Type: "get", Implicit: true},
		{Name: "create", Type: "create", Implicit: true},
		{Name: "update", Type: "update", Implicit: true},
		{Name: "delete", Type: "delete", Implicit: true},
	}

	for _, a := range actions {
		if !a.Implicit {
			t.Errorf("Action %s should be implicit", a.Name)
		}
	}
}

func TestInputSchemaStruct(t *testing.T) {
	input := InputSchema{
		Name:       "email",
		Field:      "email",
		Type:       "email",
		Required:   true,
		Default:    "",
		Prompt:     true,
		PromptText: "Enter your email:",
	}

	if input.Name != "email" {
		t.Error("InputSchema.Name not set correctly")
	}
	if input.Field != "email" {
		t.Error("InputSchema.Field not set correctly")
	}
	if input.Type != "email" {
		t.Error("InputSchema.Type not set correctly")
	}
	if !input.Required {
		t.Error("InputSchema.Required not set correctly")
	}
	if !input.Prompt {
		t.Error("InputSchema.Prompt not set correctly")
	}
	if input.PromptText != "Enter your email:" {
		t.Error("InputSchema.PromptText not set correctly")
	}
}

func TestHTTPInfoStruct(t *testing.T) {
	info := HTTPInfo{
		Method: "GET",
		Path:   "/users",
	}

	if info.Method != "GET" {
		t.Error("HTTPInfo.Method not set correctly")
	}
	if info.Path != "/users" {
		t.Error("HTTPInfo.Path not set correctly")
	}
}

func TestEndpointSchemaStruct(t *testing.T) {
	endpoint := EndpointSchema{
		Action: "list",
		Method: "GET",
		Path:   "/users",
		Auth:   "public",
	}

	if endpoint.Action != "list" {
		t.Error("EndpointSchema.Action not set correctly")
	}
	if endpoint.Method != "GET" {
		t.Error("EndpointSchema.Method not set correctly")
	}
	if endpoint.Path != "/users" {
		t.Error("EndpointSchema.Path not set correctly")
	}
	if endpoint.Auth != "public" {
		t.Error("EndpointSchema.Auth not set correctly")
	}
}

func TestEndpointSchemaHTTPMethods(t *testing.T) {
	endpoints := []EndpointSchema{
		{Action: "list", Method: "GET", Path: "/users"},
		{Action: "create", Method: "POST", Path: "/users"},
		{Action: "get", Method: "GET", Path: "/users/{id}"},
		{Action: "update", Method: "PATCH", Path: "/users/{id}"},
		{Action: "delete", Method: "DELETE", Path: "/users/{id}"},
	}

	methods := []string{"GET", "POST", "GET", "PATCH", "DELETE"}
	for i, ep := range endpoints {
		if ep.Method != methods[i] {
			t.Errorf("Endpoint %s has method %s, want %s", ep.Action, ep.Method, methods[i])
		}
	}
}
