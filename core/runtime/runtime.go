// Package runtime provides the module execution environment.
// It loads modules, manages their lifecycle, and executes actions.
package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artpar/apigate/core/analytics"
	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/events"
	"github.com/artpar/apigate/core/exporter"
	"github.com/artpar/apigate/core/registry"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/core/validation"
	"github.com/rs/zerolog"
)

// Runtime is the core execution environment for modules.
type Runtime struct {
	mu sync.RWMutex

	// registry manages module registration and path routing
	registry *registry.Registry

	// storage provides data persistence
	storage Storage

	// analytics collects execution metrics
	analytics analytics.Analytics

	// exporters export metrics to external systems
	exporters *exporter.Registry

	// validator validates input data
	validator *validation.Validator

	// channels are the communication adapters
	channels map[string]Channel

	// hooks dispatcher
	hooks *HookDispatcher

	// functions registry for "call:" hooks
	functions *FunctionRegistry

	// events bus for "emit:" hooks
	events *events.Bus

	// capabilities tracks which modules implement which capabilities
	// e.g., "payment" -> ["payment_stripe", "payment_paddle"]
	capabilities map[string][]string

	// logger for hook system
	logger zerolog.Logger

	// config
	config Config
}

// Config configures the runtime.
type Config struct {
	// ModulesDir is the directory containing module definitions.
	ModulesDir string

	// PluginsDir is the directory containing plugin modules.
	PluginsDir string

	// Analytics is the analytics collector (optional).
	Analytics analytics.Analytics

	// Logger for runtime and hook system.
	Logger zerolog.Logger
}

// Storage is the interface for data persistence.
// Implementations include SQLite, PostgreSQL, etc.
type Storage interface {
	// CreateTable creates a table for a module.
	CreateTable(ctx context.Context, mod convention.Derived) error

	// Create inserts a new record.
	Create(ctx context.Context, module string, data map[string]any) (string, error)

	// Get retrieves a record by lookup field.
	Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error)

	// List retrieves multiple records.
	List(ctx context.Context, module string, opts ListOptions) ([]map[string]any, int64, error)

	// Update modifies an existing record.
	Update(ctx context.Context, module string, id string, data map[string]any) error

	// Delete removes a record.
	Delete(ctx context.Context, module string, id string) error
}

// ListOptions configures list queries.
type ListOptions struct {
	// Limit is the maximum number of records to return.
	Limit int

	// Offset is the number of records to skip.
	Offset int

	// Filters are field-value pairs to filter by.
	Filters map[string]any

	// OrderBy is the field to sort by.
	OrderBy string

	// OrderDesc sorts in descending order.
	OrderDesc bool
}

// Channel is a communication adapter (HTTP, CLI, WebSocket, etc.)
type Channel interface {
	// Name returns the channel name.
	Name() string

	// Register registers a module with this channel.
	Register(mod convention.Derived) error

	// Start starts the channel.
	Start(ctx context.Context) error

	// Stop stops the channel.
	Stop(ctx context.Context) error
}

// HookDispatcher manages event hooks.
type HookDispatcher struct {
	handlers map[string][]HookHandler
}

// HookHandler handles a hook event.
type HookHandler func(ctx context.Context, event HookEvent) error

// HookEvent represents an event that triggers hooks.
type HookEvent struct {
	// Module that triggered the event.
	Module string

	// Action that triggered the event.
	Action string

	// Phase is "before" or "after".
	Phase string

	// Data is the action input/output.
	Data map[string]any

	// Meta carries extra data through hooks (e.g., raw API key for one-time display).
	// Hooks can read/write this to pass values back to the caller.
	Meta map[string]any
}

// New creates a new runtime.
func New(storage Storage, config Config) *Runtime {
	r := &Runtime{
		registry:     registry.New(),
		storage:      storage,
		analytics:    config.Analytics,
		validator:    validation.New(make(map[string]convention.Derived)),
		channels:     make(map[string]Channel),
		hooks:        &HookDispatcher{handlers: make(map[string][]HookHandler)},
		functions:    NewFunctionRegistry(),
		events:       events.NewBus(config.Logger),
		capabilities: make(map[string][]string),
		logger:       config.Logger,
		config:       config,
	}

	// Initialize exporter registry with analytics store
	if config.Analytics != nil {
		r.exporters = exporter.NewRegistry(config.Analytics)
	} else {
		r.exporters = exporter.NewRegistry(nil)
	}

	return r
}

// Analytics returns the analytics collector.
func (r *Runtime) Analytics() analytics.Analytics {
	return r.analytics
}

// Exporters returns the exporter registry.
func (r *Runtime) Exporters() *exporter.Registry {
	return r.exporters
}

// RegisterExporter registers a metrics exporter.
func (r *Runtime) RegisterExporter(exp exporter.Exporter) error {
	return r.exporters.Register(exp)
}

// RegisterChannel registers a communication channel.
func (r *Runtime) RegisterChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[ch.Name()] = ch
}

// LoadModule loads and registers a single module.
func (r *Runtime) LoadModule(mod schema.Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Register with the registry (checks conflicts)
	if err := r.registry.Register(mod); err != nil {
		return fmt.Errorf("register module %q: %w", mod.Name, err)
	}

	// Get the derived module
	derived, _ := r.registry.Get(mod.Name)

	// Create storage table
	if r.storage != nil {
		if err := r.storage.CreateTable(context.Background(), derived); err != nil {
			return fmt.Errorf("create table for %q: %w", mod.Name, err)
		}
	}

	// Register with all channels
	for _, ch := range r.channels {
		if err := ch.Register(derived); err != nil {
			return fmt.Errorf("register %q with channel %q: %w", mod.Name, ch.Name(), err)
		}
	}

	// Register capabilities from meta.implements
	for _, capability := range mod.Meta.Implements {
		r.capabilities[capability] = append(r.capabilities[capability], mod.Name)
		r.logger.Debug().
			Str("module", mod.Name).
			Str("capability", capability).
			Msg("registered capability provider")
	}

	// Update validator with all modules
	r.validator.UpdateModules(r.registry.All())

	return nil
}

// LoadModulesFromDir loads all modules from a directory.
func (r *Runtime) LoadModulesFromDir(dir string) error {
	modules, err := schema.ParseDir(dir)
	if err != nil {
		return fmt.Errorf("parse modules from %q: %w", dir, err)
	}

	for _, mod := range modules {
		if err := r.LoadModule(mod); err != nil {
			return err
		}
	}

	return nil
}

// Execute executes an action on a module.
func (r *Runtime) Execute(ctx context.Context, module, action string, input ActionInput) (ActionResult, error) {
	start := time.Now()

	result, err := r.executeInternal(ctx, module, action, input)

	// Record analytics (non-blocking)
	if r.analytics != nil {
		event := analytics.Event{
			Timestamp:    start,
			Channel:      input.Channel,
			Module:       module,
			Action:       action,
			RecordID:     result.ID,
			UserID:       input.Auth.UserID,
			RemoteIP:     input.RemoteIP,
			DurationNS:   time.Since(start).Nanoseconds(),
			RequestBytes: input.RequestBytes,
			Success:      err == nil,
		}
		if err != nil {
			event.Error = err.Error()
		}
		r.analytics.Record(event)
	}

	return result, err
}

// executeInternal performs the actual action execution.
func (r *Runtime) executeInternal(ctx context.Context, module, action string, input ActionInput) (ActionResult, error) {
	r.mu.RLock()
	derived, ok := r.registry.Get(module)
	r.mu.RUnlock()

	if !ok {
		return ActionResult{}, fmt.Errorf("module %q not found", module)
	}

	// Find the action
	var act *convention.DerivedAction
	for i := range derived.Actions {
		if derived.Actions[i].Name == action {
			act = &derived.Actions[i]
			break
		}
	}

	if act == nil {
		return ActionResult{}, fmt.Errorf("action %q not found in module %q", action, module)
	}

	// Create meta map for hooks to pass data back to caller
	meta := make(map[string]any)

	// Run before hooks
	if err := r.hooks.Dispatch(ctx, HookEvent{
		Module: module,
		Action: action,
		Phase:  "before",
		Data:   input.Data,
		Meta:   meta,
	}); err != nil {
		return ActionResult{}, fmt.Errorf("before hook: %w", err)
	}

	// Execute the action
	var result ActionResult
	var err error

	switch act.Type {
	case schema.ActionTypeList:
		result, err = r.executeList(ctx, derived, act, input)
	case schema.ActionTypeGet:
		result, err = r.executeGet(ctx, derived, act, input)
	case schema.ActionTypeCreate:
		result, err = r.executeCreate(ctx, derived, act, input)
	case schema.ActionTypeUpdate:
		result, err = r.executeUpdate(ctx, derived, act, input)
	case schema.ActionTypeDelete:
		result, err = r.executeDelete(ctx, derived, act, input)
	case schema.ActionTypeCustom:
		result, err = r.executeCustom(ctx, derived, act, input)
	default:
		err = fmt.Errorf("unknown action type: %v", act.Type)
	}

	if err != nil {
		return ActionResult{}, err
	}

	// Run after hooks
	if err := r.hooks.Dispatch(ctx, HookEvent{
		Module: module,
		Action: action,
		Phase:  "after",
		Data:   result.Data,
		Meta:   meta,
	}); err != nil {
		return ActionResult{}, fmt.Errorf("after hook: %w", err)
	}

	// Attach meta to result for caller to access
	result.Meta = meta

	return result, nil
}

// ActionInput contains input for an action.
type ActionInput struct {
	// Data contains the input field values.
	Data map[string]any

	// Lookup is the ID or lookup field value for get/update/delete.
	Lookup string

	// Channel is the source channel (http, cli, tty, grpc, websocket).
	Channel string

	// Auth contains authentication context.
	Auth AuthContext

	// RemoteIP is the client IP address (for HTTP requests).
	RemoteIP string

	// RequestBytes is the size of the incoming request.
	RequestBytes int64
}

// AuthContext contains authentication information.
type AuthContext struct {
	// UserID is the authenticated user ID.
	UserID string

	// Role is the user's role.
	Role string

	// IsAdmin indicates admin access.
	IsAdmin bool
}

// ActionResult contains the result of an action.
type ActionResult struct {
	// Data contains the result data (single record or list).
	Data map[string]any

	// List contains multiple records for list actions.
	List []map[string]any

	// ID is the created/updated record ID.
	ID string

	// Count is the total count for list actions.
	Count int64

	// Meta carries extra data from hooks (e.g., raw API key shown only once).
	Meta map[string]any
}

// executeList handles list actions.
func (r *Runtime) executeList(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	opts := ListOptions{
		Limit:  100,
		Offset: 0,
	}

	// Extract pagination from input
	if limit, ok := input.Data["limit"].(int); ok {
		opts.Limit = limit
	}
	if offset, ok := input.Data["offset"].(int); ok {
		opts.Offset = offset
	}
	if orderBy, ok := input.Data["order_by"].(string); ok {
		opts.OrderBy = orderBy
	}
	if orderDesc, ok := input.Data["order_desc"].(bool); ok {
		opts.OrderDesc = orderDesc
	}

	// Extract filters (either from nested "filters" key or directly from input)
	if filters, ok := input.Data["filters"].(map[string]any); ok {
		opts.Filters = filters
	} else {
		// Copy only field values, excluding pagination params
		opts.Filters = make(map[string]any)
		for k, v := range input.Data {
			if k != "limit" && k != "offset" && k != "order_by" && k != "order_desc" && k != "filters" {
				opts.Filters[k] = v
			}
		}
	}

	list, count, err := r.storage.List(ctx, mod.Source.Name, opts)
	if err != nil {
		return ActionResult{}, err
	}

	return ActionResult{List: list, Count: count}, nil
}

// executeGet handles get actions.
func (r *Runtime) executeGet(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	// Try each lookup field
	for _, lookup := range mod.Lookups {
		data, err := r.storage.Get(ctx, mod.Source.Name, lookup, input.Lookup)
		if err == nil && data != nil {
			return ActionResult{Data: data}, nil
		}
	}

	return ActionResult{}, fmt.Errorf("record not found: %s", input.Lookup)
}

// executeCreate handles create actions.
func (r *Runtime) executeCreate(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	// Validate input data
	validationResult := r.validator.ValidateCreate(mod.Source.Name, input.Data)
	if !validationResult.Valid {
		return ActionResult{}, &ValidationError{Result: validationResult}
	}

	// Resolve ref fields (lookup by name -> id)
	data := make(map[string]any)
	for k, v := range input.Data {
		data[k] = v
	}
	if err := r.resolveRefs(ctx, mod, data); err != nil {
		return ActionResult{}, err
	}

	id, err := r.storage.Create(ctx, mod.Source.Name, data)
	if err != nil {
		return ActionResult{}, err
	}

	// Fetch the created record
	result, _ := r.storage.Get(ctx, mod.Source.Name, "id", id)

	return ActionResult{ID: id, Data: result}, nil
}

// resolveRefs resolves ref field values from names to IDs.
func (r *Runtime) resolveRefs(ctx context.Context, mod convention.Derived, data map[string]any) error {
	for _, field := range mod.Fields {
		if field.Type != schema.FieldTypeRef || field.Ref == "" {
			continue
		}

		val, ok := data[field.Name]
		if !ok || val == nil {
			continue
		}

		valStr, ok := val.(string)
		if !ok || valStr == "" {
			continue
		}

		// Get the referenced module
		refMod, ok := r.registry.Get(field.Ref)
		if !ok {
			return fmt.Errorf("referenced module %q not found", field.Ref)
		}

		// Try to find the referenced record by its lookup fields
		var refID string
		for _, lookup := range refMod.Lookups {
			record, err := r.storage.Get(ctx, field.Ref, lookup, valStr)
			if err == nil && record != nil {
				if id, ok := record["id"].(string); ok {
					refID = id
					break
				}
			}
		}

		if refID == "" {
			return fmt.Errorf("%s %q not found", field.Ref, valStr)
		}

		// Replace the value with the resolved ID
		data[field.Name] = refID
	}

	return nil
}

// executeUpdate handles update actions.
func (r *Runtime) executeUpdate(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	// Validate input data
	validationResult := r.validator.ValidateUpdate(mod.Source.Name, input.Data)
	if !validationResult.Valid {
		return ActionResult{}, &ValidationError{Result: validationResult}
	}

	// Find the record first
	var id string
	for _, lookup := range mod.Lookups {
		data, err := r.storage.Get(ctx, mod.Source.Name, lookup, input.Lookup)
		if err == nil && data != nil {
			if idVal, ok := data["id"].(string); ok {
				id = idVal
				break
			}
		}
	}

	if id == "" {
		return ActionResult{}, fmt.Errorf("record not found: %s", input.Lookup)
	}

	// Resolve ref fields (lookup by name -> id)
	updateData := make(map[string]any)
	for k, v := range input.Data {
		updateData[k] = v
	}
	if err := r.resolveRefs(ctx, mod, updateData); err != nil {
		return ActionResult{}, err
	}

	if err := r.storage.Update(ctx, mod.Source.Name, id, updateData); err != nil {
		return ActionResult{}, err
	}

	// Fetch the updated record
	data, _ := r.storage.Get(ctx, mod.Source.Name, "id", id)

	return ActionResult{ID: id, Data: data}, nil
}

// executeDelete handles delete actions.
func (r *Runtime) executeDelete(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	// Find the record first
	var id string
	for _, lookup := range mod.Lookups {
		data, err := r.storage.Get(ctx, mod.Source.Name, lookup, input.Lookup)
		if err == nil && data != nil {
			if idVal, ok := data["id"].(string); ok {
				id = idVal
				break
			}
		}
	}

	if id == "" {
		return ActionResult{}, fmt.Errorf("record not found: %s", input.Lookup)
	}

	if err := r.storage.Delete(ctx, mod.Source.Name, id); err != nil {
		return ActionResult{}, err
	}

	return ActionResult{ID: id}, nil
}

// executeCustom handles custom actions.
func (r *Runtime) executeCustom(ctx context.Context, mod convention.Derived, act *convention.DerivedAction, input ActionInput) (ActionResult, error) {
	// Find the record
	var id string
	for _, lookup := range mod.Lookups {
		data, err := r.storage.Get(ctx, mod.Source.Name, lookup, input.Lookup)
		if err == nil && data != nil {
			if idVal, ok := data["id"].(string); ok {
				id = idVal
				break
			}
		}
	}

	if id == "" {
		return ActionResult{}, fmt.Errorf("record not found: %s", input.Lookup)
	}

	// Merge set values with input data
	updateData := make(map[string]any)
	for k, v := range act.Set {
		updateData[k] = v
	}
	for k, v := range input.Data {
		updateData[k] = v
	}

	if err := r.storage.Update(ctx, mod.Source.Name, id, updateData); err != nil {
		return ActionResult{}, err
	}

	// Fetch the updated record
	data, _ := r.storage.Get(ctx, mod.Source.Name, "id", id)

	return ActionResult{ID: id, Data: data}, nil
}

// ValidationError wraps validation failures.
type ValidationError struct {
	Result schema.ValidationResult
}

// Error returns the validation error message.
func (e *ValidationError) Error() string {
	return "validation failed: " + e.Result.Error()
}

// Dispatch dispatches a hook event.
func (d *HookDispatcher) Dispatch(ctx context.Context, event HookEvent) error {
	key := fmt.Sprintf("%s.%s.%s", event.Module, event.Action, event.Phase)

	handlers := d.handlers[key]
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// OnHook registers a hook handler.
func (d *HookDispatcher) OnHook(module, action, phase string, handler HookHandler) {
	key := fmt.Sprintf("%s.%s.%s", module, action, phase)
	d.handlers[key] = append(d.handlers[key], handler)
}

// Registry returns the module registry.
func (r *Runtime) Registry() *registry.Registry {
	return r.registry
}

// GetModulesWithCapability returns all modules that implement a capability.
// e.g., GetModulesWithCapability("payment") -> ["payment_stripe", "payment_paddle"]
func (r *Runtime) GetModulesWithCapability(capability string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capabilities[capability]
}

// GetCapabilities returns all registered capabilities and their providers.
func (r *Runtime) GetCapabilities() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent modification
	result := make(map[string][]string, len(r.capabilities))
	for k, v := range r.capabilities {
		providers := make([]string, len(v))
		copy(providers, v)
		result[k] = providers
	}
	return result
}

// HasCapability checks if any module provides a capability.
func (r *Runtime) HasCapability(capability string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.capabilities[capability]) > 0
}

// GetEnabledProvider returns the first enabled provider for a capability.
// Multiple providers can be registered but only one should have enabled=true.
// Returns empty string if no enabled provider found.
func (r *Runtime) GetEnabledProvider(ctx context.Context, capability string) string {
	r.mu.RLock()
	providers := r.capabilities[capability]
	r.mu.RUnlock()

	for _, moduleName := range providers {
		// Get the provider's settings to check if enabled
		if r.storage == nil {
			continue
		}

		settings, _, err := r.storage.List(ctx, moduleName, ListOptions{Limit: 1})
		if err != nil || len(settings) == 0 {
			continue
		}

		// Check the enabled field
		if enabled, ok := settings[0]["enabled"].(bool); ok && enabled {
			return moduleName
		}
		// Also check int (SQLite stores bool as int)
		if enabled, ok := settings[0]["enabled"].(int64); ok && enabled == 1 {
			return moduleName
		}
	}

	return ""
}

// GetAllEnabledProviders returns all enabled providers for a capability.
// Useful when multiple providers can be active simultaneously.
func (r *Runtime) GetAllEnabledProviders(ctx context.Context, capability string) []string {
	r.mu.RLock()
	providers := r.capabilities[capability]
	r.mu.RUnlock()

	var enabled []string
	for _, moduleName := range providers {
		if r.storage == nil {
			continue
		}

		settings, _, err := r.storage.List(ctx, moduleName, ListOptions{Limit: 1})
		if err != nil || len(settings) == 0 {
			continue
		}

		// Check the enabled field
		if e, ok := settings[0]["enabled"].(bool); ok && e {
			enabled = append(enabled, moduleName)
		}
		if e, ok := settings[0]["enabled"].(int64); ok && e == 1 {
			enabled = append(enabled, moduleName)
		}
	}

	return enabled
}

// OnHook registers a hook handler on the runtime.
func (r *Runtime) OnHook(module, action, phase string, handler HookHandler) {
	r.hooks.OnHook(module, action, phase, handler)
}

// Functions returns the function registry.
func (r *Runtime) Functions() *FunctionRegistry {
	return r.functions
}

// Events returns the event bus.
func (r *Runtime) Events() *events.Bus {
	return r.events
}

// RegisterFunction registers a callable function for "call:" hooks.
func (r *Runtime) RegisterFunction(name string, fn HookHandler) {
	r.functions.Register(name, fn)
}

// RegisterModuleHooks registers all hooks declared in a module's YAML.
// This bridges declarative YAML hooks to runtime hook handlers.
func (r *Runtime) RegisterModuleHooks(mod schema.Module) {
	for hookPhase, hooks := range mod.Hooks {
		// Parse the phase (e.g., "after_create" -> action="create", phase="after")
		action, phase := parseHookPhase(hookPhase)
		if action == "" || phase == "" {
			r.logger.Warn().
				Str("module", mod.Name).
				Str("hook_phase", hookPhase).
				Msg("invalid hook phase format, expected before_* or after_*")
			continue
		}

		for _, hook := range hooks {
			handler := r.createHookHandler(mod.Name, hook)
			if handler != nil {
				r.hooks.OnHook(mod.Name, action, phase, handler)
				r.logger.Debug().
					Str("module", mod.Name).
					Str("action", action).
					Str("phase", phase).
					Msg("registered YAML hook")
			}
		}
	}
}

// parseHookPhase parses "before_create" or "after_update" into action and phase.
func parseHookPhase(hookPhase string) (action, phase string) {
	if strings.HasPrefix(hookPhase, "before_") {
		return strings.TrimPrefix(hookPhase, "before_"), "before"
	}
	if strings.HasPrefix(hookPhase, "after_") {
		return strings.TrimPrefix(hookPhase, "after_"), "after"
	}
	return "", ""
}

// createHookHandler creates a hook handler from a YAML hook definition.
func (r *Runtime) createHookHandler(moduleName string, hook schema.Hook) HookHandler {
	// Handle shorthand "- emit: event.name" format
	if hook.Emit != "" {
		eventName := hook.Emit
		return func(ctx context.Context, event HookEvent) error {
			r.events.Publish(ctx, events.Event{
				Name:   eventName,
				Module: event.Module,
				Action: event.Action,
				Data:   event.Data,
				Meta:   event.Meta,
			})
			return nil
		}
	}

	// Handle shorthand "- call: function_name" format
	if hook.Call != "" {
		return r.createCallHandler(hook.Call)
	}

	// Handle explicit "event:" field (for type: emit)
	if hook.Event != "" {
		eventName := hook.Event
		return func(ctx context.Context, event HookEvent) error {
			r.events.Publish(ctx, events.Event{
				Name:   eventName,
				Module: event.Module,
				Action: event.Action,
				Data:   event.Data,
				Meta:   event.Meta,
			})
			return nil
		}
	}

	// Handle explicit type field for other hook types
	switch hook.Type {
	case "email":
		// Email hooks not yet implemented
		r.logger.Debug().
			Str("module", moduleName).
			Str("template", hook.Template).
			Msg("email hook registered (not yet implemented)")
		return nil

	case "webhook":
		// Webhook hooks not yet implemented
		r.logger.Debug().
			Str("module", moduleName).
			Str("url", hook.URL).
			Msg("webhook hook registered (not yet implemented)")
		return nil
	}

	return nil
}

// createCallHandler creates a handler that invokes a registered function.
func (r *Runtime) createCallHandler(funcName string) HookHandler {
	return func(ctx context.Context, event HookEvent) error {
		if !r.functions.Has(funcName) {
			r.logger.Warn().
				Str("function", funcName).
				Str("module", event.Module).
				Str("action", event.Action).
				Msg("called unregistered function (skipping)")
			return nil // Don't fail the action if function not registered
		}
		return r.functions.Call(ctx, funcName, event)
	}
}

// Start starts all channels and exporters.
func (r *Runtime) Start(ctx context.Context) error {
	// Start channels
	for _, ch := range r.channels {
		if err := ch.Start(ctx); err != nil {
			return fmt.Errorf("start channel %q: %w", ch.Name(), err)
		}
	}

	// Start exporters
	if r.exporters != nil {
		if err := r.exporters.Start(ctx); err != nil {
			return fmt.Errorf("start exporters: %w", err)
		}
	}

	return nil
}

// Stop stops all channels and exporters.
func (r *Runtime) Stop(ctx context.Context) error {
	// Stop exporters first
	if r.exporters != nil {
		if err := r.exporters.Stop(ctx); err != nil {
			return fmt.Errorf("stop exporters: %w", err)
		}
	}

	// Stop channels
	for _, ch := range r.channels {
		if err := ch.Stop(ctx); err != nil {
			return fmt.Errorf("stop channel %q: %w", ch.Name(), err)
		}
	}
	return nil
}

// ResolvedDependency represents a resolved module dependency.
type ResolvedDependency struct {
	// Name is the parameter name from the Requires definition.
	Name string

	// Capability is the capability type that was required.
	Capability string

	// ModuleName is the name of the module that provides this capability.
	ModuleName string

	// InstanceName is the specific instance name (from module settings).
	InstanceName string

	// Required indicates if this dependency was mandatory.
	Required bool
}

// DependencyContext contains resolved dependencies for a module execution.
type DependencyContext struct {
	// Dependencies maps parameter names to resolved providers.
	Dependencies map[string]ResolvedDependency

	// Runtime reference for executing dependent module actions.
	Runtime *Runtime
}

// Execute runs an action on a resolved dependency.
func (dc *DependencyContext) Execute(ctx context.Context, dependencyName, action string, input ActionInput) (ActionResult, error) {
	dep, ok := dc.Dependencies[dependencyName]
	if !ok {
		return ActionResult{}, fmt.Errorf("dependency %q not found in context", dependencyName)
	}
	return dc.Runtime.Execute(ctx, dep.ModuleName, action, input)
}

// ResolveDependencies resolves all requirements for a module.
// It returns a DependencyContext that can be used to execute actions on dependencies.
func (r *Runtime) ResolveDependencies(ctx context.Context, moduleName string) (*DependencyContext, error) {
	r.mu.RLock()
	mod, ok := r.registry.Get(moduleName)
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("module %q not found", moduleName)
	}

	dc := &DependencyContext{
		Dependencies: make(map[string]ResolvedDependency),
		Runtime:      r,
	}

	// If module has no requirements, return empty context
	if len(mod.Source.Meta.Requires) == 0 {
		return dc, nil
	}

	// Resolve each requirement
	for paramName, req := range mod.Source.Meta.Requires {
		resolved, err := r.resolveSingleDependency(ctx, paramName, req)
		if err != nil {
			if req.Required {
				return nil, fmt.Errorf("required dependency %q (%s): %w", paramName, req.Capability, err)
			}
			r.logger.Warn().
				Str("module", moduleName).
				Str("param", paramName).
				Str("capability", req.Capability).
				Err(err).
				Msg("optional dependency not resolved")
			continue
		}
		dc.Dependencies[paramName] = resolved
	}

	return dc, nil
}

// resolveSingleDependency resolves a single module requirement.
func (r *Runtime) resolveSingleDependency(ctx context.Context, paramName string, req schema.ModuleRequirement) (ResolvedDependency, error) {
	// First, try to use the default if specified
	if req.Default != "" {
		// Check if the default module exists and implements the capability
		r.mu.RLock()
		providers := r.capabilities[req.Capability]
		r.mu.RUnlock()

		for _, provider := range providers {
			if provider == req.Default {
				return ResolvedDependency{
					Name:       paramName,
					Capability: req.Capability,
					ModuleName: provider,
					Required:   req.Required,
				}, nil
			}
		}
	}

	// Find any enabled provider for this capability
	provider := r.GetEnabledProvider(ctx, req.Capability)
	if provider != "" {
		return ResolvedDependency{
			Name:       paramName,
			Capability: req.Capability,
			ModuleName: provider,
			Required:   req.Required,
		}, nil
	}

	// Fall back to default if no enabled provider found
	if req.Default != "" {
		return ResolvedDependency{
			Name:       paramName,
			Capability: req.Capability,
			ModuleName: req.Default,
			Required:   req.Required,
		}, nil
	}

	// No provider found
	r.mu.RLock()
	providers := r.capabilities[req.Capability]
	r.mu.RUnlock()

	if len(providers) == 0 {
		return ResolvedDependency{}, fmt.Errorf("no modules implement capability %q", req.Capability)
	}

	// Return first available provider as fallback
	return ResolvedDependency{
		Name:       paramName,
		Capability: req.Capability,
		ModuleName: providers[0],
		Required:   req.Required,
	}, nil
}

// ValidateModuleDependencies checks if all required dependencies for a module can be resolved.
// This should be called when a module is enabled to ensure all dependencies are available.
func (r *Runtime) ValidateModuleDependencies(ctx context.Context, moduleName string) error {
	r.mu.RLock()
	mod, ok := r.registry.Get(moduleName)
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("module %q not found", moduleName)
	}

	if len(mod.Source.Meta.Requires) == 0 {
		return nil
	}

	var missingDeps []string

	for paramName, req := range mod.Source.Meta.Requires {
		if !req.Required {
			continue
		}

		// Check if any module implements this capability
		r.mu.RLock()
		providers := r.capabilities[req.Capability]
		r.mu.RUnlock()

		if len(providers) == 0 {
			missingDeps = append(missingDeps, fmt.Sprintf("%s (%s)", paramName, req.Capability))
		}
	}

	if len(missingDeps) > 0 {
		return fmt.Errorf("missing required dependencies: %s", strings.Join(missingDeps, ", "))
	}

	return nil
}

// GetModuleDependencyInfo returns information about a module's dependencies.
func (r *Runtime) GetModuleDependencyInfo(moduleName string) map[string]DependencyInfo {
	r.mu.RLock()
	mod, ok := r.registry.Get(moduleName)
	r.mu.RUnlock()

	if !ok {
		return nil
	}

	result := make(map[string]DependencyInfo)

	for paramName, req := range mod.Source.Meta.Requires {
		info := DependencyInfo{
			ParamName:   paramName,
			Capability:  req.Capability,
			Required:    req.Required,
			Description: req.Description,
			Default:     req.Default,
		}

		// Find available providers
		r.mu.RLock()
		info.AvailableProviders = r.capabilities[req.Capability]
		r.mu.RUnlock()

		result[paramName] = info
	}

	return result
}

// DependencyInfo describes a module's dependency on a capability.
type DependencyInfo struct {
	ParamName          string
	Capability         string
	Required           bool
	Description        string
	Default            string
	AvailableProviders []string
}
