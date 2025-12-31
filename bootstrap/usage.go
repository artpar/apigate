package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// LocalUsageRecorder buffers usage events and writes them in batches to the store.
type LocalUsageRecorder struct {
	store         ports.UsageStore
	buffer        []usage.Event
	mu            sync.Mutex
	batchSize     int
	flushInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	closeOnce     sync.Once
}

// NewLocalUsageRecorder creates a new local usage recorder.
func NewLocalUsageRecorder(store ports.UsageStore, batchSize int, flushInterval time.Duration) *LocalUsageRecorder {
	if batchSize == 0 {
		batchSize = 100
	}
	if flushInterval == 0 {
		flushInterval = 10 * time.Second
	}

	r := &LocalUsageRecorder{
		store:         store,
		buffer:        make([]usage.Event, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	r.wg.Add(1)
	go r.flushLoop()

	return r
}

// Record queues a usage event for processing.
func (r *LocalUsageRecorder) Record(e usage.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buffer = append(r.buffer, e)

	if len(r.buffer) >= r.batchSize {
		r.flushLocked(context.Background())
	}
}

// Flush forces immediate processing of queued events.
func (r *LocalUsageRecorder) Flush(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.flushLocked(ctx)
}

func (r *LocalUsageRecorder) flushLocked(ctx context.Context) error {
	if len(r.buffer) == 0 {
		return nil
	}

	events := make([]usage.Event, len(r.buffer))
	copy(events, r.buffer)
	r.buffer = r.buffer[:0]

	// Write in background to not block
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r.store.RecordBatch(ctx, events)
	}()

	return nil
}

func (r *LocalUsageRecorder) flushLoop() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.Flush(context.Background())
		case <-r.stopCh:
			return
		}
	}
}

// Close stops the recorder and flushes remaining events.
func (r *LocalUsageRecorder) Close() error {
	var err error
	r.closeOnce.Do(func() {
		close(r.stopCh)
		r.wg.Wait()

		// Final flush with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		r.mu.Lock()
		defer r.mu.Unlock()

		if len(r.buffer) > 0 {
			err = r.store.RecordBatch(ctx, r.buffer)
		}
	})
	return err
}

// Ensure interface compliance.
var _ ports.UsageRecorder = (*LocalUsageRecorder)(nil)
