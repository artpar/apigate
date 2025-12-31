package adapters_test

import (
	"context"
	"strings"
	"testing"

	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/core/capability/adapters"
)

func TestMemoryCache(t *testing.T) {
	ctx := context.Background()
	cache := adapters.NewMemoryCache("test_cache")

	if cache.Name() != "test_cache" {
		t.Errorf("Name() = %v, want test_cache", cache.Name())
	}

	// Test Set and Get
	if err := cache.Set(ctx, "key1", []byte("value1"), 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("Get() = %v, want value1", string(val))
	}

	// Test Exists
	exists, err := cache.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// Test Delete
	if err := cache.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, _ = cache.Exists(ctx, "key1")
	if exists {
		t.Error("Exists() = true after delete, want false")
	}

	// Test Flush
	cache.Set(ctx, "key2", []byte("value2"), 0)
	if err := cache.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	val, _ = cache.Get(ctx, "key2")
	if val != nil {
		t.Error("Get() after Flush should return nil")
	}
}

func TestMemoryStorage(t *testing.T) {
	ctx := context.Background()
	storage := adapters.NewMemoryStorage("test_storage")

	if storage.Name() != "test_storage" {
		t.Errorf("Name() = %v, want test_storage", storage.Name())
	}

	// Test Put and Get
	if err := storage.Put(ctx, "file1.txt", []byte("content1"), "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	data, contentType, err := storage.Get(ctx, "file1.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(data) != "content1" {
		t.Errorf("Get() data = %v, want content1", string(data))
	}
	if contentType != "text/plain" {
		t.Errorf("Get() contentType = %v, want text/plain", contentType)
	}

	// Test Exists
	exists, err := storage.Exists(ctx, "file1.txt")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// Test List
	storage.Put(ctx, "dir/file2.txt", []byte("content2"), "text/plain")
	storage.Put(ctx, "dir/file3.txt", []byte("content3"), "text/plain")

	objects, err := storage.List(ctx, "dir/", 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 2 {
		t.Errorf("List() returned %d objects, want 2", len(objects))
	}

	// Test Delete
	if err := storage.Delete(ctx, "file1.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, _ = storage.Exists(ctx, "file1.txt")
	if exists {
		t.Error("Exists() = true after delete, want false")
	}

	// Test GetURL
	url, err := storage.GetURL(ctx, "dir/file2.txt", 3600)
	if err != nil {
		t.Fatalf("GetURL() error = %v", err)
	}
	if !strings.Contains(url, "memory://") {
		t.Errorf("GetURL() = %v, expected memory:// prefix", url)
	}
}

func TestMemoryQueue(t *testing.T) {
	ctx := context.Background()
	queue := adapters.NewMemoryQueue("test_queue")
	defer queue.Close()

	if queue.Name() != "test_queue" {
		t.Errorf("Name() = %v, want test_queue", queue.Name())
	}

	// Test Enqueue and Dequeue
	job := capability.Job{
		ID:      "job1",
		Type:    "test",
		Payload: map[string]any{"key": "value"},
	}

	if err := queue.Enqueue(ctx, "q1", job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Check queue length
	length, err := queue.QueueLength(ctx, "q1")
	if err != nil {
		t.Fatalf("QueueLength() error = %v", err)
	}
	if length != 1 {
		t.Errorf("QueueLength() = %d, want 1", length)
	}

	// Dequeue
	dequeued, err := queue.Dequeue(ctx, "q1", 1)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if dequeued == nil {
		t.Fatal("Dequeue() returned nil")
	}
	if dequeued.ID != "job1" {
		t.Errorf("Dequeue() job ID = %v, want job1", dequeued.ID)
	}

	// After dequeue, queue should be empty
	length, _ = queue.QueueLength(ctx, "q1")
	if length != 0 {
		t.Errorf("QueueLength() after dequeue = %d, want 0", length)
	}

	// Test Ack
	if err := queue.Ack(ctx, "q1", "job1"); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
}

func TestConsoleNotification(t *testing.T) {
	ctx := context.Background()
	notif := adapters.NewConsoleNotification("test_notif")

	if notif.Name() != "test_notif" {
		t.Errorf("Name() = %v, want test_notif", notif.Name())
	}

	// Test Send
	msg := capability.NotificationMessage{
		Title:    "Test Title",
		Message:  "Test Message",
		Severity: "info",
	}

	if err := notif.Send(ctx, msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Verify message was captured
	messages := notif.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("GetMessages() = %d messages, want 1", len(messages))
	}
	if messages[0].Title != "Test Title" {
		t.Errorf("Message title = %v, want Test Title", messages[0].Title)
	}

	// Test SendBatch
	notif.ClearMessages()
	batch := []capability.NotificationMessage{
		{Title: "Msg1", Message: "Body1", Severity: "info"},
		{Title: "Msg2", Message: "Body2", Severity: "warning"},
	}
	if err := notif.SendBatch(ctx, batch); err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}

	messages = notif.GetMessages()
	if len(messages) != 2 {
		t.Errorf("GetMessages() = %d messages, want 2", len(messages))
	}

	// Test TestConnection
	if err := notif.TestConnection(ctx); err != nil {
		t.Errorf("TestConnection() error = %v", err)
	}
}
