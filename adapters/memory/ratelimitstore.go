package memory

import (
	"context"
	"sync"

	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/ports"
)

// RateLimitStore is an in-memory implementation of ports.RateLimitStore.
type RateLimitStore struct {
	mu    sync.RWMutex
	state map[string]ratelimit.WindowState
}

// NewRateLimitStore creates a new in-memory rate limit store.
func NewRateLimitStore() *RateLimitStore {
	return &RateLimitStore{
		state: make(map[string]ratelimit.WindowState),
	}
}

// Get retrieves current rate limit state for a key.
func (s *RateLimitStore) Get(ctx context.Context, keyID string) (ratelimit.WindowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state[keyID], nil
}

// Set updates rate limit state for a key.
func (s *RateLimitStore) Set(ctx context.Context, keyID string, state ratelimit.WindowState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[keyID] = state
	return nil
}

// Clear removes all state (for testing).
func (s *RateLimitStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = make(map[string]ratelimit.WindowState)
}

// Ensure interface compliance.
var _ ports.RateLimitStore = (*RateLimitStore)(nil)
