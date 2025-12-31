package bootstrap

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

// mockRuntimeStorage implements runtime.Storage for testing.
type mockRuntimeStorage struct {
	tables map[string]bool
	data   map[string]map[string]map[string]any // module -> id -> data
}

func newMockRuntimeStorage() *mockRuntimeStorage {
	return &mockRuntimeStorage{
		tables: make(map[string]bool),
		data:   make(map[string]map[string]map[string]any),
	}
}

func (m *mockRuntimeStorage) CreateTable(ctx context.Context, mod convention.Derived) error {
	m.tables[mod.Source.Name] = true
	if m.data[mod.Source.Name] == nil {
		m.data[mod.Source.Name] = make(map[string]map[string]any)
	}
	return nil
}

func (m *mockRuntimeStorage) Create(ctx context.Context, module string, data map[string]any) (string, error) {
	if m.data[module] == nil {
		m.data[module] = make(map[string]map[string]any)
	}
	id := "id_" + module
	m.data[module][id] = data
	return id, nil
}

func (m *mockRuntimeStorage) Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error) {
	if m.data[module] == nil {
		return nil, nil
	}
	for id, data := range m.data[module] {
		if data[lookup] == value || id == value {
			result := make(map[string]any)
			for k, v := range data {
				result[k] = v
			}
			result["id"] = id
			return result, nil
		}
	}
	return nil, nil
}

func (m *mockRuntimeStorage) List(ctx context.Context, module string, opts runtime.ListOptions) ([]map[string]any, int64, error) {
	if m.data[module] == nil {
		return nil, 0, nil
	}
	var result []map[string]any
	for id, data := range m.data[module] {
		item := make(map[string]any)
		for k, v := range data {
			item[k] = v
		}
		item["id"] = id
		result = append(result, item)
	}
	return result, int64(len(result)), nil
}

func (m *mockRuntimeStorage) Update(ctx context.Context, module string, id string, data map[string]any) error {
	if m.data[module] == nil {
		return nil
	}
	if m.data[module][id] == nil {
		m.data[module][id] = make(map[string]any)
	}
	for k, v := range data {
		m.data[module][id][k] = v
	}
	return nil
}

func (m *mockRuntimeStorage) Delete(ctx context.Context, module string, id string) error {
	if m.data[module] != nil {
		delete(m.data[module], id)
	}
	return nil
}

func TestCoreModules(t *testing.T) {
	modules := CoreModules()

	if len(modules) == 0 {
		t.Fatal("CoreModules should return at least one module")
	}

	// Verify expected modules are present
	expectedModules := []string{"user", "plan", "api_key", "route", "upstream", "setting"}
	moduleMap := make(map[string]bool)
	for _, mod := range modules {
		moduleMap[mod.Name] = true
	}

	for _, expected := range expectedModules {
		if !moduleMap[expected] {
			t.Errorf("CoreModules should include %s module", expected)
		}
	}
}

func TestCoreModulesDir(t *testing.T) {
	dir := CoreModulesDir()
	if dir == "" {
		t.Error("CoreModulesDir should return a non-empty path")
	}

	expectedPath := filepath.Join("core", "modules")
	if dir != expectedPath {
		t.Errorf("CoreModulesDir should return %q, got %q", expectedPath, dir)
	}
}

func TestCoreUserModule(t *testing.T) {
	mod := coreUserModule()

	if mod.Name != "user" {
		t.Errorf("expected module name 'user', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"email", "password_hash", "name", "stripe_id", "plan_id", "status"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("user module should have field %q", field)
		}
	}

	// Verify actions
	expectedActions := []string{"activate", "suspend", "cancel", "set_password"}
	for _, action := range expectedActions {
		if _, ok := mod.Actions[action]; !ok {
			t.Errorf("user module should have action %q", action)
		}
	}
}

func TestCorePlanModule(t *testing.T) {
	mod := corePlanModule()

	if mod.Name != "plan" {
		t.Errorf("expected module name 'plan', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"name", "rate_limit_per_minute", "requests_per_month", "price_monthly", "enabled"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("plan module should have field %q", field)
		}
	}

	// Verify actions
	if _, ok := mod.Actions["enable"]; !ok {
		t.Error("plan module should have enable action")
	}
	if _, ok := mod.Actions["disable"]; !ok {
		t.Error("plan module should have disable action")
	}
}

func TestCoreAPIKeyModule(t *testing.T) {
	mod := coreAPIKeyModule()

	if mod.Name != "api_key" {
		t.Errorf("expected module name 'api_key', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"user_id", "hash", "prefix", "name", "scopes", "expires_at"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("api_key module should have field %q", field)
		}
	}

	// Verify revoke action
	if _, ok := mod.Actions["revoke"]; !ok {
		t.Error("api_key module should have revoke action")
	}
}

func TestCoreRouteModule(t *testing.T) {
	mod := coreRouteModule()

	if mod.Name != "route" {
		t.Errorf("expected module name 'route', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"name", "path_pattern", "match_type", "methods", "upstream_id", "enabled"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("route module should have field %q", field)
		}
	}

	// Verify hooks are defined
	if len(mod.Hooks) == 0 {
		t.Error("route module should have hooks for router reload")
	}
}

func TestCoreUpstreamModule(t *testing.T) {
	mod := coreUpstreamModule()

	if mod.Name != "upstream" {
		t.Errorf("expected module name 'upstream', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"name", "base_url", "timeout_ms", "max_idle_conns", "auth_type", "enabled"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("upstream module should have field %q", field)
		}
	}

	// Verify hooks are defined
	if len(mod.Hooks) == 0 {
		t.Error("upstream module should have hooks for router reload")
	}
}

func TestCoreSettingModule(t *testing.T) {
	mod := coreSettingModule()

	if mod.Name != "setting" {
		t.Errorf("expected module name 'setting', got %q", mod.Name)
	}

	// Verify schema fields
	expectedFields := []string{"key", "value", "encrypted"}
	for _, field := range expectedFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("setting module should have field %q", field)
		}
	}
}

func TestModuleRuntime_NewModuleRuntime(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	if mr.Runtime == nil {
		t.Error("ModuleRuntime should have a runtime")
	}
	if mr.Storage == nil {
		t.Error("ModuleRuntime should have storage")
	}
	if mr.HTTP == nil {
		t.Error("ModuleRuntime should have HTTP channel")
	}
	if mr.CLI == nil {
		t.Error("ModuleRuntime should have CLI channel")
	}
}

func TestModuleRuntime_LoadModules(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load core modules
	err = mr.LoadModules(context.Background(), ModuleConfig{
		EmbeddedModules: CoreModules(),
	})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	// Verify modules are loaded
	modules := mr.Modules()
	if len(modules) == 0 {
		t.Error("should have loaded at least one module")
	}
}

func TestModuleRuntime_Handler(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	handler := mr.Handler()
	if handler == nil {
		t.Error("Handler should return a non-nil HTTP handler")
	}
}

func TestModuleRuntime_MetricsHandler(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// MetricsHandler should not be nil since we register a prometheus exporter
	handler := mr.MetricsHandler()
	if handler == nil {
		t.Error("MetricsHandler should return a non-nil handler when prometheus exporter is registered")
	}
}

func TestModuleRuntime_StartStop(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}

	ctx := context.Background()

	// Start should not error
	err = mr.Start(ctx)
	if err != nil {
		t.Errorf("Start should not error: %v", err)
	}

	// Stop should not error
	err = mr.Stop(ctx)
	if err != nil {
		t.Errorf("Stop should not error: %v", err)
	}
}

func TestModuleRuntime_GetModule(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load core modules
	err = mr.LoadModules(context.Background(), ModuleConfig{
		EmbeddedModules: CoreModules(),
	})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	// Get a known module
	mod, ok := mr.GetModule("user")
	if !ok {
		t.Error("GetModule should find 'user' module")
	}
	if mod.Source.Name != "user" {
		t.Errorf("expected module name 'user', got %q", mod.Source.Name)
	}

	// Try to get non-existent module
	_, ok = mr.GetModule("nonexistent")
	if ok {
		t.Error("GetModule should not find 'nonexistent' module")
	}
}

func TestModuleRuntime_GetHTTPPaths(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load core modules
	err = mr.LoadModules(context.Background(), ModuleConfig{
		EmbeddedModules: CoreModules(),
	})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	paths := mr.GetHTTPPaths()
	// Should have some paths registered
	if len(paths) == 0 {
		t.Error("GetHTTPPaths should return registered paths")
	}
}

func TestModuleRuntime_GetCLIPaths(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load core modules
	err = mr.LoadModules(context.Background(), ModuleConfig{
		EmbeddedModules: CoreModules(),
	})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	paths := mr.GetCLIPaths()
	// Should have some CLI paths registered
	if len(paths) == 0 {
		t.Error("GetCLIPaths should return registered CLI paths")
	}
}

func TestModuleRuntime_LoadModulesFromDir_NonExistent(t *testing.T) {
	// Create temp database
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load from non-existent directory should not fail (just log warning)
	err = mr.LoadModules(context.Background(), ModuleConfig{
		ModulesDir: "/nonexistent/path",
	})
	if err != nil {
		t.Errorf("LoadModules from non-existent dir should not error: %v", err)
	}
}

func TestModuleRuntime_LoadModulesFromDir_Valid(t *testing.T) {
	// Create temp database and modules directory
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	modulesDir := filepath.Join(dir, "modules")
	os.MkdirAll(modulesDir, 0755)

	// Create a simple module YAML
	moduleYAML := `
module: test_module

meta:
  description: A test module

schema:
  name:
    type: string
    required: true
`
	err := os.WriteFile(filepath.Join(modulesDir, "test.yaml"), []byte(moduleYAML), 0644)
	if err != nil {
		t.Fatalf("write module yaml: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// Load from directory
	err = mr.LoadModules(context.Background(), ModuleConfig{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Errorf("LoadModules should not error: %v", err)
	}

	// Verify module was loaded
	_, ok := mr.GetModule("test_module")
	if !ok {
		t.Error("test_module should be loaded from YAML")
	}
}

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	if truePtr == nil || *truePtr != true {
		t.Error("boolPtr(true) should return pointer to true")
	}

	falsePtr := boolPtr(false)
	if falsePtr == nil || *falsePtr != false {
		t.Error("boolPtr(false) should return pointer to false")
	}
}

func TestCountHooks(t *testing.T) {
	hooks := map[string][]schema.Hook{
		"after_create": {{Call: "func1"}, {Call: "func2"}},
		"after_update": {{Call: "func3"}},
	}

	count := countHooks(hooks)
	if count != 3 {
		t.Errorf("countHooks should return 3, got %d", count)
	}

	// Empty hooks
	count = countHooks(map[string][]schema.Hook{})
	if count != 0 {
		t.Errorf("countHooks with empty map should return 0, got %d", count)
	}
}

func TestRuntimeStorageAdapter(t *testing.T) {
	// This tests the runtimeStorageAdapter that wraps SQLiteStore
	// Since it delegates to SQLiteStore, we just verify the adapter works

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	logger := zerolog.Nop()

	mr, err := NewModuleRuntime(db, nil, logger, ModuleConfig{})
	if err != nil {
		t.Fatalf("NewModuleRuntime: %v", err)
	}
	defer mr.Stop(context.Background())

	// The adapter is used internally by the runtime
	// We verify it works by loading and using modules
	err = mr.LoadModules(context.Background(), ModuleConfig{
		EmbeddedModules: CoreModules(),
	})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	// If we got here without errors, the adapter is working
}
