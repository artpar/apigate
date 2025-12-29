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
