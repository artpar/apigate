package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	yaml := `
module: plan

schema:
  name:        { type: string }
  rate_limit:  { type: int, default: 60 }
  enabled:     { type: bool, default: true }

actions:
  enable:  { set: { enabled: true } }
  disable: { set: { enabled: false } }

channels:
  http:
    serve:
      enabled: true
  cli:
    serve:
      enabled: true
`

	mod, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if mod.Name != "plan" {
		t.Errorf("Name = %q, want %q", mod.Name, "plan")
	}

	if len(mod.Schema) != 3 {
		t.Errorf("Schema has %d fields, want 3", len(mod.Schema))
	}

	if _, ok := mod.Schema["name"]; !ok {
		t.Error("Schema missing 'name' field")
	}

	if _, ok := mod.Actions["enable"]; !ok {
		t.Error("Actions missing 'enable' action")
	}

	if !mod.Channels.HTTP.Serve.Enabled {
		t.Error("HTTP serve should be enabled")
	}

	if !mod.Channels.CLI.Serve.Enabled {
		t.Error("CLI serve should be enabled")
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid minimal",
			yaml: `
module: test
schema:
  name: { type: string }
`,
			wantErr: false,
		},
		{
			name: "missing module name",
			yaml: `
schema:
  name: { type: string }
`,
			wantErr: true,
		},
		{
			name: "empty schema",
			yaml: `
module: test
schema: {}
`,
			wantErr: true,
		},
		{
			name: "invalid field type",
			yaml: `
module: test
schema:
  name: { type: invalid_type }
`,
			wantErr: true,
		},
		{
			name: "enum without values",
			yaml: `
module: test
schema:
  status: { type: enum }
`,
			wantErr: true,
		},
		{
			name: "enum with values",
			yaml: `
module: test
schema:
  status: { type: enum, values: [a, b, c] }
`,
			wantErr: false,
		},
		{
			name: "ref without target",
			yaml: `
module: test
schema:
  user: { type: ref }
`,
			wantErr: true,
		},
		{
			name: "ref with target",
			yaml: `
module: test
schema:
  user: { type: ref, to: user }
`,
			wantErr: false,
		},
		{
			name: "action references non-existent field",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  bad: { set: { nonexistent: value } }
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldTypes(t *testing.T) {
	tests := []struct {
		fieldType FieldType
		sqlType   string
	}{
		{FieldTypeString, "TEXT"},
		{FieldTypeInt, "INTEGER"},
		{FieldTypeFloat, "REAL"},
		{FieldTypeBool, "INTEGER"},
		{FieldTypeTimestamp, "TEXT"},
		{FieldTypeEmail, "TEXT"},
		{FieldTypeJSON, "TEXT"},
		{FieldTypeBytes, "BLOB"},
		{FieldTypeSecret, "BLOB"},
	}

	for _, tt := range tests {
		t.Run(string(tt.fieldType), func(t *testing.T) {
			f := Field{Type: tt.fieldType}
			if got := f.SQLType(); got != tt.sqlType {
				t.Errorf("SQLType() = %q, want %q", got, tt.sqlType)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "schema_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("valid file", func(t *testing.T) {
		content := `
module: test
schema:
  name: { type: string }
`
		path := filepath.Join(tmpDir, "test.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		mod, err := ParseFile(path)
		if err != nil {
			t.Fatalf("ParseFile failed: %v", err)
		}
		if mod.Name != "test" {
			t.Errorf("Module name = %q, want %q", mod.Name, "test")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseFile(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("ParseFile should fail for non-existent file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		content := `
module: test
schema: [invalid yaml
`
		path := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Error("ParseFile should fail for invalid YAML")
		}
	})

	t.Run("yml extension", func(t *testing.T) {
		content := `
module: test_yml
schema:
  name: { type: string }
`
		path := filepath.Join(tmpDir, "test.yml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		mod, err := ParseFile(path)
		if err != nil {
			t.Fatalf("ParseFile failed: %v", err)
		}
		if mod.Name != "test_yml" {
			t.Errorf("Module name = %q, want %q", mod.Name, "test_yml")
		}
	})
}

func TestParseDir(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "schema_dir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.Mkdir(emptyDir, 0755); err != nil {
			t.Fatalf("Failed to create empty dir: %v", err)
		}

		modules, err := ParseDir(emptyDir)
		if err != nil {
			t.Fatalf("ParseDir failed: %v", err)
		}
		if len(modules) != 0 {
			t.Errorf("Expected 0 modules, got %d", len(modules))
		}
	})

	t.Run("directory with yaml files", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "modules")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}

		// Create multiple module files
		files := map[string]string{
			"user.yaml": `
module: user
schema:
  name: { type: string }
`,
			"plan.yaml": `
module: plan
schema:
  title: { type: string }
`,
		}

		for name, content := range files {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write file %s: %v", name, err)
			}
		}

		modules, err := ParseDir(dir)
		if err != nil {
			t.Fatalf("ParseDir failed: %v", err)
		}
		if len(modules) != 2 {
			t.Errorf("Expected 2 modules, got %d", len(modules))
		}
	})

	t.Run("directory with subdirectories", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "nested")
		subDir := filepath.Join(dir, "submodules")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dirs: %v", err)
		}

		// Create files in both directories
		rootContent := `
module: root
schema:
  name: { type: string }
`
		subContent := `
module: sub
schema:
  title: { type: string }
`
		if err := os.WriteFile(filepath.Join(dir, "root.yaml"), []byte(rootContent), 0644); err != nil {
			t.Fatalf("Failed to write root file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "sub.yaml"), []byte(subContent), 0644); err != nil {
			t.Fatalf("Failed to write sub file: %v", err)
		}

		modules, err := ParseDir(dir)
		if err != nil {
			t.Fatalf("ParseDir failed: %v", err)
		}
		if len(modules) != 2 {
			t.Errorf("Expected 2 modules (including subdirectory), got %d", len(modules))
		}
	})

	t.Run("non-yaml files ignored", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "mixed")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}

		yamlContent := `
module: test
schema:
  name: { type: string }
`
		if err := os.WriteFile(filepath.Join(dir, "module.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write yaml file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644); err != nil {
			t.Fatalf("Failed to write txt file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to write go file: %v", err)
		}

		modules, err := ParseDir(dir)
		if err != nil {
			t.Fatalf("ParseDir failed: %v", err)
		}
		if len(modules) != 1 {
			t.Errorf("Expected 1 module (only yaml file), got %d", len(modules))
		}
	})

	t.Run("directory not found", func(t *testing.T) {
		_, err := ParseDir(filepath.Join(tmpDir, "nonexistent"))
		if err == nil {
			t.Error("ParseDir should fail for non-existent directory")
		}
	})

	t.Run("invalid yaml in directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "invalid")
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}

		content := `
module: test
schema: [invalid
`
		if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err = ParseDir(dir)
		if err == nil {
			t.Error("ParseDir should fail for invalid YAML file")
		}
	})
}

func TestValidateDefault(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid int default",
			yaml: `
module: test
schema:
  count: { type: int, default: 10 }
`,
			wantErr: false,
		},
		{
			name: "valid bool default",
			yaml: `
module: test
schema:
  enabled: { type: bool, default: true }
`,
			wantErr: false,
		},
		{
			name: "valid enum default",
			yaml: `
module: test
schema:
  status: { type: enum, values: [active, inactive], default: active }
`,
			wantErr: false,
		},
		{
			name: "invalid int default (string)",
			yaml: `
module: test
schema:
  count: { type: int, default: "ten" }
`,
			wantErr: true,
		},
		{
			name: "invalid bool default (string)",
			yaml: `
module: test
schema:
  enabled: { type: bool, default: "yes" }
`,
			wantErr: true,
		},
		{
			name: "invalid enum default (not in values)",
			yaml: `
module: test
schema:
  status: { type: enum, values: [active, inactive], default: pending }
`,
			wantErr: true,
		},
		{
			name: "valid float default parsed as int",
			yaml: `
module: test
schema:
  count: { type: int, default: 10.0 }
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid module name",
			yaml: `
module: user
schema:
  name: { type: string }
`,
			wantErr: false,
		},
		{
			name: "valid module name with underscore",
			yaml: `
module: user_profile
schema:
  name: { type: string }
`,
			wantErr: false,
		},
		{
			name: "valid module name starting with underscore",
			yaml: `
module: _internal
schema:
  name: { type: string }
`,
			wantErr: false,
		},
		{
			name: "valid module name with numbers",
			yaml: `
module: user2
schema:
  name: { type: string }
`,
			wantErr: false,
		},
		{
			name: "invalid module name starting with number",
			yaml: `
module: 2users
schema:
  name: { type: string }
`,
			wantErr: true,
		},
		{
			name: "invalid module name with hyphen",
			yaml: `
module: user-profile
schema:
  name: { type: string }
`,
			wantErr: true,
		},
		{
			name: "invalid module name with space",
			yaml: `
module: user profile
schema:
  name: { type: string }
`,
			wantErr: true,
		},
		{
			name: "invalid field name with hyphen",
			yaml: `
module: test
schema:
  user-name: { type: string }
`,
			wantErr: true,
		},
		{
			name: "invalid action name with hyphen",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  do-something: { set: { name: done } }
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCapability(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid capability",
			yaml: `
capability: payment
description: Payment processing interface
actions:
  charge:
    input:
      - name: amount
        type: int
`,
			wantErr: false,
		},
		{
			name: "capability without name",
			yaml: `
capability: ""
description: Empty capability name
`,
			wantErr: true,
		},
		{
			name: "capability with invalid name",
			yaml: `
capability: payment-provider
description: Invalid capability name
`,
			wantErr: true,
		},
		{
			name: "capability with invalid action name",
			yaml: `
capability: payment
actions:
  do-charge: {}
`,
			wantErr: true,
		},
		{
			name: "capability without schema (allowed)",
			yaml: `
capability: cache
description: Caching interface
actions:
  get: {}
  set: {}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "action with valid set field",
			yaml: `
module: test
schema:
  status: { type: string }
actions:
  activate: { set: { status: active } }
`,
			wantErr: false,
		},
		{
			name: "action with invalid set field",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  activate: { set: { status: active } }
`,
			wantErr: true,
		},
		{
			name: "action with valid input field reference",
			yaml: `
module: test
schema:
  email: { type: email }
actions:
  update_email:
    input:
      - field: email
`,
			wantErr: false,
		},
		{
			name: "action with invalid input field reference",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  update_email:
    input:
      - field: email
`,
			wantErr: true,
		},
		{
			name: "action with input name only (no field reference)",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  custom:
    input:
      - name: custom_param
        type: string
`,
			wantErr: false,
		},
		{
			name: "action with output fields (not validated against schema)",
			yaml: `
module: test
schema:
  name: { type: string }
actions:
  custom:
    output:
      - name: result
        type: string
      - name: custom_field
        type: int
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidFieldType(t *testing.T) {
	validTypes := []string{
		"string", "int", "float", "bool", "timestamp", "duration",
		"json", "bytes", "email", "url", "uuid", "enum", "ref",
		"secret", "strings", "ints",
	}

	for _, ft := range validTypes {
		yaml := `
module: test
schema:
  field: { type: ` + ft
		if ft == "enum" {
			yaml += `, values: [a, b]`
		} else if ft == "ref" {
			yaml += `, to: other`
		}
		yaml += ` }
`
		t.Run(ft, func(t *testing.T) {
			_, err := Parse([]byte(yaml))
			if err != nil {
				t.Errorf("Field type %q should be valid, got error: %v", ft, err)
			}
		})
	}

	invalidTypes := []string{
		"invalid", "text", "number", "boolean", "array", "object",
	}

	for _, ft := range invalidTypes {
		yaml := `
module: test
schema:
  field: { type: ` + ft + ` }
`
		t.Run(ft+"_invalid", func(t *testing.T) {
			_, err := Parse([]byte(yaml))
			if err == nil {
				t.Errorf("Field type %q should be invalid", ft)
			}
		})
	}
}

func TestParseInvalidYAML(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "completely invalid yaml",
			input: "{{{{invalid",
		},
		{
			name:  "invalid indentation",
			input: "module: test\n  schema:\n name: string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.input))
			if err == nil {
				t.Error("Parse should fail for invalid YAML")
			}
		})
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	// Test that multiple validation errors are collected
	yaml := `
module: ""
schema: {}
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("Parse should fail with multiple validation errors")
	}

	// The error should mention both issues
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error message should not be empty")
	}
}
