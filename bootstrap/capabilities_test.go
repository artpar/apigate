package bootstrap

import (
	"context"
	"testing"

	"github.com/artpar/apigate/domain/settings"
	"github.com/rs/zerolog"
)

func TestNewCapabilityContainer(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	// Verify capabilities are registered
	caps := container.ListCapabilities()
	if len(caps) == 0 {
		t.Error("expected at least one capability registered")
	}

	// Test cache provider is available
	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache.Name() != "default" {
		t.Errorf("Cache().Name() = %v, want default", cache.Name())
	}

	// Test storage provider is available
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}
	if storage.Name() != "default" {
		t.Errorf("Storage().Name() = %v, want default", storage.Name())
	}

	// Test queue provider is available
	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if queue.Name() != "default" {
		t.Errorf("Queue().Name() = %v, want default", queue.Name())
	}

	// Test notification provider is available
	notification, err := container.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}
	if notification.Name() != "default" {
		t.Errorf("Notification().Name() = %v, want default", notification.Name())
	}
}

func TestCapabilityContainer_CacheOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}

	// Test Set and Get
	key := "test_key"
	value := []byte("test_value")

	if err := cache.Set(ctx, key, value, 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", string(got), string(value))
	}
}

func TestCapabilityContainer_StorageOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	storage, err := container.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}

	// Test Put and Get
	key := "test/file.txt"
	data := []byte("file content")
	contentType := "text/plain"

	if err := storage.Put(ctx, key, data, contentType); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	gotData, gotContentType, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(gotData) != string(data) {
		t.Errorf("Get() data = %v, want %v", string(gotData), string(data))
	}
	if gotContentType != contentType {
		t.Errorf("Get() contentType = %v, want %v", gotContentType, contentType)
	}
}

func TestCapabilityContainer_QueueOperations(t *testing.T) {
	logger := zerolog.Nop()
	s := settings.Settings{}

	container, err := NewCapabilityContainer(CapabilityConfig{
		Settings: s,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewCapabilityContainer() error = %v", err)
	}
	defer container.Close()

	ctx := context.Background()
	queue, err := container.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	// Verify queue operations work
	length, err := queue.QueueLength(ctx, "test_queue")
	if err != nil {
		t.Fatalf("QueueLength() error = %v", err)
	}
	if length != 0 {
		t.Errorf("QueueLength() = %d, want 0", length)
	}
}
