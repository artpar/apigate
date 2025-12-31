package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/settings"
	"github.com/rs/zerolog"
)

// mockSettingsStore implements ports.SettingsStore for testing.
type mockSettingsStore struct {
	mu     sync.RWMutex
	data   settings.Settings
	getErr error
	setErr error
}

func newMockSettingsStore() *mockSettingsStore {
	return &mockSettingsStore{
		data: make(settings.Settings),
	}
}

func (m *mockSettingsStore) Get(ctx context.Context, key string) (settings.Setting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return settings.Setting{}, m.getErr
	}
	v, ok := m.data[key]
	if !ok {
		return settings.Setting{}, errors.New("not found")
	}
	return settings.Setting{Key: key, Value: v}, nil
}

func (m *mockSettingsStore) GetAll(ctx context.Context) (settings.Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	result := make(settings.Settings, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result, nil
}

func (m *mockSettingsStore) GetByPrefix(ctx context.Context, prefix string) (settings.Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	result := make(settings.Settings)
	for k, v := range m.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v
		}
	}
	return result, nil
}

func (m *mockSettingsStore) Set(ctx context.Context, key, value string, encrypted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.data[key] = value
	return nil
}

func (m *mockSettingsStore) SetBatch(ctx context.Context, s settings.Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	for k, v := range s {
		m.data[k] = v
	}
	return nil
}

func (m *mockSettingsStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func TestNewSettingsService(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Should have default values
	settings := svc.Get()
	if settings.Get("server.port") != "8080" {
		t.Errorf("expected default port 8080, got %s", settings.Get("server.port"))
	}
}

func TestSettingsService_Load(t *testing.T) {
	store := newMockSettingsStore()
	store.data["server.port"] = "9000"
	store.data["portal.app_name"] = "MyApp"
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	err := svc.Load(ctx)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should merge with defaults
	settings := svc.Get()
	if settings.Get("server.port") != "9000" {
		t.Errorf("expected loaded port 9000, got %s", settings.Get("server.port"))
	}
	if settings.Get("portal.app_name") != "MyApp" {
		t.Errorf("expected loaded app name MyApp, got %s", settings.Get("portal.app_name"))
	}
	// Default should still be present
	if settings.Get("server.host") != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", settings.Get("server.host"))
	}
}

func TestSettingsService_Load_Error(t *testing.T) {
	store := newMockSettingsStore()
	store.getErr = errors.New("database error")
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	err := svc.Load(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSettingsService_Get(t *testing.T) {
	store := newMockSettingsStore()
	store.data["custom.key"] = "custom_value"
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()
	_ = svc.Load(ctx)

	// Get returns a copy
	settings1 := svc.Get()
	settings2 := svc.Get()

	// Modifying one should not affect the other
	settings1["custom.key"] = "modified"

	if settings2.Get("custom.key") != "custom_value" {
		t.Error("Get should return a copy, not the original")
	}
}

func TestSettingsService_GetValue(t *testing.T) {
	store := newMockSettingsStore()
	store.data["test.key"] = "test_value"
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()
	_ = svc.Load(ctx)

	value := svc.GetValue("test.key")
	if value != "test_value" {
		t.Errorf("expected test_value, got %s", value)
	}

	// Non-existent key should return empty (from defaults or empty)
	value = svc.GetValue("nonexistent.key")
	if value != "" {
		t.Errorf("expected empty string for nonexistent key, got %s", value)
	}
}

func TestSettingsService_Set(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	err := svc.Set(ctx, "custom.key", "custom_value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be reflected in cache
	value := svc.GetValue("custom.key")
	if value != "custom_value" {
		t.Errorf("expected custom_value, got %s", value)
	}

	// Should be in store
	if store.data["custom.key"] != "custom_value" {
		t.Error("value not stored")
	}
}

func TestSettingsService_Set_SensitiveKey(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	// Setting a sensitive key should work
	err := svc.Set(ctx, settings.KeyAuthJWTSecret, "my-secret")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value := svc.GetValue(settings.KeyAuthJWTSecret)
	if value != "my-secret" {
		t.Errorf("expected my-secret, got %s", value)
	}
}

func TestSettingsService_Set_Error(t *testing.T) {
	store := newMockSettingsStore()
	store.setErr = errors.New("store error")
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	err := svc.Set(ctx, "key", "value")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSettingsService_SetBatch(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	batch := settings.Settings{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := svc.SetBatch(ctx, batch)
	if err != nil {
		t.Fatalf("SetBatch failed: %v", err)
	}

	// All should be in cache
	if svc.GetValue("key1") != "value1" {
		t.Error("key1 not set")
	}
	if svc.GetValue("key2") != "value2" {
		t.Error("key2 not set")
	}
	if svc.GetValue("key3") != "value3" {
		t.Error("key3 not set")
	}
}

func TestSettingsService_SetBatch_Error(t *testing.T) {
	store := newMockSettingsStore()
	store.setErr = errors.New("batch error")
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	batch := settings.Settings{"key": "value"}
	err := svc.SetBatch(ctx, batch)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSettingsService_GetByPrefix(t *testing.T) {
	store := newMockSettingsStore()
	store.data["email.from_address"] = "test@example.com"
	store.data["email.from_name"] = "Test"
	store.data["server.port"] = "8080"
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()
	_ = svc.Load(ctx)

	emailSettings := svc.GetByPrefix("email.")
	// At least our 2 custom ones should be present (defaults may add more)
	if len(emailSettings) < 2 {
		t.Errorf("expected at least 2 email settings, got %d", len(emailSettings))
	}
	if emailSettings.Get("email.from_address") != "test@example.com" {
		t.Error("email.from_address not found")
	}
	if emailSettings.Get("email.from_name") != "Test" {
		t.Error("email.from_name not found")
	}
}

func TestSettingsService_GetByPrefix_Empty(t *testing.T) {
	store := newMockSettingsStore()
	store.data["server.port"] = "8080"
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()
	_ = svc.Load(ctx)

	// Prefix that doesn't match anything custom (only defaults with that prefix)
	customSettings := svc.GetByPrefix("custom.")
	if len(customSettings) != 0 {
		t.Errorf("expected 0 custom settings, got %d", len(customSettings))
	}
}

func TestSettingsService_Store(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)

	returnedStore := svc.Store()
	if returnedStore != store {
		t.Error("Store should return the underlying store")
	}
}

func TestSettingsService_ConcurrentAccess(t *testing.T) {
	store := newMockSettingsStore()
	logger := zerolog.Nop()

	svc := app.NewSettingsService(store, logger)
	ctx := context.Background()

	// Concurrent reads and writes
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func(n int) {
			key := "concurrent.key" + settingsItoa(n)
			_ = svc.Set(ctx, key, "value")
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		go func() {
			_ = svc.Get()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func settingsItoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return settingsItoa(n/10) + string(rune('0'+n%10))
}
