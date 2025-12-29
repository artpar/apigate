package schema

import (
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
