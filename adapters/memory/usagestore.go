package memory

import (
	"context"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// UsageStore is an in-memory implementation of ports.UsageStore.
type UsageStore struct {
	mu     sync.RWMutex
	events []usage.Event
}

// NewUsageStore creates a new in-memory usage store.
func NewUsageStore() *UsageStore {
	return &UsageStore{
		events: make([]usage.Event, 0),
	}
}

// RecordBatch stores multiple usage events.
func (s *UsageStore) RecordBatch(ctx context.Context, events []usage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, events...)
	return nil
}

// GetSummary returns aggregated usage for a period.
func (s *UsageStore) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []usage.Event
	for _, e := range s.events {
		if e.UserID == userID && !e.Timestamp.Before(start) && !e.Timestamp.After(end) {
			matching = append(matching, e)
		}
	}

	return usage.Aggregate(matching, start, end), nil
}

// GetHistory returns usage summaries for past periods.
func (s *UsageStore) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Group events by month
	byMonth := make(map[string][]usage.Event)
	for _, e := range s.events {
		if e.UserID == userID {
			key := e.Timestamp.Format("2006-01")
			byMonth[key] = append(byMonth[key], e)
		}
	}

	var summaries []usage.Summary
	for _, events := range byMonth {
		if len(events) > 0 {
			start, end := usage.PeriodBounds(events[0].Timestamp)
			summaries = append(summaries, usage.Aggregate(events, start, end))
		}
	}

	// Limit to requested periods
	if periods > 0 && len(summaries) > periods {
		summaries = summaries[:periods]
	}

	return summaries, nil
}

// GetRecentRequests returns recent request logs.
func (s *UsageStore) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []usage.Event
	for i := len(s.events) - 1; i >= 0 && len(matching) < limit; i-- {
		if s.events[i].UserID == userID {
			matching = append(matching, s.events[i])
		}
	}

	return matching, nil
}

// GetAll returns all events (for testing).
func (s *UsageStore) GetAll() []usage.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]usage.Event{}, s.events...)
}

// Drain returns all events and clears the store (for testing).
func (s *UsageStore) Drain() []usage.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.events
	s.events = make([]usage.Event, 0)
	return events
}

// Clear removes all events (for testing).
func (s *UsageStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = make([]usage.Event, 0)
}

// Ensure interface compliance.
var _ ports.UsageStore = (*UsageStore)(nil)
