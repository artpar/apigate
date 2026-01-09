package sqlite_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/ports"
)

func TestInviteStore_CreateAndGetByTokenHash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	token := "test-token-123"
	hash := sha256.Sum256([]byte(token))

	invite := ports.AdminInvite{
		ID:        "invite-1",
		Email:     "admin@example.com",
		TokenHash: hash[:],
		CreatedBy: "system",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second),
	}

	if err := store.Create(ctx, invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	got, err := store.GetByTokenHash(ctx, hash[:])
	if err != nil {
		t.Fatalf("get by token hash: %v", err)
	}

	if got.ID != invite.ID {
		t.Errorf("ID = %s, want %s", got.ID, invite.ID)
	}
	if got.Email != invite.Email {
		t.Errorf("Email = %s, want %s", got.Email, invite.Email)
	}
	if got.CreatedBy != invite.CreatedBy {
		t.Errorf("CreatedBy = %s, want %s", got.CreatedBy, invite.CreatedBy)
	}
}

func TestInviteStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	// Create multiple invites
	for i := 0; i < 3; i++ {
		token := "token-" + string(rune('a'+i))
		hash := sha256.Sum256([]byte(token))
		invite := ports.AdminInvite{
			ID:        "invite-" + string(rune('1'+i)),
			Email:     "admin" + string(rune('1'+i)) + "@example.com",
			TokenHash: hash[:],
			CreatedBy: "system",
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
			ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		}
		if err := store.Create(ctx, invite); err != nil {
			t.Fatalf("create invite %d: %v", i, err)
		}
	}

	// List with limit
	invites, err := store.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}

	if len(invites) != 2 {
		t.Errorf("got %d invites, want 2", len(invites))
	}
}

func TestInviteStore_MarkUsed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	token := "token-for-use"
	hash := sha256.Sum256([]byte(token))
	invite := ports.AdminInvite{
		ID:        "invite-to-use",
		Email:     "use@example.com",
		TokenHash: hash[:],
		CreatedBy: "system",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
	}

	if err := store.Create(ctx, invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	usedAt := time.Now().UTC()
	if err := store.MarkUsed(ctx, invite.ID, usedAt); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	got, err := store.GetByTokenHash(ctx, hash[:])
	if err != nil {
		t.Fatalf("get after mark used: %v", err)
	}

	if got.UsedAt == nil {
		t.Error("UsedAt should not be nil after MarkUsed")
	}
}

func TestInviteStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	token := "token-to-delete"
	hash := sha256.Sum256([]byte(token))
	invite := ports.AdminInvite{
		ID:        "invite-to-delete",
		Email:     "delete@example.com",
		TokenHash: hash[:],
		CreatedBy: "system",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
	}

	if err := store.Create(ctx, invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	if err := store.Delete(ctx, invite.ID); err != nil {
		t.Fatalf("delete invite: %v", err)
	}

	_, err := store.GetByTokenHash(ctx, hash[:])
	if err == nil {
		t.Error("expected error getting deleted invite")
	}
}

func TestInviteStore_DeleteExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	// Create expired invite
	expiredToken := "expired-token"
	expiredHash := sha256.Sum256([]byte(expiredToken))
	expired := ports.AdminInvite{
		ID:        "invite-expired",
		Email:     "expired@example.com",
		TokenHash: expiredHash[:],
		CreatedBy: "system",
		CreatedAt: time.Now().UTC().Add(-72 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-24 * time.Hour), // expired
	}
	if err := store.Create(ctx, expired); err != nil {
		t.Fatalf("create expired invite: %v", err)
	}

	// Create valid invite
	validToken := "valid-token"
	validHash := sha256.Sum256([]byte(validToken))
	valid := ports.AdminInvite{
		ID:        "invite-valid",
		Email:     "valid@example.com",
		TokenHash: validHash[:],
		CreatedBy: "system",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
	}
	if err := store.Create(ctx, valid); err != nil {
		t.Fatalf("create valid invite: %v", err)
	}

	// Delete expired
	deleted, err := store.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("delete expired: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify expired is gone
	_, err = store.GetByTokenHash(ctx, expiredHash[:])
	if err == nil {
		t.Error("expected error getting expired invite")
	}

	// Verify valid still exists
	_, err = store.GetByTokenHash(ctx, validHash[:])
	if err != nil {
		t.Errorf("valid invite should still exist: %v", err)
	}
}

func TestInviteStore_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	// Initial count should be 0
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	// Add invites
	for i := 0; i < 5; i++ {
		token := "token-count-" + string(rune('a'+i))
		hash := sha256.Sum256([]byte(token))
		invite := ports.AdminInvite{
			ID:        "invite-count-" + string(rune('a'+i)),
			Email:     "count" + string(rune('a'+i)) + "@example.com",
			TokenHash: hash[:],
			CreatedBy: "system",
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		}
		if err := store.Create(ctx, invite); err != nil {
			t.Fatalf("create invite %d: %v", i, err)
		}
	}

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("count after creates: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestInviteStore_GetByTokenHash_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	token := "nonexistent-token"
	hash := sha256.Sum256([]byte(token))

	_, err := store.GetByTokenHash(ctx, hash[:])
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func TestInviteStore_ListWithOffset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInviteStore(db.DB)
	ctx := context.Background()

	// Create 5 invites
	for i := 0; i < 5; i++ {
		token := "token-offset-" + string(rune('a'+i))
		hash := sha256.Sum256([]byte(token))
		invite := ports.AdminInvite{
			ID:        "invite-offset-" + string(rune('a'+i)),
			Email:     "offset" + string(rune('a'+i)) + "@example.com",
			TokenHash: hash[:],
			CreatedBy: "system",
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
			ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
		}
		if err := store.Create(ctx, invite); err != nil {
			t.Fatalf("create invite %d: %v", i, err)
		}
	}

	// List with offset
	invites, err := store.List(ctx, 10, 2)
	if err != nil {
		t.Fatalf("list with offset: %v", err)
	}

	if len(invites) != 3 {
		t.Errorf("got %d invites, want 3 (after offset 2)", len(invites))
	}
}
