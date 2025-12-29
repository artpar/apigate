package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/ports"
)

// KeyStore tests

func TestKeyStore_Create(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	k := key.Key{
		ID:     "key1",
		UserID: "user1",
		Prefix: "ak_abc123",
	}

	err := store.Create(ctx, k)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	keys, _ := store.ListByUser(ctx, "user1")
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
}

func TestKeyStore_Get(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", Prefix: "prefix1"})
	store.Create(ctx, key.Key{ID: "k2", Prefix: "prefix1"})
	store.Create(ctx, key.Key{ID: "k3", Prefix: "prefix2"})

	keys, err := store.Get(ctx, "prefix1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys with prefix1, got %d", len(keys))
	}
}

func TestKeyStore_ListByUser(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})
	store.Create(ctx, key.Key{ID: "k2", UserID: "user1"})
	store.Create(ctx, key.Key{ID: "k3", UserID: "user2"})

	keys, _ := store.ListByUser(ctx, "user1")
	if len(keys) != 2 {
		t.Errorf("expected 2 keys for user1, got %d", len(keys))
	}
}

func TestKeyStore_Revoke(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})

	now := time.Now()
	err := store.Revoke(ctx, "k1", now)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	keys, _ := store.ListByUser(ctx, "user1")
	if keys[0].RevokedAt == nil {
		t.Error("expected RevokedAt to be set")
	}
}

func TestKeyStore_UpdateLastUsed(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})

	now := time.Now()
	err := store.UpdateLastUsed(ctx, "k1", now)
	if err != nil {
		t.Fatalf("UpdateLastUsed failed: %v", err)
	}

	keys, _ := store.ListByUser(ctx, "user1")
	if keys[0].LastUsed == nil {
		t.Error("expected LastUsed to be set")
	}
}

func TestKeyStore_GetAll(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1"})
	store.Create(ctx, key.Key{ID: "k2"})
	store.Create(ctx, key.Key{ID: "k3"})

	all := store.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 keys, got %d", len(all))
	}
}

func TestKeyStore_Clear(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1"})
	store.Create(ctx, key.Key{ID: "k2"})

	store.Clear()

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 keys after Clear, got %d", len(all))
	}
}

// UserStore tests

func TestUserStore_Create(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	user := ports.User{ID: "u1", Email: "test@example.com"}
	err := store.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, _ := store.Get(ctx, "u1")
	if got.Email != "test@example.com" {
		t.Errorf("Email = %s", got.Email)
	}
}

func TestUserStore_Create_DuplicateEmail(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})
	err := store.Create(ctx, ports.User{ID: "u2", Email: "test@example.com"})

	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestUserStore_Get(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com", Name: "Test"})

	user, err := store.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if user.Name != "Test" {
		t.Errorf("Name = %s", user.Name)
	}
}

func TestUserStore_Get_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})

	user, err := store.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if user.ID != "u1" {
		t.Errorf("ID = %s", user.ID)
	}
}

func TestUserStore_GetByEmail_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	_, err := store.GetByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for nonexistent email")
	}
}

func TestUserStore_Update(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "old@example.com", Name: "Old"})

	err := store.Update(ctx, ports.User{ID: "u1", Email: "new@example.com", Name: "New"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	user, _ := store.Get(ctx, "u1")
	if user.Name != "New" {
		t.Errorf("Name = %s", user.Name)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %s", user.Email)
	}
}

func TestUserStore_Update_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	err := store.Update(ctx, ports.User{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_Delete(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})

	err := store.Delete(ctx, "u1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(ctx, "u1")
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestUserStore_Delete_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_List(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})
	store.Create(ctx, ports.User{ID: "u3", Email: "c@example.com"})

	users, _ := store.List(ctx, 10, 0)
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestUserStore_List_Pagination(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		store.Create(ctx, ports.User{ID: string(rune('a' + i)), Email: string(rune('a'+i)) + "@example.com"})
	}

	users, _ := store.List(ctx, 3, 0)
	if len(users) != 3 {
		t.Errorf("limit: expected 3 users, got %d", len(users))
	}

	users, _ = store.List(ctx, 3, 8)
	if len(users) != 2 {
		t.Errorf("offset: expected 2 users, got %d", len(users))
	}

	users, _ = store.List(ctx, 3, 100)
	if len(users) != 0 {
		t.Errorf("large offset: expected 0 users, got %d", len(users))
	}
}

func TestUserStore_Count(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	count, _ = store.Count(ctx)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestUserStore_GetAll(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	all := store.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 users, got %d", len(all))
	}
}

func TestUserStore_Clear(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})
	store.Clear()

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0 after Clear, got %d", count)
	}
}

// RateLimitStore tests

func TestRateLimitStore_GetSet(t *testing.T) {
	store := memory.NewRateLimitStore()
	ctx := context.Background()

	state := ratelimit.WindowState{
		Count:     10,
		WindowEnd: time.Now().Add(time.Minute),
		BurstUsed: 2,
	}

	err := store.Set(ctx, "key1", state)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Count != 10 {
		t.Errorf("Count = %d, want 10", got.Count)
	}
	if got.BurstUsed != 2 {
		t.Errorf("BurstUsed = %d, want 2", got.BurstUsed)
	}
}

func TestRateLimitStore_Get_NotFound(t *testing.T) {
	store := memory.NewRateLimitStore()
	ctx := context.Background()

	state, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Should return zero state
	if state.Count != 0 {
		t.Errorf("expected zero count for nonexistent key")
	}
}

func TestRateLimitStore_Clear(t *testing.T) {
	store := memory.NewRateLimitStore()
	ctx := context.Background()

	store.Set(ctx, "key1", ratelimit.WindowState{Count: 5})
	store.Set(ctx, "key2", ratelimit.WindowState{Count: 10})

	store.Clear()

	state, _ := store.Get(ctx, "key1")
	if state.Count != 0 {
		t.Errorf("expected 0 after Clear, got %d", state.Count)
	}
}
