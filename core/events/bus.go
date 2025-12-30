// Package events provides a simple event bus for publish/subscribe patterns.
// Events are emitted via "emit:" directives in YAML hooks.
package events

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

// Event represents a published event.
type Event struct {
	// Name is the event name (e.g., "user.created", "route.updated").
	Name string

	// Module is the source module that emitted the event.
	Module string

	// Action is the action that triggered the event.
	Action string

	// Data contains the event payload (typically the record data).
	Data map[string]any

	// Meta contains additional metadata.
	Meta map[string]any
}

// Handler is a function that processes an event.
type Handler func(ctx context.Context, event Event) error

// Bus is a simple publish/subscribe event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   zerolog.Logger
}

// NewBus creates a new event bus.
func NewBus(logger zerolog.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

// Subscribe registers a handler for an event.
// The handler will be called whenever the event is published.
// Supports wildcard subscriptions:
//   - "user.created" - exact match
//   - "user.*" - all user events
//   - "*" - all events
func (b *Bus) Subscribe(event string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[event] = append(b.handlers[event], handler)
}

// Publish emits an event to all matching handlers.
// Handlers are called synchronously in registration order.
// If any handler returns an error, publishing continues but errors are logged.
func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Log the event emission
	b.logger.Debug().
		Str("event", event.Name).
		Str("module", event.Module).
		Str("action", event.Action).
		Msg("event emitted")

	// Collect matching handlers
	var matched []Handler

	// Exact match
	if handlers, ok := b.handlers[event.Name]; ok {
		matched = append(matched, handlers...)
	}

	// Module wildcard (e.g., "user.*")
	if len(event.Name) > 0 {
		parts := splitEvent(event.Name)
		if len(parts) >= 1 {
			wildcard := parts[0] + ".*"
			if handlers, ok := b.handlers[wildcard]; ok {
				matched = append(matched, handlers...)
			}
		}
	}

	// Global wildcard
	if handlers, ok := b.handlers["*"]; ok {
		matched = append(matched, handlers...)
	}

	// Call all matched handlers
	for _, handler := range matched {
		if err := handler(ctx, event); err != nil {
			b.logger.Error().
				Err(err).
				Str("event", event.Name).
				Msg("event handler error")
		}
	}
}

// PublishAsync emits an event asynchronously.
// The function returns immediately; handlers run in a goroutine.
func (b *Bus) PublishAsync(ctx context.Context, event Event) {
	go b.Publish(ctx, event)
}

// HasSubscribers checks if any handlers are registered for an event.
func (b *Bus) HasSubscribers(event string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.handlers[event]) > 0 {
		return true
	}

	// Check wildcards
	parts := splitEvent(event)
	if len(parts) >= 1 {
		wildcard := parts[0] + ".*"
		if len(b.handlers[wildcard]) > 0 {
			return true
		}
	}

	return len(b.handlers["*"]) > 0
}

// splitEvent splits an event name by "."
func splitEvent(name string) []string {
	var parts []string
	start := 0
	for i, c := range name {
		if c == '.' {
			parts = append(parts, name[start:i])
			start = i + 1
		}
	}
	if start < len(name) {
		parts = append(parts, name[start:])
	}
	return parts
}
