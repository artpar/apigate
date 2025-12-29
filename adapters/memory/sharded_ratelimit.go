package memory

import (
	"context"
	"hash/fnv"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/ports"
)

// rateLimitShard is a single shard of the rate limit store.
type rateLimitShard struct {
	mu    sync.RWMutex
	state map[string]ratelimit.WindowState
}

// ShardedRateLimitStore is a production-ready sharded in-memory rate limit store.
// Uses sharding to reduce lock contention for high throughput.
type ShardedRateLimitStore struct {
	shards    []*rateLimitShard
	numShards int
	cleanup   *time.Ticker
	done      chan struct{}
}

// ShardedRateLimitConfig configures the sharded rate limit store.
type ShardedRateLimitConfig struct {
	NumShards       int           // Number of shards (default: 32)
	CleanupInterval time.Duration // How often to clean expired windows (default: 5m)
}

// NewShardedRateLimitStore creates a new sharded in-memory rate limit store.
func NewShardedRateLimitStore(cfg ShardedRateLimitConfig) *ShardedRateLimitStore {
	if cfg.NumShards <= 0 {
		cfg.NumShards = 32
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}

	s := &ShardedRateLimitStore{
		shards:    make([]*rateLimitShard, cfg.NumShards),
		numShards: cfg.NumShards,
		done:      make(chan struct{}),
	}

	for i := range s.shards {
		s.shards[i] = &rateLimitShard{
			state: make(map[string]ratelimit.WindowState),
		}
	}

	// Start background cleanup
	s.cleanup = time.NewTicker(cfg.CleanupInterval)
	go s.cleanupLoop()

	return s
}

// getShard returns the shard for a given key using consistent hashing.
func (s *ShardedRateLimitStore) getShard(keyID string) *rateLimitShard {
	h := fnv.New32a()
	h.Write([]byte(keyID))
	return s.shards[h.Sum32()%uint32(s.numShards)]
}

// Get retrieves current rate limit state for a key.
func (s *ShardedRateLimitStore) Get(ctx context.Context, keyID string) (ratelimit.WindowState, error) {
	shard := s.getShard(keyID)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return shard.state[keyID], nil
}

// Set updates rate limit state for a key.
func (s *ShardedRateLimitStore) Set(ctx context.Context, keyID string, state ratelimit.WindowState) error {
	shard := s.getShard(keyID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.state[keyID] = state
	return nil
}

// GetAndCheck atomically gets state, checks rate limit, and updates.
// This is more efficient than separate Get + Check + Set for high throughput.
func (s *ShardedRateLimitStore) GetAndCheck(ctx context.Context, keyID string, cfg ratelimit.Config, now time.Time) (ratelimit.CheckResult, error) {
	shard := s.getShard(keyID)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	state := shard.state[keyID]
	result, newState := ratelimit.Check(state, cfg, now)
	shard.state[keyID] = newState

	return result, nil
}

// cleanupLoop periodically removes expired rate limit entries.
func (s *ShardedRateLimitStore) cleanupLoop() {
	for {
		select {
		case <-s.cleanup.C:
			s.doCleanup()
		case <-s.done:
			return
		}
	}
}

// doCleanup removes rate limit states for expired windows.
func (s *ShardedRateLimitStore) doCleanup() {
	now := time.Now()
	// Remove entries that expired more than 1 hour ago
	cutoff := now.Add(-time.Hour)

	for _, shard := range s.shards {
		shard.mu.Lock()
		for key, state := range shard.state {
			if !state.WindowEnd.IsZero() && state.WindowEnd.Before(cutoff) {
				delete(shard.state, key)
			}
		}
		shard.mu.Unlock()
	}
}

// Close stops the cleanup goroutine.
func (s *ShardedRateLimitStore) Close() error {
	close(s.done)
	s.cleanup.Stop()
	return nil
}

// Clear removes all state (for testing).
func (s *ShardedRateLimitStore) Clear() {
	for _, shard := range s.shards {
		shard.mu.Lock()
		shard.state = make(map[string]ratelimit.WindowState)
		shard.mu.Unlock()
	}
}

// Len returns the total number of entries across all shards (for testing).
func (s *ShardedRateLimitStore) Len() int {
	total := 0
	for _, shard := range s.shards {
		shard.mu.RLock()
		total += len(shard.state)
		shard.mu.RUnlock()
	}
	return total
}

// Ensure interface compliance.
var _ ports.RateLimitStore = (*ShardedRateLimitStore)(nil)
