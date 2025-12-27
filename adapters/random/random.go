// Package random provides Random implementations.
package random

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Real uses crypto/rand for secure randomness.
type Real struct{}

// Bytes generates n cryptographically secure random bytes.
func (Real) Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// String generates a random hex string of n characters.
func (r Real) String(n int) (string, error) {
	// We need n/2 bytes to get n hex chars
	bytes := (n + 1) / 2
	b, err := r.Bytes(bytes)
	if err != nil {
		return "", err
	}
	s := hex.EncodeToString(b)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}

// Fake provides deterministic randomness for testing.
type Fake struct {
	mu      sync.Mutex
	counter int
	values  [][]byte // Preset values to return
	index   int
}

// NewFake creates a fake random source.
func NewFake() *Fake {
	return &Fake{}
}

// WithValues sets preset byte values to return.
func (f *Fake) WithValues(values ...[]byte) *Fake {
	f.values = values
	f.index = 0
	return f
}

// Bytes returns preset bytes or deterministic bytes based on counter.
func (f *Fake) Bytes(n int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Return preset value if available
	if f.index < len(f.values) {
		v := f.values[f.index]
		f.index++
		if len(v) >= n {
			return v[:n], nil
		}
		// Pad if needed
		result := make([]byte, n)
		copy(result, v)
		return result, nil
	}

	// Generate deterministic bytes
	f.counter++
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte((f.counter + i) % 256)
	}
	return b, nil
}

// String returns a deterministic hex string.
func (f *Fake) String(n int) (string, error) {
	bytes := (n + 1) / 2
	b, err := f.Bytes(bytes)
	if err != nil {
		return "", err
	}
	s := hex.EncodeToString(b)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}

// Reset resets the fake to initial state.
func (f *Fake) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counter = 0
	f.index = 0
}
