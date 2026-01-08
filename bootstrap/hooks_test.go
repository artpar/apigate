package bootstrap

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/events"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/domain/webhook"
	"github.com/artpar/apigate/ports"
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

func TestNoopEmailSender(t *testing.T) {
	sender := &noopEmailSender{}
	ctx := context.Background()

	t.Run("Send", func(t *testing.T) {
		msg := ports.EmailMessage{
			To:       "test@example.com",
			Subject:  "Test",
			HTMLBody: "<p>Test body</p>",
			TextBody: "Test body",
		}
		err := sender.Send(ctx, msg)
		if err != nil {
			t.Errorf("Send should return nil: %v", err)
		}
	})

	t.Run("SendVerification", func(t *testing.T) {
		err := sender.SendVerification(ctx, "test@example.com", "Test", "token123")
		if err != nil {
			t.Errorf("SendVerification should return nil: %v", err)
		}
	})

	t.Run("SendPasswordReset", func(t *testing.T) {
		err := sender.SendPasswordReset(ctx, "test@example.com", "Test", "token123")
		if err != nil {
			t.Errorf("SendPasswordReset should return nil: %v", err)
		}
	})

	t.Run("SendWelcome", func(t *testing.T) {
		err := sender.SendWelcome(ctx, "test@example.com", "Test")
		if err != nil {
			t.Errorf("SendWelcome should return nil: %v", err)
		}
	})
}

func TestSubscribeWebhooksToEvents_NilModuleRuntime(t *testing.T) {
	logger := zerolog.Nop()
	app := &App{
		ModuleRuntime: nil,
		Logger:        logger,
	}

	// Should not panic with nil ModuleRuntime
	app.subscribeWebhooksToEvents()
}

func TestSubscribeWebhooksToEvents_NilRuntime(t *testing.T) {
	logger := zerolog.Nop()
	app := &App{
		ModuleRuntime: &ModuleRuntime{
			Runtime: nil,
			Logger:  logger,
		},
		Logger: logger,
	}

	// Should not panic with nil Runtime inside ModuleRuntime
	app.subscribeWebhooksToEvents()
}

func TestMapEventToWebhookType(t *testing.T) {
	tests := []struct {
		name     string
		event    events.Event
		expected string
	}{
		{"api_key.created", events.Event{Name: "api_key.created"}, string(webhook.EventKeyCreated)},
		{"key.created", events.Event{Name: "key.created"}, string(webhook.EventKeyCreated)},
		{"api_key.revoked", events.Event{Name: "api_key.revoked"}, string(webhook.EventKeyRevoked)},
		{"key.revoked", events.Event{Name: "key.revoked"}, string(webhook.EventKeyRevoked)},
		{"user.plan_changed", events.Event{Name: "user.plan_changed"}, string(webhook.EventPlanChanged)},
		{"plan.changed", events.Event{Name: "plan.changed"}, string(webhook.EventPlanChanged)},
		{"subscription.created", events.Event{Name: "subscription.created"}, string(webhook.EventSubscriptionStart)},
		{"subscription.started", events.Event{Name: "subscription.started"}, string(webhook.EventSubscriptionStart)},
		{"subscription.cancelled", events.Event{Name: "subscription.cancelled"}, string(webhook.EventSubscriptionEnd)},
		{"subscription.ended", events.Event{Name: "subscription.ended"}, string(webhook.EventSubscriptionEnd)},
		{"subscription.renewed", events.Event{Name: "subscription.renewed"}, string(webhook.EventSubscriptionRenew)},
		{"payment.succeeded", events.Event{Name: "payment.succeeded"}, string(webhook.EventPaymentSuccess)},
		{"payment.success", events.Event{Name: "payment.success"}, string(webhook.EventPaymentSuccess)},
		{"payment.failed", events.Event{Name: "payment.failed"}, string(webhook.EventPaymentFailed)},
		{"invoice.created", events.Event{Name: "invoice.created"}, string(webhook.EventInvoiceCreated)},
		{"usage.threshold", events.Event{Name: "usage.threshold"}, string(webhook.EventUsageThreshold)},
		{"usage.limit", events.Event{Name: "usage.limit"}, string(webhook.EventUsageLimit)},
		{"unknown.event", events.Event{Name: "unknown.event"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapEventToWebhookType(tt.event)
			if result != tt.expected {
				t.Errorf("mapEventToWebhookType(%q) = %q, want %q", tt.event.Name, result, tt.expected)
			}
		})
	}
}

// mockTestEmailSender implements ports.EmailSender for testing with tracking.
type mockTestEmailSender struct {
	sendVerificationCalled bool
	sendPasswordResetCalled bool
	sendWelcomeCalled       bool
	sendCalled              bool
	lastEmail              string
	lastToken              string
}

func (m *mockTestEmailSender) Send(ctx context.Context, msg ports.EmailMessage) error {
	m.sendCalled = true
	return nil
}

func (m *mockTestEmailSender) SendVerification(ctx context.Context, to, name, token string) error {
	m.sendVerificationCalled = true
	m.lastEmail = to
	m.lastToken = token
	return nil
}

func (m *mockTestEmailSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	m.sendPasswordResetCalled = true
	m.lastEmail = to
	m.lastToken = token
	return nil
}

func (m *mockTestEmailSender) SendWelcome(ctx context.Context, to, name string) error {
	m.sendWelcomeCalled = true
	m.lastEmail = to
	return nil
}

// mockTestPlanStore implements ports.PlanStore for testing.
type mockTestPlanStore struct {
	clearOtherDefaultsCalled bool
	lastExceptID            string
}

func (m *mockTestPlanStore) List(ctx context.Context) ([]ports.Plan, error) {
	return nil, nil
}

func (m *mockTestPlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	return ports.Plan{}, nil
}

func (m *mockTestPlanStore) Create(ctx context.Context, p ports.Plan) error {
	return nil
}

func (m *mockTestPlanStore) Update(ctx context.Context, p ports.Plan) error {
	return nil
}

func (m *mockTestPlanStore) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockTestPlanStore) ClearOtherDefaults(ctx context.Context, exceptID string) error {
	m.clearOtherDefaultsCalled = true
	m.lastExceptID = exceptID
	return nil
}

func TestSendVerificationEmailFunction_WithEmailSender(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	// Set up mock email sender
	mockEmail := &mockTestEmailSender{}
	oldEmailSender := emailSender
	emailSender = mockEmail
	defer func() { emailSender = oldEmailSender }()

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "user",
		Action: "create",
		Phase:  "after",
		Data: map[string]any{
			"email": "test@example.com",
			"name":  "Test User",
		},
		Meta: map[string]any{
			"verification_token": "token123",
		},
	}

	err := rt.Functions().Call(ctx, "send_verification_email", event)
	if err != nil {
		t.Errorf("send_verification_email should not error: %v", err)
	}

	if !mockEmail.sendVerificationCalled {
		t.Error("send_verification_email should call SendVerification")
	}

	if mockEmail.lastEmail != "test@example.com" {
		t.Errorf("SendVerification called with email %q, want %q", mockEmail.lastEmail, "test@example.com")
	}

	if mockEmail.lastToken != "token123" {
		t.Errorf("SendVerification called with token %q, want %q", mockEmail.lastToken, "token123")
	}
}

func TestSendVerificationEmailFunction_NoEmail(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	mockEmail := &mockTestEmailSender{}
	oldEmailSender := emailSender
	emailSender = mockEmail
	defer func() { emailSender = oldEmailSender }()

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "user",
		Action: "create",
		Phase:  "after",
		Data:   map[string]any{},
		Meta:   map[string]any{},
	}

	err := rt.Functions().Call(ctx, "send_verification_email", event)
	if err != nil {
		t.Errorf("send_verification_email should not error: %v", err)
	}

	if mockEmail.sendVerificationCalled {
		t.Error("send_verification_email should not call SendVerification when no email")
	}
}

func TestClearOtherDefaultsFunction_WithPlanStore(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	// Set up mock plan store
	mockStore := &mockTestPlanStore{}
	oldPlanStore := planStore
	planStore = mockStore
	defer func() { planStore = oldPlanStore }()

	RegisterHooks(rt, logger)

	ctx := context.Background()
	event := runtime.HookEvent{
		Module: "plan",
		Action: "set_default",
		Phase:  "after",
		Data: map[string]any{
			"id": "plan123",
		},
		Meta: map[string]any{},
	}

	err := rt.Functions().Call(ctx, "clear_other_defaults", event)
	if err != nil {
		t.Errorf("clear_other_defaults should not error: %v", err)
	}

	if !mockStore.clearOtherDefaultsCalled {
		t.Error("clear_other_defaults should call ClearOtherDefaults")
	}

	if mockStore.lastExceptID != "plan123" {
		t.Errorf("ClearOtherDefaults called with ID %q, want %q", mockStore.lastExceptID, "plan123")
	}
}

func TestClearOtherDefaultsFunction_NoPlanID(t *testing.T) {
	storage := &mockRuntimeStorage{}
	logger := zerolog.Nop()

	rt := runtime.New(storage, runtime.Config{
		Logger: logger,
	})

	mockStore := &mockTestPlanStore{}
	oldPlanStore := planStore
	planStore = mockStore
	defer func() { planStore = oldPlanStore }()

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

	if mockStore.clearOtherDefaultsCalled {
		t.Error("clear_other_defaults should not call ClearOtherDefaults when no ID")
	}
}

func TestSetPlanStore(t *testing.T) {
	oldPlanStore := planStore
	defer func() { planStore = oldPlanStore }()

	planStore = nil
	mockStore := &mockTestPlanStore{}
	SetPlanStore(mockStore)

	if planStore == nil {
		t.Error("SetPlanStore should set the global plan store")
	}
}

func TestSetEmailSender(t *testing.T) {
	oldEmailSender := emailSender
	defer func() { emailSender = oldEmailSender }()

	emailSender = nil
	mockEmail := &mockTestEmailSender{}
	SetEmailSender(mockEmail)

	if emailSender != mockEmail {
		t.Error("SetEmailSender should set the global email sender")
	}
}
