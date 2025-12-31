package capability

import (
	"fmt"
	"sync"
)

// Registry manages registered capability providers.
// It tracks which modules provide which capabilities and their enabled state.
// Thread-safe for concurrent access.
type Registry struct {
	mu sync.RWMutex

	// providers maps instance name -> ProviderInfo
	providers map[string]ProviderInfo

	// byCapability maps capability type -> list of provider names
	byCapability map[string][]string

	// byModule maps module name -> list of provider names
	byModule map[string][]string
}

// NewRegistry creates a new capability registry.
func NewRegistry() *Registry {
	return &Registry{
		providers:    make(map[string]ProviderInfo),
		byCapability: make(map[string][]string),
		byModule:     make(map[string][]string),
	}
}

// Register adds a provider to the registry.
// Returns error if a provider with the same name already exists.
func (r *Registry) Register(info ProviderInfo) error {
	if err := info.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[info.Name]; exists {
		return fmt.Errorf("provider %q already registered", info.Name)
	}

	r.providers[info.Name] = info

	// Index by capability
	capKey := info.CapabilityKey()
	r.byCapability[capKey] = append(r.byCapability[capKey], info.Name)

	// Index by module
	r.byModule[info.Module] = append(r.byModule[info.Module], info.Name)

	return nil
}

// Unregister removes a provider from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %q not found", name)
	}

	delete(r.providers, name)

	// Remove from capability index
	capKey := info.CapabilityKey()
	r.byCapability[capKey] = removeFromSlice(r.byCapability[capKey], name)

	// Remove from module index
	r.byModule[info.Module] = removeFromSlice(r.byModule[info.Module], name)

	return nil
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (ProviderInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.providers[name]
	return info, ok
}

// GetByCapability returns all providers that implement a capability.
func (r *Registry) GetByCapability(cap Type) []ProviderInfo {
	return r.GetByCustomCapability(cap.String())
}

// GetByCustomCapability returns all providers for a capability key (including custom).
func (r *Registry) GetByCustomCapability(capKey string) []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.byCapability[capKey]
	result := make([]ProviderInfo, 0, len(names))
	for _, name := range names {
		if info, ok := r.providers[name]; ok {
			result = append(result, info)
		}
	}
	return result
}

// GetEnabled returns all enabled providers for a capability.
func (r *Registry) GetEnabled(cap Type) []ProviderInfo {
	providers := r.GetByCapability(cap)
	result := make([]ProviderInfo, 0)
	for _, p := range providers {
		if p.Enabled {
			result = append(result, p)
		}
	}
	return result
}

// GetDefault returns the default provider for a capability.
// If no default is set, returns the first enabled provider.
func (r *Registry) GetDefault(cap Type) (ProviderInfo, bool) {
	providers := r.GetByCapability(cap)

	// First, look for explicit default
	for _, p := range providers {
		if p.IsDefault && p.Enabled {
			return p, true
		}
	}

	// Fall back to first enabled
	for _, p := range providers {
		if p.Enabled {
			return p, true
		}
	}

	return ProviderInfo{}, false
}

// SetEnabled updates the enabled state of a provider.
func (r *Registry) SetEnabled(name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("provider %q not found", name)
	}

	info.Enabled = enabled
	r.providers[name] = info
	return nil
}

// SetDefault sets a provider as the default for its capability.
// Clears the default flag from other providers of the same capability.
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("provider %q not found", name)
	}

	capKey := info.CapabilityKey()

	// Clear default from all providers of this capability
	for _, pName := range r.byCapability[capKey] {
		p := r.providers[pName]
		p.IsDefault = false
		r.providers[pName] = p
	}

	// Set new default
	info.IsDefault = true
	r.providers[name] = info

	return nil
}

// GetByModule returns all providers from a specific module.
func (r *Registry) GetByModule(module string) []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.byModule[module]
	result := make([]ProviderInfo, 0, len(names))
	for _, name := range names {
		if info, ok := r.providers[name]; ok {
			result = append(result, info)
		}
	}
	return result
}

// ListCapabilities returns all registered capability types (including custom).
func (r *Registry) ListCapabilities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.byCapability))
	for cap := range r.byCapability {
		result = append(result, cap)
	}
	return result
}

// All returns all registered providers.
func (r *Registry) All() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProviderInfo, 0, len(r.providers))
	for _, info := range r.providers {
		result = append(result, info)
	}
	return result
}

// HasCapability checks if any provider implements a capability.
func (r *Registry) HasCapability(cap Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byCapability[cap.String()]) > 0
}

// Helper to remove an element from a slice
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
