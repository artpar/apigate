// Package registry manages module registration and conflict detection.
// It ensures modules don't claim conflicting paths and provides
// lookup capabilities for the runtime.
package registry

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// Registry manages registered modules and their path claims.
type Registry struct {
	mu sync.RWMutex

	// modules by name
	modules map[string]convention.Derived

	// paths indexed by type and key
	paths map[schema.PathType]map[string]schema.PathClaim

	// tables to modules
	tables map[string]string
}

// New creates a new registry.
func New() *Registry {
	return &Registry{
		modules: make(map[string]convention.Derived),
		paths:   make(map[schema.PathType]map[string]schema.PathClaim),
		tables:  make(map[string]string),
	}
}

// Register registers a module and its derived paths.
// Returns an error if any conflicts are detected.
func (r *Registry) Register(mod schema.Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate module name
	if _, exists := r.modules[mod.Name]; exists {
		return fmt.Errorf("module %q already registered", mod.Name)
	}

	// Derive the full module
	derived := convention.Derive(mod)

	// Check for table name conflict
	if existing, exists := r.tables[derived.Table]; exists {
		return fmt.Errorf("table %q already claimed by module %q", derived.Table, existing)
	}

	// Check for path conflicts
	conflicts := r.detectConflicts(derived.Paths)
	if len(conflicts) > 0 {
		return &ConflictError{Conflicts: conflicts}
	}

	// Register the module
	r.modules[mod.Name] = derived
	r.tables[derived.Table] = mod.Name

	// Register all paths
	for _, path := range derived.Paths {
		if r.paths[path.Type] == nil {
			r.paths[path.Type] = make(map[string]schema.PathClaim)
		}
		r.paths[path.Type][path.Key()] = path
	}

	return nil
}

// Unregister removes a module from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	derived, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %q not registered", name)
	}

	// Remove paths
	for _, path := range derived.Paths {
		if r.paths[path.Type] != nil {
			delete(r.paths[path.Type], path.Key())
		}
	}

	// Remove table mapping
	delete(r.tables, derived.Table)

	// Remove module
	delete(r.modules, name)

	return nil
}

// Get returns a registered module by name.
func (r *Registry) Get(name string) (convention.Derived, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mod, ok := r.modules[name]
	return mod, ok
}

// List returns all registered modules.
func (r *Registry) List() []convention.Derived {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]convention.Derived, 0, len(r.modules))
	for _, mod := range r.modules {
		modules = append(modules, mod)
	}

	// Sort by name for consistent ordering
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Source.Name < modules[j].Source.Name
	})

	return modules
}

// All returns all registered modules as a map keyed by name.
func (r *Registry) All() map[string]convention.Derived {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]convention.Derived, len(r.modules))
	for name, mod := range r.modules {
		result[name] = mod
	}
	return result
}

// LookupPath finds the module and action for a given path.
func (r *Registry) LookupPath(pathType schema.PathType, method, path string) (module string, action string, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths, exists := r.paths[pathType]
	if !exists {
		return "", "", false
	}

	// Build key based on path type
	var key string
	switch pathType {
	case schema.PathTypeHTTP:
		key = fmt.Sprintf("http:%s:%s", method, schema.NormalizePath(path))
	case schema.PathTypeCLI:
		key = fmt.Sprintf("cli:%s", path)
	default:
		key = fmt.Sprintf("%s:%s", pathType, path)
	}

	claim, ok := paths[key]
	if !ok {
		// Try pattern matching for parameterized paths
		claim, ok = r.matchPath(pathType, key)
		if !ok {
			return "", "", false
		}
	}

	return claim.Module, claim.Action, true
}

// matchPath attempts pattern matching for parameterized paths.
func (r *Registry) matchPath(pathType schema.PathType, key string) (schema.PathClaim, bool) {
	paths := r.paths[pathType]

	for claimKey, claim := range paths {
		if matchPattern(claimKey, key) {
			return claim, true
		}
	}

	return schema.PathClaim{}, false
}

// matchPattern checks if a key matches a pattern with {param} placeholders.
func matchPattern(pattern, key string) bool {
	patternParts := strings.Split(pattern, "/")
	keyParts := strings.Split(key, "/")

	if len(patternParts) != len(keyParts) {
		return false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue // Parameter matches anything
		}
		if part != keyParts[i] {
			return false
		}
	}

	return true
}

// detectConflicts checks for path conflicts without modifying the registry.
func (r *Registry) detectConflicts(newPaths []schema.PathClaim) []schema.PathConflict {
	var conflicts []schema.PathConflict

	for _, path := range newPaths {
		if existing, exists := r.paths[path.Type]; exists {
			if existingClaim, ok := existing[path.Key()]; ok {
				conflicts = append(conflicts, schema.PathConflict{
					Type:   path.Type,
					Path:   path.Path,
					Claims: []schema.PathClaim{existingClaim, path},
				})
			}
		}
	}

	return conflicts
}

// GetHTTPPaths returns all HTTP path claims.
func (r *Registry) GetHTTPPaths() []schema.PathClaim {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]schema.PathClaim, 0)
	if httpPaths, ok := r.paths[schema.PathTypeHTTP]; ok {
		for _, p := range httpPaths {
			paths = append(paths, p)
		}
	}

	// Sort by path for consistent ordering
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Path < paths[j].Path
	})

	return paths
}

// GetCLIPaths returns all CLI path claims.
func (r *Registry) GetCLIPaths() []schema.PathClaim {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]schema.PathClaim, 0)
	if cliPaths, ok := r.paths[schema.PathTypeCLI]; ok {
		for _, p := range cliPaths {
			paths = append(paths, p)
		}
	}

	// Sort by path for consistent ordering
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Path < paths[j].Path
	})

	return paths
}

// ConflictError represents one or more path conflicts.
type ConflictError struct {
	Conflicts []schema.PathConflict
}

// Error returns the conflict error message.
func (e *ConflictError) Error() string {
	var msgs []string
	for _, c := range e.Conflicts {
		msgs = append(msgs, c.Error())
	}
	return fmt.Sprintf("path conflicts detected:\n  - %s", strings.Join(msgs, "\n  - "))
}

// HasConflicts returns true if there are any conflicts.
func (e *ConflictError) HasConflicts() bool {
	return len(e.Conflicts) > 0
}
