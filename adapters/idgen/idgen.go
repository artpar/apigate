// Package idgen provides ID generation implementations.
package idgen

import (
	"sync/atomic"

	"github.com/artpar/apigate/ports"
	"github.com/google/uuid"
)

// UUID generates UUIDs.
type UUID struct{}

// New generates a new UUID v4.
func (UUID) New() string {
	return uuid.New().String()
}

// Ensure interface compliance.
var _ ports.IDGenerator = UUID{}

// Sequential generates sequential IDs (for testing).
type Sequential struct {
	prefix  string
	counter uint64
}

// NewSequential creates a sequential ID generator.
func NewSequential(prefix string) *Sequential {
	return &Sequential{prefix: prefix}
}

// New generates the next sequential ID.
func (s *Sequential) New() string {
	n := atomic.AddUint64(&s.counter, 1)
	return s.prefix + uitoa(n)
}

// Reset resets the counter (for testing).
func (s *Sequential) Reset() {
	atomic.StoreUint64(&s.counter, 0)
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Ensure interface compliance.
var _ ports.IDGenerator = (*Sequential)(nil)
