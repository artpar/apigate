package bootstrap

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/runtime"
	"github.com/rs/zerolog"
)

// mockRouterReloader implements RouterReloader for testing.
type mockRouterReloader struct {
	reloadCalled bool
	reloadErr    error
}

func (m *mockRouterReloader) Reload(ctx context.Context) error {
	m.reloadCalled = true
	return m.reloadErr
}

// mockPlanReloader implements PlanReloader for testing.
type mockPlanReloader struct {
	reloadCalled bool
	reloadErr    error
}

func (m *mockPlanReloader) ReloadPlans(ctx context.Context) error {
	m.reloadCalled = true
	return m.reloadErr
}

func TestSetRouterReloader(t *testing.T) {
	// Clear any existing reloader
	routerReloader = nil

	mock := &mockRouterReloader{}
	SetRouterReloader(mock)

	if routerReloader != mock {
		t.Error("SetRouterReloader should set the global router reloader")
	}

	// Cleanup
	routerReloader = nil
}

func TestSetPlanReloader(t *testing.T) {
	// Clear any existing reloader
	planReloader = nil

	mock := &mockPlanReloader{}
	SetPlanReloader(mock)

	if planReloader != mock {
		t.Error("SetPlanReloader should set the global plan reloader")
	}

	// Cleanup
	planReloader = nil
}

func TestTriggerPlanReload_NilReloader(t *testing.T) {
	// Clear any existing reloader
	planReloader = nil

	err := TriggerPlanReload(context.Background())
	if err != nil {
		t.Errorf("TriggerPlanReload with nil reloader should return nil, got: %v", err)
	}
}

func TestTriggerPlanReload_WithReloader(t *testing.T) {
	mock := &mockPlanReloader{}
	planReloader = mock

	err := TriggerPlanReload(context.Background())
	if err != nil {
		t.Errorf("TriggerPlanReload should return nil, got: %v", err)
	}

	if !mock.reloadCalled {
		t.Error("TriggerPlanReload should call ReloadPlans on the reloader")
	}

	// Cleanup
	planReloader = nil
}

func TestRegisterHooks(t *testing.T) {
	// Create a mock storage for the runtime
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	// Register hooks
	RegisterHooks(rt, logger)

	// Verify functions are registered
	functions := rt.Functions()

	// Check that built-in functions are registered
	if !functions.Has("reload_router") {
		t.Error("reload_router function should be registered")
	}
	if !functions.Has("send_verification_email") {
		t.Error("send_verification_email function should be registered")
	}
	if !functions.Has("clear_other_defaults") {
		t.Error("clear_other_defaults function should be registered")
	}
	if !functions.Has("sync_to_stripe") {
		t.Error("sync_to_stripe function should be registered")
	}
	if !functions.Has("reload_plans") {
		t.Error("reload_plans function should be registered")
	}
}

func TestReloadRouterFunction(t *testing.T) {
	// Create a mock storage for the runtime
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	// Set up mock router reloader
	mock := &mockRouterReloader{}
	routerReloader = mock

	// Register hooks
	RegisterHooks(rt, logger)

	// Call the reload_router function
	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "route",
		Action: "create",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "reload_router", event)
	if err != nil {
		t.Errorf("reload_router function should not error: %v", err)
	}

	if !mock.reloadCalled {
		t.Error("reload_router should call Reload on the router reloader")
	}

	// Cleanup
	routerReloader = nil
}

func TestReloadRouterFunction_NilReloader(t *testing.T) {
	// Clear any existing reloader
	routerReloader = nil

	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "route",
		Action: "create",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "reload_router", event)
	if err != nil {
		t.Errorf("reload_router with nil reloader should not error: %v", err)
	}
}

func TestReloadPlansFunction(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	// Set up mock plan reloader
	mock := &mockPlanReloader{}
	planReloader = mock

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "plan",
		Action: "create",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "reload_plans", event)
	if err != nil {
		t.Errorf("reload_plans function should not error: %v", err)
	}

	if !mock.reloadCalled {
		t.Error("reload_plans should call ReloadPlans on the plan reloader")
	}

	// Cleanup
	planReloader = nil
}

func TestReloadPlansFunction_NilReloader(t *testing.T) {
	planReloader = nil

	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "plan",
		Action: "create",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "reload_plans", event)
	if err != nil {
		t.Errorf("reload_plans with nil reloader should not error: %v", err)
	}
}

func TestSendVerificationEmailFunction(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "user",
		Action: "create",
		Phase:  "after",
		Data: map[string]any{
			"email": "test@example.com",
		},
		Meta: map[string]any{},
	}

	err := rt.Functions().Call(ctx, "send_verification_email", event)
	if err != nil {
		t.Errorf("send_verification_email should not error: %v", err)
	}
}

func TestClearOtherDefaultsFunction(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "plan",
		Action: "set_default",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "clear_other_defaults", event)
	if err != nil {
		t.Errorf("clear_other_defaults should not error: %v", err)
	}
}

func TestSyncToStripeFunction(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "plan",
		Action: "update",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "sync_to_stripe", event)
	if err != nil {
		t.Errorf("sync_to_stripe should not error: %v", err)
	}
}

func TestAPIKeyBeforeCreate(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	RegisterHooks(rt, logger)

	// Test by executing the hook through a simulated create event
	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "api_key",
		Action: "create",
		Phase:  "before",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	// The hook dispatcher will call our registered handler
	// We need to verify that hash, prefix, and raw_key are set
	handler := apiKeyBeforeCreate(logger)
	err := handler(ctx, event)
	if err != nil {
		t.Errorf("apiKeyBeforeCreate should not error: %v", err)
	}

	// Verify hash is set
	if _, ok := event.Data["hash"]; !ok {
		t.Error("apiKeyBeforeCreate should set hash in data")
	}

	// Verify prefix is set
	if prefix, ok := event.Data["prefix"].(string); !ok || prefix == "" {
		t.Error("apiKeyBeforeCreate should set prefix in data")
	}

	// Verify raw_key is set in meta
	if rawKey, ok := event.Meta["raw_key"].(string); !ok || rawKey == "" {
		t.Error("apiKeyBeforeCreate should set raw_key in meta")
	}
}

func TestUserBeforeSetPassword(t *testing.T) {
	logger := zerolog.Nop()

	handler := userBeforeSetPassword(logger)

	t.Run("with password", func(t *testing.T) {
		event := runtime.HookEvent{
			Module: "user",
			Action: "set_password",
			Phase:  "before",
			Data: map[string]any{
				"password": "mysecretpassword",
			},
			Meta: map[string]any{},
		}

		err := handler(context.Background(), event)
		if err != nil {
			t.Errorf("userBeforeSetPassword should not error: %v", err)
		}

		// Verify password is removed
		if _, ok := event.Data["password"]; ok {
			t.Error("userBeforeSetPassword should remove password from data")
		}

		// Verify password_hash is set
		if hash, ok := event.Data["password_hash"].(string); !ok || hash == "" {
			t.Error("userBeforeSetPassword should set password_hash in data")
		}
	})

	t.Run("without password", func(t *testing.T) {
		event := runtime.HookEvent{
			Module: "user",
			Action: "set_password",
			Phase:  "before",
			Data:   map[string]any{},
			Meta:   map[string]any{},
		}

		err := handler(context.Background(), event)
		if err != nil {
			t.Errorf("userBeforeSetPassword with no password should not error: %v", err)
		}

		// Verify password_hash is not set
		if _, ok := event.Data["password_hash"]; ok {
			t.Error("userBeforeSetPassword without password should not set password_hash")
		}
	})

	t.Run("with empty password", func(t *testing.T) {
		event := runtime.HookEvent{
			Module: "user",
			Action: "set_password",
			Phase:  "before",
			Data: map[string]any{
				"password": "",
			},
			Meta: map[string]any{},
		}

		err := handler(context.Background(), event)
		if err != nil {
			t.Errorf("userBeforeSetPassword with empty password should not error: %v", err)
		}

		// Verify password_hash is not set
		if _, ok := event.Data["password_hash"]; ok {
			t.Error("userBeforeSetPassword with empty password should not set password_hash")
		}
	})
}
