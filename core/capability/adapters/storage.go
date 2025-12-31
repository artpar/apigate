package adapters

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// In-Memory Storage Implementation
// =============================================================================

// MemoryStorage provides an in-memory storage implementation.
// Suitable for development, testing, and single-instance deployments.
type MemoryStorage struct {
	name string
	mu   sync.RWMutex
	data map[string]storageEntry
}

type storageEntry struct {
	data        []byte
	contentType string
}

// NewMemoryStorage creates a new in-memory storage provider.
func NewMemoryStorage(name string) *MemoryStorage {
	return &MemoryStorage{
		name: name,
		data: make(map[string]storageEntry),
	}
}

func (s *MemoryStorage) Name() string {
	return s.name
}

func (s *MemoryStorage) Put(ctx context.Context, key string, data []byte, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = storageEntry{
		data:        data,
		contentType: contentType,
	}
	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, key string) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[key]
	if !ok {
		return nil, "", fmt.Errorf("key not found: %s", key)
	}
	return entry.data, entry.contentType, nil
}

func (s *MemoryStorage) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

func (s *MemoryStorage) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok, nil
}

func (s *MemoryStorage) List(ctx context.Context, prefix string, limit int) ([]capability.StorageObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []capability.StorageObject
	count := 0
	for key, entry := range s.data {
		if limit > 0 && count >= limit {
			break
		}
		if len(prefix) == 0 || (len(key) >= len(prefix) && key[:len(prefix)] == prefix) {
			result = append(result, capability.StorageObject{
				Key:         key,
				Size:        int64(len(entry.data)),
				ContentType: entry.contentType,
			})
			count++
		}
	}
	return result, nil
}

func (s *MemoryStorage) GetURL(ctx context.Context, key string, expiresIn int) (string, error) {
	// In-memory storage doesn't support signed URLs
	return fmt.Sprintf("memory://%s/%s", s.name, key), nil
}

func (s *MemoryStorage) PutStream(ctx context.Context, key string, reader capability.Reader, contentType string) error {
	data, err := io.ReadAll(reader.(io.Reader))
	if err != nil {
		return err
	}
	return s.Put(ctx, key, data, contentType)
}

// Ensure MemoryStorage implements capability.StorageProvider
var _ capability.StorageProvider = (*MemoryStorage)(nil)
