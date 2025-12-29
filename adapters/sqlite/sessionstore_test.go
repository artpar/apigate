package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/auth"
)

func TestSessionStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	session := auth.Session{
		ID:        "sess_test123",
		UserID:    "user_123",
		Email:     "test@example.com",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Create
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, session.ID)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("UserID = %s, want %s", retrieved.UserID, session.UserID)
	}
	if retrieved.Email != session.Email {
		t.Errorf("Email = %s, want %s", retrieved.Email, session.Email)
	}
	if retrieved.IPAddress != session.IPAddress {
		t.Errorf("IPAddress = %s, want %s", retrieved.IPAddress, session.IPAddress)
	}
	if retrieved.UserAgent != session.UserAgent {
		t.Errorf("UserAgent = %s, want %s", retrieved.UserAgent, session.UserAgent)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	session := auth.Session{
		ID:        "sess_delete",
		UserID:    "user_del",
		Email:     "delete@example.com",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete
	if err := store.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := store.Get(ctx, session.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestSessionStore_DeleteByUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	userID := "user_multi_session"

	// Create multiple sessions for the same user
	for i := 0; i < 3; i++ {
		session := auth.Session{
			ID:        "sess_multi_" + string(rune('a'+i)),
			UserID:    userID,
			Email:     "multi@example.com",
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Delete by user
	if err := store.DeleteByUser(ctx, userID); err != nil {
		t.Fatalf("DeleteByUser failed: %v", err)
	}

	// Verify all are gone
	for i := 0; i < 3; i++ {
		_, err := store.Get(ctx, "sess_multi_"+string(rune('a'+i)))
		if err != sqlite.ErrNotFound {
			t.Errorf("Session %d should be deleted", i)
		}
	}
}

func TestSessionStore_DeleteExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	// Create expired session (use UTC to match store comparison)
	expired := auth.Session{
		ID:        "sess_expired",
		UserID:    "user_exp",
		Email:     "expired@example.com",
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // already expired
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := store.Create(ctx, expired); err != nil {
		t.Fatalf("Create expired failed: %v", err)
	}

	// Create valid session (use UTC to match store comparison)
	valid := auth.Session{
		ID:        "sess_valid",
		UserID:    "user_val",
		Email:     "valid@example.com",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, valid); err != nil {
		t.Fatalf("Create valid failed: %v", err)
	}

	// Delete expired
	deleted, err := store.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Deleted = %d, want 1", deleted)
	}

	// Verify expired is gone
	_, err = store.Get(ctx, expired.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound for expired session, got %v", err)
	}

	// Verify valid still exists
	_, err = store.Get(ctx, valid.ID)
	if err != nil {
		t.Errorf("Valid session should still exist: %v", err)
	}
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestSessionStore_Delete_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestSessionStore_NullableFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSessionStore(db)
	ctx := context.Background()

	// Create session without optional fields
	session := auth.Session{
		ID:        "sess_nullable",
		UserID:    "user_null",
		Email:     "null@example.com",
		IPAddress: "",
		UserAgent: "",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get and verify empty strings
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.IPAddress != "" {
		t.Errorf("IPAddress should be empty, got %s", retrieved.IPAddress)
	}
	if retrieved.UserAgent != "" {
		t.Errorf("UserAgent should be empty, got %s", retrieved.UserAgent)
	}
}
