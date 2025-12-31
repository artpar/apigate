// Package app contains the SettingsService for loading and managing settings.
package app

import (
	"context"
	"sync"

	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// SettingsService provides access to application settings.
type SettingsService struct {
	store  ports.SettingsStore
	logger zerolog.Logger
	mu     sync.RWMutex
	cache  settings.Settings
}

// NewSettingsService creates a new settings service.
func NewSettingsService(store ports.SettingsStore, logger zerolog.Logger) *SettingsService {
	return &SettingsService{
		store:  store,
		logger: logger,
		cache:  settings.Defaults(),
	}
}

// Load loads all settings from the store and merges with defaults.
func (s *SettingsService) Load(ctx context.Context) error {
	loaded, err := s.store.GetAll(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cache = settings.Merge(loaded)
	s.mu.Unlock()

	s.logger.Info().Int("count", len(loaded)).Msg("settings loaded from database")
	return nil
}

// Get returns the current settings (read from cache).
func (s *SettingsService) Get() settings.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent mutation
	result := make(settings.Settings, len(s.cache))
	for k, v := range s.cache {
		result[k] = v
	}
	return result
}

// GetValue returns a single setting value.
func (s *SettingsService) GetValue(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache.Get(key)
}

// Set updates a setting in both cache and store.
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	encrypted := settings.IsSensitive(key)
	if err := s.store.Set(ctx, key, value, encrypted); err != nil {
		return err
	}

	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()

	s.logger.Debug().Str("key", key).Msg("setting updated")
	return nil
}

// SetBatch updates multiple settings.
func (s *SettingsService) SetBatch(ctx context.Context, batch settings.Settings) error {
	if err := s.store.SetBatch(ctx, batch); err != nil {
		return err
	}

	s.mu.Lock()
	for k, v := range batch {
		s.cache[k] = v
	}
	s.mu.Unlock()

	s.logger.Debug().Int("count", len(batch)).Msg("settings batch updated")
	return nil
}

// GetByPrefix returns all settings with a given prefix.
func (s *SettingsService) GetByPrefix(prefix string) settings.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(settings.Settings)
	for k, v := range s.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result[k] = v
		}
	}
	return result
}

// Store returns the underlying settings store for direct access.
func (s *SettingsService) Store() ports.SettingsStore {
	return s.store
}
