package openapi

import (
	"encoding/json"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Helper to create a test module
func createTestModule(name string, fields map[string]schema.Field, actions map[string]schema.Action) schema.Module {
	return schema.Module{
		Name:    name,
		Schema:  fields,
		Actions: actions,
		Meta: schema.ModuleMeta{
			Description: "Test module for " + name,
		},
	}
}

// Helper to derive a module
func deriveModule(mod schema.Module) convention.Derived {
	return convention.Derive(mod)
}

// TestNewGenerator tests the NewGenerator function
func TestNewGenerator(t *testing.T) {
	modules := make(map[string]convention.Derived)
	gen := NewGenerator(modules)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}

	if gen.modules == nil {
		t.Error("modules map should not be nil")
	}

	if gen.info.Title != "APIGate API" {
		t.Errorf("expected default title 'APIGate API', got %q", gen.info.Title)
	}

	if gen.info.Version != "1.0.0" {
		t.Errorf("expected default version '1.0.0', got %q", gen.info.Version)
	}
}

// TestSetInfo tests the SetInfo method
func TestSetInfo(t *testing.T) {
	gen := NewGenerator(nil)

	info := Info{
		Title:       "Custom API",
		Description: "A custom API description",
		Version:     "2.0.0",
		Contact: &Contact{
			Name:  "Test Contact",
			Email: "test@example.com",
			URL:   "https://example.com",
		},
		License: &License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	}

	gen.SetInfo(info)

	if gen.info.Title != "Custom API" {
		t.Errorf("expected title 'Custom API', got %q", gen.info.Title)
	}

	if gen.info.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", gen.info.Version)
	}

	if gen.info.Contact == nil || gen.info.Contact.Name != "Test Contact" {
		t.Error("contact info not set correctly")
	}

	if gen.info.License == nil || gen.info.License.Name != "MIT" {
		t.Error("license info not set correctly")
	}
}

// TestAddServer tests the AddServer method
func TestAddServer(t *testing.T) {
	gen := NewGenerator(nil)

	gen.AddServer("https://api.example.com", "Production")
	gen.AddServer("https://staging.example.com", "Staging")

	if len(gen.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(gen.servers))
	}

	if gen.servers[0].URL != "https://api.example.com" {
		t.Errorf("expected first server URL 'https://api.example.com', got %q", gen.servers[0].URL)
	}

	if gen.servers[1].Description != "Staging" {
		t.Errorf("expected second server description 'Staging', got %q", gen.servers[1].Description)
	}
}

// TestGenerateEmptyModules tests generating spec with no modules
func TestGenerateEmptyModules(t *testing.T) {
	modules := make(map[string]convention.Derived)
	gen := NewGenerator(modules)

	spec := gen.Generate()

	if spec == nil {
		t.Fatal("Generate returned nil")
	}

	if spec.OpenAPI != "3.0.3" {
		t.Errorf("expected OpenAPI version '3.0.3', got %q", spec.OpenAPI)
	}

	if spec.Paths == nil {
		t.Error("Paths should not be nil")
	}

	if spec.Components.Schemas == nil {
		t.Error("Components.Schemas should not be nil")
	}

	// Check security schemes
	if _, ok := spec.Components.SecuritySchemes["bearerAuth"]; !ok {
		t.Error("bearerAuth security scheme should be present")
	}

	if _, ok := spec.Components.SecuritySchemes["apiKey"]; !ok {
		t.Error("apiKey security scheme should be present")
	}
}

// TestGenerateWithModule tests generating spec with a simple module
func TestGenerateWithModule(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail, Unique: true, Lookup: true},
		"name":  {Type: schema.FieldTypeString},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	// Check tags
	if len(spec.Tags) == 0 {
		t.Error("expected at least one tag")
	}

	foundUserTag := false
	for _, tag := range spec.Tags {
		if tag.Name == "user" {
			foundUserTag = true
			break
		}
	}
	if !foundUserTag {
		t.Error("expected 'user' tag")
	}

	// Check schemas
	if _, ok := spec.Components.Schemas["User"]; !ok {
		t.Error("expected 'User' schema")
	}

	if _, ok := spec.Components.Schemas["UserCreate"]; !ok {
		t.Error("expected 'UserCreate' schema")
	}

	if _, ok := spec.Components.Schemas["UserUpdate"]; !ok {
		t.Error("expected 'UserUpdate' schema")
	}

	if _, ok := spec.Components.Schemas["UserList"]; !ok {
		t.Error("expected 'UserList' schema")
	}
}

// TestFieldToSchema tests fieldToSchema for all field types
func TestFieldToSchema(t *testing.T) {
	tests := []struct {
		name           string
		field          convention.DerivedField
		expectedType   string
		expectedFormat string
	}{
		{
			name:         "string field",
			field:        convention.DerivedField{Name: "test", Type: schema.FieldTypeString},
			expectedType: "string",
		},
		{
			name:         "int field",
			field:        convention.DerivedField{Name: "count", Type: schema.FieldTypeInt},
			expectedType: "integer",
		},
		{
			name:           "float field",
			field:          convention.DerivedField{Name: "price", Type: schema.FieldTypeFloat},
			expectedType:   "number",
			expectedFormat: "float",
		},
		{
			name:         "bool field",
			field:        convention.DerivedField{Name: "active", Type: schema.FieldTypeBool},
			expectedType: "boolean",
		},
		{
			name:           "email field",
			field:          convention.DerivedField{Name: "email", Type: schema.FieldTypeEmail},
			expectedType:   "string",
			expectedFormat: "email",
		},
		{
			name:           "url field",
			field:          convention.DerivedField{Name: "website", Type: schema.FieldTypeURL},
			expectedType:   "string",
			expectedFormat: "uri",
		},
		{
			name:           "timestamp field",
			field:          convention.DerivedField{Name: "created", Type: schema.FieldTypeTimestamp},
			expectedType:   "string",
			expectedFormat: "date-time",
		},
		{
			name:         "duration field",
			field:        convention.DerivedField{Name: "timeout", Type: schema.FieldTypeDuration},
			expectedType: "string",
		},
		{
			name:           "uuid field",
			field:          convention.DerivedField{Name: "id", Type: schema.FieldTypeUUID},
			expectedType:   "string",
			expectedFormat: "uuid",
		},
		{
			name:         "json field",
			field:        convention.DerivedField{Name: "config", Type: schema.FieldTypeJSON},
			expectedType: "object",
		},
		{
			name:           "bytes field",
			field:          convention.DerivedField{Name: "data", Type: schema.FieldTypeBytes},
			expectedType:   "string",
			expectedFormat: "byte",
		},
		{
			name:           "secret field",
			field:          convention.DerivedField{Name: "password", Type: schema.FieldTypeSecret},
			expectedType:   "string",
			expectedFormat: "password",
		},
		{
			name:         "enum field",
			field:        convention.DerivedField{Name: "status", Type: schema.FieldTypeEnum, Values: []string{"active", "inactive"}},
			expectedType: "string",
		},
		{
			name:         "ref field",
			field:        convention.DerivedField{Name: "user_id", Type: schema.FieldTypeRef, Ref: "user"},
			expectedType: "string",
		},
		{
			name:         "strings field",
			field:        convention.DerivedField{Name: "tags", Type: schema.FieldTypeStrings},
			expectedType: "array",
		},
		{
			name:         "ints field",
			field:        convention.DerivedField{Name: "scores", Type: schema.FieldTypeInts},
			expectedType: "array",
		},
		{
			name:         "unknown field type",
			field:        convention.DerivedField{Name: "unknown", Type: "unknown_type"},
			expectedType: "string",
		},
	}

	gen := NewGenerator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := gen.fieldToSchema(tt.field)

			if s.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, s.Type)
			}

			if tt.expectedFormat != "" && s.Format != tt.expectedFormat {
				t.Errorf("expected format %q, got %q", tt.expectedFormat, s.Format)
			}
		})
	}
}

// TestFieldToSchemaWithEnum tests enum field schema generation
func TestFieldToSchemaWithEnum(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name:   "status",
		Type:   schema.FieldTypeEnum,
		Values: []string{"pending", "active", "completed"},
	}

	s := gen.fieldToSchema(field)

	if s.Type != "string" {
		t.Errorf("expected type 'string', got %q", s.Type)
	}

	if len(s.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(s.Enum))
	}

	if s.Example != "pending" {
		t.Errorf("expected example 'pending', got %v", s.Example)
	}
}

// TestFieldToSchemaWithDefault tests field schema with default value
func TestFieldToSchemaWithDefault(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name:    "retries",
		Type:    schema.FieldTypeInt,
		Default: 3,
	}

	s := gen.fieldToSchema(field)

	if s.Default != 3 {
		t.Errorf("expected default 3, got %v", s.Default)
	}

	// Default should be used as example
	if s.Example != 3 {
		t.Errorf("expected example 3, got %v", s.Example)
	}
}

// TestFieldToSchemaWithRef tests reference field schema
func TestFieldToSchemaWithRef(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name: "user_id",
		Type: schema.FieldTypeRef,
		Ref:  "user",
	}

	s := gen.fieldToSchema(field)

	if s.Type != "string" {
		t.Errorf("expected type 'string', got %q", s.Type)
	}

	if s.Description == "" {
		t.Error("expected description for ref field")
	}
}

// TestGenerateExample tests contextual example generation
func TestGenerateExample(t *testing.T) {
	gen := NewGenerator(nil)

	tests := []struct {
		fieldName    string
		fieldType    schema.FieldType
		expectedType interface{}
	}{
		{"username", schema.FieldTypeString, "John Doe"},              // contains "name" and "user"
		{"user_email", schema.FieldTypeString, "user@example.com"},    // contains "email"
		{"api_url", schema.FieldTypeString, "https://api.example.com"}, // contains "url"
		{"base_path", schema.FieldTypeString, "/api/v1/resource"},     // contains "path"
		{"http_method", schema.FieldTypeString, "GET"},                // contains "method"
		{"server_port", schema.FieldTypeInt, 8080},                    // contains "port"
		{"server_host", schema.FieldTypeString, "localhost"},          // contains "host"
		{"request_timeout", schema.FieldTypeString, "30s"},            // contains "timeout"
		{"max_limit", schema.FieldTypeInt, 100},                       // contains "limit"
		{"total_count", schema.FieldTypeInt, 10},                      // contains "count"
		{"request_rate", schema.FieldTypeInt, 60},                     // contains "rate"
		{"unit_price", schema.FieldTypeFloat, 999},                    // contains "price"
		{"item_description", schema.FieldTypeString, "A brief description of this item"},
		{"api_key", schema.FieldTypeString, "ak_example_key_prefix"}, // contains "key"
		{"client_secret", schema.FieldTypeString, "********"},        // contains "secret"
		{"record_id", schema.FieldTypeUUID, "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := convention.DerivedField{
				Name: tt.fieldName,
				Type: tt.fieldType,
			}
			example := gen.generateExample(field, "default")

			if example != tt.expectedType {
				t.Errorf("expected example %v, got %v", tt.expectedType, example)
			}
		})
	}
}

// TestApplyConstraint tests constraint application to schema
func TestApplyConstraint(t *testing.T) {
	gen := NewGenerator(nil)

	t.Run("min_length constraint", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMinLength,
			Value: 5,
		})

		if s.MinLength == nil || *s.MinLength != 5 {
			t.Error("min_length not applied correctly")
		}
	})

	t.Run("max_length constraint", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMaxLength,
			Value: 100,
		})

		if s.MaxLength == nil || *s.MaxLength != 100 {
			t.Error("max_length not applied correctly")
		}
	})

	t.Run("min constraint with float64", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMin,
			Value: 10.5,
		})

		if s.Minimum == nil || *s.Minimum != 10.5 {
			t.Error("min not applied correctly")
		}
	})

	t.Run("min constraint with int", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMin,
			Value: 10,
		})

		if s.Minimum == nil || *s.Minimum != 10.0 {
			t.Error("min not applied correctly for int")
		}
	})

	t.Run("max constraint with float64", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMax,
			Value: 100.5,
		})

		if s.Maximum == nil || *s.Maximum != 100.5 {
			t.Error("max not applied correctly")
		}
	})

	t.Run("max constraint with int", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMax,
			Value: 100,
		})

		if s.Maximum == nil || *s.Maximum != 100.0 {
			t.Error("max not applied correctly for int")
		}
	})

	t.Run("pattern constraint", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintPattern,
			Value: "^[a-z]+$",
		})

		if s.Pattern != "^[a-z]+$" {
			t.Error("pattern not applied correctly")
		}
	})

	t.Run("constraint with invalid value type", func(t *testing.T) {
		s := &Schema{}
		gen.applyConstraint(s, schema.Constraint{
			Type:  schema.ConstraintMinLength,
			Value: "not-an-int",
		})

		// Should not crash, just skip
		if s.MinLength != nil {
			t.Error("should not apply constraint with invalid value")
		}
	})
}

// TestFieldToSchemaWithConstraints tests field schema with constraints
func TestFieldToSchemaWithConstraints(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name: "password",
		Type: schema.FieldTypeString,
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintMinLength, Value: 8},
			{Type: schema.ConstraintMaxLength, Value: 128},
			{Type: schema.ConstraintPattern, Value: "^[A-Za-z0-9]+$"},
		},
	}

	s := gen.fieldToSchema(field)

	if s.MinLength == nil || *s.MinLength != 8 {
		t.Error("min_length constraint not applied")
	}

	if s.MaxLength == nil || *s.MaxLength != 128 {
		t.Error("max_length constraint not applied")
	}

	if s.Pattern != "^[A-Za-z0-9]+$" {
		t.Error("pattern constraint not applied")
	}
}

// TestBuildCreateSchema tests buildCreateSchema
func TestBuildCreateSchema(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail, Required: boolPtr(true)},
		"name":  {Type: schema.FieldTypeString},
		"role":  {Type: schema.FieldTypeString, Internal: true},
	}, nil)

	derived := deriveModule(userModule)
	gen := NewGenerator(nil)

	s := gen.buildCreateSchema(derived)

	if s.Type != "object" {
		t.Errorf("expected type 'object', got %q", s.Type)
	}

	// Should have email and name, but not role (internal)
	if _, ok := s.Properties["email"]; !ok {
		t.Error("expected email property")
	}

	if _, ok := s.Properties["name"]; !ok {
		t.Error("expected name property")
	}

	if _, ok := s.Properties["role"]; ok {
		t.Error("internal field 'role' should not be in create schema")
	}

	// Check required fields
	hasEmailRequired := false
	for _, r := range s.Required {
		if r == "email" {
			hasEmailRequired = true
		}
	}
	if !hasEmailRequired {
		t.Error("email should be in required list")
	}
}

// TestBuildUpdateSchema tests buildUpdateSchema
func TestBuildUpdateSchema(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail, Required: boolPtr(true)},
		"name":  {Type: schema.FieldTypeString},
		"hash":  {Type: schema.FieldTypeSecret},
	}, nil)

	derived := deriveModule(userModule)
	gen := NewGenerator(nil)

	s := gen.buildUpdateSchema(derived)

	if s.Type != "object" {
		t.Errorf("expected type 'object', got %q", s.Type)
	}

	// Should have email and name, but not hash (internal/secret)
	if _, ok := s.Properties["email"]; !ok {
		t.Error("expected email property")
	}

	if _, ok := s.Properties["name"]; !ok {
		t.Error("expected name property")
	}

	// Secret fields are internal
	if _, ok := s.Properties["hash"]; ok {
		t.Error("secret field 'hash' should not be in update schema")
	}

	// Update schema should not have required fields
	if len(s.Required) > 0 {
		t.Error("update schema should not have required fields")
	}
}

// TestBuildResponseSchema tests buildResponseSchema
func TestBuildResponseSchema(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
		"name":  {Type: schema.FieldTypeString},
		"hash":  {Type: schema.FieldTypeSecret},
	}, nil)

	derived := deriveModule(userModule)
	gen := NewGenerator(nil)

	s := gen.buildResponseSchema(derived)

	if s.Type != "object" {
		t.Errorf("expected type 'object', got %q", s.Type)
	}

	// Should have email and name
	if _, ok := s.Properties["email"]; !ok {
		t.Error("expected email property")
	}

	if _, ok := s.Properties["name"]; !ok {
		t.Error("expected name property")
	}

	// Secret/internal fields should not be in response
	if _, ok := s.Properties["hash"]; ok {
		t.Error("secret field 'hash' should not be in response schema")
	}
}

// TestGenerateWithCustomAction tests custom action path generation
func TestGenerateWithCustomAction(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email":  {Type: schema.FieldTypeEmail},
		"status": {Type: schema.FieldTypeEnum, Values: []string{"active", "inactive"}},
	}, map[string]schema.Action{
		"activate": {
			Set:         map[string]string{"status": "active"},
			Description: "Activate the user",
			Input: []schema.ActionInput{
				{Name: "reason", Type: "string", Required: true},
			},
		},
	})

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	// Check for custom action path
	activatePath := "/mod/users/{id}/activate"
	pathItem, ok := spec.Paths[activatePath]
	if !ok {
		t.Errorf("expected path %q to exist", activatePath)
	}

	if pathItem.Post == nil {
		t.Error("expected POST operation for activate action")
	}

	if pathItem.Post.RequestBody == nil {
		t.Error("expected request body for action with inputs")
	}
}

// TestGenerateWithCustomBasePath tests custom base path
func TestGenerateWithCustomBasePath(t *testing.T) {
	userModule := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"email": {Type: schema.FieldTypeEmail},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{
				Serve: schema.HTTPServe{
					BasePath: "/api/v2/users",
				},
			},
		},
	}

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	// Check for custom base path
	if _, ok := spec.Paths["/api/v2/users"]; !ok {
		t.Error("expected custom base path /api/v2/users")
	}
}

// TestGeneratePaths tests all CRUD path generation
func TestGeneratePaths(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail, Lookup: true},
		"name":  {Type: schema.FieldTypeString},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	basePath := "/mod/users"

	// Check list path
	if path, ok := spec.Paths[basePath]; ok {
		if path.Get == nil {
			t.Error("expected GET operation on base path (list)")
		}
		if path.Post == nil {
			t.Error("expected POST operation on base path (create)")
		}
	} else {
		t.Errorf("expected base path %q", basePath)
	}

	// Check item path
	itemPath := basePath + "/{id}"
	if path, ok := spec.Paths[itemPath]; ok {
		if path.Get == nil {
			t.Error("expected GET operation on item path")
		}
		if path.Put == nil {
			t.Error("expected PUT operation on item path")
		}
		if path.Patch == nil {
			t.Error("expected PATCH operation on item path")
		}
		if path.Delete == nil {
			t.Error("expected DELETE operation on item path")
		}
	} else {
		t.Errorf("expected item path %q", itemPath)
	}
}

// TestListOperationParameters tests list operation has correct parameters
func TestListOperationParameters(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email":  {Type: schema.FieldTypeEmail, Lookup: true},
		"status": {Type: schema.FieldTypeEnum, Values: []string{"active", "inactive"}, Lookup: true},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	basePath := "/mod/users"
	path := spec.Paths[basePath]

	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	// Check for limit and offset parameters
	hasLimit := false
	hasOffset := false
	hasEmail := false
	hasStatus := false

	for _, param := range path.Get.Parameters {
		switch param.Name {
		case "limit":
			hasLimit = true
		case "offset":
			hasOffset = true
		case "email":
			hasEmail = true
		case "status":
			hasStatus = true
		}
	}

	if !hasLimit {
		t.Error("expected limit parameter")
	}
	if !hasOffset {
		t.Error("expected offset parameter")
	}
	if !hasEmail {
		t.Error("expected email lookup parameter")
	}
	if !hasStatus {
		t.Error("expected status lookup parameter")
	}
}

// TestOperationSecurity tests that operations have security requirements
func TestOperationSecurity(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	basePath := "/mod/users"
	path := spec.Paths[basePath]

	if path.Get == nil {
		t.Fatal("expected GET operation")
	}

	if len(path.Get.Security) == 0 {
		t.Error("expected security requirements")
	}
}

// TestToJSON tests JSON serialization
func TestToJSON(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	jsonData, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if parsed["openapi"] != "3.0.3" {
		t.Error("expected openapi version in JSON")
	}
}

// TestToJSONCompact tests compact JSON serialization
func TestToJSONCompact(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	compactJSON, err := spec.ToJSONCompact()
	if err != nil {
		t.Fatalf("ToJSONCompact failed: %v", err)
	}

	prettyJSON, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Compact should be smaller (no indentation)
	if len(compactJSON) >= len(prettyJSON) {
		t.Error("compact JSON should be smaller than pretty JSON")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(compactJSON, &parsed); err != nil {
		t.Fatalf("invalid compact JSON output: %v", err)
	}
}

// TestMultipleModules tests generation with multiple modules
func TestMultipleModules(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	productModule := createTestModule("product", map[string]schema.Field{
		"name":  {Type: schema.FieldTypeString},
		"price": {Type: schema.FieldTypeFloat},
	}, nil)

	orderModule := createTestModule("order", map[string]schema.Field{
		"total":  {Type: schema.FieldTypeFloat},
		"status": {Type: schema.FieldTypeEnum, Values: []string{"pending", "completed"}},
	}, nil)

	modules := map[string]convention.Derived{
		"user":    deriveModule(userModule),
		"product": deriveModule(productModule),
		"order":   deriveModule(orderModule),
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	// Should have 3 tags
	if len(spec.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(spec.Tags))
	}

	// Should have schemas for all modules
	expectedSchemas := []string{
		"User", "UserCreate", "UserUpdate", "UserList",
		"Product", "ProductCreate", "ProductUpdate", "ProductList",
		"Order", "OrderCreate", "OrderUpdate", "OrderList",
	}

	for _, name := range expectedSchemas {
		if _, ok := spec.Components.Schemas[name]; !ok {
			t.Errorf("expected schema %q", name)
		}
	}
}

// TestSpecServers tests servers are included in spec
func TestSpecServers(t *testing.T) {
	gen := NewGenerator(nil)
	gen.AddServer("https://api.example.com", "Production")

	spec := gen.Generate()

	if len(spec.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(spec.Servers))
	}

	if spec.Servers[0].URL != "https://api.example.com" {
		t.Error("server URL not set correctly")
	}
}

// TestSpecInfo tests info is included in spec
func TestSpecInfo(t *testing.T) {
	gen := NewGenerator(nil)
	gen.SetInfo(Info{
		Title:       "Test API",
		Description: "Test description",
		Version:     "1.2.3",
	})

	spec := gen.Generate()

	if spec.Info.Title != "Test API" {
		t.Error("info title not set correctly")
	}

	if spec.Info.Version != "1.2.3" {
		t.Error("info version not set correctly")
	}
}

// TestArrayFieldSchemas tests array field schema items
func TestArrayFieldSchemas(t *testing.T) {
	gen := NewGenerator(nil)

	t.Run("strings array", func(t *testing.T) {
		field := convention.DerivedField{
			Name: "tags",
			Type: schema.FieldTypeStrings,
		}
		s := gen.fieldToSchema(field)

		if s.Type != "array" {
			t.Errorf("expected type 'array', got %q", s.Type)
		}

		if s.Items == nil || s.Items.Type != "string" {
			t.Error("expected items type 'string'")
		}
	})

	t.Run("ints array", func(t *testing.T) {
		field := convention.DerivedField{
			Name: "scores",
			Type: schema.FieldTypeInts,
		}
		s := gen.fieldToSchema(field)

		if s.Type != "array" {
			t.Errorf("expected type 'array', got %q", s.Type)
		}

		if s.Items == nil || s.Items.Type != "integer" {
			t.Error("expected items type 'integer'")
		}
	})
}

// TestCustomActionWithoutInputs tests custom action without inputs
func TestCustomActionWithoutInputs(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"status": {Type: schema.FieldTypeEnum, Values: []string{"active", "inactive"}},
	}, map[string]schema.Action{
		"deactivate": {
			Set:         map[string]string{"status": "inactive"},
			Description: "Deactivate the user",
		},
	})

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	deactivatePath := "/mod/users/{id}/deactivate"
	pathItem, ok := spec.Paths[deactivatePath]
	if !ok {
		t.Errorf("expected path %q to exist", deactivatePath)
	}

	if pathItem.Post == nil {
		t.Error("expected POST operation for deactivate action")
	}

	// Should not have request body when no inputs
	if pathItem.Post.RequestBody != nil {
		t.Error("expected no request body for action without inputs")
	}
}

// TestFieldWithDescription tests field description in schema
func TestFieldWithDescription(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name:        "email",
		Type:        schema.FieldTypeEmail,
		Description: "User's email address",
	}

	s := gen.fieldToSchema(field)

	if s.Description != "User's email address" {
		t.Errorf("expected description 'User's email address', got %q", s.Description)
	}
}

// TestRefFieldWithoutDescription tests ref field generates description
func TestRefFieldWithoutDescription(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name: "user_id",
		Type: schema.FieldTypeRef,
		Ref:  "user",
	}

	s := gen.fieldToSchema(field)

	if s.Description == "" {
		t.Error("expected auto-generated description for ref field")
	}

	// Should mention the ref target
	if !contains(s.Description, "user") {
		t.Error("description should mention ref target")
	}
}

// TestEmptyEnumValues tests enum with no values
func TestEmptyEnumValues(t *testing.T) {
	gen := NewGenerator(nil)

	field := convention.DerivedField{
		Name:   "status",
		Type:   schema.FieldTypeEnum,
		Values: []string{},
	}

	s := gen.fieldToSchema(field)

	if s.Type != "string" {
		t.Errorf("expected type 'string', got %q", s.Type)
	}

	if len(s.Enum) != 0 {
		t.Error("expected empty enum")
	}
}

// TestOperationResponses tests operation response codes
func TestOperationResponses(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	basePath := "/mod/users"
	itemPath := basePath + "/{id}"

	// Check list response codes
	if list := spec.Paths[basePath].Get; list != nil {
		if _, ok := list.Responses["200"]; !ok {
			t.Error("expected 200 response for list")
		}
		if _, ok := list.Responses["401"]; !ok {
			t.Error("expected 401 response for list")
		}
	}

	// Check create response codes
	if create := spec.Paths[basePath].Post; create != nil {
		if _, ok := create.Responses["201"]; !ok {
			t.Error("expected 201 response for create")
		}
		if _, ok := create.Responses["400"]; !ok {
			t.Error("expected 400 response for create")
		}
	}

	// Check get response codes
	if get := spec.Paths[itemPath].Get; get != nil {
		if _, ok := get.Responses["200"]; !ok {
			t.Error("expected 200 response for get")
		}
		if _, ok := get.Responses["404"]; !ok {
			t.Error("expected 404 response for get")
		}
	}

	// Check delete response codes
	if del := spec.Paths[itemPath].Delete; del != nil {
		if _, ok := del.Responses["204"]; !ok {
			t.Error("expected 204 response for delete")
		}
		if _, ok := del.Responses["404"]; !ok {
			t.Error("expected 404 response for delete")
		}
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

// Helper to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestImplicitFieldsExcluded tests implicit fields are excluded from create/update schemas
func TestImplicitFieldsExcluded(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	gen := NewGenerator(nil)

	createSchema := gen.buildCreateSchema(derived)

	// Implicit fields (id, created_at, updated_at) should not be in create schema
	if _, ok := createSchema.Properties["id"]; ok {
		t.Error("id should not be in create schema")
	}
	if _, ok := createSchema.Properties["created_at"]; ok {
		t.Error("created_at should not be in create schema")
	}
	if _, ok := createSchema.Properties["updated_at"]; ok {
		t.Error("updated_at should not be in create schema")
	}
}

// TestListSchemaStructure tests the list response schema structure
func TestListSchemaStructure(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	listSchema := spec.Components.Schemas["UserList"]
	if listSchema == nil {
		t.Fatal("expected UserList schema")
	}

	if listSchema.Type != "object" {
		t.Errorf("expected type 'object', got %q", listSchema.Type)
	}

	// Check data property
	if data, ok := listSchema.Properties["data"]; ok {
		if data.Type != "array" {
			t.Error("data should be an array")
		}
		if data.Items == nil || data.Items.Ref != "#/components/schemas/User" {
			t.Error("data items should reference User schema")
		}
	} else {
		t.Error("expected data property")
	}

	// Check count property
	if count, ok := listSchema.Properties["count"]; ok {
		if count.Type != "integer" {
			t.Error("count should be integer")
		}
	} else {
		t.Error("expected count property")
	}
}

// TestOperationIDs tests that operation IDs are generated correctly
func TestOperationIDs(t *testing.T) {
	userModule := createTestModule("user", map[string]schema.Field{
		"email": {Type: schema.FieldTypeEmail},
	}, nil)

	derived := deriveModule(userModule)
	modules := map[string]convention.Derived{
		"user": derived,
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	basePath := "/mod/users"
	itemPath := basePath + "/{id}"

	if spec.Paths[basePath].Get != nil && spec.Paths[basePath].Get.OperationID != "listUser" {
		t.Errorf("expected operationId 'listUser', got %q", spec.Paths[basePath].Get.OperationID)
	}

	if spec.Paths[basePath].Post != nil && spec.Paths[basePath].Post.OperationID != "createUser" {
		t.Errorf("expected operationId 'createUser', got %q", spec.Paths[basePath].Post.OperationID)
	}

	if spec.Paths[itemPath].Get != nil && spec.Paths[itemPath].Get.OperationID != "getUser" {
		t.Errorf("expected operationId 'getUser', got %q", spec.Paths[itemPath].Get.OperationID)
	}

	if spec.Paths[itemPath].Put != nil && spec.Paths[itemPath].Put.OperationID != "updateUser" {
		t.Errorf("expected operationId 'updateUser', got %q", spec.Paths[itemPath].Put.OperationID)
	}

	if spec.Paths[itemPath].Delete != nil && spec.Paths[itemPath].Delete.OperationID != "deleteUser" {
		t.Errorf("expected operationId 'deleteUser', got %q", spec.Paths[itemPath].Delete.OperationID)
	}
}

// TestSecuritySchemes tests security scheme definitions
func TestSecuritySchemes(t *testing.T) {
	gen := NewGenerator(nil)
	spec := gen.Generate()

	// Check bearer auth
	bearerAuth, ok := spec.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("expected bearerAuth security scheme")
	}

	if bearerAuth.Type != "http" {
		t.Errorf("expected type 'http', got %q", bearerAuth.Type)
	}

	if bearerAuth.Scheme != "bearer" {
		t.Errorf("expected scheme 'bearer', got %q", bearerAuth.Scheme)
	}

	if bearerAuth.BearerFormat != "JWT" {
		t.Errorf("expected bearerFormat 'JWT', got %q", bearerAuth.BearerFormat)
	}

	// Check API key
	apiKey, ok := spec.Components.SecuritySchemes["apiKey"]
	if !ok {
		t.Fatal("expected apiKey security scheme")
	}

	if apiKey.Type != "apiKey" {
		t.Errorf("expected type 'apiKey', got %q", apiKey.Type)
	}

	if apiKey.In != "header" {
		t.Errorf("expected in 'header', got %q", apiKey.In)
	}

	if apiKey.Name != "X-API-Key" {
		t.Errorf("expected name 'X-API-Key', got %q", apiKey.Name)
	}
}

// TestModulesSortedInOutput tests that modules are processed in sorted order
func TestModulesSortedInOutput(t *testing.T) {
	// Create modules with names that would sort differently
	modules := map[string]convention.Derived{
		"zebra": deriveModule(createTestModule("zebra", map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		}, nil)),
		"apple": deriveModule(createTestModule("apple", map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		}, nil)),
		"mango": deriveModule(createTestModule("mango", map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		}, nil)),
	}

	gen := NewGenerator(modules)
	spec := gen.Generate()

	// Tags should be in alphabetical order
	if len(spec.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(spec.Tags))
	}

	expectedOrder := []string{"apple", "mango", "zebra"}
	for i, expected := range expectedOrder {
		if spec.Tags[i].Name != expected {
			t.Errorf("expected tag %d to be %q, got %q", i, expected, spec.Tags[i].Name)
		}
	}
}
