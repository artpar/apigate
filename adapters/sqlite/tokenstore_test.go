package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestTokenStore_CreateAndGetByHash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	// Generate a token
	rawToken := "test-token-value"
	hash, _ := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.DefaultCost)

	token := auth.Token{
		ID:        "tok_test123",
		UserID:    "user_123",
		Email:     "test@example.com",
		Type:      auth.TokenTypeEmailVerification,
		Hash:      hash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	// Create
	err := store.Create(ctx, token)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by hash
	retrieved, err := store.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}

	if retrieved.ID != token.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, token.ID)
	}
	if retrieved.UserID != token.UserID {
		t.Errorf("UserID = %s, want %s", retrieved.UserID, token.UserID)
	}
	if retrieved.Email != token.Email {
		t.Errorf("Email = %s, want %s", retrieved.Email, token.Email)
	}
	if retrieved.Type != token.Type {
		t.Errorf("Type = %s, want %s", retrieved.Type, token.Type)
	}
	if retrieved.UsedAt != nil {
		t.Error("UsedAt should be nil")
	}
}

func TestTokenStore_GetByUserAndType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	userID := "user_456"
	hash1, _ := bcrypt.GenerateFromPassword([]byte("token1"), bcrypt.DefaultCost)
	hash2, _ := bcrypt.GenerateFromPassword([]byte("token2"), bcrypt.DefaultCost)

	// Create first token
	token1 := auth.Token{
		ID:        "tok_first",
		UserID:    userID,
		Email:     "test@example.com",
		Type:      auth.TokenTypePasswordReset,
		Hash:      hash1,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now().Add(-time.Minute), // older
	}
	if err := store.Create(ctx, token1); err != nil {
		t.Fatalf("Create token1 failed: %v", err)
	}

	// Create second token (newer)
	token2 := auth.Token{
		ID:        "tok_second",
		UserID:    userID,
		Email:     "test@example.com",
		Type:      auth.TokenTypePasswordReset,
		Hash:      hash2,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	if err := store.Create(ctx, token2); err != nil {
		t.Fatalf("Create token2 failed: %v", err)
	}

	// Should get the latest token
	retrieved, err := store.GetByUserAndType(ctx, userID, auth.TokenTypePasswordReset)
	if err != nil {
		t.Fatalf("GetByUserAndType failed: %v", err)
	}

	if retrieved.ID != "tok_second" {
		t.Errorf("Should get latest token, got %s", retrieved.ID)
	}
}

func TestTokenStore_MarkUsed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("token"), bcrypt.DefaultCost)
	token := auth.Token{
		ID:        "tok_markused",
		UserID:    "user_789",
		Email:     "test@example.com",
		Type:      auth.TokenTypeEmailVerification,
		Hash:      hash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, token); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Mark as used
	usedAt := time.Now()
	if err := store.MarkUsed(ctx, token.ID, usedAt); err != nil {
		t.Fatalf("MarkUsed failed: %v", err)
	}

	// Verify
	retrieved, err := store.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}

	if retrieved.UsedAt == nil {
		t.Error("UsedAt should not be nil")
	}
}

func TestTokenStore_DeleteExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	hash1, _ := bcrypt.GenerateFromPassword([]byte("expired"), bcrypt.DefaultCost)
	hash2, _ := bcrypt.GenerateFromPassword([]byte("valid"), bcrypt.DefaultCost)

	// Create expired token (use UTC to match store comparison)
	expired := auth.Token{
		ID:        "tok_expired",
		UserID:    "user_exp",
		Email:     "expired@example.com",
		Type:      auth.TokenTypeEmailVerification,
		Hash:      hash1,
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // already expired
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := store.Create(ctx, expired); err != nil {
		t.Fatalf("Create expired failed: %v", err)
	}

	// Create valid token (use UTC to match store comparison)
	valid := auth.Token{
		ID:        "tok_valid",
		UserID:    "user_val",
		Email:     "valid@example.com",
		Type:      auth.TokenTypeEmailVerification,
		Hash:      hash2,
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
	_, err = store.GetByHash(ctx, hash1)
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound for expired token, got %v", err)
	}

	// Verify valid still exists
	_, err = store.GetByHash(ctx, hash2)
	if err != nil {
		t.Errorf("Valid token should still exist: %v", err)
	}
}

func TestTokenStore_DeleteByUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	userID := "user_delete"
	hash1, _ := bcrypt.GenerateFromPassword([]byte("t1"), bcrypt.DefaultCost)
	hash2, _ := bcrypt.GenerateFromPassword([]byte("t2"), bcrypt.DefaultCost)

	// Create two tokens for the user
	for i, h := range [][]byte{hash1, hash2} {
		token := auth.Token{
			ID:        "tok_del_" + string(rune('a'+i)),
			UserID:    userID,
			Email:     "delete@example.com",
			Type:      auth.TokenTypeEmailVerification,
			Hash:      h,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}
		if err := store.Create(ctx, token); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Delete by user
	if err := store.DeleteByUser(ctx, userID); err != nil {
		t.Fatalf("DeleteByUser failed: %v", err)
	}

	// Verify both are gone
	_, err := store.GetByHash(ctx, hash1)
	if err != sqlite.ErrNotFound {
		t.Error("First token should be deleted")
	}

	_, err = store.GetByHash(ctx, hash2)
	if err != sqlite.ErrNotFound {
		t.Error("Second token should be deleted")
	}
}

func TestTokenStore_GetByHash_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	_, err := store.GetByHash(ctx, []byte("nonexistent"))
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestTokenStore_MarkUsed_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewTokenStore(db)
	ctx := context.Background()

	err := store.MarkUsed(ctx, "nonexistent", time.Now())
	if err != sqlite.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}
