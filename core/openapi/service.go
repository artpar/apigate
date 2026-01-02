// Package openapi provides OpenAPI 3.0 specification generation.
// This file provides a unified service that merges module and route specs.
package openapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// Service provides unified OpenAPI specification generation with caching.
type Service struct {
	routeStore    ports.RouteStore
	upstreamStore ports.UpstreamStore
	moduleGetter  func() map[string]convention.Derived
	appName       string
	logger        zerolog.Logger

	cache atomic.Pointer[cachedSpec]
	mu    sync.Mutex // Protects cache generation
}

// cachedSpec holds a cached OpenAPI spec with metadata.
type cachedSpec struct {
	spec        *Spec
	generatedAt time.Time
	dataHash    string // Hash of routes + upstreams for change detection
}

// ServiceConfig contains configuration for the OpenAPI service.
type ServiceConfig struct {
	RouteStore    ports.RouteStore
	UpstreamStore ports.UpstreamStore
	ModuleGetter  func() map[string]convention.Derived
	AppName       string
	Logger        zerolog.Logger
}

// NewService creates a new OpenAPI service.
func NewService(cfg ServiceConfig) *Service {
	appName := cfg.AppName
	if appName == "" {
		appName = "APIGate"
	}

	return &Service{
		routeStore:    cfg.RouteStore,
		upstreamStore: cfg.UpstreamStore,
		moduleGetter:  cfg.ModuleGetter,
		appName:       appName,
		logger:        cfg.Logger,
	}
}

// GetUnifiedSpec returns a unified OpenAPI spec containing both module and route APIs.
// The spec is cached and automatically invalidated when data changes.
func (s *Service) GetUnifiedSpec(ctx context.Context, baseURL string) *Spec {
	// Check cache validity
	cached := s.cache.Load()
	if cached != nil && time.Since(cached.generatedAt) < 30*time.Second {
		// Return cached spec with updated server URL
		spec := s.cloneSpecWithServer(cached.spec, baseURL)
		return spec
	}

	// Generate new spec (with mutex to prevent thundering herd)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring lock
	cached = s.cache.Load()
	if cached != nil && time.Since(cached.generatedAt) < 30*time.Second {
		spec := s.cloneSpecWithServer(cached.spec, baseURL)
		return spec
	}

	// Load data
	routes, upstreams := s.loadData(ctx)

	// Compute hash for change detection
	dataHash := s.computeDataHash(routes, upstreams)

	// Check if data hasn't changed (just refresh timestamp)
	if cached != nil && cached.dataHash == dataHash {
		newCached := &cachedSpec{
			spec:        cached.spec,
			generatedAt: time.Now(),
			dataHash:    dataHash,
		}
		s.cache.Store(newCached)
		return s.cloneSpecWithServer(cached.spec, baseURL)
	}

	// Generate new spec
	spec := s.generateSpec(routes, upstreams, baseURL)

	// Cache it
	s.cache.Store(&cachedSpec{
		spec:        spec,
		generatedAt: time.Now(),
		dataHash:    dataHash,
	})

	return spec
}

// InvalidateCache forces the next GetUnifiedSpec call to regenerate the spec.
func (s *Service) InvalidateCache() {
	s.cache.Store(nil)
	s.logger.Debug().Msg("OpenAPI cache invalidated")
}

// loadData loads routes and upstreams from stores.
func (s *Service) loadData(ctx context.Context) ([]route.Route, map[string]route.Upstream) {
	routes, err := s.routeStore.List(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load routes for OpenAPI spec")
		routes = nil
	}

	upstreamsList, err := s.upstreamStore.List(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load upstreams for OpenAPI spec")
		upstreamsList = nil
	}

	// Convert to map for easy lookup
	upstreams := make(map[string]route.Upstream, len(upstreamsList))
	for _, u := range upstreamsList {
		upstreams[u.ID] = u
	}

	return routes, upstreams
}

// computeDataHash creates a hash of routes and upstreams for change detection.
func (s *Service) computeDataHash(routes []route.Route, upstreams map[string]route.Upstream) string {
	hasher := sha256.New()

	// Sort routes for consistent hashing
	sortedRoutes := make([]route.Route, len(routes))
	copy(sortedRoutes, routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].ID < sortedRoutes[j].ID
	})

	for _, r := range sortedRoutes {
		hasher.Write([]byte(r.ID))
		hasher.Write([]byte(r.PathPattern))
		hasher.Write([]byte(r.Description))
		if r.Enabled {
			hasher.Write([]byte("1"))
		} else {
			hasher.Write([]byte("0"))
		}
	}

	// Sort upstreams for consistent hashing
	var upstreamIDs []string
	for id := range upstreams {
		upstreamIDs = append(upstreamIDs, id)
	}
	sort.Strings(upstreamIDs)

	for _, id := range upstreamIDs {
		u := upstreams[id]
		hasher.Write([]byte(u.ID))
		hasher.Write([]byte(u.Name))
		hasher.Write([]byte(string(u.AuthType)))
		if u.Enabled {
			hasher.Write([]byte("1"))
		} else {
			hasher.Write([]byte("0"))
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// generateSpec creates the unified OpenAPI spec.
func (s *Service) generateSpec(routes []route.Route, upstreams map[string]route.Upstream, baseURL string) *Spec {
	// Create unified spec
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       s.appName + " API",
			Description: "Complete API documentation for " + s.appName + ", including management APIs and proxied routes.",
			Version:     "1.0.0",
			Contact: &Contact{
				Name: s.appName + " Support",
			},
			License: &License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []Server{
			{URL: baseURL, Description: "Current server"},
		},
		Paths: make(map[string]PathItem),
		Components: Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
		Tags: make([]Tag, 0),
	}

	// Add default security schemes
	spec.Components.SecuritySchemes["apiKey"] = SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        "X-API-Key",
		Description: "API key for authenticating requests",
	}
	spec.Components.SecuritySchemes["bearerAuth"] = SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT authentication for admin endpoints",
	}

	// Generate module spec if available
	if s.moduleGetter != nil {
		modules := s.moduleGetter()
		if len(modules) > 0 {
			moduleGen := NewGenerator(modules)
			moduleSpec := moduleGen.Generate()

			// Merge module paths
			for path, item := range moduleSpec.Paths {
				spec.Paths[path] = item
			}

			// Merge module schemas
			for name, schema := range moduleSpec.Components.Schemas {
				spec.Components.Schemas[name] = schema
			}

			// Merge module tags
			spec.Tags = append(spec.Tags, moduleSpec.Tags...)
		}
	}

	// Generate route spec
	if len(routes) > 0 {
		routeGen := NewRouteGenerator(routes, upstreams)
		routeSpec := routeGen.Generate()

		// Merge route paths
		for path, item := range routeSpec.Paths {
			// If path already exists, merge operations
			if existing, ok := spec.Paths[path]; ok {
				if item.Get != nil && existing.Get == nil {
					existing.Get = item.Get
				}
				if item.Post != nil && existing.Post == nil {
					existing.Post = item.Post
				}
				if item.Put != nil && existing.Put == nil {
					existing.Put = item.Put
				}
				if item.Patch != nil && existing.Patch == nil {
					existing.Patch = item.Patch
				}
				if item.Delete != nil && existing.Delete == nil {
					existing.Delete = item.Delete
				}
				spec.Paths[path] = existing
			} else {
				spec.Paths[path] = item
			}
		}

		// Merge route security schemes
		for name, scheme := range routeSpec.Components.SecuritySchemes {
			if _, exists := spec.Components.SecuritySchemes[name]; !exists {
				spec.Components.SecuritySchemes[name] = scheme
			}
		}

		// Merge route tags (add upstream-based tags)
		existingTags := make(map[string]bool)
		for _, tag := range spec.Tags {
			existingTags[tag.Name] = true
		}
		for _, tag := range routeSpec.Tags {
			if !existingTags[tag.Name] {
				spec.Tags = append(spec.Tags, tag)
				existingTags[tag.Name] = true
			}
		}

		// Add tags for each upstream
		for _, u := range upstreams {
			if u.Enabled && !existingTags[u.Name] {
				spec.Tags = append(spec.Tags, Tag{
					Name:        u.Name,
					Description: u.Description,
				})
				existingTags[u.Name] = true
			}
		}
	}

	// Sort tags alphabetically
	sort.Slice(spec.Tags, func(i, j int) bool {
		return spec.Tags[i].Name < spec.Tags[j].Name
	})

	return spec
}

// cloneSpecWithServer creates a copy of the spec with the given server URL.
func (s *Service) cloneSpecWithServer(spec *Spec, baseURL string) *Spec {
	// Deep clone using JSON (simple but effective)
	data, err := json.Marshal(spec)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to clone OpenAPI spec")
		return spec
	}

	var cloned Spec
	if err := json.Unmarshal(data, &cloned); err != nil {
		s.logger.Error().Err(err).Msg("Failed to unmarshal cloned OpenAPI spec")
		return spec
	}

	// Update server URL
	if len(cloned.Servers) > 0 {
		cloned.Servers[0].URL = baseURL
	} else {
		cloned.Servers = []Server{{URL: baseURL, Description: "Current server"}}
	}

	return &cloned
}
