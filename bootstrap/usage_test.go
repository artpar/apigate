package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/usage"
)

// mockUsageStore implements ports.UsageStore for testing.
type mockUsageStore struct {
	mu           sync.Mutex
	batchRecords [][]usage.Event
	recordErr    error
}

func (m *mockUsageStore) RecordBatch(ctx context.Context, events []usage.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	// Make a copy of events to avoid race conditions
	eventsCopy := make([]usage.Event, len(events))
	copy(eventsCopy, events)
	m.batchRecords = append(m.batchRecords, eventsCopy)
	return nil
}

func (m *mockUsageStore) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	return usage.Summary{}, nil
}

func (m *mockUsageStore) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	return nil, nil
}

func (m *mockUsageStore) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	return nil, nil
}

func (m *mockUsageStore) getTotalRecordedEvents() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := 0
	for _, batch := range m.batchRecords {
		total += len(batch)
	}
	return total
}

func TestNewLocalUsageRecorder(t *testing.T) {
	store := &mockUsageStore{}

	recorder := NewLocalUsageRecorder(store, 10, 100*time.Millisecond)
	if recorder == nil {
		t.Fatal("NewLocalUsageRecorder should return a recorder")
	}

	if recorder.batchSize != 10 {
		t.Errorf("batchSize should be 10, got %d", recorder.batchSize)
	}

	if recorder.flushInterval != 100*time.Millisecond {
		t.Errorf("flushInterval should be 100ms, got %v", recorder.flushInterval)
	}

	// Cleanup
	recorder.Close()
}

func TestNewLocalUsageRecorder_Defaults(t *testing.T) {
	store := &mockUsageStore{}

	// Test with 0 values to use defaults
	recorder := NewLocalUsageRecorder(store, 0, 0)
	if recorder == nil {
		t.Fatal("NewLocalUsageRecorder should return a recorder")
	}

	if recorder.batchSize != 100 {
		t.Errorf("default batchSize should be 100, got %d", recorder.batchSize)
	}

	if recorder.flushInterval != 10*time.Second {
		t.Errorf("default flushInterval should be 10s, got %v", recorder.flushInterval)
	}

	// Cleanup
	recorder.Close()
}

func TestLocalUsageRecorder_Record(t *testing.T) {
	store := &mockUsageStore{}
	recorder := NewLocalUsageRecorder(store, 10, 100*time.Millisecond)
	defer recorder.Close()

	// Record an event
	event := usage.Event{
		UserID:     "user1",
		KeyID:      "key1",
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}

	recorder.Record(event)

	// Wait for flush loop to process
	time.Sleep(200 * time.Millisecond)

	// Force flush
	recorder.Flush(context.Background())

	// Wait a bit for async processing
	time.Sleep(50 * time.Millisecond)

	if store.getTotalRecordedEvents() < 1 {
		t.Error("Record should eventually store the event")
	}
}

func TestLocalUsageRecorder_BatchFlush(t *testing.T) {
	store := &mockUsageStore{}
	batchSize := 5
	recorder := NewLocalUsageRecorder(store, batchSize, 10*time.Second)
	defer recorder.Close()

	// Record exactly batchSize events to trigger auto-flush
	for i := 0; i < batchSize; i++ {
		recorder.Record(usage.Event{
			UserID:     "user1",
			KeyID:      "key1",
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	if store.getTotalRecordedEvents() < batchSize {
		t.Errorf("expected at least %d events to be recorded after batch, got %d", batchSize, store.getTotalRecordedEvents())
	}
}

func TestLocalUsageRecorder_Flush(t *testing.T) {
	store := &mockUsageStore{}
	recorder := NewLocalUsageRecorder(store, 100, 10*time.Second)
	defer recorder.Close()

	// Record some events
	for i := 0; i < 3; i++ {
		recorder.Record(usage.Event{
			UserID:     "user1",
			KeyID:      "key1",
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	// Flush manually
	err := recorder.Flush(context.Background())
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	if store.getTotalRecordedEvents() < 3 {
		t.Errorf("expected at least 3 events after flush, got %d", store.getTotalRecordedEvents())
	}
}

func TestLocalUsageRecorder_FlushEmpty(t *testing.T) {
	store := &mockUsageStore{}
	recorder := NewLocalUsageRecorder(store, 100, 10*time.Second)
	defer recorder.Close()

	// Flush with no events
	err := recorder.Flush(context.Background())
	if err != nil {
		t.Errorf("Flush with no events should not error: %v", err)
	}

	// Should have no records
	if store.getTotalRecordedEvents() != 0 {
		t.Errorf("expected 0 events after empty flush, got %d", store.getTotalRecordedEvents())
	}
}

func TestLocalUsageRecorder_Close(t *testing.T) {
	store := &mockUsageStore{}
	recorder := NewLocalUsageRecorder(store, 100, 10*time.Second)

	// Record some events
	for i := 0; i < 5; i++ {
		recorder.Record(usage.Event{
			UserID:     "user1",
			KeyID:      "key1",
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	// Close should flush remaining events
	err := recorder.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Final events should be recorded synchronously
	if store.getTotalRecordedEvents() < 5 {
		t.Errorf("Close should flush all remaining events, got %d", store.getTotalRecordedEvents())
	}
}

func TestLocalUsageRecorder_FlushLoop(t *testing.T) {
	store := &mockUsageStore{}
	// Short flush interval for testing
	recorder := NewLocalUsageRecorder(store, 100, 50*time.Millisecond)
	defer recorder.Close()

	// Record a few events
	for i := 0; i < 3; i++ {
		recorder.Record(usage.Event{
			UserID:     "user1",
			KeyID:      "key1",
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Timestamp:  time.Now(),
		})
	}

	// Wait for flush loop to trigger
	time.Sleep(100 * time.Millisecond)

	if store.getTotalRecordedEvents() < 3 {
		t.Errorf("flush loop should have flushed events, got %d", store.getTotalRecordedEvents())
	}
}

func TestLocalUsageRecorder_ConcurrentRecord(t *testing.T) {
	store := &mockUsageStore{}
	recorder := NewLocalUsageRecorder(store, 100, 10*time.Second)
	defer recorder.Close()

	// Record events concurrently
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				recorder.Record(usage.Event{
					UserID:     "user1",
					KeyID:      "key1",
					Path:       "/api/test",
					Method:     "GET",
					StatusCode: 200,
					Timestamp:  time.Now(),
				})
			}
		}(i)
	}
	wg.Wait()

	// Flush all events
	recorder.Flush(context.Background())

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Close to get final count
	recorder.Close()

	total := store.getTotalRecordedEvents()
	if total < 100 {
		t.Errorf("expected at least 100 events after concurrent recording, got %d", total)
	}
}
