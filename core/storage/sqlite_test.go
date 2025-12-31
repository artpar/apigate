package storage

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

func TestSQLiteStore(t *testing.T) {
	// Create in-memory database
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	// Create a test module
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name":  {Type: schema.FieldTypeString},
			"price": {Type: schema.FieldTypeInt, Default: 0},
			"sku":   {Type: schema.FieldTypeString, Unique: true, Lookup: true},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	// Test CreateTable
	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Test Create
	id, err := store.Create(ctx, "product", map[string]any{
		"name":  "Widget",
		"price": 100,
		"sku":   "WGT-001",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id == "" {
		t.Error("Create returned empty ID")
	}

	// Test Get by id
	record, err := store.Get(ctx, "product", "id", id)
	if err != nil {
		t.Fatalf("Get by id failed: %v", err)
	}
	if record == nil {
		t.Fatal("Get returned nil record")
	}
	if record["name"] != "Widget" {
		t.Errorf("name = %v, want Widget", record["name"])
	}
	if record["sku"] != "WGT-001" {
		t.Errorf("sku = %v, want WGT-001", record["sku"])
	}

	// Test Get by lookup field
	record, err = store.Get(ctx, "product", "sku", "WGT-001")
	if err != nil {
		t.Fatalf("Get by sku failed: %v", err)
	}
	if record == nil {
		t.Fatal("Get by sku returned nil record")
	}
	if record["name"] != "Widget" {
		t.Errorf("name = %v, want Widget", record["name"])
	}

	// Test Update
	if err := store.Update(ctx, "product", id, map[string]any{
		"price": 150,
	}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	record, _ = store.Get(ctx, "product", "id", id)
	if price, ok := record["price"].(int64); !ok || price != 150 {
		t.Errorf("price = %v, want 150", record["price"])
	}

	// Test List
	store.Create(ctx, "product", map[string]any{
		"name":  "Gadget",
		"price": 200,
		"sku":   "GDT-001",
	})

	list, count, err := store.List(ctx, "product", ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}

	// Test List with filter
	list, count, err = store.List(ctx, "product", ListOptions{
		Filters: map[string]any{"name": "Widget"},
	})
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}
	if count != 1 {
		t.Errorf("filtered count = %d, want 1", count)
	}

	// Test Delete
	if err := store.Delete(ctx, "product", id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	record, _ = store.Get(ctx, "product", "id", id)
	if record != nil {
		t.Error("Record still exists after delete")
	}
}

func TestBuildCreateTableSQL(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"email":    {Type: schema.FieldTypeEmail, Unique: true, Lookup: true},
			"password": {Type: schema.FieldTypeSecret},
			"status":   {Type: schema.FieldTypeEnum, Values: []string{"active", "inactive"}, Default: "active"},
			"plan":     {Type: schema.FieldTypeRef, To: "plan"},
		},
	}

	derived := convention.Derive(mod)
	sql := BuildCreateTableSQL(derived)

	// Check SQL contains expected parts
	expectedParts := []string{
		"CREATE TABLE IF NOT EXISTS users",
		"id TEXT PRIMARY KEY",
		"email TEXT",
		"password BLOB",
		"status TEXT",
		"plan TEXT",
		"created_at TEXT",
		"updated_at TEXT",
		"UNIQUE(email)",
		"FOREIGN KEY(plan) REFERENCES plans(id)",
	}

	for _, part := range expectedParts {
		if !containsString(sql, part) {
			t.Errorf("SQL missing expected part: %s\nGot: %s", part, sql)
		}
	}
}

func TestBuildIndexSQL(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"email": {Type: schema.FieldTypeEmail, Unique: true, Lookup: true},
			"name":  {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	indexes := BuildIndexSQL(derived)

	if len(indexes) != 1 {
		t.Errorf("len(indexes) = %d, want 1", len(indexes))
	}

	if len(indexes) > 0 && !containsString(indexes[0], "idx_users_email") {
		t.Errorf("Index missing expected name, got: %s", indexes[0])
	}
}

func TestDefaultValues(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "config",
		Schema: map[string]schema.Field{
			"key":     {Type: schema.FieldTypeString},
			"enabled": {Type: schema.FieldTypeBool, Default: true},
			"count":   {Type: schema.FieldTypeInt, Default: 10},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Create without providing default fields
	id, err := store.Create(ctx, "config", map[string]any{
		"key": "test",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	record, _ := store.Get(ctx, "config", "id", id)

	// Check defaults were applied
	if enabled, ok := record["enabled"].(bool); !ok || !enabled {
		t.Errorf("enabled = %v, want true", record["enabled"])
	}
	if count, ok := record["count"].(int64); !ok || count != 10 {
		t.Errorf("count = %v, want 10", record["count"])
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestNewSQLiteStoreFromDB tests creating a store from an existing DB connection
func TestNewSQLiteStoreFromDB(t *testing.T) {
	// First create a store to get a DB connection
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	db := store.DB()
	if db == nil {
		t.Fatal("DB() returned nil")
	}

	// Create a new store from the existing DB
	store2 := NewSQLiteStoreFromDB(db)
	if store2 == nil {
		t.Fatal("NewSQLiteStoreFromDB returned nil")
	}

	// Verify the DB is accessible
	if store2.DB() != db {
		t.Error("NewSQLiteStoreFromDB did not preserve DB connection")
	}
}

// TestDBMethod tests the DB() method
func TestDBMethod(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	db := store.DB()
	if db == nil {
		t.Fatal("DB() returned nil")
	}

	// Test that we can use the returned DB
	_, err = db.Exec("CREATE TABLE test (id TEXT)")
	if err != nil {
		t.Errorf("Failed to use returned DB: %v", err)
	}
}

// TestCreateUnregisteredModule tests Create with an unregistered module
func TestCreateUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = store.Create(ctx, "nonexistent", map[string]any{"name": "test"})
	if err == nil {
		t.Error("Expected error for unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestGetUnregisteredModule tests Get with an unregistered module
func TestGetUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = store.Get(ctx, "nonexistent", "id", "123")
	if err == nil {
		t.Error("Expected error for unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestListUnregisteredModule tests List with an unregistered module
func TestListUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_, _, err = store.List(ctx, "nonexistent", ListOptions{})
	if err == nil {
		t.Error("Expected error for unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestUpdateUnregisteredModule tests Update with an unregistered module
func TestUpdateUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	err = store.Update(ctx, "nonexistent", "123", map[string]any{"name": "test"})
	if err == nil {
		t.Error("Expected error for unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestDeleteUnregisteredModule tests Delete with an unregistered module
func TestDeleteUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	err = store.Delete(ctx, "nonexistent", "123")
	if err == nil {
		t.Error("Expected error for unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestUpdateNonExistentRecord tests Update on a non-existent record
func TestUpdateNonExistentRecord(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	err = store.Update(ctx, "product", "nonexistent-id", map[string]any{"name": "test"})
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
	if !containsString(err.Error(), "record not found") {
		t.Errorf("Error should mention 'record not found', got: %v", err)
	}
}

// TestDeleteNonExistentRecord tests Delete on a non-existent record
func TestDeleteNonExistentRecord(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	err = store.Delete(ctx, "product", "nonexistent-id")
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
	if !containsString(err.Error(), "record not found") {
		t.Errorf("Error should mention 'record not found', got: %v", err)
	}
}

// TestUpdateEmptyData tests Update with no data to update
func TestUpdateEmptyData(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	id, err := store.Create(ctx, "product", map[string]any{"name": "test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update with empty data (only id and created_at which are skipped)
	err = store.Update(ctx, "product", id, map[string]any{
		"id":         "ignored",
		"created_at": "ignored",
	})
	if err != nil {
		t.Errorf("Update with empty data should succeed: %v", err)
	}
}

// TestUpdateUnknownField tests Update with unknown field names
func TestUpdateUnknownField(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	id, err := store.Create(ctx, "product", map[string]any{"name": "test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update with unknown field (should be skipped)
	err = store.Update(ctx, "product", id, map[string]any{
		"unknown_field": "ignored",
		"name":          "updated",
	})
	if err != nil {
		t.Errorf("Update with unknown field should succeed: %v", err)
	}

	// Verify name was updated
	record, _ := store.Get(ctx, "product", "id", id)
	if record["name"] != "updated" {
		t.Errorf("name = %v, want 'updated'", record["name"])
	}
}

// TestCreateMissingRequiredField tests Create with missing required field
func TestCreateMissingRequiredField(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	requiredTrue := true
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name":        {Type: schema.FieldTypeString, Required: &requiredTrue},
			"description": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Create without required field
	_, err = store.Create(ctx, "product", map[string]any{"description": "test"})
	if err == nil {
		t.Error("Expected error for missing required field")
	}
	if !containsString(err.Error(), "required field") {
		t.Errorf("Error should mention 'required field', got: %v", err)
	}
}

// TestCreateWithProvidedID tests Create with a pre-provided ID
func TestCreateWithProvidedID(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	customID := "custom-uuid-1234"
	id, err := store.Create(ctx, "product", map[string]any{
		"id":   customID,
		"name": "test",
	})
	if err != nil {
		t.Fatalf("Create with custom ID failed: %v", err)
	}
	if id != customID {
		t.Errorf("id = %v, want %v", id, customID)
	}

	// Verify we can retrieve by custom ID
	record, _ := store.Get(ctx, "product", "id", customID)
	if record == nil {
		t.Fatal("Record with custom ID not found")
	}
}

// TestListWithOrderBy tests List with ordering options
func TestListWithOrderBy(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name":  {Type: schema.FieldTypeString},
			"price": {Type: schema.FieldTypeInt},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Create products
	store.Create(ctx, "product", map[string]any{"name": "A", "price": 100})
	store.Create(ctx, "product", map[string]any{"name": "B", "price": 200})
	store.Create(ctx, "product", map[string]any{"name": "C", "price": 50})

	// Test order by name ascending
	list, _, err := store.List(ctx, "product", ListOptions{
		OrderBy:   "name",
		OrderDesc: false,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) > 0 && list[0]["name"] != "A" {
		t.Errorf("First item should be 'A', got: %v", list[0]["name"])
	}

	// Test order by price descending
	list, _, err = store.List(ctx, "product", ListOptions{
		OrderBy:   "price",
		OrderDesc: true,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) > 0 && list[0]["name"] != "B" {
		t.Errorf("First item should be 'B' (highest price), got: %v", list[0]["name"])
	}

	// Test with invalid orderBy field (should fall back to created_at)
	list, _, err = store.List(ctx, "product", ListOptions{
		OrderBy: "invalid_field",
	})
	if err != nil {
		t.Fatalf("List with invalid orderBy failed: %v", err)
	}
	// Should not error, just use created_at
}

// TestListWithPagination tests List with limit and offset
func TestListWithPagination(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	derived := convention.Derive(mod)
	ctx := context.Background()

	if err := store.CreateTable(ctx, derived); err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Create 5 products
	for i := 1; i <= 5; i++ {
		store.Create(ctx, "product", map[string]any{"name": string(rune('A' + i - 1))})
	}

	// Test with limit
	list, count, err := store.List(ctx, "product", ListOptions{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}

	// Test with offset
	list, _, err = store.List(ctx, "product", ListOptions{
		Limit:  2,
		Offset: 2,
	})
	if err != nil {
		t.Fatalf("List with offset failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}

// TestConvertValueBool tests convertValue for bool type
func TestConvertValueBool(t *testing.T) {
	field := convention.DerivedField{Type: "bool"}

	tests := []struct {
		input    any
		expected any
	}{
		{true, 1},
		{false, 0},
		{"true", 1},
		{"1", 1},
		{"false", 0},
		{"other", 0},
		{nil, nil},
		{123, 0}, // non-bool, non-string defaults to 0
	}

	for _, tt := range tests {
		result := convertValue(tt.input, field)
		if result != tt.expected {
			t.Errorf("convertValue(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestConvertValueSecret tests convertValue for secret/bytes type
func TestConvertValueSecret(t *testing.T) {
	secretField := convention.DerivedField{Type: "secret"}
	bytesField := convention.DerivedField{Type: "bytes"}

	// Test string conversion to bytes
	result := convertValue("password123", secretField)
	if bytes, ok := result.([]byte); !ok || string(bytes) != "password123" {
		t.Errorf("convertValue(string) for secret type failed: %v", result)
	}

	// Test bytes passthrough
	input := []byte("binary data")
	result = convertValue(input, bytesField)
	if bytes, ok := result.([]byte); !ok || string(bytes) != "binary data" {
		t.Errorf("convertValue([]byte) for bytes type failed: %v", result)
	}

	// Test nil
	result = convertValue(nil, secretField)
	if result != nil {
		t.Errorf("convertValue(nil) should return nil, got: %v", result)
	}
}

// TestConvertFromDBBool tests convertFromDB for bool type
func TestConvertFromDBBool(t *testing.T) {
	field := convention.DerivedField{Type: "bool"}

	tests := []struct {
		input    any
		expected any
	}{
		{int64(1), true},
		{int64(0), false},
		{int(1), true},
		{int(0), false},
		{nil, nil},
		{"other", false}, // Unknown type defaults to false
	}

	for _, tt := range tests {
		result := convertFromDB(tt.input, field)
		if result != tt.expected {
			t.Errorf("convertFromDB(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestConvertFromDBSecret tests convertFromDB for secret/bytes type
func TestConvertFromDBSecret(t *testing.T) {
	secretField := convention.DerivedField{Type: "secret"}
	bytesField := convention.DerivedField{Type: "bytes"}

	// Test bytes passthrough
	input := []byte("secret data")
	result := convertFromDB(input, secretField)
	if bytes, ok := result.([]byte); !ok || string(bytes) != "secret data" {
		t.Errorf("convertFromDB([]byte) for secret type failed: %v", result)
	}

	// Test string conversion to bytes (legacy)
	result = convertFromDB("legacy string", bytesField)
	if bytes, ok := result.([]byte); !ok || string(bytes) != "legacy string" {
		t.Errorf("convertFromDB(string) for bytes type should convert to []byte: %v", result)
	}

	// Test nil
	result = convertFromDB(nil, secretField)
	if result != nil {
		t.Errorf("convertFromDB(nil) should return nil, got: %v", result)
	}

	// Test other types (should pass through)
	result = convertFromDB(123, secretField)
	if result != 123 {
		t.Errorf("convertFromDB(int) should pass through, got: %v", result)
	}
}

// TestConvertFromDBDefault tests convertFromDB for default types
func TestConvertFromDBDefault(t *testing.T) {
	field := convention.DerivedField{Type: "string"}

	// Test byte slice to string conversion
	input := []byte("hello")
	result := convertFromDB(input, field)
	if result != "hello" {
		t.Errorf("convertFromDB([]byte) for string type = %v, want 'hello'", result)
	}

	// Test other types pass through
	result = convertFromDB("test", field)
	if result != "test" {
		t.Errorf("convertFromDB(string) = %v, want 'test'", result)
	}

	result = convertFromDB(123, field)
	if result != 123 {
		t.Errorf("convertFromDB(int) = %v, want 123", result)
	}
}

// TestValidateReferences tests reference validation
func TestValidateReferences(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create parent module (plan)
	planMod := schema.Module{
		Name: "plan",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}
	planDerived := convention.Derive(planMod)
	if err := store.CreateTable(ctx, planDerived); err != nil {
		t.Fatalf("CreateTable for plan failed: %v", err)
	}

	// Create child module (user with plan reference)
	userMod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
			"plan": {Type: schema.FieldTypeRef, To: "plan"},
		},
	}
	userDerived := convention.Derive(userMod)
	if err := store.CreateTable(ctx, userDerived); err != nil {
		t.Fatalf("CreateTable for user failed: %v", err)
	}

	// Create a plan
	planID, err := store.Create(ctx, "plan", map[string]any{"name": "Premium"})
	if err != nil {
		t.Fatalf("Create plan failed: %v", err)
	}

	// Create user with valid reference
	_, err = store.Create(ctx, "user", map[string]any{
		"name": "John",
		"plan": planID,
	})
	if err != nil {
		t.Errorf("Create with valid reference should succeed: %v", err)
	}

	// Create user with invalid reference
	_, err = store.Create(ctx, "user", map[string]any{
		"name": "Jane",
		"plan": "nonexistent-plan-id",
	})
	if err == nil {
		t.Error("Expected error for invalid reference")
	}
	if !containsString(err.Error(), "does not exist") {
		t.Errorf("Error should mention 'does not exist', got: %v", err)
	}

	// Create user with empty reference (now fails with FK constraint - empty string is not a valid FK)
	_, err = store.Create(ctx, "user", map[string]any{
		"name": "Bob",
		"plan": "",
	})
	if err == nil {
		t.Errorf("Create with empty reference should fail with FK constraint")
	}

	// Create user with nil reference (should succeed)
	_, err = store.Create(ctx, "user", map[string]any{
		"name": "Alice",
	})
	if err != nil {
		t.Errorf("Create with nil reference should succeed: %v", err)
	}
}

// TestValidateReferencesUnregisteredModule tests reference validation with unregistered target
func TestValidateReferencesUnregisteredModule(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create module with reference to unregistered module
	userMod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name":    {Type: schema.FieldTypeString},
			"company": {Type: schema.FieldTypeRef, To: "company"}, // company not registered
		},
	}
	userDerived := convention.Derive(userMod)
	if err := store.CreateTable(ctx, userDerived); err != nil {
		t.Fatalf("CreateTable for user failed: %v", err)
	}

	// Try to create with reference to unregistered module
	_, err = store.Create(ctx, "user", map[string]any{
		"name":    "John",
		"company": "some-company-id",
	})
	if err == nil {
		t.Error("Expected error for reference to unregistered module")
	}
	if !containsString(err.Error(), "not registered") {
		t.Errorf("Error should mention 'not registered', got: %v", err)
	}
}

// TestUpdateWithReferences tests Update with reference validation
func TestUpdateWithReferences(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create parent module (plan)
	planMod := schema.Module{
		Name: "plan",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}
	planDerived := convention.Derive(planMod)
	if err := store.CreateTable(ctx, planDerived); err != nil {
		t.Fatalf("CreateTable for plan failed: %v", err)
	}

	// Create child module (user with plan reference)
	userMod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
			"plan": {Type: schema.FieldTypeRef, To: "plan"},
		},
	}
	userDerived := convention.Derive(userMod)
	if err := store.CreateTable(ctx, userDerived); err != nil {
		t.Fatalf("CreateTable for user failed: %v", err)
	}

	// Create a plan
	planID, _ := store.Create(ctx, "plan", map[string]any{"name": "Premium"})

	// Create user without plan
	userID, _ := store.Create(ctx, "user", map[string]any{"name": "John"})

	// Update with valid reference
	err = store.Update(ctx, "user", userID, map[string]any{"plan": planID})
	if err != nil {
		t.Errorf("Update with valid reference should succeed: %v", err)
	}

	// Update with invalid reference
	err = store.Update(ctx, "user", userID, map[string]any{"plan": "nonexistent-plan-id"})
	if err == nil {
		t.Error("Expected error for invalid reference")
	}
}
