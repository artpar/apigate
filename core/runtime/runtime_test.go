package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"github.com/rs/zerolog"
)

// mockStorage implements the Storage interface for testing
type mockStorage struct {
	createTableErr error
	createErr      error
	getErr         error
	listErr        error
	updateErr      error
	deleteErr      error

	createdModule string
	createdData   map[string]any
	lastID        string

	getModule string
	getLookup string
	getValue  string
	getData   map[string]any

	// getDataByLookup allows returning different data based on lookup key/value
	getDataByLookup map[string]map[string]map[string]map[string]any

	listModule  string
	listOpts    ListOptions
	listData    []map[string]any
	listCount   int64

	updateModule string
	updateID     string
	updateData   map[string]any

	deleteModule string
	deleteID     string
}

func (m *mockStorage) CreateTable(ctx context.Context, mod convention.Derived) error {
	return m.createTableErr
}

func (m *mockStorage) Create(ctx context.Context, module string, data map[string]any) (string, error) {
	m.createdModule = module
	m.createdData = data
	if m.createErr != nil {
		return "", m.createErr
	}
	m.lastID = "generated-id"
	return m.lastID, nil
}

func (m *mockStorage) Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error) {
	m.getModule = module
	m.getLookup = lookup
	m.getValue = value
	if m.getErr != nil {
		return nil, m.getErr
	}
	// Check for module-specific lookup data
	if m.getDataByLookup != nil {
		if moduleLookups, ok := m.getDataByLookup[module]; ok {
			if lookupData, ok := moduleLookups[lookup]; ok {
				if data, ok := lookupData[value]; ok {
					return data, nil
				}
			}
		}
	}
	return m.getData, nil
}

func (m *mockStorage) List(ctx context.Context, module string, opts ListOptions) ([]map[string]any, int64, error) {
	m.listModule = module
	m.listOpts = opts
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.listData, m.listCount, nil
}

func (m *mockStorage) Update(ctx context.Context, module string, id string, data map[string]any) error {
	m.updateModule = module
	m.updateID = id
	m.updateData = data
	return m.updateErr
}

func (m *mockStorage) Delete(ctx context.Context, module string, id string) error {
	m.deleteModule = module
	m.deleteID = id
	return m.deleteErr
}

// mockChannel implements the Channel interface for testing
type mockChannel struct {
	name        string
	registerErr error
	startErr    error
	stopErr     error
	started     bool
	stopped     bool
	registered  []string
}

func (m *mockChannel) Name() string {
	return m.name
}

func (m *mockChannel) Register(mod convention.Derived) error {
	if m.registerErr != nil {
		return m.registerErr
	}
	m.registered = append(m.registered, mod.Source.Name)
	return nil
}

func (m *mockChannel) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockChannel) Stop(ctx context.Context) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func newTestRuntime() *Runtime {
	return New(nil, Config{
		Logger: zerolog.Nop(),
	})
}

func newTestRuntimeWithStorage(storage Storage) *Runtime {
	return New(storage, Config{
		Logger: zerolog.Nop(),
	})
}

func TestNew(t *testing.T) {
	t.Run("with nil storage", func(t *testing.T) {
		r := New(nil, Config{
			Logger: zerolog.Nop(),
		})

		if r == nil {
			t.Fatal("New returned nil")
		}
		if r.registry == nil {
			t.Error("Runtime.registry should be initialized")
		}
		if r.channels == nil {
			t.Error("Runtime.channels should be initialized")
		}
		if r.hooks == nil {
			t.Error("Runtime.hooks should be initialized")
		}
		if r.functions == nil {
			t.Error("Runtime.functions should be initialized")
		}
		if r.events == nil {
			t.Error("Runtime.events should be initialized")
		}
		if r.capabilities == nil {
			t.Error("Runtime.capabilities should be initialized")
		}
		if r.exporters == nil {
			t.Error("Runtime.exporters should be initialized")
		}
	})

	t.Run("with storage", func(t *testing.T) {
		storage := &mockStorage{}
		r := New(storage, Config{
			Logger: zerolog.Nop(),
		})

		if r.storage != storage {
			t.Error("Runtime.storage should be set")
		}
	})
}

func TestRuntime_RegisterChannel(t *testing.T) {
	r := newTestRuntime()

	ch := &mockChannel{name: "http"}
	r.RegisterChannel(ch)

	if len(r.channels) != 1 {
		t.Errorf("Runtime.channels should have 1 channel, got %d", len(r.channels))
	}
	if r.channels["http"] != ch {
		t.Error("Runtime.channels should contain registered channel")
	}
}

func TestRuntime_LoadModule(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}

		err := r.LoadModule(mod)
		if err != nil {
			t.Errorf("LoadModule returned error: %v", err)
		}

		derived, ok := r.registry.Get("user")
		if !ok {
			t.Error("Module should be registered")
		}
		if derived.Source.Name != "user" {
			t.Errorf("Derived.Source.Name = %q, want %q", derived.Source.Name, "user")
		}
	})

	t.Run("with channel registration", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		ch := &mockChannel{name: "http"}
		r.RegisterChannel(ch)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}

		err := r.LoadModule(mod)
		if err != nil {
			t.Errorf("LoadModule returned error: %v", err)
		}

		if len(ch.registered) != 1 || ch.registered[0] != "user" {
			t.Errorf("Channel should have registered module, got %v", ch.registered)
		}
	})

	t.Run("with capabilities", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment", "webhook_provider"},
			},
		}

		err := r.LoadModule(mod)
		if err != nil {
			t.Errorf("LoadModule returned error: %v", err)
		}

		providers := r.GetModulesWithCapability("payment")
		if len(providers) != 1 || providers[0] != "payment_stripe" {
			t.Errorf("GetModulesWithCapability = %v, want [payment_stripe]", providers)
		}

		providers = r.GetModulesWithCapability("webhook_provider")
		if len(providers) != 1 || providers[0] != "payment_stripe" {
			t.Errorf("GetModulesWithCapability = %v, want [payment_stripe]", providers)
		}
	})

	t.Run("create table error", func(t *testing.T) {
		storage := &mockStorage{
			createTableErr: errors.New("create table failed"),
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}

		err := r.LoadModule(mod)
		if err == nil {
			t.Error("LoadModule should return error when CreateTable fails")
		}
	})

	t.Run("channel registration error", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		ch := &mockChannel{
			name:        "http",
			registerErr: errors.New("registration failed"),
		}
		r.RegisterChannel(ch)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}

		err := r.LoadModule(mod)
		if err == nil {
			t.Error("LoadModule should return error when channel registration fails")
		}
	})
}

func TestRuntime_Execute(t *testing.T) {
	t.Run("module not found", func(t *testing.T) {
		r := newTestRuntime()

		_, err := r.Execute(context.Background(), "nonexistent", "get", ActionInput{})
		if err == nil {
			t.Error("Execute should return error for nonexistent module")
		}
	})

	t.Run("action not found", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, err := r.Execute(context.Background(), "user", "nonexistent_action", ActionInput{})
		if err == nil {
			t.Error("Execute should return error for nonexistent action")
		}
	})

	t.Run("list action", func(t *testing.T) {
		storage := &mockStorage{
			listData: []map[string]any{
				{"id": "1", "name": "John"},
				{"id": "2", "name": "Jane"},
			},
			listCount: 2,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		result, err := r.Execute(context.Background(), "user", "list", ActionInput{})
		if err != nil {
			t.Errorf("Execute returned error: %v", err)
		}
		if len(result.List) != 2 {
			t.Errorf("Result.List length = %d, want 2", len(result.List))
		}
		if result.Count != 2 {
			t.Errorf("Result.Count = %d, want 2", result.Count)
		}
	})

	t.Run("get action", func(t *testing.T) {
		storage := &mockStorage{
			getData: map[string]any{"id": "1", "name": "John"},
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		result, err := r.Execute(context.Background(), "user", "get", ActionInput{Lookup: "1"})
		if err != nil {
			t.Errorf("Execute returned error: %v", err)
		}
		if result.Data["name"] != "John" {
			t.Errorf("Result.Data[name] = %v, want John", result.Data["name"])
		}
	})

	t.Run("get action not found", func(t *testing.T) {
		storage := &mockStorage{
			getData: nil,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, err := r.Execute(context.Background(), "user", "get", ActionInput{Lookup: "nonexistent"})
		if err == nil {
			t.Error("Execute should return error for not found record")
		}
	})

	t.Run("create action", func(t *testing.T) {
		storage := &mockStorage{
			getData: map[string]any{"id": "generated-id", "name": "John"},
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		result, err := r.Execute(context.Background(), "user", "create", ActionInput{
			Data: map[string]any{"name": "John"},
		})
		if err != nil {
			t.Errorf("Execute returned error: %v", err)
		}
		if result.ID != "generated-id" {
			t.Errorf("Result.ID = %q, want generated-id", result.ID)
		}
		if storage.createdData["name"] != "John" {
			t.Errorf("Storage.createdData[name] = %v, want John", storage.createdData["name"])
		}
	})

	t.Run("update action", func(t *testing.T) {
		storage := &mockStorage{
			getData: map[string]any{"id": "1", "name": "John Updated"},
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		result, err := r.Execute(context.Background(), "user", "update", ActionInput{
			Lookup: "1",
			Data:   map[string]any{"name": "John Updated"},
		})
		if err != nil {
			t.Errorf("Execute returned error: %v", err)
		}
		if result.ID != "1" {
			t.Errorf("Result.ID = %q, want 1", result.ID)
		}
	})

	t.Run("update action not found", func(t *testing.T) {
		storage := &mockStorage{
			getData: nil,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, err := r.Execute(context.Background(), "user", "update", ActionInput{
			Lookup: "nonexistent",
			Data:   map[string]any{"name": "Updated"},
		})
		if err == nil {
			t.Error("Execute should return error for not found record")
		}
	})

	t.Run("delete action", func(t *testing.T) {
		storage := &mockStorage{
			getData: map[string]any{"id": "1", "name": "John"},
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		result, err := r.Execute(context.Background(), "user", "delete", ActionInput{Lookup: "1"})
		if err != nil {
			t.Errorf("Execute returned error: %v", err)
		}
		if result.ID != "1" {
			t.Errorf("Result.ID = %q, want 1", result.ID)
		}
		if storage.deleteID != "1" {
			t.Errorf("Storage.deleteID = %q, want 1", storage.deleteID)
		}
	})

	t.Run("delete action not found", func(t *testing.T) {
		storage := &mockStorage{
			getData: nil,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, err := r.Execute(context.Background(), "user", "delete", ActionInput{Lookup: "nonexistent"})
		if err == nil {
			t.Error("Execute should return error for not found record")
		}
	})
}

func TestRuntime_ExecuteList_Pagination(t *testing.T) {
	storage := &mockStorage{
		listData:  []map[string]any{},
		listCount: 0,
	}
	r := newTestRuntimeWithStorage(storage)

	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}
	_ = r.LoadModule(mod)

	_, _ = r.Execute(context.Background(), "user", "list", ActionInput{
		Data: map[string]any{
			"limit":      50,
			"offset":     10,
			"order_by":   "name",
			"order_desc": true,
		},
	})

	if storage.listOpts.Limit != 50 {
		t.Errorf("ListOpts.Limit = %d, want 50", storage.listOpts.Limit)
	}
	if storage.listOpts.Offset != 10 {
		t.Errorf("ListOpts.Offset = %d, want 10", storage.listOpts.Offset)
	}
	if storage.listOpts.OrderBy != "name" {
		t.Errorf("ListOpts.OrderBy = %q, want name", storage.listOpts.OrderBy)
	}
	if !storage.listOpts.OrderDesc {
		t.Error("ListOpts.OrderDesc should be true")
	}
}

func TestRuntime_ExecuteList_Filters(t *testing.T) {
	t.Run("nested filters", func(t *testing.T) {
		storage := &mockStorage{
			listData:  []map[string]any{},
			listCount: 0,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, _ = r.Execute(context.Background(), "user", "list", ActionInput{
			Data: map[string]any{
				"filters": map[string]any{
					"status": "active",
				},
			},
		})

		if storage.listOpts.Filters["status"] != "active" {
			t.Errorf("ListOpts.Filters[status] = %v, want active", storage.listOpts.Filters["status"])
		}
	})

	t.Run("direct filters", func(t *testing.T) {
		storage := &mockStorage{
			listData:  []map[string]any{},
			listCount: 0,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name":   {Type: schema.FieldTypeString},
				"status": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		_, _ = r.Execute(context.Background(), "user", "list", ActionInput{
			Data: map[string]any{
				"status": "active",
				"name":   "John",
			},
		})

		if storage.listOpts.Filters["status"] != "active" {
			t.Errorf("ListOpts.Filters[status] = %v, want active", storage.listOpts.Filters["status"])
		}
		if storage.listOpts.Filters["name"] != "John" {
			t.Errorf("ListOpts.Filters[name] = %v, want John", storage.listOpts.Filters["name"])
		}
	})
}

func TestHookDispatcher_Dispatch(t *testing.T) {
	t.Run("no handlers", func(t *testing.T) {
		d := &HookDispatcher{handlers: make(map[string][]HookHandler)}

		err := d.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "before",
		})

		if err != nil {
			t.Errorf("Dispatch should not return error with no handlers, got: %v", err)
		}
	})

	t.Run("single handler", func(t *testing.T) {
		d := &HookDispatcher{handlers: make(map[string][]HookHandler)}

		handlerCalled := false
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			handlerCalled = true
			return nil
		})

		err := d.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "before",
		})

		if err != nil {
			t.Errorf("Dispatch returned error: %v", err)
		}
		if !handlerCalled {
			t.Error("Handler should be called")
		}
	})

	t.Run("multiple handlers", func(t *testing.T) {
		d := &HookDispatcher{handlers: make(map[string][]HookHandler)}

		callOrder := []int{}
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			callOrder = append(callOrder, 1)
			return nil
		})
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			callOrder = append(callOrder, 2)
			return nil
		})

		_ = d.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "before",
		})

		if len(callOrder) != 2 {
			t.Errorf("Both handlers should be called, got %d calls", len(callOrder))
		}
		if callOrder[0] != 1 || callOrder[1] != 2 {
			t.Errorf("Handlers should be called in order, got %v", callOrder)
		}
	})

	t.Run("handler returns error", func(t *testing.T) {
		d := &HookDispatcher{handlers: make(map[string][]HookHandler)}

		expectedErr := errors.New("handler error")
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			return expectedErr
		})

		err := d.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "before",
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("Dispatch should return handler error, got: %v", err)
		}
	})

	t.Run("error stops chain", func(t *testing.T) {
		d := &HookDispatcher{handlers: make(map[string][]HookHandler)}

		secondCalled := false
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			return errors.New("first error")
		})
		d.OnHook("user", "create", "before", func(ctx context.Context, event HookEvent) error {
			secondCalled = true
			return nil
		})

		_ = d.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "before",
		})

		if secondCalled {
			t.Error("Second handler should not be called after first returns error")
		}
	})
}

func TestRuntime_OnHook(t *testing.T) {
	r := newTestRuntime()

	handlerCalled := false
	r.OnHook("user", "create", "after", func(ctx context.Context, event HookEvent) error {
		handlerCalled = true
		return nil
	})

	storage := &mockStorage{
		getData: map[string]any{"id": "1", "name": "John"},
	}
	r.storage = storage

	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}
	_ = r.LoadModule(mod)

	_, _ = r.Execute(context.Background(), "user", "create", ActionInput{
		Data: map[string]any{"name": "John"},
	})

	if !handlerCalled {
		t.Error("Hook handler should be called after create")
	}
}

func TestRuntime_Capabilities(t *testing.T) {
	t.Run("GetModulesWithCapability", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod1 := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		mod2 := schema.Module{
			Name: "payment_paddle",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}

		_ = r.LoadModule(mod1)
		_ = r.LoadModule(mod2)

		providers := r.GetModulesWithCapability("payment")
		if len(providers) != 2 {
			t.Errorf("GetModulesWithCapability should return 2 providers, got %d", len(providers))
		}
	})

	t.Run("GetCapabilities", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment", "webhook"},
			},
		}
		_ = r.LoadModule(mod)

		caps := r.GetCapabilities()
		if len(caps) != 2 {
			t.Errorf("GetCapabilities should return 2 capabilities, got %d", len(caps))
		}
		if len(caps["payment"]) != 1 {
			t.Errorf("GetCapabilities[payment] should have 1 provider, got %d", len(caps["payment"]))
		}
	})

	t.Run("HasCapability", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		if r.HasCapability("payment") {
			t.Error("HasCapability should return false for empty capabilities")
		}

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		_ = r.LoadModule(mod)

		if !r.HasCapability("payment") {
			t.Error("HasCapability should return true after loading module with capability")
		}
		if r.HasCapability("nonexistent") {
			t.Error("HasCapability should return false for nonexistent capability")
		}
	})
}

func TestRuntime_GetEnabledProvider(t *testing.T) {
	t.Run("no storage", func(t *testing.T) {
		r := newTestRuntime()

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"enabled": {Type: schema.FieldTypeBool},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		// Load without storage to skip CreateTable
		r.registry.Register(mod)
		r.capabilities["payment"] = []string{"payment_stripe"}

		provider := r.GetEnabledProvider(context.Background(), "payment")
		if provider != "" {
			t.Errorf("GetEnabledProvider should return empty string with no storage, got %q", provider)
		}
	})

	t.Run("enabled provider found", func(t *testing.T) {
		storage := &mockStorage{
			listData: []map[string]any{
				{"id": "1", "enabled": true},
			},
			listCount: 1,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"enabled": {Type: schema.FieldTypeBool},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		_ = r.LoadModule(mod)

		provider := r.GetEnabledProvider(context.Background(), "payment")
		if provider != "payment_stripe" {
			t.Errorf("GetEnabledProvider = %q, want payment_stripe", provider)
		}
	})

	t.Run("enabled as int64", func(t *testing.T) {
		storage := &mockStorage{
			listData: []map[string]any{
				{"id": "1", "enabled": int64(1)},
			},
			listCount: 1,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"enabled": {Type: schema.FieldTypeBool},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		_ = r.LoadModule(mod)

		provider := r.GetEnabledProvider(context.Background(), "payment")
		if provider != "payment_stripe" {
			t.Errorf("GetEnabledProvider = %q, want payment_stripe", provider)
		}
	})

	t.Run("no enabled provider", func(t *testing.T) {
		storage := &mockStorage{
			listData: []map[string]any{
				{"id": "1", "enabled": false},
			},
			listCount: 1,
		}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"enabled": {Type: schema.FieldTypeBool},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		_ = r.LoadModule(mod)

		provider := r.GetEnabledProvider(context.Background(), "payment")
		if provider != "" {
			t.Errorf("GetEnabledProvider should return empty string, got %q", provider)
		}
	})
}

func TestRuntime_GetAllEnabledProviders(t *testing.T) {
	storage := &mockStorage{
		listData: []map[string]any{
			{"id": "1", "enabled": true},
		},
		listCount: 1,
	}
	r := newTestRuntimeWithStorage(storage)

	mod1 := schema.Module{
		Name: "payment_stripe",
		Schema: map[string]schema.Field{
			"enabled": {Type: schema.FieldTypeBool},
		},
		Meta: schema.ModuleMeta{
			Implements: []string{"payment"},
		},
	}
	mod2 := schema.Module{
		Name: "payment_paddle",
		Schema: map[string]schema.Field{
			"enabled": {Type: schema.FieldTypeBool},
		},
		Meta: schema.ModuleMeta{
			Implements: []string{"payment"},
		},
	}

	_ = r.LoadModule(mod1)
	_ = r.LoadModule(mod2)

	providers := r.GetAllEnabledProviders(context.Background(), "payment")
	if len(providers) != 2 {
		t.Errorf("GetAllEnabledProviders should return 2 providers, got %d", len(providers))
	}
}

func TestRuntime_Functions(t *testing.T) {
	r := newTestRuntime()

	if r.Functions() == nil {
		t.Error("Functions() should not return nil")
	}

	handlerCalled := false
	r.RegisterFunction("test_func", func(ctx context.Context, event HookEvent) error {
		handlerCalled = true
		return nil
	})

	if !r.Functions().Has("test_func") {
		t.Error("Function should be registered")
	}

	_ = r.Functions().Call(context.Background(), "test_func", HookEvent{})
	if !handlerCalled {
		t.Error("Function should be called")
	}
}

func TestRuntime_Events(t *testing.T) {
	r := newTestRuntime()

	if r.Events() == nil {
		t.Error("Events() should not return nil")
	}
}

func TestRuntime_Analytics(t *testing.T) {
	r := newTestRuntime()

	if r.Analytics() != nil {
		t.Error("Analytics() should return nil when not configured")
	}
}

func TestRuntime_Exporters(t *testing.T) {
	r := newTestRuntime()

	if r.Exporters() == nil {
		t.Error("Exporters() should not return nil")
	}
}

func TestRuntime_Registry(t *testing.T) {
	r := newTestRuntime()

	if r.Registry() == nil {
		t.Error("Registry() should not return nil")
	}
}

func TestRuntime_StartStop(t *testing.T) {
	t.Run("start and stop channels", func(t *testing.T) {
		r := newTestRuntime()

		ch := &mockChannel{name: "http"}
		r.RegisterChannel(ch)

		err := r.Start(context.Background())
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
		if !ch.started {
			t.Error("Channel should be started")
		}

		err = r.Stop(context.Background())
		if err != nil {
			t.Errorf("Stop returned error: %v", err)
		}
		if !ch.stopped {
			t.Error("Channel should be stopped")
		}
	})

	t.Run("start error", func(t *testing.T) {
		r := newTestRuntime()

		ch := &mockChannel{name: "http", startErr: errors.New("start failed")}
		r.RegisterChannel(ch)

		err := r.Start(context.Background())
		if err == nil {
			t.Error("Start should return error when channel fails to start")
		}
	})

	t.Run("stop error", func(t *testing.T) {
		r := newTestRuntime()

		ch := &mockChannel{name: "http", stopErr: errors.New("stop failed")}
		r.RegisterChannel(ch)

		err := r.Stop(context.Background())
		if err == nil {
			t.Error("Stop should return error when channel fails to stop")
		}
	})
}

func TestValidationError(t *testing.T) {
	validationResult := schema.ValidationResult{
		Valid: false,
		Errors: []schema.ConstraintError{
			{Field: "name", Message: "is required"},
		},
	}

	err := &ValidationError{Result: validationResult}

	errStr := err.Error()
	if errStr == "" {
		t.Error("ValidationError.Error() should not be empty")
	}
	if errStr != "validation failed: name: is required" {
		t.Errorf("ValidationError.Error() = %q, unexpected format", errStr)
	}
}

func TestParseHookPhase(t *testing.T) {
	tests := []struct {
		input         string
		expectedAction string
		expectedPhase  string
	}{
		{"before_create", "create", "before"},
		{"after_create", "create", "after"},
		{"before_update", "update", "before"},
		{"after_update", "update", "after"},
		{"before_delete", "delete", "before"},
		{"after_delete", "delete", "after"},
		{"invalid", "", ""},
		{"", "", ""},
		{"before_", "", "before"},
		{"after_", "", "after"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			action, phase := parseHookPhase(tc.input)
			if action != tc.expectedAction {
				t.Errorf("parseHookPhase(%q) action = %q, want %q", tc.input, action, tc.expectedAction)
			}
			if phase != tc.expectedPhase {
				t.Errorf("parseHookPhase(%q) phase = %q, want %q", tc.input, phase, tc.expectedPhase)
			}
		})
	}
}

func TestRuntime_RegisterModuleHooks(t *testing.T) {
	t.Run("emit hook", func(t *testing.T) {
		r := newTestRuntime()

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
			Hooks: map[string][]schema.Hook{
				"after_create": {
					{Emit: "user.created"},
				},
			},
		}

		r.RegisterModuleHooks(mod)

		// The hook should be registered - verify by checking that dispatch doesn't error
		err := r.hooks.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "after",
			Data:   map[string]any{},
			Meta:   map[string]any{},
		})
		if err != nil {
			t.Errorf("Dispatch returned error: %v", err)
		}
	})

	t.Run("call hook", func(t *testing.T) {
		r := newTestRuntime()

		funcCalled := false
		r.RegisterFunction("process_user", func(ctx context.Context, event HookEvent) error {
			funcCalled = true
			return nil
		})

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
			Hooks: map[string][]schema.Hook{
				"after_create": {
					{Call: "process_user"},
				},
			},
		}

		r.RegisterModuleHooks(mod)

		_ = r.hooks.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "after",
		})

		if !funcCalled {
			t.Error("Function should be called via hook")
		}
	})

	t.Run("call unregistered function", func(t *testing.T) {
		r := newTestRuntime()

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
			Hooks: map[string][]schema.Hook{
				"after_create": {
					{Call: "nonexistent_func"},
				},
			},
		}

		r.RegisterModuleHooks(mod)

		// Should not error even if function doesn't exist
		err := r.hooks.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "after",
		})
		if err != nil {
			t.Errorf("Dispatch should not error for unregistered function, got: %v", err)
		}
	})

	t.Run("event hook", func(t *testing.T) {
		r := newTestRuntime()

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
			Hooks: map[string][]schema.Hook{
				"after_create": {
					{Event: "user.created"},
				},
			},
		}

		r.RegisterModuleHooks(mod)

		// Should work without error
		err := r.hooks.Dispatch(context.Background(), HookEvent{
			Module: "user",
			Action: "create",
			Phase:  "after",
			Data:   map[string]any{},
			Meta:   map[string]any{},
		})
		if err != nil {
			t.Errorf("Dispatch returned error: %v", err)
		}
	})

	t.Run("invalid hook phase", func(t *testing.T) {
		r := newTestRuntime()

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
			Hooks: map[string][]schema.Hook{
				"invalid_phase": {
					{Emit: "user.created"},
				},
			},
		}

		// Should not panic
		r.RegisterModuleHooks(mod)
	})
}

func TestRuntime_CustomAction(t *testing.T) {
	storage := &mockStorage{
		getData: map[string]any{"id": "1", "status": "inactive"},
	}
	r := newTestRuntimeWithStorage(storage)

	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name":   {Type: schema.FieldTypeString},
			"status": {Type: schema.FieldTypeString},
		},
		Actions: map[string]schema.Action{
			"activate": {
				Set: map[string]string{
					"status": "active",
				},
			},
		},
	}
	_ = r.LoadModule(mod)

	result, err := r.Execute(context.Background(), "user", "activate", ActionInput{
		Lookup: "1",
	})
	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}
	if result.ID != "1" {
		t.Errorf("Result.ID = %q, want 1", result.ID)
	}

	// Verify the status was set
	if storage.updateData["status"] != "active" {
		t.Errorf("Storage.updateData[status] = %v, want active", storage.updateData["status"])
	}
}

func TestRuntime_ResolveDependencies(t *testing.T) {
	t.Run("module not found", func(t *testing.T) {
		r := newTestRuntime()

		_, err := r.ResolveDependencies(context.Background(), "nonexistent")
		if err == nil {
			t.Error("ResolveDependencies should return error for nonexistent module")
		}
	})

	t.Run("module with no requirements", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		dc, err := r.ResolveDependencies(context.Background(), "user")
		if err != nil {
			t.Errorf("ResolveDependencies returned error: %v", err)
		}
		if len(dc.Dependencies) != 0 {
			t.Errorf("DependencyContext should have no dependencies, got %d", len(dc.Dependencies))
		}
	})
}

func TestRuntime_ValidateModuleDependencies(t *testing.T) {
	t.Run("module not found", func(t *testing.T) {
		r := newTestRuntime()

		err := r.ValidateModuleDependencies(context.Background(), "nonexistent")
		if err == nil {
			t.Error("ValidateModuleDependencies should return error for nonexistent module")
		}
	})

	t.Run("module with no requirements", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "user",
			Schema: map[string]schema.Field{
				"name": {Type: schema.FieldTypeString},
			},
		}
		_ = r.LoadModule(mod)

		err := r.ValidateModuleDependencies(context.Background(), "user")
		if err != nil {
			t.Errorf("ValidateModuleDependencies returned error: %v", err)
		}
	})

	t.Run("missing required dependency", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		mod := schema.Module{
			Name: "checkout",
			Schema: map[string]schema.Field{
				"amount": {Type: schema.FieldTypeFloat},
			},
			Meta: schema.ModuleMeta{
				Requires: map[string]schema.ModuleRequirement{
					"payment": {
						Capability: "payment",
						Required:   true,
					},
				},
			},
		}
		_ = r.LoadModule(mod)

		err := r.ValidateModuleDependencies(context.Background(), "checkout")
		if err == nil {
			t.Error("ValidateModuleDependencies should return error for missing required dependency")
		}
	})
}

func TestRuntime_GetModuleDependencyInfo(t *testing.T) {
	t.Run("module not found", func(t *testing.T) {
		r := newTestRuntime()

		info := r.GetModuleDependencyInfo("nonexistent")
		if info != nil {
			t.Error("GetModuleDependencyInfo should return nil for nonexistent module")
		}
	})

	t.Run("module with requirements", func(t *testing.T) {
		storage := &mockStorage{}
		r := newTestRuntimeWithStorage(storage)

		paymentMod := schema.Module{
			Name: "payment_stripe",
			Schema: map[string]schema.Field{
				"api_key": {Type: schema.FieldTypeString},
			},
			Meta: schema.ModuleMeta{
				Implements: []string{"payment"},
			},
		}
		_ = r.LoadModule(paymentMod)

		checkoutMod := schema.Module{
			Name: "checkout",
			Schema: map[string]schema.Field{
				"amount": {Type: schema.FieldTypeFloat},
			},
			Meta: schema.ModuleMeta{
				Requires: map[string]schema.ModuleRequirement{
					"payment": {
						Capability:  "payment",
						Required:    true,
						Description: "Payment processor",
						Default:     "payment_stripe",
					},
				},
			},
		}
		_ = r.LoadModule(checkoutMod)

		info := r.GetModuleDependencyInfo("checkout")
		if info == nil {
			t.Fatal("GetModuleDependencyInfo should not return nil")
		}

		paymentInfo, ok := info["payment"]
		if !ok {
			t.Fatal("GetModuleDependencyInfo should contain payment dependency")
		}
		if paymentInfo.Capability != "payment" {
			t.Errorf("DependencyInfo.Capability = %q, want payment", paymentInfo.Capability)
		}
		if !paymentInfo.Required {
			t.Error("DependencyInfo.Required should be true")
		}
		if paymentInfo.Default != "payment_stripe" {
			t.Errorf("DependencyInfo.Default = %q, want payment_stripe", paymentInfo.Default)
		}
		if len(paymentInfo.AvailableProviders) != 1 {
			t.Errorf("DependencyInfo.AvailableProviders should have 1 provider, got %d", len(paymentInfo.AvailableProviders))
		}
	})
}

func TestDependencyContext_Execute(t *testing.T) {
	t.Run("dependency not found", func(t *testing.T) {
		r := newTestRuntime()

		dc := &DependencyContext{
			Dependencies: make(map[string]ResolvedDependency),
			Runtime:      r,
		}

		_, err := dc.Execute(context.Background(), "nonexistent", "get", ActionInput{})
		if err == nil {
			t.Error("Execute should return error for nonexistent dependency")
		}
	})
}

func TestListOptions(t *testing.T) {
	opts := ListOptions{
		Limit:     10,
		Offset:    5,
		Filters:   map[string]any{"status": "active"},
		OrderBy:   "name",
		OrderDesc: true,
	}

	if opts.Limit != 10 {
		t.Errorf("ListOptions.Limit = %d, want 10", opts.Limit)
	}
	if opts.Offset != 5 {
		t.Errorf("ListOptions.Offset = %d, want 5", opts.Offset)
	}
	if opts.OrderBy != "name" {
		t.Errorf("ListOptions.OrderBy = %q, want name", opts.OrderBy)
	}
	if !opts.OrderDesc {
		t.Error("ListOptions.OrderDesc should be true")
	}
}

func TestActionInput(t *testing.T) {
	input := ActionInput{
		Data:   map[string]any{"name": "John"},
		Lookup: "123",
		Channel: "http",
		Auth: AuthContext{
			UserID:  "user1",
			Role:    "admin",
			IsAdmin: true,
		},
		RemoteIP:     "192.168.1.1",
		RequestBytes: 1024,
	}

	if input.Lookup != "123" {
		t.Errorf("ActionInput.Lookup = %q, want 123", input.Lookup)
	}
	if input.Channel != "http" {
		t.Errorf("ActionInput.Channel = %q, want http", input.Channel)
	}
	if input.Auth.UserID != "user1" {
		t.Errorf("ActionInput.Auth.UserID = %q, want user1", input.Auth.UserID)
	}
	if !input.Auth.IsAdmin {
		t.Error("ActionInput.Auth.IsAdmin should be true")
	}
	if input.RemoteIP != "192.168.1.1" {
		t.Errorf("ActionInput.RemoteIP = %q, want 192.168.1.1", input.RemoteIP)
	}
	if input.RequestBytes != 1024 {
		t.Errorf("ActionInput.RequestBytes = %d, want 1024", input.RequestBytes)
	}
}

func TestActionResult(t *testing.T) {
	result := ActionResult{
		Data: map[string]any{"name": "John"},
		List: []map[string]any{
			{"id": "1"},
			{"id": "2"},
		},
		ID:    "123",
		Count: 2,
		Meta:  map[string]any{"raw_key": "secret123"},
	}

	if result.ID != "123" {
		t.Errorf("ActionResult.ID = %q, want 123", result.ID)
	}
	if result.Count != 2 {
		t.Errorf("ActionResult.Count = %d, want 2", result.Count)
	}
	if len(result.List) != 2 {
		t.Errorf("ActionResult.List length = %d, want 2", len(result.List))
	}
	if result.Meta["raw_key"] != "secret123" {
		t.Errorf("ActionResult.Meta[raw_key] = %v, want secret123", result.Meta["raw_key"])
	}
}

func TestRuntime_RegisterExporter(t *testing.T) {
	r := newTestRuntime()

	// Create a mock exporter
	exp := &mockExporter{name: "test-exporter"}

	// Register should succeed
	err := r.RegisterExporter(exp)
	if err != nil {
		t.Errorf("RegisterExporter() error = %v", err)
	}

	// Exporter should be in the registry
	exporters := r.Exporters()
	if exporters == nil {
		t.Error("Exporters() should not be nil")
	}
}

func TestRuntime_LoadModulesFromDir_NonExistent(t *testing.T) {
	r := newTestRuntime()

	// Loading from non-existent directory should error
	err := r.LoadModulesFromDir("/nonexistent/path/to/modules")
	if err == nil {
		t.Error("LoadModulesFromDir() should error for non-existent directory")
	}
}

// mockExporter implements the Exporter interface for testing
type mockExporter struct {
	name    string
	started bool
	stopped bool
}

func (e *mockExporter) Name() string {
	return e.name
}

func (e *mockExporter) Start(ctx context.Context) error {
	e.started = true
	return nil
}

func (e *mockExporter) Stop(ctx context.Context) error {
	e.stopped = true
	return nil
}
