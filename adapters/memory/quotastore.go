package memory

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// quotaShard is a single shard of the quota store.
type quotaShard struct {
	mu    sync.RWMutex
	state map[string]ports.QuotaState
}

// QuotaStore is a sharded in-memory implementation of ports.QuotaStore.
// Uses sharding to reduce lock contention for high throughput.
type QuotaStore struct {
	shards     []*quotaShard
	numShards  int
	usageStore ports.UsageStore // For syncing with persistent storage
	cleanup    *time.Ticker
	done       chan struct{}
}

// QuotaStoreConfig configures the quota store.
type QuotaStoreConfig struct {
	NumShards       int           // Number of shards (default: 32)
	CleanupInterval time.Duration // How often to clean old periods (default: 1h)
	UsageStore      ports.UsageStore // Optional: for syncing with persistent storage
}

// NewQuotaStore creates a new sharded in-memory quota store.
func NewQuotaStore(cfg QuotaStoreConfig) *QuotaStore {
	if cfg.NumShards <= 0 {
		cfg.NumShards = 32
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = time.Hour
	}

	s := &QuotaStore{
		shards:     make([]*quotaShard, cfg.NumShards),
		numShards:  cfg.NumShards,
		usageStore: cfg.UsageStore,
		done:       make(chan struct{}),
	}

	for i := range s.shards {
		s.shards[i] = &quotaShard{
			state: make(map[string]ports.QuotaState),
		}
	}

	// Start background cleanup
	s.cleanup = time.NewTicker(cfg.CleanupInterval)
	go s.cleanupLoop()

	return s
}

// key generates the map key for a user and period.
func (s *QuotaStore) key(userID string, periodStart time.Time) string {
	return fmt.Sprintf("%s:%s", userID, periodStart.Format("2006-01"))
}

// getShard returns the shard for a given key using consistent hashing.
func (s *QuotaStore) getShard(key string) *quotaShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.shards[h.Sum32()%uint32(s.numShards)]
}

// Get retrieves current quota state for a user's billing period.
func (s *QuotaStore) Get(ctx context.Context, userID string, periodStart time.Time) (ports.QuotaState, error) {
	k := s.key(userID, periodStart)
	shard := s.getShard(k)

	shard.mu.RLock()
	state, ok := shard.state[k]
	shard.mu.RUnlock()

	if ok {
		return state, nil
	}

	// Not in memory, try to load from usage store
	if s.usageStore != nil {
		periodEnd := periodStart.AddDate(0, 1, 0)
		summary, err := s.usageStore.GetSummary(ctx, userID, periodStart, periodEnd)
		if err != nil {
			// Return empty state on error, will be filled on first increment
			return ports.QuotaState{
				UserID:      userID,
				PeriodStart: periodStart,
			}, nil
		}

		state = ports.QuotaState{
			UserID:       userID,
			PeriodStart:  periodStart,
			RequestCount: summary.RequestCount,
			ComputeUnits: summary.ComputeUnits,
			BytesUsed:    summary.BytesIn + summary.BytesOut,
			LastUpdated:  time.Now(),
		}

		// Cache it
		shard.mu.Lock()
		shard.state[k] = state
		shard.mu.Unlock()

		return state, nil
	}

	// Return empty state
	return ports.QuotaState{
		UserID:      userID,
		PeriodStart: periodStart,
	}, nil
}

// Increment atomically adds to quota counters, returns new state.
func (s *QuotaStore) Increment(ctx context.Context, userID string, periodStart time.Time,
	requests int64, computeUnits float64, bytes int64) (ports.QuotaState, error) {

	k := s.key(userID, periodStart)
	shard := s.getShard(k)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	state, ok := shard.state[k]
	if !ok {
		state = ports.QuotaState{
			UserID:      userID,
			PeriodStart: periodStart,
		}
	}

	state.RequestCount += requests
	state.ComputeUnits += computeUnits
	state.BytesUsed += bytes
	state.LastUpdated = time.Now()

	shard.state[k] = state
	return state, nil
}

// Sync reconciles quota state from usage store (background job).
func (s *QuotaStore) Sync(ctx context.Context, userID string, periodStart time.Time, summary usage.Summary) error {
	k := s.key(userID, periodStart)
	shard := s.getShard(k)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	state := ports.QuotaState{
		UserID:       userID,
		PeriodStart:  periodStart,
		RequestCount: summary.RequestCount,
		ComputeUnits: summary.ComputeUnits,
		BytesUsed:    summary.BytesIn + summary.BytesOut,
		LastUpdated:  time.Now(),
	}

	shard.state[k] = state
	return nil
}

// cleanupLoop periodically removes old period entries.
func (s *QuotaStore) cleanupLoop() {
	for {
		select {
		case <-s.cleanup.C:
			s.doCleanup()
		case <-s.done:
			return
		}
	}
}

// doCleanup removes quota states for periods older than 2 months.
func (s *QuotaStore) doCleanup() {
	now := time.Now()
	cutoff := now.AddDate(0, -2, 0)

	for _, shard := range s.shards {
		shard.mu.Lock()
		for k, state := range shard.state {
			if state.PeriodStart.Before(cutoff) {
				delete(shard.state, k)
			}
		}
		shard.mu.Unlock()
	}
}

// Close stops the cleanup goroutine.
func (s *QuotaStore) Close() error {
	close(s.done)
	s.cleanup.Stop()
	return nil
}

// Clear removes all state (for testing).
func (s *QuotaStore) Clear() {
	for _, shard := range s.shards {
		shard.mu.Lock()
		shard.state = make(map[string]ports.QuotaState)
		shard.mu.Unlock()
	}
}

// Len returns the total number of entries across all shards (for testing).
func (s *QuotaStore) Len() int {
	total := 0
	for _, shard := range s.shards {
		shard.mu.RLock()
		total += len(shard.state)
		shard.mu.RUnlock()
	}
	return total
}

// Ensure interface compliance.
var _ ports.QuotaStore = (*QuotaStore)(nil)
