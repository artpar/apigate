// Package clock provides Clock implementations.
package clock

import (
	"sync"
	"time"
)

// Real returns the actual current time.
type Real struct{}

// Now returns the current time.
func (Real) Now() time.Time {
	return time.Now()
}

// Fake provides a controllable clock for testing.
type Fake struct {
	mu      sync.RWMutex
	current time.Time
}

// NewFake creates a fake clock set to the given time.
func NewFake(t time.Time) *Fake {
	return &Fake{current: t}
}

// Now returns the fake current time.
func (f *Fake) Now() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// Set sets the fake current time.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = t
}

// Advance moves the fake time forward by duration d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = f.current.Add(d)
}
