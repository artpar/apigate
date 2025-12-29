// Package testgen generates Go test files from module schemas.
// It auto-generates tests for validation rules, constraints, CRUD operations, and custom actions.
package testgen

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
	"time"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Generator generates test files from module schemas.
type Generator struct {
	modules    map[string]convention.Derived
	packageName string
}

// NewGenerator creates a new test generator.
func NewGenerator(modules map[string]convention.Derived) *Generator {
	return &Generator{
		modules:    modules,
		packageName: "generated_test",
	}
}

// SetPackageName sets the package name for generated tests.
func (g *Generator) SetPackageName(name string) {
	g.packageName = name
}

// GenerateModule generates tests for a single module.
func (g *Generator) GenerateModule(moduleName string) ([]byte, error) {
	mod, ok := g.modules[moduleName]
	if !ok {
		return nil, fmt.Errorf("module %q not found", moduleName)
	}

	return g.generateModuleTests(mod)
}

// GenerateAll generates tests for all modules.
func (g *Generator) GenerateAll() (map[string][]byte, error) {
	result := make(map[string][]byte)

	for name, mod := range g.modules {
		code, err := g.generateModuleTests(mod)
		if err != nil {
			return nil, fmt.Errorf("generating tests for %s: %w", name, err)
		}
		result[name] = code
	}

	return result, nil
}

// TestCase represents a single test case.
type TestCase struct {
	Name        string
	Description string
	Module      string
	Action      string
	Input       map[string]any
	ExpectError bool
	ErrorMatch  string
	Setup       string
	Cleanup     string
}

// ModuleTestData holds all test data for a module.
type ModuleTestData struct {
	PackageName   string
	ModuleName    string
	ModuleTitle   string
	Plural        string
	GeneratedAt   string
	TestCases     []TestCase
	Fields        []FieldTestData
	Actions       []ActionTestData
	HasCustom     bool
	RequiredFields []string
	UniqueFields   []string
	EnumFields     []EnumFieldData
}

// FieldTestData holds test data for a field.
type FieldTestData struct {
	Name       string
	Type       string
	Required   bool
	Unique     bool
	IsEnum     bool
	Values     []string
	Constraints []ConstraintTestData
}

// ConstraintTestData holds test data for a constraint.
type ConstraintTestData struct {
	Type    string
	Value   any
	Message string
}

// EnumFieldData holds enum field data.
type EnumFieldData struct {
	Name   string
	Values []string
}

// ActionTestData holds test data for an action.
type ActionTestData struct {
	Name        string
	Type        string
	Description string
	HasInput    bool
}

// generateModuleTests generates the test file for a module.
func (g *Generator) generateModuleTests(mod convention.Derived) ([]byte, error) {
	data := g.buildTestData(mod)

	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"title":     strings.Title,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"camel":     toCamelCase,
		"snake":     toSnakeCase,
		"quote":     func(s string) string { return fmt.Sprintf("%q", s) },
		"join":      strings.Join,
		"hasPrefix": strings.HasPrefix,
	}).Parse(testTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted if formatting fails (for debugging)
		return buf.Bytes(), nil
	}

	return formatted, nil
}

// buildTestData builds the test data from a module.
func (g *Generator) buildTestData(mod convention.Derived) ModuleTestData {
	data := ModuleTestData{
		PackageName: g.packageName,
		ModuleName:  mod.Source.Name,
		ModuleTitle: strings.Title(mod.Source.Name),
		Plural:      mod.Plural,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	// Collect field information
	for _, field := range mod.Fields {
		if field.Internal || field.Implicit {
			continue
		}

		fd := FieldTestData{
			Name:     field.Name,
			Type:     string(field.Type),
			Required: field.Required,
			Unique:   field.Unique,
			IsEnum:   field.Type == schema.FieldTypeEnum,
			Values:   field.Values,
		}

		// Add constraints
		for _, c := range field.Constraints {
			fd.Constraints = append(fd.Constraints, ConstraintTestData{
				Type:    string(c.Type),
				Value:   c.Value,
				Message: c.Message,
			})
		}

		data.Fields = append(data.Fields, fd)

		if field.Required {
			data.RequiredFields = append(data.RequiredFields, field.Name)
		}
		if field.Unique {
			data.UniqueFields = append(data.UniqueFields, field.Name)
		}
		if field.Type == schema.FieldTypeEnum {
			data.EnumFields = append(data.EnumFields, EnumFieldData{
				Name:   field.Name,
				Values: field.Values,
			})
		}
	}

	// Collect action information
	for _, action := range mod.Actions {
		ad := ActionTestData{
			Name:        action.Name,
			Type:        action.Type.String(),
			Description: action.Description,
			HasInput:    len(action.Input) > 0,
		}
		data.Actions = append(data.Actions, ad)

		if action.Type == schema.ActionTypeCustom {
			data.HasCustom = true
		}
	}

	return data
}

// toCamelCase converts snake_case to CamelCase.
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// toSnakeCase converts CamelCase to snake_case.
func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if i > 0 && c >= 'A' && c <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, byte(c))
	}
	return strings.ToLower(string(result))
}

const testTemplate = `// Code generated by apigate test generate. DO NOT EDIT.
// Generated at: {{.GeneratedAt}}
// Module: {{.ModuleName}}

package {{.PackageName}}

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/core/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = strings.Contains // Avoid unused import error

// TestContext provides a context with timeout for tests.
func testContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx
}

// validInput returns a valid input map for creating a {{.ModuleName}} record.
func valid{{.ModuleTitle}}Input() map[string]any {
	return map[string]any{
		{{- range .Fields}}
		{{- if .Required}}
		{{- if eq .Type "string"}}
		"{{.Name}}": "test-value",
		{{- else if eq .Type "email"}}
		"{{.Name}}": "test@example.com",
		{{- else if eq .Type "int"}}
		"{{.Name}}": 100,
		{{- else if eq .Type "float"}}
		"{{.Name}}": 99.99,
		{{- else if eq .Type "bool"}}
		"{{.Name}}": true,
		{{- else if eq .Type "enum"}}
		"{{.Name}}": "{{index .Values 0}}",
		{{- else if eq .Type "url"}}
		"{{.Name}}": "https://example.com",
		{{- else if eq .Type "uuid"}}
		"{{.Name}}": "00000000-0000-0000-0000-000000000001",
		{{- else}}
		"{{.Name}}": "test-value",
		{{- end}}
		{{- end}}
		{{- end}}
	}
}

// ============================================================================
// Required Field Tests
// ============================================================================

{{range .RequiredFields}}
func Test{{$.ModuleTitle}}_{{camel .}}_Required(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create input without the required field
	input := valid{{$.ModuleTitle}}Input()
	delete(input, "{{.}}")

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	require.Error(t, err, "Expected error when {{.}} is missing")
	assert.Contains(t, err.Error(), "{{.}}", "Error should mention the missing field")
}
{{end}}

// ============================================================================
// Unique Constraint Tests
// ============================================================================

{{range .UniqueFields}}
func Test{{$.ModuleTitle}}_{{camel .}}_Unique(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create first record
	input1 := valid{{$.ModuleTitle}}Input()
	{{- if eq . "email"}}
	input1["{{.}}"] = "unique-test-1@example.com"
	{{- else}}
	input1["{{.}}"] = "unique-test-value-1"
	{{- end}}

	result1, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input1,
		Channel: "test",
	})
	require.NoError(t, err, "First create should succeed")
	require.NotEmpty(t, result1.ID, "Should return ID")

	// Cleanup after test
	defer func() {
		rt.Execute(ctx, "{{$.ModuleName}}", "delete", runtime.ActionInput{
			Lookup:  result1.ID,
			Channel: "test",
		})
	}()

	// Try to create duplicate
	input2 := valid{{$.ModuleTitle}}Input()
	input2["{{.}}"] = input1["{{.}}"]

	_, err = rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input2,
		Channel: "test",
	})

	require.Error(t, err, "Duplicate {{.}} should fail")
	assert.Contains(t, strings.ToLower(err.Error()), "unique", "Error should mention unique constraint")
}
{{end}}

// ============================================================================
// Enum Field Tests
// ============================================================================

{{range .EnumFields}}
func Test{{$.ModuleTitle}}_{{camel .Name}}_ValidEnum(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	validValues := []string{ {{range .Values}}"{{.}}", {{end}} }

	for _, val := range validValues {
		input := valid{{$.ModuleTitle}}Input()
		input["{{.Name}}"] = val

		result, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
			Data:    input,
			Channel: "test",
		})

		if err == nil && result.ID != "" {
			// Cleanup
			rt.Execute(ctx, "{{$.ModuleName}}", "delete", runtime.ActionInput{
				Lookup:  result.ID,
				Channel: "test",
			})
		}

		assert.NoError(t, err, "Valid enum value %q should be accepted", val)
	}
}

func Test{{$.ModuleTitle}}_{{camel .Name}}_InvalidEnum(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{.Name}}"] = "INVALID_ENUM_VALUE_12345"

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	require.Error(t, err, "Invalid enum value should be rejected")
}
{{end}}

// ============================================================================
// Type Validation Tests
// ============================================================================

{{range .Fields}}
{{- if eq .Type "email"}}
func Test{{$.ModuleTitle}}_{{camel .Name}}_InvalidEmail(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	invalidEmails := []string{
		"not-an-email",
		"missing@domain",
		"@nodomain.com",
		"spaces in@email.com",
	}

	for _, email := range invalidEmails {
		input := valid{{$.ModuleTitle}}Input()
		input["{{.Name}}"] = email

		_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
			Data:    input,
			Channel: "test",
		})

		assert.Error(t, err, "Invalid email %q should be rejected", email)
	}
}
{{end}}
{{- if eq .Type "url"}}
func Test{{$.ModuleTitle}}_{{camel .Name}}_InvalidURL(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	invalidURLs := []string{
		"not-a-url",
		"ftp://missing-http.com",
		"://no-scheme.com",
	}

	for _, url := range invalidURLs {
		input := valid{{$.ModuleTitle}}Input()
		input["{{.Name}}"] = url

		_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
			Data:    input,
			Channel: "test",
		})

		assert.Error(t, err, "Invalid URL %q should be rejected", url)
	}
}
{{end}}
{{- if eq .Type "int"}}
func Test{{$.ModuleTitle}}_{{camel .Name}}_InvalidInt(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{.Name}}"] = "not-a-number"

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Invalid integer should be rejected")
}
{{end}}
{{end}}

// ============================================================================
// Constraint Tests
// ============================================================================

{{range .Fields}}
{{- range .Constraints}}
{{- if eq .Type "min_length"}}
func Test{{$.ModuleTitle}}_{{camel $.Name}}_MinLength(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{$.Name}}"] = "" // Too short

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Value shorter than min_length should be rejected")
}
{{end}}
{{- if eq .Type "max_length"}}
func Test{{$.ModuleTitle}}_{{camel $.Name}}_MaxLength(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	// Create a string longer than max length
	longValue := strings.Repeat("x", {{.Value}} + 10)
	input["{{$.Name}}"] = longValue

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Value longer than max_length should be rejected")
}
{{end}}
{{- if eq .Type "min"}}
func Test{{$.ModuleTitle}}_{{camel $.Name}}_Min(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{$.Name}}"] = {{.Value}} - 1 // Below minimum

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Value below minimum should be rejected")
}
{{end}}
{{- if eq .Type "max"}}
func Test{{$.ModuleTitle}}_{{camel $.Name}}_Max(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{$.Name}}"] = {{.Value}} + 1 // Above maximum

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Value above maximum should be rejected")
}
{{end}}
{{- if eq .Type "pattern"}}
func Test{{$.ModuleTitle}}_{{camel $.Name}}_Pattern(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{$.ModuleTitle}}Input()
	input["{{$.Name}}"] = "!!!invalid-pattern-value!!!"

	_, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	assert.Error(t, err, "Value not matching pattern should be rejected")
}
{{end}}
{{- end}}
{{- end}}

// ============================================================================
// CRUD Operation Tests
// ============================================================================

func Test{{.ModuleTitle}}_Create(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	input := valid{{.ModuleTitle}}Input()

	result, err := rt.Execute(ctx, "{{.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})

	require.NoError(t, err, "Create should succeed")
	require.NotEmpty(t, result.ID, "Should return ID")

	// Cleanup
	_, err = rt.Execute(ctx, "{{.ModuleName}}", "delete", runtime.ActionInput{
		Lookup:  result.ID,
		Channel: "test",
	})
	assert.NoError(t, err, "Cleanup should succeed")
}

func Test{{.ModuleTitle}}_Get(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create a record first
	input := valid{{.ModuleTitle}}Input()
	createResult, err := rt.Execute(ctx, "{{.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, createResult.ID)

	defer func() {
		rt.Execute(ctx, "{{.ModuleName}}", "delete", runtime.ActionInput{
			Lookup:  createResult.ID,
			Channel: "test",
		})
	}()

	// Get the record
	getResult, err := rt.Execute(ctx, "{{.ModuleName}}", "get", runtime.ActionInput{
		Lookup:  createResult.ID,
		Channel: "test",
	})

	require.NoError(t, err, "Get should succeed")
	require.NotNil(t, getResult.Data, "Should return data")
}

func Test{{.ModuleTitle}}_Get_NotFound(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	_, err := rt.Execute(ctx, "{{.ModuleName}}", "get", runtime.ActionInput{
		Lookup:  "nonexistent-id-12345",
		Channel: "test",
	})

	require.Error(t, err, "Get nonexistent record should fail")
}

func Test{{.ModuleTitle}}_Update(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create a record first
	input := valid{{.ModuleTitle}}Input()
	createResult, err := rt.Execute(ctx, "{{.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, createResult.ID)

	defer func() {
		rt.Execute(ctx, "{{.ModuleName}}", "delete", runtime.ActionInput{
			Lookup:  createResult.ID,
			Channel: "test",
		})
	}()

	// Update the record
	updateData := map[string]any{
		{{- range .Fields}}
		{{- if and (not .Required) (eq .Type "string")}}
		"{{.Name}}": "updated-value",
		{{- end}}
		{{- end}}
	}

	updateResult, err := rt.Execute(ctx, "{{.ModuleName}}", "update", runtime.ActionInput{
		Lookup:  createResult.ID,
		Data:    updateData,
		Channel: "test",
	})

	require.NoError(t, err, "Update should succeed")
	assert.Equal(t, createResult.ID, updateResult.ID, "ID should match")
}

func Test{{.ModuleTitle}}_Delete(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create a record first
	input := valid{{.ModuleTitle}}Input()
	createResult, err := rt.Execute(ctx, "{{.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, createResult.ID)

	// Delete the record
	_, err = rt.Execute(ctx, "{{.ModuleName}}", "delete", runtime.ActionInput{
		Lookup:  createResult.ID,
		Channel: "test",
	})
	require.NoError(t, err, "Delete should succeed")

	// Verify it's gone
	_, err = rt.Execute(ctx, "{{.ModuleName}}", "get", runtime.ActionInput{
		Lookup:  createResult.ID,
		Channel: "test",
	})
	require.Error(t, err, "Get deleted record should fail")
}

func Test{{.ModuleTitle}}_Delete_NotFound(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	_, err := rt.Execute(ctx, "{{.ModuleName}}", "delete", runtime.ActionInput{
		Lookup:  "nonexistent-id-12345",
		Channel: "test",
	})

	require.Error(t, err, "Delete nonexistent record should fail")
}

func Test{{.ModuleTitle}}_List(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	result, err := rt.Execute(ctx, "{{.ModuleName}}", "list", runtime.ActionInput{
		Data: map[string]any{
			"limit":  10,
			"offset": 0,
		},
		Channel: "test",
	})

	require.NoError(t, err, "List should succeed")
	assert.NotNil(t, result.List, "Should return list")
	assert.GreaterOrEqual(t, result.Count, 0, "Count should be non-negative")
}

func Test{{.ModuleTitle}}_List_WithPagination(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// First page
	result1, err := rt.Execute(ctx, "{{.ModuleName}}", "list", runtime.ActionInput{
		Data: map[string]any{
			"limit":  5,
			"offset": 0,
		},
		Channel: "test",
	})
	require.NoError(t, err)

	// Second page
	result2, err := rt.Execute(ctx, "{{.ModuleName}}", "list", runtime.ActionInput{
		Data: map[string]any{
			"limit":  5,
			"offset": 5,
		},
		Channel: "test",
	})
	require.NoError(t, err)

	// Both should succeed
	assert.NotNil(t, result1.List)
	assert.NotNil(t, result2.List)
}

{{if .HasCustom}}
// ============================================================================
// Custom Action Tests
// ============================================================================

{{range .Actions}}
{{- if eq .Type "custom"}}
func Test{{$.ModuleTitle}}_{{camel .Name}}(t *testing.T) {
	rt := getTestRuntime(t)
	ctx := testContext()

	// Create a record first
	input := valid{{$.ModuleTitle}}Input()
	createResult, err := rt.Execute(ctx, "{{$.ModuleName}}", "create", runtime.ActionInput{
		Data:    input,
		Channel: "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, createResult.ID)

	defer func() {
		rt.Execute(ctx, "{{$.ModuleName}}", "delete", runtime.ActionInput{
			Lookup:  createResult.ID,
			Channel: "test",
		})
	}()

	// Execute custom action: {{.Name}}
	{{if .HasInput}}
	actionInput := map[string]any{
		// TODO: Add required input fields for {{.Name}}
	}
	{{else}}
	actionInput := map[string]any{}
	{{end}}

	_, err = rt.Execute(ctx, "{{$.ModuleName}}", "{{.Name}}", runtime.ActionInput{
		Lookup:  createResult.ID,
		Data:    actionInput,
		Channel: "test",
	})

	// Action should complete (success or expected failure)
	// Adjust assertion based on expected behavior
	_ = err // Check error or assert.NoError based on action requirements
}
{{end}}
{{- end}}
{{end}}

// ============================================================================
// Helper: Get Test Runtime
// ============================================================================

// getTestRuntime returns the runtime for testing.
// This should be implemented to return your initialized runtime.
func getTestRuntime(t *testing.T) *runtime.Runtime {
	t.Helper()
	// TODO: Initialize and return your test runtime
	// Example:
	// rt, err := bootstrap.NewRuntime(config)
	// require.NoError(t, err)
	// return rt
	t.Skip("getTestRuntime not implemented - implement to run generated tests")
	return nil
}
`
