package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/artpar/apigate/ports"
)

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("not found")

// UserStore is an in-memory implementation of ports.UserStore.
type UserStore struct {
	mu    sync.RWMutex
	users map[string]ports.User // by ID
	byEmail map[string]string   // email -> ID
}

// NewUserStore creates a new in-memory user store.
func NewUserStore() *UserStore {
	return &UserStore{
		users:   make(map[string]ports.User),
		byEmail: make(map[string]string),
	}
}

// Get retrieves a user by ID.
func (s *UserStore) Get(ctx context.Context, id string) (ports.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return ports.User{}, ErrNotFound
	}
	return u, nil
}

// GetByEmail retrieves a user by email.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byEmail[email]
	if !ok {
		return ports.User{}, ErrNotFound
	}
	return s.users[id], nil
}

// Create stores a new user.
func (s *UserStore) Create(ctx context.Context, u ports.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate email
	if _, exists := s.byEmail[u.Email]; exists {
		return errors.New("email already exists")
	}

	s.users[u.ID] = u
	s.byEmail[u.Email] = u.ID
	return nil
}

// Update modifies an existing user.
func (s *UserStore) Update(ctx context.Context, u ports.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.users[u.ID]
	if !ok {
		return ErrNotFound
	}

	// Update email index if changed
	if old.Email != u.Email {
		delete(s.byEmail, old.Email)
		s.byEmail[u.Email] = u.ID
	}

	s.users[u.ID] = u
	return nil
}

// List returns users with pagination.
func (s *UserStore) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all users
	all := make([]ports.User, 0, len(s.users))
	for _, u := range s.users {
		all = append(all, u)
	}

	// Apply offset
	if offset >= len(all) {
		return nil, nil
	}
	all = all[offset:]

	// Apply limit
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}

	return all, nil
}

// Count returns total user count.
func (s *UserStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users), nil
}

// Delete removes a user.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return ErrNotFound
	}

	delete(s.byEmail, u.Email)
	delete(s.users, id)
	return nil
}

// GetAll returns all users (for testing).
func (s *UserStore) GetAll() []ports.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ports.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

// Clear removes all users (for testing).
func (s *UserStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]ports.User)
	s.byEmail = make(map[string]string)
}

// Ensure interface compliance.
var _ ports.UserStore = (*UserStore)(nil)
