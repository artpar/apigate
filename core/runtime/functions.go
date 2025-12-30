// Package runtime provides the function registry for hook call: directives.
package runtime

import (
	"context"
	"fmt"
	"sync"
)

// FunctionRegistry manages callable functions for hooks.
// Functions are registered by name and invoked via "call:" directives in YAML.
type FunctionRegistry struct {
	mu    sync.RWMutex
	funcs map[string]HookHandler
}

// NewFunctionRegistry creates a new function registry.
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		funcs: make(map[string]HookHandler),
	}
}

// Register adds a function to the registry.
// The function receives the HookEvent with module context and can access/modify data.
func (r *FunctionRegistry) Register(name string, fn HookHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = fn
}

// Call invokes a registered function by name.
// Returns an error if the function is not found.
func (r *FunctionRegistry) Call(ctx context.Context, name string, event HookEvent) error {
	r.mu.RLock()
	fn, ok := r.funcs[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("function %q not registered", name)
	}

	return fn(ctx, event)
}

// Has checks if a function is registered.
func (r *FunctionRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.funcs[name]
	return ok
}

// List returns all registered function names.
func (r *FunctionRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	return names
}
