package adapters

import (
	"context"
	"sync"
	"time"

	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/ports"
)

// CacheAdapter wraps a ports.CacheProvider to implement capability.CacheProvider.
type CacheAdapter struct {
	inner ports.CacheProvider
}

// WrapCache creates a capability.CacheProvider from a ports.CacheProvider.
func WrapCache(inner ports.CacheProvider) *CacheAdapter {
	return &CacheAdapter{inner: inner}
}

func (a *CacheAdapter) Name() string {
	return a.inner.Name()
}

func (a *CacheAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	return a.inner.Get(ctx, key)
}

func (a *CacheAdapter) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	return a.inner.Set(ctx, key, value, ttlSeconds)
}

func (a *CacheAdapter) Delete(ctx context.Context, key string) error {
	return a.inner.Delete(ctx, key)
}

func (a *CacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.inner.Exists(ctx, key)
}

func (a *CacheAdapter) Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error) {
	return a.inner.Increment(ctx, key, delta, ttlSeconds)
}

func (a *CacheAdapter) Flush(ctx context.Context) error {
	return a.inner.Flush(ctx)
}

func (a *CacheAdapter) Close() error {
	return a.inner.Close()
}

// Ensure CacheAdapter implements capability.CacheProvider
var _ capability.CacheProvider = (*CacheAdapter)(nil)

// =============================================================================
// In-Memory Cache Implementation
// =============================================================================

// MemoryCache provides an in-memory cache implementation.
// Suitable for development, testing, and single-instance deployments.
type MemoryCache struct {
	name string
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(name string) *MemoryCache {
	return &MemoryCache{
		name: name,
		data: make(map[string]cacheEntry),
	}
}

func (c *MemoryCache) Name() string {
	return c.name
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, nil
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	return entry.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cacheEntry{value: value}
	if ttlSeconds > 0 {
		entry.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	c.data[key] = entry
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return false, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return false, nil
	}
	return true, nil
}

func (c *MemoryCache) Increment(ctx context.Context, key string, delta int64, ttlSeconds int) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var current int64
	if entry, ok := c.data[key]; ok {
		if entry.expiresAt.IsZero() || time.Now().Before(entry.expiresAt) {
			// Parse current value
			for _, b := range entry.value {
				current = current*10 + int64(b-'0')
			}
		}
	}

	newVal := current + delta
	entry := cacheEntry{value: []byte(intToString(newVal))}
	if ttlSeconds > 0 {
		entry.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	c.data[key] = entry

	return newVal, nil
}

func (c *MemoryCache) Flush(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
	return nil
}

func (c *MemoryCache) Close() error {
	return nil
}

// intToString converts int64 to string without importing strconv
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Ensure MemoryCache implements capability.CacheProvider
var _ capability.CacheProvider = (*MemoryCache)(nil)
