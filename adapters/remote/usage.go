package remote

import (
	"context"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// UsageRecorder sends usage events to an external HTTP service.
// This enables customers to use their own analytics/metering system.
//
// API Contract:
//
//	POST /usage/events
//	Request:  {"events": [...]}
//	Response: {"received": 5}
//
//	POST /usage/event
//	Request:  {"event": {...}}
//	Response: {}
type UsageRecorder struct {
	client    *Client
	buffer    []usage.Event
	mu        sync.Mutex
	batchSize int
	flushInterval time.Duration
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// UsageRecorderConfig configures the usage recorder.
type UsageRecorderConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// NewUsageRecorder creates a remote usage recorder.
func NewUsageRecorder(client *Client, cfg UsageRecorderConfig) *UsageRecorder {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = 10 * time.Second
	}

	r := &UsageRecorder{
		client:    client,
		buffer:    make([]usage.Event, 0, cfg.BatchSize),
		batchSize: cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		stopCh:    make(chan struct{}),
	}

	r.wg.Add(1)
	go r.flushLoop()

	return r
}

// RemoteUsageEvent is the wire format for usage events.
type RemoteUsageEvent struct {
	ID             string    `json:"id"`
	KeyID          string    `json:"key_id"`
	UserID         string    `json:"user_id"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	StatusCode     int       `json:"status_code"`
	LatencyMs      int64     `json:"latency_ms"`
	RequestBytes   int64     `json:"request_bytes"`
	ResponseBytes  int64     `json:"response_bytes"`
	CostMultiplier float64   `json:"cost_multiplier"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// Record queues a usage event for processing.
func (r *UsageRecorder) Record(e usage.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buffer = append(r.buffer, e)

	if len(r.buffer) >= r.batchSize {
		r.flushLocked(context.Background())
	}
}

// Flush forces immediate processing of queued events.
func (r *UsageRecorder) Flush(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.flushLocked(ctx)
}

func (r *UsageRecorder) flushLocked(ctx context.Context) error {
	if len(r.buffer) == 0 {
		return nil
	}

	events := make([]RemoteUsageEvent, len(r.buffer))
	for i, e := range r.buffer {
		events[i] = RemoteUsageEvent{
			ID:             e.ID,
			KeyID:          e.KeyID,
			UserID:         e.UserID,
			Method:         e.Method,
			Path:           e.Path,
			StatusCode:     e.StatusCode,
			LatencyMs:      e.LatencyMs,
			RequestBytes:   e.RequestBytes,
			ResponseBytes:  e.ResponseBytes,
			CostMultiplier: e.CostMultiplier,
			IPAddress:      e.IPAddress,
			UserAgent:      e.UserAgent,
			Timestamp:      e.Timestamp,
		}
	}

	req := map[string]interface{}{
		"events": events,
	}

	err := r.client.Request(ctx, "POST", "/usage/events", req, nil)
	if err != nil {
		// Keep events in buffer for retry
		return err
	}

	r.buffer = r.buffer[:0]
	return nil
}

func (r *UsageRecorder) flushLoop() {
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
func (r *UsageRecorder) Close() error {
	close(r.stopCh)
	r.wg.Wait()
	return r.Flush(context.Background())
}

// Ensure interface compliance.
var _ ports.UsageRecorder = (*UsageRecorder)(nil)
