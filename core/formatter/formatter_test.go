package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"gopkg.in/yaml.v3"
)

// Helper function to create a test Derived module
func createTestDerived() convention.Derived {
	return convention.Derived{
		Source: schema.Module{
			Name: "user",
		},
		Plural: "users",
		Table:  "users",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeUUID, Internal: false},
			{Name: "name", Type: schema.FieldTypeString, Internal: false},
			{Name: "email", Type: schema.FieldTypeEmail, Internal: false},
			{Name: "age", Type: schema.FieldTypeInt, Internal: false},
			{Name: "password_hash", Type: schema.FieldTypeSecret, Internal: true},
			{Name: "internal_notes", Type: schema.FieldTypeString, Internal: true},
		},
	}
}

// Helper function to create test records
func createTestRecords() []map[string]any {
	return []map[string]any{
		{"id": "uuid-1", "name": "Alice", "email": "alice@example.com", "age": 30},
		{"id": "uuid-2", "name": "Bob", "email": "bob@example.com", "age": 25},
	}
}

// ===========================================
// Registry Tests
// ===========================================

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.formatters == nil {
		t.Fatal("formatters map should be initialized")
	}
	if r.defaultFmt != "table" {
		t.Errorf("default format should be 'table', got %q", r.defaultFmt)
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	// Register a formatter
	f := NewTableFormatter()
	err := r.Register(f)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Try to register the same formatter again
	err = r.Register(f)
	if err == nil {
		t.Fatal("expected error when registering duplicate formatter")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("error message should mention 'already registered', got: %v", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	f := NewTableFormatter()
	_ = r.Register(f)

	// Get existing formatter
	got, ok := r.Get("table")
	if !ok {
		t.Fatal("expected to find 'table' formatter")
	}
	if got.Name() != "table" {
		t.Errorf("expected name 'table', got %q", got.Name())
	}

	// Get non-existing formatter
	_, ok = r.Get("nonexistent")
	if ok {
		t.Fatal("expected not to find 'nonexistent' formatter")
	}
}

func TestRegistry_Default(t *testing.T) {
	r := NewRegistry()

	// Empty registry returns nil
	d := r.Default()
	if d != nil {
		t.Fatal("expected nil default for empty registry")
	}

	// Register table formatter
	tableF := NewTableFormatter()
	_ = r.Register(tableF)

	// Default should return table formatter
	d = r.Default()
	if d == nil {
		t.Fatal("expected non-nil default")
	}
	if d.Name() != "table" {
		t.Errorf("expected default 'table', got %q", d.Name())
	}

	// Register json formatter and set as default
	jsonF := NewJSONFormatter()
	_ = r.Register(jsonF)
	_ = r.SetDefault("json")

	d = r.Default()
	if d.Name() != "json" {
		t.Errorf("expected default 'json', got %q", d.Name())
	}
}

func TestRegistry_Default_Fallback(t *testing.T) {
	r := NewRegistry()

	// Register only JSON formatter
	jsonF := NewJSONFormatter()
	_ = r.Register(jsonF)

	// Default is "table" but not registered, should fallback to first available
	d := r.Default()
	if d == nil {
		t.Fatal("expected fallback default formatter")
	}
	// Should get json since it's the only one
	if d.Name() != "json" {
		t.Errorf("expected fallback to 'json', got %q", d.Name())
	}
}

func TestRegistry_SetDefault(t *testing.T) {
	r := NewRegistry()
	f := NewTableFormatter()
	_ = r.Register(f)

	// Set valid default
	err := r.SetDefault("table")
	if err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	// Set invalid default
	err = r.SetDefault("nonexistent")
	if err == nil {
		t.Fatal("expected error when setting nonexistent default")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error message should mention 'not registered', got: %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	names := r.List()
	if len(names) != 0 {
		t.Fatalf("expected empty list, got %v", names)
	}

	// Register formatters
	_ = r.Register(NewTableFormatter())
	_ = r.Register(NewJSONFormatter())
	_ = r.Register(NewYAMLFormatter())

	names = r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 formatters, got %d", len(names))
	}

	// Check all are present
	nameMap := make(map[string]bool)
	for _, n := range names {
		nameMap[n] = true
	}
	for _, expected := range []string{"table", "json", "yaml"} {
		if !nameMap[expected] {
			t.Errorf("expected %q in list", expected)
		}
	}
}

// ===========================================
// Global Functions Tests
// ===========================================

func TestGlobalFunctions(t *testing.T) {
	// Save and restore the default registry
	originalRegistry := DefaultRegistry
	defer func() { DefaultRegistry = originalRegistry }()

	DefaultRegistry = NewRegistry()

	// Test Register
	f := NewTableFormatter()
	err := Register(f)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Test Get
	got, ok := Get("table")
	if !ok {
		t.Fatal("expected to find 'table' formatter")
	}
	if got.Name() != "table" {
		t.Errorf("expected 'table', got %q", got.Name())
	}

	// Test Default
	d := Default()
	if d == nil {
		t.Fatal("expected non-nil default")
	}

	// Test List
	names := List()
	if len(names) != 1 || names[0] != "table" {
		t.Errorf("expected ['table'], got %v", names)
	}
}

// ===========================================
// TableFormatter Tests
// ===========================================

func TestNewTableFormatter(t *testing.T) {
	f := NewTableFormatter()
	if f == nil {
		t.Fatal("NewTableFormatter returned nil")
	}
}

func TestTableFormatter_Name(t *testing.T) {
	f := NewTableFormatter()
	if f.Name() != "table" {
		t.Errorf("expected 'table', got %q", f.Name())
	}
}

func TestTableFormatter_Description(t *testing.T) {
	f := NewTableFormatter()
	if f.Description() != "Aligned text table output" {
		t.Errorf("unexpected description: %q", f.Description())
	}
}

func TestTableFormatter_FormatList_Empty(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, []map[string]any{}, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No records found") {
		t.Errorf("expected 'No records found' message, got: %q", output)
	}
}

func TestTableFormatter_FormatList_WithRecords(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, records, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	output := buf.String()
	// Should contain header and values
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %q", output)
	}
	if !strings.Contains(output, "Bob") {
		t.Errorf("expected 'Bob' in output, got: %q", output)
	}
}

func TestTableFormatter_FormatList_WithColumns(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	opts := FormatOptions{
		Columns: []string{"name", "email"},
	}

	err := f.FormatList(&buf, mod, records, opts)
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	output := buf.String()
	// Should contain NAME and EMAIL headers
	if !strings.Contains(output, "NAME") {
		t.Errorf("expected 'NAME' header, got: %q", output)
	}
	if !strings.Contains(output, "EMAIL") {
		t.Errorf("expected 'EMAIL' header, got: %q", output)
	}
}

func TestTableFormatter_FormatList_NoHeader(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	opts := FormatOptions{
		NoHeader: true,
		Columns:  []string{"name"},
	}

	err := f.FormatList(&buf, mod, records, opts)
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	output := buf.String()
	// Should not contain header
	if strings.Contains(output, "NAME") {
		t.Errorf("expected no 'NAME' header with NoHeader option, got: %q", output)
	}
	// But should still contain values
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %q", output)
	}
}

func TestTableFormatter_FormatRecord_Nil(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, nil, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Record not found") {
		t.Errorf("expected 'Record not found' message, got: %q", output)
	}
}

func TestTableFormatter_FormatRecord_WithData(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	record := map[string]any{
		"id":    "uuid-1",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, record, FormatOptions{
		Columns: []string{"name", "email"},
	})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	output := buf.String()
	// Should contain key-value format
	if !strings.Contains(output, "Name:") {
		t.Errorf("expected 'Name:' label, got: %q", output)
	}
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected 'Alice' value, got: %q", output)
	}
}

func TestTableFormatter_FormatError(t *testing.T) {
	f := NewTableFormatter()
	var buf bytes.Buffer

	testErr := errors.New("test error message")
	err := f.FormatError(&buf, testErr)
	if err != nil {
		t.Fatalf("FormatError failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Errorf("expected 'Error:' prefix, got: %q", output)
	}
	if !strings.Contains(output, "test error message") {
		t.Errorf("expected error message, got: %q", output)
	}
}

func TestTableFormatter_FormatValue(t *testing.T) {
	f := NewTableFormatter()

	tests := []struct {
		name     string
		val      any
		maxWidth int
		expected string
	}{
		{"nil", nil, 0, "-"},
		{"string", "hello", 0, "hello"},
		{"bool true", true, 0, "yes"},
		{"bool false", false, 0, "no"},
		{"bytes", []byte{1, 2, 3}, 0, "[binary]"},
		{"float whole", float64(42), 0, "42"},
		{"float decimal", float64(3.14159), 0, "3.14"},
		{"int slice", []int{1, 2, 3}, 0, "[1,2,3]"},
		{"truncate", "this is a very long string", 10, "this is..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatValue(tt.val, tt.maxWidth)
			if got != tt.expected {
				t.Errorf("formatValue(%v, %d) = %q, want %q", tt.val, tt.maxWidth, got, tt.expected)
			}
		})
	}
}

func TestTableFormatter_FormatLabel(t *testing.T) {
	f := NewTableFormatter()

	tests := []struct {
		name     string
		expected string
	}{
		{"name", "Name"},
		{"email_address", "Email Address"},
		{"created_at", "Created At"},
		{"id", "Id"},
		{"password_hash", "Password Hash"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatLabel(tt.name)
			if got != tt.expected {
				t.Errorf("formatLabel(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestTableFormatter_ResolveColumns_WithRequested(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()

	// When columns are requested, use them
	cols := f.resolveColumns(mod, []string{"name", "email"})
	if len(cols) != 2 || cols[0] != "name" || cols[1] != "email" {
		t.Errorf("expected [name, email], got %v", cols)
	}
}

func TestTableFormatter_ResolveColumns_Default(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()

	// When no columns requested, exclude internal and secret fields
	cols := f.resolveColumns(mod, nil)

	// Should include id, name, email, age but not password_hash or internal_notes
	colMap := make(map[string]bool)
	for _, c := range cols {
		colMap[c] = true
	}

	if !colMap["id"] || !colMap["name"] || !colMap["email"] || !colMap["age"] {
		t.Errorf("expected public fields, got %v", cols)
	}
	if colMap["password_hash"] || colMap["internal_notes"] {
		t.Errorf("should not include internal/secret fields, got %v", cols)
	}
}

// ===========================================
// JSONFormatter Tests
// ===========================================

func TestNewJSONFormatter(t *testing.T) {
	f := NewJSONFormatter()
	if f == nil {
		t.Fatal("NewJSONFormatter returned nil")
	}
}

func TestJSONFormatter_Name(t *testing.T) {
	f := NewJSONFormatter()
	if f.Name() != "json" {
		t.Errorf("expected 'json', got %q", f.Name())
	}
}

func TestJSONFormatter_Description(t *testing.T) {
	f := NewJSONFormatter()
	if f.Description() != "JSON output format" {
		t.Errorf("unexpected description: %q", f.Description())
	}
}

func TestJSONFormatter_FormatList(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, records, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	// Parse the output
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["module"] != "user" {
		t.Errorf("expected module 'user', got %v", result["module"])
	}
	if result["count"] != float64(2) {
		t.Errorf("expected count 2, got %v", result["count"])
	}
	data, ok := result["data"].([]any)
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 data items, got %v", result["data"])
	}
}

func TestJSONFormatter_FormatList_Compact(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, records, FormatOptions{Compact: true})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	output := buf.String()
	// Compact should not have indentation
	if strings.Contains(output, "  ") {
		t.Errorf("compact output should not have indentation")
	}
}

func TestJSONFormatter_FormatList_WithColumns(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	opts := FormatOptions{
		Columns: []string{"name"},
	}

	err := f.FormatList(&buf, mod, records, opts)
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	data := result["data"].([]any)
	firstRecord := data[0].(map[string]any)

	// Should only have 'name' field
	if _, ok := firstRecord["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := firstRecord["email"]; ok {
		t.Error("should not have 'email' field when filtered")
	}
}

func TestJSONFormatter_FormatRecord_Nil(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, nil, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["data"] != nil {
		t.Errorf("expected nil data for nil record, got %v", result["data"])
	}
}

func TestJSONFormatter_FormatRecord_WithData(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	record := map[string]any{
		"id":    "uuid-1",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, record, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["module"] != "user" {
		t.Errorf("expected module 'user', got %v", result["module"])
	}

	data := result["data"].(map[string]any)
	if data["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", data["name"])
	}
}

func TestJSONFormatter_FormatRecord_WithColumns(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	record := map[string]any{
		"id":    "uuid-1",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	var buf bytes.Buffer

	opts := FormatOptions{
		Columns: []string{"name"},
	}

	err := f.FormatRecord(&buf, mod, record, opts)
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	data := result["data"].(map[string]any)
	if _, ok := data["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := data["email"]; ok {
		t.Error("should not have 'email' field when filtered")
	}
}

func TestJSONFormatter_FormatError(t *testing.T) {
	f := NewJSONFormatter()
	var buf bytes.Buffer

	testErr := errors.New("test error message")
	err := f.FormatError(&buf, testErr)
	if err != nil {
		t.Fatalf("FormatError failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["error"] != "test error message" {
		t.Errorf("expected error 'test error message', got %v", result["error"])
	}
}

func TestJSONFormatter_RemoveInternal(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()

	records := []map[string]any{
		{
			"id":             "uuid-1",
			"name":           "Alice",
			"password_hash":  "secret",
			"internal_notes": "internal",
		},
	}

	result := f.removeInternal(mod, records)
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}

	// Should have id and name but not password_hash or internal_notes
	if _, ok := result[0]["id"]; !ok {
		t.Error("expected 'id' field")
	}
	if _, ok := result[0]["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := result[0]["password_hash"]; ok {
		t.Error("should not have 'password_hash' (secret) field")
	}
	if _, ok := result[0]["internal_notes"]; ok {
		t.Error("should not have 'internal_notes' (internal) field")
	}
}

// ===========================================
// YAMLFormatter Tests
// ===========================================

func TestNewYAMLFormatter(t *testing.T) {
	f := NewYAMLFormatter()
	if f == nil {
		t.Fatal("NewYAMLFormatter returned nil")
	}
}

func TestYAMLFormatter_Name(t *testing.T) {
	f := NewYAMLFormatter()
	if f.Name() != "yaml" {
		t.Errorf("expected 'yaml', got %q", f.Name())
	}
}

func TestYAMLFormatter_Description(t *testing.T) {
	f := NewYAMLFormatter()
	if f.Description() != "YAML output format" {
		t.Errorf("unexpected description: %q", f.Description())
	}
}

func TestYAMLFormatter_FormatList(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, records, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	// Parse the output
	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if result["module"] != "user" {
		t.Errorf("expected module 'user', got %v", result["module"])
	}
	if result["count"] != 2 {
		t.Errorf("expected count 2, got %v", result["count"])
	}
	data, ok := result["data"].([]any)
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 data items, got %v", result["data"])
	}
}

func TestYAMLFormatter_FormatList_WithColumns(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	records := createTestRecords()
	var buf bytes.Buffer

	opts := FormatOptions{
		Columns: []string{"name"},
	}

	err := f.FormatList(&buf, mod, records, opts)
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	data := result["data"].([]any)
	firstRecord := data[0].(map[string]any)

	// Should only have 'name' field
	if _, ok := firstRecord["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := firstRecord["email"]; ok {
		t.Error("should not have 'email' field when filtered")
	}
}

func TestYAMLFormatter_FormatRecord_Nil(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, nil, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if result["data"] != nil {
		t.Errorf("expected nil data for nil record, got %v", result["data"])
	}
}

func TestYAMLFormatter_FormatRecord_WithData(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	record := map[string]any{
		"id":    "uuid-1",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, record, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if result["module"] != "user" {
		t.Errorf("expected module 'user', got %v", result["module"])
	}

	data := result["data"].(map[string]any)
	if data["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", data["name"])
	}
}

func TestYAMLFormatter_FormatRecord_WithColumns(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	record := map[string]any{
		"id":    "uuid-1",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	var buf bytes.Buffer

	opts := FormatOptions{
		Columns: []string{"name"},
	}

	err := f.FormatRecord(&buf, mod, record, opts)
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	data := result["data"].(map[string]any)
	if _, ok := data["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := data["email"]; ok {
		t.Error("should not have 'email' field when filtered")
	}
}

func TestYAMLFormatter_FormatError(t *testing.T) {
	f := NewYAMLFormatter()
	var buf bytes.Buffer

	testErr := errors.New("test error message")
	err := f.FormatError(&buf, testErr)
	if err != nil {
		t.Fatalf("FormatError failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if result["error"] != "test error message" {
		t.Errorf("expected error 'test error message', got %v", result["error"])
	}
}

func TestYAMLFormatter_RemoveInternal(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()

	records := []map[string]any{
		{
			"id":             "uuid-1",
			"name":           "Alice",
			"password_hash":  "secret",
			"internal_notes": "internal",
		},
	}

	result := f.removeInternal(mod, records)
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}

	// Should have id and name but not password_hash or internal_notes
	if _, ok := result[0]["id"]; !ok {
		t.Error("expected 'id' field")
	}
	if _, ok := result[0]["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := result[0]["password_hash"]; ok {
		t.Error("should not have 'password_hash' (secret) field")
	}
	if _, ok := result[0]["internal_notes"]; ok {
		t.Error("should not have 'internal_notes' (internal) field")
	}
}

func TestYAMLFormatter_FilterRecords_WithColumns(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()

	records := []map[string]any{
		{"id": "uuid-1", "name": "Alice", "email": "alice@example.com"},
		{"id": "uuid-2", "name": "Bob", "email": "bob@example.com"},
	}

	result := f.filterRecords(mod, records, []string{"name"})
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}

	// Each record should only have 'name'
	for i, r := range result {
		if _, ok := r["name"]; !ok {
			t.Errorf("record %d: expected 'name' field", i)
		}
		if _, ok := r["email"]; ok {
			t.Errorf("record %d: should not have 'email' field", i)
		}
	}
}

// ===========================================
// FormatOptions Tests
// ===========================================

func TestFormatOptions_Defaults(t *testing.T) {
	opts := FormatOptions{}

	if opts.NoHeader != false {
		t.Error("NoHeader should default to false")
	}
	if opts.Compact != false {
		t.Error("Compact should default to false")
	}
	if opts.Color != false {
		t.Error("Color should default to false")
	}
	if opts.MaxWidth != 0 {
		t.Error("MaxWidth should default to 0")
	}
	if opts.Columns != nil {
		t.Error("Columns should default to nil")
	}
}

// ===========================================
// Edge Cases and Integration Tests
// ===========================================

func TestTableFormatter_EmptyRecord(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatRecord(&buf, mod, map[string]any{}, FormatOptions{
		Columns: []string{"name", "email"},
	})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	output := buf.String()
	// Should show labels with "-" for missing values
	if !strings.Contains(output, "-") {
		t.Errorf("expected '-' for empty values, got: %q", output)
	}
}

func TestJSONFormatter_EmptyRecords(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, []map[string]any{}, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", result["count"])
	}
	data := result["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data, got %v", data)
	}
}

func TestYAMLFormatter_EmptyRecords(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()
	var buf bytes.Buffer

	err := f.FormatList(&buf, mod, []map[string]any{}, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	if result["count"] != 0 {
		t.Errorf("expected count 0, got %v", result["count"])
	}
}

func TestTableFormatter_MaxWidth_EdgeCases(t *testing.T) {
	f := NewTableFormatter()

	// Test exactly at max width
	val := f.formatValue("123456789", 9)
	if val != "123456789" {
		t.Errorf("expected '123456789', got %q", val)
	}

	// Test just over max width
	val = f.formatValue("1234567890", 9)
	if val != "123456..." {
		t.Errorf("expected '123456...', got %q", val)
	}

	// Test very small max width
	val = f.formatValue("hello", 4)
	if val != "h..." {
		t.Errorf("expected 'h...', got %q", val)
	}
}

func TestJSONFormatter_FilterRecord_NonexistentColumn(t *testing.T) {
	f := NewJSONFormatter()
	mod := createTestDerived()

	record := map[string]any{
		"id":   "uuid-1",
		"name": "Alice",
	}

	// Filter with nonexistent column
	result := f.filterRecord(mod, record, []string{"name", "nonexistent"})

	if _, ok := result["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := result["nonexistent"]; ok {
		t.Error("should not have 'nonexistent' field since it doesn't exist in record")
	}
}

func TestYAMLFormatter_FilterRecord_NonexistentColumn(t *testing.T) {
	f := NewYAMLFormatter()
	mod := createTestDerived()

	record := map[string]any{
		"id":   "uuid-1",
		"name": "Alice",
	}

	// Filter with nonexistent column
	result := f.filterRecord(mod, record, []string{"name", "nonexistent"})

	if _, ok := result["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := result["nonexistent"]; ok {
		t.Error("should not have 'nonexistent' field since it doesn't exist in record")
	}
}

// ===========================================
// Concurrency Tests
// ===========================================

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewTableFormatter())

	// Run multiple goroutines accessing the registry
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = r.Get("table")
				_ = r.List()
				_ = r.Default()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ===========================================
// Interface Compliance Tests
// ===========================================

func TestTableFormatter_ImplementsInterface(t *testing.T) {
	var _ Formatter = (*TableFormatter)(nil)
}

func TestJSONFormatter_ImplementsInterface(t *testing.T) {
	var _ Formatter = (*JSONFormatter)(nil)
}

func TestYAMLFormatter_ImplementsInterface(t *testing.T) {
	var _ Formatter = (*YAMLFormatter)(nil)
}

// ===========================================
// Special Value Tests
// ===========================================

func TestTableFormatter_SpecialValues(t *testing.T) {
	f := NewTableFormatter()
	mod := createTestDerived()

	record := map[string]any{
		"id":      "uuid-1",
		"name":    nil,
		"active":  true,
		"balance": 123.45,
		"data":    []byte{1, 2, 3},
	}

	var buf bytes.Buffer
	err := f.FormatRecord(&buf, mod, record, FormatOptions{
		Columns: []string{"name", "active", "balance", "data"},
	})
	if err != nil {
		t.Fatalf("FormatRecord failed: %v", err)
	}

	output := buf.String()

	// nil should show as "-"
	if !strings.Contains(output, "-") {
		t.Errorf("expected '-' for nil value, got: %q", output)
	}

	// true should show as "yes"
	if !strings.Contains(output, "yes") {
		t.Errorf("expected 'yes' for true value, got: %q", output)
	}

	// bytes should show as "[binary]"
	if !strings.Contains(output, "[binary]") {
		t.Errorf("expected '[binary]' for bytes value, got: %q", output)
	}
}

func TestTableFormatter_ComplexJSONValue(t *testing.T) {
	f := NewTableFormatter()

	// Test nested map
	val := f.formatValue(map[string]any{"key": "value"}, 0)
	if !strings.Contains(val, "key") || !strings.Contains(val, "value") {
		t.Errorf("expected JSON for map, got: %q", val)
	}

	// Test slice
	val = f.formatValue([]string{"a", "b", "c"}, 0)
	if !strings.Contains(val, "a") {
		t.Errorf("expected JSON for slice, got: %q", val)
	}
}

func TestTableFormatter_FloatFormatting(t *testing.T) {
	f := NewTableFormatter()

	tests := []struct {
		val      float64
		expected string
	}{
		{0.0, "0"},
		{1.0, "1"},
		{100.0, "100"},
		{-5.0, "-5"},
		{3.14, "3.14"},
		{0.01, "0.01"},
		{123.456, "123.46"},
	}

	for _, tt := range tests {
		got := f.formatValue(tt.val, 0)
		if got != tt.expected {
			t.Errorf("formatValue(%v) = %q, want %q", tt.val, got, tt.expected)
		}
	}
}

// ===========================================
// Empty/Nil Module Tests
// ===========================================

func TestTableFormatter_EmptyModule(t *testing.T) {
	f := NewTableFormatter()
	mod := convention.Derived{
		Source: schema.Module{Name: "empty"},
		Fields: []convention.DerivedField{},
	}

	var buf bytes.Buffer
	records := []map[string]any{{"any": "value"}}

	// Should not panic with empty fields
	err := f.FormatList(&buf, mod, records, FormatOptions{})
	if err != nil {
		t.Fatalf("FormatList failed: %v", err)
	}
}

func TestJSONFormatter_RemoveInternalSingle_EmptyModule(t *testing.T) {
	f := NewJSONFormatter()
	mod := convention.Derived{
		Source: schema.Module{Name: "empty"},
		Fields: []convention.DerivedField{},
	}

	record := map[string]any{
		"id":   "uuid-1",
		"name": "Alice",
	}

	// With no fields defined, all fields should be kept
	result := f.removeInternalSingle(mod, record)
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}
}
