package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// testLogger returns a disabled logger for tests
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

// TestNewBus verifies that NewBus creates a properly initialized Bus
func TestNewBus(t *testing.T) {
	logger := testLogger()
	bus := NewBus(logger)

	if bus == nil {
		t.Fatal("NewBus returned nil")
	}

	if bus.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if len(bus.handlers) != 0 {
		t.Error("handlers map should be empty on creation")
	}
}

// TestSubscribe verifies that Subscribe correctly registers handlers
func TestSubscribe(t *testing.T) {
	bus := NewBus(testLogger())

	// Subscribe to an event
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		return nil
	})

	// Verify handler was registered
	if len(bus.handlers["test.event"]) != 1 {
		t.Errorf("expected 1 handler, got %d", len(bus.handlers["test.event"]))
	}
}

// TestSubscribeMultipleHandlers verifies multiple handlers for same event
func TestSubscribeMultipleHandlers(t *testing.T) {
	bus := NewBus(testLogger())

	callOrder := []int{}
	var mu sync.Mutex

	// Subscribe multiple handlers
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		callOrder = append(callOrder, 1)
		mu.Unlock()
		return nil
	})

	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		callOrder = append(callOrder, 2)
		mu.Unlock()
		return nil
	})

	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		callOrder = append(callOrder, 3)
		mu.Unlock()
		return nil
	})

	// Verify all handlers registered
	if len(bus.handlers["test.event"]) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(bus.handlers["test.event"]))
	}

	// Publish and verify order
	bus.Publish(context.Background(), Event{Name: "test.event"})

	if len(callOrder) != 3 {
		t.Errorf("expected 3 calls, got %d", len(callOrder))
	}

	// Verify handlers called in registration order
	for i, order := range callOrder {
		if order != i+1 {
			t.Errorf("expected call order %d at position %d, got %d", i+1, i, order)
		}
	}
}

// TestPublishExactMatch verifies exact event name matching
func TestPublishExactMatch(t *testing.T) {
	bus := NewBus(testLogger())

	var receivedEvent Event
	called := false

	bus.Subscribe("user.created", func(ctx context.Context, event Event) error {
		called = true
		receivedEvent = event
		return nil
	})

	// Publish matching event
	testEvent := Event{
		Name:   "user.created",
		Module: "users",
		Action: "create",
		Data:   map[string]any{"id": "123"},
		Meta:   map[string]any{"timestamp": time.Now()},
	}
	bus.Publish(context.Background(), testEvent)

	if !called {
		t.Error("handler was not called for exact match")
	}

	if receivedEvent.Name != testEvent.Name {
		t.Errorf("expected event name %s, got %s", testEvent.Name, receivedEvent.Name)
	}
	if receivedEvent.Module != testEvent.Module {
		t.Errorf("expected module %s, got %s", testEvent.Module, receivedEvent.Module)
	}
	if receivedEvent.Action != testEvent.Action {
		t.Errorf("expected action %s, got %s", testEvent.Action, receivedEvent.Action)
	}
}

// TestPublishNoMatch verifies handler is not called for non-matching events
func TestPublishNoMatch(t *testing.T) {
	bus := NewBus(testLogger())

	called := false
	bus.Subscribe("user.created", func(ctx context.Context, event Event) error {
		called = true
		return nil
	})

	// Publish non-matching event
	bus.Publish(context.Background(), Event{Name: "user.deleted"})

	if called {
		t.Error("handler should not be called for non-matching event")
	}
}

// TestPublishWildcardModule verifies module wildcard matching (e.g., "user.*")
func TestPublishWildcardModule(t *testing.T) {
	bus := NewBus(testLogger())

	events := []string{}
	var mu sync.Mutex

	// Subscribe to wildcard
	bus.Subscribe("user.*", func(ctx context.Context, event Event) error {
		mu.Lock()
		events = append(events, event.Name)
		mu.Unlock()
		return nil
	})

	// Publish various user events
	bus.Publish(context.Background(), Event{Name: "user.created"})
	bus.Publish(context.Background(), Event{Name: "user.updated"})
	bus.Publish(context.Background(), Event{Name: "user.deleted"})
	// This should not match
	bus.Publish(context.Background(), Event{Name: "order.created"})

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d: %v", len(events), events)
	}
}

// TestPublishGlobalWildcard verifies global wildcard matching ("*")
func TestPublishGlobalWildcard(t *testing.T) {
	bus := NewBus(testLogger())

	var count int32

	// Subscribe to all events
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Publish various events
	bus.Publish(context.Background(), Event{Name: "user.created"})
	bus.Publish(context.Background(), Event{Name: "order.created"})
	bus.Publish(context.Background(), Event{Name: "system.startup"})
	bus.Publish(context.Background(), Event{Name: "single"})

	if count != 4 {
		t.Errorf("expected 4 events, got %d", count)
	}
}

// TestPublishMultipleWildcardsMatch verifies event matches multiple patterns
func TestPublishMultipleWildcardsMatch(t *testing.T) {
	bus := NewBus(testLogger())

	var count int32

	// Subscribe to exact, module wildcard, and global wildcard
	bus.Subscribe("user.created", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	bus.Subscribe("user.*", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Publish event that matches all three
	bus.Publish(context.Background(), Event{Name: "user.created"})

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}

// TestPublishHandlerError verifies errors are logged but publishing continues
func TestPublishHandlerError(t *testing.T) {
	bus := NewBus(testLogger())

	calls := []int{}
	var mu sync.Mutex

	// First handler succeeds
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, 1)
		mu.Unlock()
		return nil
	})

	// Second handler fails
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, 2)
		mu.Unlock()
		return errors.New("handler error")
	})

	// Third handler should still be called
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		mu.Lock()
		calls = append(calls, 3)
		mu.Unlock()
		return nil
	})

	bus.Publish(context.Background(), Event{Name: "test.event"})

	// All handlers should have been called
	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
	}
}

// TestPublishAsync verifies asynchronous event publishing
func TestPublishAsync(t *testing.T) {
	bus := NewBus(testLogger())

	done := make(chan bool, 1)

	bus.Subscribe("async.test", func(ctx context.Context, event Event) error {
		done <- true
		return nil
	})

	// Publish async
	bus.PublishAsync(context.Background(), Event{Name: "async.test"})

	// Wait for handler to be called (with timeout)
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Error("handler was not called within timeout")
	}
}

// TestPublishAsyncReturnsImmediately verifies PublishAsync returns immediately
func TestPublishAsyncReturnsImmediately(t *testing.T) {
	bus := NewBus(testLogger())

	bus.Subscribe("slow.event", func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	start := time.Now()
	bus.PublishAsync(context.Background(), Event{Name: "slow.event"})
	elapsed := time.Since(start)

	// Should return much faster than the handler sleep time
	if elapsed > 50*time.Millisecond {
		t.Errorf("PublishAsync took %v, expected immediate return", elapsed)
	}
}

// TestHasSubscribersExactMatch verifies HasSubscribers for exact matches
func TestHasSubscribersExactMatch(t *testing.T) {
	bus := NewBus(testLogger())

	// No subscribers initially
	if bus.HasSubscribers("test.event") {
		t.Error("should have no subscribers initially")
	}

	// Add subscriber
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		return nil
	})

	// Now should have subscribers
	if !bus.HasSubscribers("test.event") {
		t.Error("should have subscriber after Subscribe")
	}

	// Different event should still have no subscribers
	if bus.HasSubscribers("other.event") {
		t.Error("should have no subscribers for different event")
	}
}

// TestHasSubscribersWildcardModule verifies HasSubscribers with module wildcards
func TestHasSubscribersWildcardModule(t *testing.T) {
	bus := NewBus(testLogger())

	// Subscribe to user wildcard
	bus.Subscribe("user.*", func(ctx context.Context, event Event) error {
		return nil
	})

	// Should match user events
	if !bus.HasSubscribers("user.created") {
		t.Error("should match user.* wildcard for user.created")
	}
	if !bus.HasSubscribers("user.updated") {
		t.Error("should match user.* wildcard for user.updated")
	}

	// Should not match other modules
	if bus.HasSubscribers("order.created") {
		t.Error("should not match user.* wildcard for order.created")
	}
}

// TestHasSubscribersGlobalWildcard verifies HasSubscribers with global wildcard
func TestHasSubscribersGlobalWildcard(t *testing.T) {
	bus := NewBus(testLogger())

	// Subscribe to all events
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		return nil
	})

	// Should match any event
	if !bus.HasSubscribers("user.created") {
		t.Error("should match * wildcard for user.created")
	}
	if !bus.HasSubscribers("order.deleted") {
		t.Error("should match * wildcard for order.deleted")
	}
	if !bus.HasSubscribers("anything") {
		t.Error("should match * wildcard for anything")
	}
}

// TestSplitEvent verifies the splitEvent helper function
func TestSplitEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "two parts",
			input:    "user.created",
			expected: []string{"user", "created"},
		},
		{
			name:     "three parts",
			input:    "module.action.sub",
			expected: []string{"module", "action", "sub"},
		},
		{
			name:     "single part",
			input:    "event",
			expected: []string{"event"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "trailing dot",
			input:    "user.",
			expected: []string{"user"},
		},
		{
			name:     "leading dot",
			input:    ".user",
			expected: []string{"", "user"},
		},
		{
			name:     "multiple consecutive dots",
			input:    "a..b",
			expected: []string{"a", "", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitEvent(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitEvent(%q) returned %d parts, expected %d: got %v, expected %v",
					tt.input, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("splitEvent(%q)[%d] = %q, expected %q",
						tt.input, i, part, tt.expected[i])
				}
			}
		})
	}
}

// TestEventStruct verifies Event struct fields
func TestEventStruct(t *testing.T) {
	event := Event{
		Name:   "user.created",
		Module: "users",
		Action: "create",
		Data:   map[string]any{"id": "123", "name": "test"},
		Meta:   map[string]any{"ip": "127.0.0.1"},
	}

	if event.Name != "user.created" {
		t.Errorf("expected Name 'user.created', got %q", event.Name)
	}
	if event.Module != "users" {
		t.Errorf("expected Module 'users', got %q", event.Module)
	}
	if event.Action != "create" {
		t.Errorf("expected Action 'create', got %q", event.Action)
	}
	if event.Data["id"] != "123" {
		t.Errorf("expected Data['id'] '123', got %v", event.Data["id"])
	}
	if event.Meta["ip"] != "127.0.0.1" {
		t.Errorf("expected Meta['ip'] '127.0.0.1', got %v", event.Meta["ip"])
	}
}

// TestConcurrentSubscribeAndPublish verifies thread safety
func TestConcurrentSubscribeAndPublish(t *testing.T) {
	bus := NewBus(testLogger())
	var wg sync.WaitGroup
	var count int64

	// Start multiple goroutines subscribing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bus.Subscribe("concurrent.test", func(ctx context.Context, event Event) error {
				atomic.AddInt64(&count, 1)
				return nil
			})
		}(i)
	}

	// Wait for subscriptions
	wg.Wait()

	// Publish concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), Event{Name: "concurrent.test"})
		}()
	}

	wg.Wait()

	// Each of 10 publishes should trigger 10 handlers = 100 calls
	expected := int64(100)
	if count != expected {
		t.Errorf("expected %d handler calls, got %d", expected, count)
	}
}

// TestPublishWithContext verifies context is passed to handlers
func TestPublishWithContext(t *testing.T) {
	bus := NewBus(testLogger())

	type contextKey string
	key := contextKey("test-key")
	expectedValue := "test-value"

	var receivedValue any

	bus.Subscribe("context.test", func(ctx context.Context, event Event) error {
		receivedValue = ctx.Value(key)
		return nil
	})

	ctx := context.WithValue(context.Background(), key, expectedValue)
	bus.Publish(ctx, Event{Name: "context.test"})

	if receivedValue != expectedValue {
		t.Errorf("expected context value %q, got %v", expectedValue, receivedValue)
	}
}

// TestPublishEmptyEventName verifies behavior with empty event name
func TestPublishEmptyEventName(t *testing.T) {
	bus := NewBus(testLogger())

	exactCalled := false
	wildcardCalled := false

	bus.Subscribe("", func(ctx context.Context, event Event) error {
		exactCalled = true
		return nil
	})

	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		wildcardCalled = true
		return nil
	})

	bus.Publish(context.Background(), Event{Name: ""})

	if !exactCalled {
		t.Error("exact match handler for empty string should be called")
	}
	if !wildcardCalled {
		t.Error("global wildcard should match empty event name")
	}
}

// TestHasSubscribersEmptyEvent verifies HasSubscribers with empty event name
func TestHasSubscribersEmptyEvent(t *testing.T) {
	bus := NewBus(testLogger())

	// No subscribers
	if bus.HasSubscribers("") {
		t.Error("should have no subscribers for empty event")
	}

	// Add global wildcard
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		return nil
	})

	// Global wildcard should match empty event
	if !bus.HasSubscribers("") {
		t.Error("global wildcard should match empty event")
	}
}

// TestPublishSinglePartEvent verifies events without dots
func TestPublishSinglePartEvent(t *testing.T) {
	bus := NewBus(testLogger())

	exactCalled := false
	wildcardCalled := false

	bus.Subscribe("startup", func(ctx context.Context, event Event) error {
		exactCalled = true
		return nil
	})

	bus.Subscribe("startup.*", func(ctx context.Context, event Event) error {
		wildcardCalled = true
		return nil
	})

	bus.Publish(context.Background(), Event{Name: "startup"})

	if !exactCalled {
		t.Error("exact match handler should be called")
	}
	// Note: The current implementation does match "startup.*" for "startup" event
	// because splitEvent("startup") returns ["startup"] with len >= 1,
	// so the wildcard "startup.*" is checked and matched.
	if !wildcardCalled {
		t.Error("module wildcard should match single-part event (current behavior)")
	}
}

// TestPublishNilData verifies publishing events with nil Data and Meta
func TestPublishNilData(t *testing.T) {
	bus := NewBus(testLogger())

	called := false
	bus.Subscribe("nil.test", func(ctx context.Context, event Event) error {
		called = true
		if event.Data != nil {
			t.Error("Data should be nil")
		}
		if event.Meta != nil {
			t.Error("Meta should be nil")
		}
		return nil
	})

	bus.Publish(context.Background(), Event{Name: "nil.test"})

	if !called {
		t.Error("handler should be called")
	}
}

// TestMultipleDifferentEvents verifies handling of different event types
func TestMultipleDifferentEvents(t *testing.T) {
	bus := NewBus(testLogger())

	counts := make(map[string]int)
	var mu sync.Mutex

	bus.Subscribe("user.created", func(ctx context.Context, event Event) error {
		mu.Lock()
		counts["user.created"]++
		mu.Unlock()
		return nil
	})

	bus.Subscribe("user.updated", func(ctx context.Context, event Event) error {
		mu.Lock()
		counts["user.updated"]++
		mu.Unlock()
		return nil
	})

	bus.Subscribe("order.created", func(ctx context.Context, event Event) error {
		mu.Lock()
		counts["order.created"]++
		mu.Unlock()
		return nil
	})

	// Publish different events
	bus.Publish(context.Background(), Event{Name: "user.created"})
	bus.Publish(context.Background(), Event{Name: "user.created"})
	bus.Publish(context.Background(), Event{Name: "user.updated"})
	bus.Publish(context.Background(), Event{Name: "order.created"})
	bus.Publish(context.Background(), Event{Name: "order.created"})
	bus.Publish(context.Background(), Event{Name: "order.created"})

	if counts["user.created"] != 2 {
		t.Errorf("expected 2 user.created events, got %d", counts["user.created"])
	}
	if counts["user.updated"] != 1 {
		t.Errorf("expected 1 user.updated event, got %d", counts["user.updated"])
	}
	if counts["order.created"] != 3 {
		t.Errorf("expected 3 order.created events, got %d", counts["order.created"])
	}
}

// TestHandlerReceivesCorrectData verifies event data is passed correctly
func TestHandlerReceivesCorrectData(t *testing.T) {
	bus := NewBus(testLogger())

	var receivedEvent Event

	bus.Subscribe("data.test", func(ctx context.Context, event Event) error {
		receivedEvent = event
		return nil
	})

	originalEvent := Event{
		Name:   "data.test",
		Module: "test-module",
		Action: "test-action",
		Data: map[string]any{
			"string":  "value",
			"number":  42,
			"boolean": true,
			"nested":  map[string]any{"key": "nested-value"},
		},
		Meta: map[string]any{
			"requestId": "req-123",
		},
	}

	bus.Publish(context.Background(), originalEvent)

	// Verify all fields match
	if receivedEvent.Name != originalEvent.Name {
		t.Errorf("Name mismatch: got %q, want %q", receivedEvent.Name, originalEvent.Name)
	}
	if receivedEvent.Module != originalEvent.Module {
		t.Errorf("Module mismatch: got %q, want %q", receivedEvent.Module, originalEvent.Module)
	}
	if receivedEvent.Action != originalEvent.Action {
		t.Errorf("Action mismatch: got %q, want %q", receivedEvent.Action, originalEvent.Action)
	}
	if receivedEvent.Data["string"] != "value" {
		t.Errorf("Data['string'] mismatch: got %v", receivedEvent.Data["string"])
	}
	if receivedEvent.Data["number"] != 42 {
		t.Errorf("Data['number'] mismatch: got %v", receivedEvent.Data["number"])
	}
	if receivedEvent.Meta["requestId"] != "req-123" {
		t.Errorf("Meta['requestId'] mismatch: got %v", receivedEvent.Meta["requestId"])
	}
}

// TestWildcardDoesNotMatchDifferentModule verifies wildcard scoping
func TestWildcardDoesNotMatchDifferentModule(t *testing.T) {
	bus := NewBus(testLogger())

	called := false
	bus.Subscribe("user.*", func(ctx context.Context, event Event) error {
		called = true
		return nil
	})

	// Publish order event - should not match user.*
	bus.Publish(context.Background(), Event{Name: "order.created"})

	if called {
		t.Error("user.* should not match order.created")
	}
}

// TestDeepEventName verifies events with multiple dot separators
func TestDeepEventName(t *testing.T) {
	bus := NewBus(testLogger())

	exactCalled := false
	wildcardCalled := false
	globalCalled := false

	bus.Subscribe("module.action.sub.deep", func(ctx context.Context, event Event) error {
		exactCalled = true
		return nil
	})

	bus.Subscribe("module.*", func(ctx context.Context, event Event) error {
		wildcardCalled = true
		return nil
	})

	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		globalCalled = true
		return nil
	})

	bus.Publish(context.Background(), Event{Name: "module.action.sub.deep"})

	if !exactCalled {
		t.Error("exact match should be called")
	}
	if !wildcardCalled {
		t.Error("module.* should match module.action.sub.deep")
	}
	if !globalCalled {
		t.Error("* should match any event")
	}
}

// BenchmarkPublish benchmarks the Publish function
func BenchmarkPublish(b *testing.B) {
	bus := NewBus(testLogger())

	// Add some handlers
	for i := 0; i < 5; i++ {
		bus.Subscribe("bench.event", func(ctx context.Context, event Event) error {
			return nil
		})
	}

	event := Event{Name: "bench.event", Module: "bench", Action: "test"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(ctx, event)
	}
}

// BenchmarkPublishWithWildcards benchmarks Publish with wildcard matching
func BenchmarkPublishWithWildcards(b *testing.B) {
	bus := NewBus(testLogger())

	// Add exact match, wildcard, and global handlers
	bus.Subscribe("bench.event", func(ctx context.Context, event Event) error {
		return nil
	})
	bus.Subscribe("bench.*", func(ctx context.Context, event Event) error {
		return nil
	})
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		return nil
	})

	event := Event{Name: "bench.event", Module: "bench", Action: "test"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(ctx, event)
	}
}

// BenchmarkHasSubscribers benchmarks the HasSubscribers function
func BenchmarkHasSubscribers(b *testing.B) {
	bus := NewBus(testLogger())

	bus.Subscribe("bench.event", func(ctx context.Context, event Event) error {
		return nil
	})
	bus.Subscribe("bench.*", func(ctx context.Context, event Event) error {
		return nil
	})
	bus.Subscribe("*", func(ctx context.Context, event Event) error {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.HasSubscribers("bench.event")
	}
}
