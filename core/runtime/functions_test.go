package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestNewFunctionRegistry(t *testing.T) {
	registry := NewFunctionRegistry()

	if registry == nil {
		t.Fatal("NewFunctionRegistry returned nil")
	}

	if registry.funcs == nil {
		t.Error("FunctionRegistry.funcs map should be initialized")
	}

	if len(registry.funcs) != 0 {
		t.Errorf("FunctionRegistry.funcs should be empty, got %d", len(registry.funcs))
	}
}

func TestFunctionRegistry_Register(t *testing.T) {
	registry := NewFunctionRegistry()

	handlerCalled := false
	handler := func(ctx context.Context, event HookEvent) error {
		handlerCalled = true
		return nil
	}

	registry.Register("test_func", handler)

	if !registry.Has("test_func") {
		t.Error("Function should be registered")
	}

	// Verify the handler is the one we registered by calling it
	err := registry.Call(context.Background(), "test_func", HookEvent{})
	if err != nil {
		t.Errorf("Call returned unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestFunctionRegistry_Register_Overwrite(t *testing.T) {
	registry := NewFunctionRegistry()

	firstCalled := false
	secondCalled := false

	registry.Register("test_func", func(ctx context.Context, event HookEvent) error {
		firstCalled = true
		return nil
	})

	registry.Register("test_func", func(ctx context.Context, event HookEvent) error {
		secondCalled = true
		return nil
	})

	_ = registry.Call(context.Background(), "test_func", HookEvent{})

	if firstCalled {
		t.Error("First handler should not be called after overwrite")
	}
	if !secondCalled {
		t.Error("Second handler should be called")
	}
}

func TestFunctionRegistry_Call(t *testing.T) {
	t.Run("successful call", func(t *testing.T) {
		registry := NewFunctionRegistry()

		var receivedEvent HookEvent
		registry.Register("test_func", func(ctx context.Context, event HookEvent) error {
			receivedEvent = event
			return nil
		})

		testEvent := HookEvent{
			Module: "test_module",
			Action: "create",
			Phase:  "before",
			Data:   map[string]any{"key": "value"},
			Meta:   map[string]any{"meta_key": "meta_value"},
		}

		err := registry.Call(context.Background(), "test_func", testEvent)
		if err != nil {
			t.Errorf("Call returned unexpected error: %v", err)
		}

		if receivedEvent.Module != "test_module" {
			t.Errorf("Event.Module = %q, want %q", receivedEvent.Module, "test_module")
		}
		if receivedEvent.Action != "create" {
			t.Errorf("Event.Action = %q, want %q", receivedEvent.Action, "create")
		}
		if receivedEvent.Phase != "before" {
			t.Errorf("Event.Phase = %q, want %q", receivedEvent.Phase, "before")
		}
		if receivedEvent.Data["key"] != "value" {
			t.Errorf("Event.Data[key] = %v, want %q", receivedEvent.Data["key"], "value")
		}
	})

	t.Run("function not found", func(t *testing.T) {
		registry := NewFunctionRegistry()

		err := registry.Call(context.Background(), "nonexistent", HookEvent{})
		if err == nil {
			t.Error("Call should return error for unregistered function")
		}
		if err.Error() != `function "nonexistent" not registered` {
			t.Errorf("Error = %q, want %q", err.Error(), `function "nonexistent" not registered`)
		}
	})

	t.Run("function returns error", func(t *testing.T) {
		registry := NewFunctionRegistry()

		expectedErr := errors.New("handler error")
		registry.Register("error_func", func(ctx context.Context, event HookEvent) error {
			return expectedErr
		})

		err := registry.Call(context.Background(), "error_func", HookEvent{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("Call should return handler error, got: %v", err)
		}
	})

	t.Run("context passed to handler", func(t *testing.T) {
		registry := NewFunctionRegistry()

		type contextKey string
		key := contextKey("test_key")
		ctx := context.WithValue(context.Background(), key, "test_value")

		var receivedCtx context.Context
		registry.Register("ctx_func", func(ctx context.Context, event HookEvent) error {
			receivedCtx = ctx
			return nil
		})

		_ = registry.Call(ctx, "ctx_func", HookEvent{})

		if receivedCtx.Value(key) != "test_value" {
			t.Error("Context should be passed to handler")
		}
	})
}

func TestFunctionRegistry_Has(t *testing.T) {
	registry := NewFunctionRegistry()

	if registry.Has("test_func") {
		t.Error("Has should return false for unregistered function")
	}

	registry.Register("test_func", func(ctx context.Context, event HookEvent) error {
		return nil
	})

	if !registry.Has("test_func") {
		t.Error("Has should return true for registered function")
	}

	if registry.Has("other_func") {
		t.Error("Has should return false for other unregistered function")
	}
}

func TestFunctionRegistry_List(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry := NewFunctionRegistry()

		list := registry.List()
		if len(list) != 0 {
			t.Errorf("List should be empty, got %d items", len(list))
		}
	})

	t.Run("registry with functions", func(t *testing.T) {
		registry := NewFunctionRegistry()
		handler := func(ctx context.Context, event HookEvent) error { return nil }

		registry.Register("func1", handler)
		registry.Register("func2", handler)
		registry.Register("func3", handler)

		list := registry.List()
		if len(list) != 3 {
			t.Errorf("List should have 3 items, got %d", len(list))
		}

		// Check that all functions are present (order may vary due to map iteration)
		found := make(map[string]bool)
		for _, name := range list {
			found[name] = true
		}

		for _, expected := range []string{"func1", "func2", "func3"} {
			if !found[expected] {
				t.Errorf("List should contain %q", expected)
			}
		}
	})
}

func TestFunctionRegistry_Concurrency(t *testing.T) {
	registry := NewFunctionRegistry()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4 types of operations

	// Concurrent Register operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				registry.Register("func_"+string(rune(id))+"_"+string(rune(j)), func(ctx context.Context, event HookEvent) error {
					return nil
				})
			}
		}(i)
	}

	// Concurrent Has operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				registry.Has("some_func")
			}
		}()
	}

	// Concurrent List operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = registry.List()
			}
		}()
	}

	// Concurrent Call operations (on pre-registered function)
	registry.Register("concurrent_test", func(ctx context.Context, event HookEvent) error {
		return nil
	})
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = registry.Call(context.Background(), "concurrent_test", HookEvent{})
			}
		}()
	}

	wg.Wait()
	// If we get here without race conditions, the test passes
}

func TestHookEvent(t *testing.T) {
	event := HookEvent{
		Module: "user",
		Action: "create",
		Phase:  "after",
		Data: map[string]any{
			"name":  "John",
			"email": "john@example.com",
		},
		Meta: map[string]any{
			"request_id": "abc123",
		},
	}

	if event.Module != "user" {
		t.Errorf("HookEvent.Module = %q, want %q", event.Module, "user")
	}
	if event.Action != "create" {
		t.Errorf("HookEvent.Action = %q, want %q", event.Action, "create")
	}
	if event.Phase != "after" {
		t.Errorf("HookEvent.Phase = %q, want %q", event.Phase, "after")
	}
	if event.Data["name"] != "John" {
		t.Errorf("HookEvent.Data[name] = %v, want %q", event.Data["name"], "John")
	}
	if event.Meta["request_id"] != "abc123" {
		t.Errorf("HookEvent.Meta[request_id] = %v, want %q", event.Meta["request_id"], "abc123")
	}
}

func TestFunctionRegistry_EmptyFunctionName(t *testing.T) {
	registry := NewFunctionRegistry()

	registry.Register("", func(ctx context.Context, event HookEvent) error {
		return nil
	})

	// Should work with empty string as function name (edge case)
	if !registry.Has("") {
		t.Error("Has should return true for empty string function name")
	}

	err := registry.Call(context.Background(), "", HookEvent{})
	if err != nil {
		t.Errorf("Call should succeed for empty string function name, got error: %v", err)
	}
}

func TestFunctionRegistry_NilHandler(t *testing.T) {
	registry := NewFunctionRegistry()

	// Registering nil handler - this is allowed but will panic when called
	registry.Register("nil_func", nil)

	if !registry.Has("nil_func") {
		t.Error("Has should return true for nil handler")
	}

	// Calling nil handler should panic, we test this as expected behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("Call with nil handler should panic")
		}
	}()

	_ = registry.Call(context.Background(), "nil_func", HookEvent{})
}
