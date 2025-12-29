// Package formatter provides a pluggable output formatting system.
// Formatters convert structured data to various output formats (table, json, yaml, csv, etc.)
package formatter

import (
	"fmt"
	"io"
	"sync"

	"github.com/artpar/apigate/core/convention"
)

// Formatter converts structured data to a specific output format.
type Formatter interface {
	// Name returns the formatter name (e.g., "table", "json", "yaml").
	Name() string

	// Description returns a human-readable description.
	Description() string

	// FormatList formats a list of records.
	FormatList(w io.Writer, mod convention.Derived, records []map[string]any, opts FormatOptions) error

	// FormatRecord formats a single record.
	FormatRecord(w io.Writer, mod convention.Derived, record map[string]any, opts FormatOptions) error

	// FormatError formats an error.
	FormatError(w io.Writer, err error) error
}

// FormatOptions configures formatting behavior.
type FormatOptions struct {
	// Columns specifies which fields to include (nil = all non-internal).
	Columns []string

	// NoHeader disables header row for tabular formats.
	NoHeader bool

	// Compact minimizes whitespace (for json/yaml).
	Compact bool

	// Color enables ANSI color output.
	Color bool

	// MaxWidth truncates long values (0 = no limit).
	MaxWidth int
}

// Registry manages registered formatters.
type Registry struct {
	mu         sync.RWMutex
	formatters map[string]Formatter
	defaultFmt string
}

// NewRegistry creates a new formatter registry.
func NewRegistry() *Registry {
	return &Registry{
		formatters: make(map[string]Formatter),
		defaultFmt: "table",
	}
}

// Register adds a formatter to the registry.
func (r *Registry) Register(f Formatter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.formatters[f.Name()]; exists {
		return fmt.Errorf("formatter %q already registered", f.Name())
	}

	r.formatters[f.Name()] = f
	return nil
}

// Get returns a formatter by name.
func (r *Registry) Get(name string) (Formatter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.formatters[name]
	return f, ok
}

// Default returns the default formatter.
func (r *Registry) Default() Formatter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.formatters[r.defaultFmt]
	if !ok {
		// Fallback to first available
		for _, fmt := range r.formatters {
			return fmt
		}
		return nil
	}
	return f
}

// SetDefault sets the default formatter.
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.formatters[name]; !exists {
		return fmt.Errorf("formatter %q not registered", name)
	}

	r.defaultFmt = name
	return nil
}

// List returns all registered formatter names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.formatters))
	for name := range r.formatters {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the global formatter registry.
var DefaultRegistry = NewRegistry()

// Register adds a formatter to the default registry.
func Register(f Formatter) error {
	return DefaultRegistry.Register(f)
}

// Get returns a formatter from the default registry.
func Get(name string) (Formatter, bool) {
	return DefaultRegistry.Get(name)
}

// Default returns the default formatter from the default registry.
func Default() Formatter {
	return DefaultRegistry.Default()
}

// List returns all formatter names from the default registry.
func List() []string {
	return DefaultRegistry.List()
}
