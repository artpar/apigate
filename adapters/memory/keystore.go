// Package memory provides in-memory implementations for testing.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
)

// KeyStore is an in-memory implementation of ports.KeyStore.
type KeyStore struct {
	mu   sync.RWMutex
	keys map[string]key.Key // by ID
}

// NewKeyStore creates a new in-memory key store.
func NewKeyStore() *KeyStore {
	return &KeyStore{
		keys: make(map[string]key.Key),
	}
}

// Get retrieves keys matching a prefix.
func (s *KeyStore) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []key.Key
	for _, k := range s.keys {
		if k.Prefix == prefix {
			result = append(result, k)
		}
	}
	return result, nil
}

// Create stores a new key.
func (s *KeyStore) Create(ctx context.Context, k key.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[k.ID] = k
	return nil
}

// Revoke marks a key as revoked.
func (s *KeyStore) Revoke(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if k, ok := s.keys[id]; ok {
		k.RevokedAt = &at
		s.keys[id] = k
	}
	return nil
}

// ListByUser returns all keys for a user.
func (s *KeyStore) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []key.Key
	for _, k := range s.keys {
		if k.UserID == userID {
			result = append(result, k)
		}
	}
	return result, nil
}

// UpdateLastUsed updates the last used timestamp.
func (s *KeyStore) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if k, ok := s.keys[id]; ok {
		k.LastUsed = &at
		s.keys[id] = k
	}
	return nil
}

// GetAll returns all keys (for testing).
func (s *KeyStore) GetAll() []key.Key {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]key.Key, 0, len(s.keys))
	for _, k := range s.keys {
		result = append(result, k)
	}
	return result
}

// Clear removes all keys (for testing).
func (s *KeyStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = make(map[string]key.Key)
}

// Ensure interface compliance.
var _ ports.KeyStore = (*KeyStore)(nil)
