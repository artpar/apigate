package testgen

import (
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "User"},
		{"user_name", "UserName"},
		{"api_key", "ApiKey"},
		{"first_name", "FirstName"},
		{"my_long_variable_name", "MyLongVariableName"},
		{"single", "Single"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserName", "user_name"},
		{"APIKey", "a_p_i_key"},
		{"FirstName", "first_name"},
		{"lowercase", "lowercase"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewGenerator(t *testing.T) {
	modules := map[string]convention.Derived{}
	gen := NewGenerator(modules)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.packageName != "generated_test" {
		t.Errorf("packageName = %q, want %q", gen.packageName, "generated_test")
	}
}

func TestGenerator_SetPackageName(t *testing.T) {
	gen := NewGenerator(nil)
	gen.SetPackageName("custom_test")

	if gen.packageName != "custom_test" {
		t.Errorf("packageName = %q, want %q", gen.packageName, "custom_test")
	}
}

func TestGenerator_GenerateModule_NotFound(t *testing.T) {
	modules := map[string]convention.Derived{}
	gen := NewGenerator(modules)

	_, err := gen.GenerateModule("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent module")
	}
}

func TestGenerator_GenerateModule_Simple(t *testing.T) {
	modules := map[string]convention.Derived{
		"test_module": {
			Source: schema.Module{
				Name:        "test_module",
				Description: "Test module",
			},
			Plural: "test_modules",
			Fields: []convention.DerivedField{
				{
					Name:     "name",
					Type:     schema.FieldTypeString,
					Required: true,
				},
			},
		},
	}

	gen := NewGenerator(modules)
	code, err := gen.GenerateModule("test_module")

	if err != nil {
		t.Fatalf("GenerateModule failed: %v", err)
	}
	if len(code) == 0 {
		t.Error("GenerateModule returned empty code")
	}

	// Check that generated code contains expected elements
	codeStr := string(code)
	if !contains(codeStr, "package generated_test") {
		t.Error("generated code missing package declaration")
	}
	if !contains(codeStr, "test_module") {
		t.Error("generated code missing module name")
	}
}

func TestGenerator_GenerateAll_Empty(t *testing.T) {
	modules := map[string]convention.Derived{}
	gen := NewGenerator(modules)

	result, err := gen.GenerateAll()
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("GenerateAll returned %d modules, want 0", len(result))
	}
}

func TestGenerator_GenerateAll_MultipleModules(t *testing.T) {
	modules := map[string]convention.Derived{
		"module_a": {
			Source: schema.Module{Name: "module_a"},
			Plural: "module_as",
		},
		"module_b": {
			Source: schema.Module{Name: "module_b"},
			Plural: "module_bs",
		},
	}

	gen := NewGenerator(modules)
	result, err := gen.GenerateAll()

	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GenerateAll returned %d modules, want 2", len(result))
	}
	if _, ok := result["module_a"]; !ok {
		t.Error("GenerateAll missing module_a")
	}
	if _, ok := result["module_b"]; !ok {
		t.Error("GenerateAll missing module_b")
	}
}

func TestGenerator_BuildTestData_RequiredFields(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "email", Type: schema.FieldTypeEmail, Required: true},
			{Name: "name", Type: schema.FieldTypeString, Required: true},
			{Name: "age", Type: schema.FieldTypeInt, Required: false},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	if len(data.RequiredFields) != 2 {
		t.Errorf("RequiredFields has %d items, want 2", len(data.RequiredFields))
	}
}

func TestGenerator_BuildTestData_UniqueFields(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "email", Type: schema.FieldTypeEmail, Unique: true},
			{Name: "name", Type: schema.FieldTypeString, Unique: false},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	if len(data.UniqueFields) != 1 {
		t.Errorf("UniqueFields has %d items, want 1", len(data.UniqueFields))
	}
}

func TestGenerator_BuildTestData_EnumFields(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "task"},
		Plural: "tasks",
		Fields: []convention.DerivedField{
			{
				Name:   "status",
				Type:   schema.FieldTypeEnum,
				Values: []string{"pending", "active", "done"},
			},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	if len(data.EnumFields) != 1 {
		t.Errorf("EnumFields has %d items, want 1", len(data.EnumFields))
	}
	if len(data.EnumFields[0].Values) != 3 {
		t.Errorf("EnumFields[0].Values has %d items, want 3", len(data.EnumFields[0].Values))
	}
}

func TestGenerator_BuildTestData_CustomAction(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{
				Name: "activate",
				Type: schema.ActionTypeCustom,
			},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	if !data.HasCustom {
		t.Error("HasCustom should be true")
	}
}

func TestGenerator_BuildTestData_InternalFieldsSkipped(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeString, Internal: true},
			{Name: "created_at", Type: schema.FieldTypeString, Implicit: true},
			{Name: "name", Type: schema.FieldTypeString, Required: true},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	// Only "name" should be in Fields (internal and implicit are skipped)
	if len(data.Fields) != 1 {
		t.Errorf("Fields has %d items, want 1 (internal/implicit should be skipped)", len(data.Fields))
	}
}

func TestGenerator_BuildTestData_FieldConstraints(t *testing.T) {
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{
				Name: "name",
				Type: schema.FieldTypeString,
				Constraints: []schema.Constraint{
					{Type: schema.ConstraintMinLength, Value: 3, Message: "name too short"},
					{Type: schema.ConstraintMaxLength, Value: 100, Message: "name too long"},
				},
			},
		},
	}

	gen := NewGenerator(nil)
	data := gen.buildTestData(mod)

	if len(data.Fields) != 1 {
		t.Fatalf("Fields has %d items, want 1", len(data.Fields))
	}
	if len(data.Fields[0].Constraints) != 2 {
		t.Errorf("Constraints has %d items, want 2", len(data.Fields[0].Constraints))
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
