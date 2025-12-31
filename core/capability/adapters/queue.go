package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// In-Memory Queue Implementation
// =============================================================================

// MemoryQueue provides an in-memory queue implementation.
// Suitable for development, testing, and single-instance deployments.
type MemoryQueue struct {
	name   string
	mu     sync.RWMutex
	queues map[string]*queueData
	closed bool
}

type queueData struct {
	jobs     []capability.Job
	inflight map[string]capability.Job // jobs being processed
	cond     *sync.Cond
	mu       sync.Mutex
}

func newQueueData() *queueData {
	q := &queueData{
		jobs:     make([]capability.Job, 0),
		inflight: make(map[string]capability.Job),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// NewMemoryQueue creates a new in-memory queue provider.
func NewMemoryQueue(name string) *MemoryQueue {
	return &MemoryQueue{
		name:   name,
		queues: make(map[string]*queueData),
	}
}

func (q *MemoryQueue) Name() string {
	return q.name
}

func (q *MemoryQueue) getQueue(name string) *queueData {
	q.mu.Lock()
	defer q.mu.Unlock()

	if queue, ok := q.queues[name]; ok {
		return queue
	}
	queue := newQueueData()
	q.queues[name] = queue
	return queue
}

func (q *MemoryQueue) Enqueue(ctx context.Context, queueName string, job capability.Job) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	queue := q.getQueue(queueName)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	// Generate ID if not set
	if job.ID == "" {
		job.ID = generateJobID()
	}

	queue.jobs = append(queue.jobs, job)
	queue.cond.Signal()
	return nil
}

func (q *MemoryQueue) EnqueueDelayed(ctx context.Context, queueName string, job capability.Job, delaySeconds int) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	// For memory queue, we use a simple goroutine to delay
	go func() {
		select {
		case <-time.After(time.Duration(delaySeconds) * time.Second):
			q.Enqueue(context.Background(), queueName, job)
		case <-ctx.Done():
			// Context cancelled, don't enqueue
		}
	}()
	return nil
}

func (q *MemoryQueue) Dequeue(ctx context.Context, queueName string, timeoutSeconds int) (*capability.Job, error) {
	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	queue := q.getQueue(queueName)
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	queue.mu.Lock()
	defer queue.mu.Unlock()

	for len(queue.jobs) == 0 {
		if time.Now().After(deadline) {
			return nil, nil // Timeout, no job available
		}

		// Wait with timeout
		done := make(chan struct{})
		go func() {
			queue.cond.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Got a signal, check for jobs
		case <-ctx.Done():
			queue.cond.Signal() // Wake up the waiting goroutine
			return nil, ctx.Err()
		case <-time.After(time.Until(deadline)):
			queue.cond.Signal() // Wake up the waiting goroutine
			return nil, nil     // Timeout
		}
	}

	// Pop first job
	job := queue.jobs[0]
	queue.jobs = queue.jobs[1:]

	// Track as inflight
	queue.inflight[job.ID] = job

	return &job, nil
}

func (q *MemoryQueue) Ack(ctx context.Context, queueName string, jobID string) error {
	queue := q.getQueue(queueName)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	delete(queue.inflight, jobID)
	return nil
}

func (q *MemoryQueue) Nack(ctx context.Context, queueName string, jobID string) error {
	queue := q.getQueue(queueName)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	if job, ok := queue.inflight[jobID]; ok {
		job.Retries++
		queue.jobs = append(queue.jobs, job)
		delete(queue.inflight, jobID)
		queue.cond.Signal()
	}
	return nil
}

func (q *MemoryQueue) QueueLength(ctx context.Context, queueName string) (int64, error) {
	queue := q.getQueue(queueName)
	queue.mu.Lock()
	defer queue.mu.Unlock()

	return int64(len(queue.jobs)), nil
}

func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true

	// Wake up all waiting dequeuers
	for _, queue := range q.queues {
		queue.cond.Broadcast()
	}

	return nil
}

// Helper to generate unique job IDs
var jobIDCounter int64
var jobIDMu sync.Mutex

func generateJobID() string {
	jobIDMu.Lock()
	defer jobIDMu.Unlock()
	jobIDCounter++
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), jobIDCounter)
}

// Ensure MemoryQueue implements capability.QueueProvider
var _ capability.QueueProvider = (*MemoryQueue)(nil)
