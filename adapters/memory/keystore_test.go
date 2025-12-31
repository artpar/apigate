package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/key"
)

func TestKeyStore_NewKeyStore(t *testing.T) {
	store := memory.NewKeyStore()
	if store == nil {
		t.Fatal("NewKeyStore returned nil")
	}

	// Verify it's empty
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("new store should be empty, got %d keys", len(all))
	}
}

func TestKeyStore_CreateAndGet(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	k := key.Key{
		ID:        "key-001",
		UserID:    "user-001",
		Prefix:    "ak_test1234",
		Name:      "Test Key",
		Scopes:    []string{"read", "write"},
		CreatedAt: time.Now(),
	}

	if err := store.Create(ctx, k); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by prefix
	keys, err := store.Get(ctx, "ak_test1234")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ID != "key-001" {
		t.Errorf("expected ID 'key-001', got '%s'", keys[0].ID)
	}
	if keys[0].Name != "Test Key" {
		t.Errorf("expected Name 'Test Key', got '%s'", keys[0].Name)
	}
}

func TestKeyStore_Get_EmptyPrefix(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Create keys with different prefixes
	store.Create(ctx, key.Key{ID: "k1", Prefix: "prefix1"})
	store.Create(ctx, key.Key{ID: "k2", Prefix: "prefix2"})

	// Get with non-matching prefix
	keys, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for nonexistent prefix, got %d", len(keys))
	}
}

func TestKeyStore_Get_MultipleMatches(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Create multiple keys with same prefix (different IDs)
	store.Create(ctx, key.Key{ID: "k1", Prefix: "same_prefix", UserID: "user1"})
	store.Create(ctx, key.Key{ID: "k2", Prefix: "same_prefix", UserID: "user2"})
	store.Create(ctx, key.Key{ID: "k3", Prefix: "same_prefix", UserID: "user3"})

	keys, err := store.Get(ctx, "same_prefix")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys with same prefix, got %d", len(keys))
	}
}

func TestKeyStore_ListByUser_NoKeys(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	keys, err := store.ListByUser(ctx, "nonexistent-user")
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for nonexistent user, got %d", len(keys))
	}
}

func TestKeyStore_ListByUser_MultipleKeys(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Create keys for different users
	store.Create(ctx, key.Key{ID: "k1", UserID: "user1", Name: "Key 1"})
	store.Create(ctx, key.Key{ID: "k2", UserID: "user1", Name: "Key 2"})
	store.Create(ctx, key.Key{ID: "k3", UserID: "user1", Name: "Key 3"})
	store.Create(ctx, key.Key{ID: "k4", UserID: "user2", Name: "Key 4"})

	keys, err := store.ListByUser(ctx, "user1")
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys for user1, got %d", len(keys))
	}
}

func TestKeyStore_Revoke_ExistingKey(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})

	revokeTime := time.Now()
	err := store.Revoke(ctx, "k1", revokeTime)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	keys, _ := store.ListByUser(ctx, "user1")
	if len(keys) != 1 {
		t.Fatal("expected 1 key")
	}
	if keys[0].RevokedAt == nil {
		t.Error("expected RevokedAt to be set")
	}
	if !keys[0].RevokedAt.Equal(revokeTime) {
		t.Errorf("RevokedAt = %v, want %v", *keys[0].RevokedAt, revokeTime)
	}
}

func TestKeyStore_Revoke_NonExistentKey(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Revoking non-existent key should not error (idempotent)
	err := store.Revoke(ctx, "nonexistent", time.Now())
	if err != nil {
		t.Errorf("Revoke on non-existent key should not error: %v", err)
	}
}

func TestKeyStore_UpdateLastUsed_ExistingKey(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})

	lastUsedTime := time.Now()
	err := store.UpdateLastUsed(ctx, "k1", lastUsedTime)
	if err != nil {
		t.Fatalf("UpdateLastUsed failed: %v", err)
	}

	keys, _ := store.ListByUser(ctx, "user1")
	if len(keys) != 1 {
		t.Fatal("expected 1 key")
	}
	if keys[0].LastUsed == nil {
		t.Error("expected LastUsed to be set")
	}
	if !keys[0].LastUsed.Equal(lastUsedTime) {
		t.Errorf("LastUsed = %v, want %v", *keys[0].LastUsed, lastUsedTime)
	}
}

func TestKeyStore_UpdateLastUsed_NonExistentKey(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Updating non-existent key should not error (idempotent)
	err := store.UpdateLastUsed(ctx, "nonexistent", time.Now())
	if err != nil {
		t.Errorf("UpdateLastUsed on non-existent key should not error: %v", err)
	}
}

func TestKeyStore_UpdateLastUsed_MultipleTimes(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1", UserID: "user1"})

	// Update multiple times
	time1 := time.Now()
	store.UpdateLastUsed(ctx, "k1", time1)

	time2 := time1.Add(time.Hour)
	store.UpdateLastUsed(ctx, "k1", time2)

	keys, _ := store.ListByUser(ctx, "user1")
	if !keys[0].LastUsed.Equal(time2) {
		t.Errorf("LastUsed should be updated to latest time")
	}
}

func TestKeyStore_GetAll_Empty(t *testing.T) {
	store := memory.NewKeyStore()

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 keys, got %d", len(all))
	}
}

func TestKeyStore_GetAll_WithKeys(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1"})
	store.Create(ctx, key.Key{ID: "k2"})
	store.Create(ctx, key.Key{ID: "k3"})
	store.Create(ctx, key.Key{ID: "k4"})
	store.Create(ctx, key.Key{ID: "k5"})

	all := store.GetAll()
	if len(all) != 5 {
		t.Errorf("expected 5 keys, got %d", len(all))
	}
}

func TestKeyStore_Clear(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1"})
	store.Create(ctx, key.Key{ID: "k2"})
	store.Create(ctx, key.Key{ID: "k3"})

	store.Clear()

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 keys after Clear, got %d", len(all))
	}
}

func TestKeyStore_Clear_MultipleTimes(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	store.Create(ctx, key.Key{ID: "k1"})
	store.Clear()
	store.Clear() // Should not panic

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 keys, got %d", len(all))
	}
}

func TestKeyStore_CreateOverwrite(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Create a key
	store.Create(ctx, key.Key{ID: "k1", Name: "Original"})

	// Create with same ID (overwrites)
	store.Create(ctx, key.Key{ID: "k1", Name: "Updated"})

	all := store.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 key, got %d", len(all))
	}
	if all[0].Name != "Updated" {
		t.Errorf("expected Name 'Updated', got '%s'", all[0].Name)
	}
}

func TestKeyStore_ConcurrentAccess(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			k := key.Key{
				ID:     string(rune('a' + idx%26)) + string(rune('0'+idx/26)),
				UserID: "user1",
				Prefix: "prefix",
			}
			store.Create(ctx, k)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.ListByUser(ctx, "user1")
			store.Get(ctx, "prefix")
			store.GetAll()
		}()
	}

	wg.Wait()
}

func TestKeyStore_KeyWithExpiration(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	expiry := time.Now().Add(24 * time.Hour)
	k := key.Key{
		ID:        "k1",
		UserID:    "user1",
		ExpiresAt: &expiry,
	}

	store.Create(ctx, k)

	keys, _ := store.ListByUser(ctx, "user1")
	if keys[0].ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if !keys[0].ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", *keys[0].ExpiresAt, expiry)
	}
}

func TestKeyStore_KeyWithScopes(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	scopes := []string{"read", "write", "admin"}
	k := key.Key{
		ID:     "k1",
		UserID: "user1",
		Scopes: scopes,
	}

	store.Create(ctx, k)

	keys, _ := store.ListByUser(ctx, "user1")
	if len(keys[0].Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(keys[0].Scopes))
	}
}

func TestKeyStore_FullKeyLifecycle(t *testing.T) {
	store := memory.NewKeyStore()
	ctx := context.Background()

	// Create
	k := key.Key{
		ID:        "lifecycle-key",
		UserID:    "user1",
		Prefix:    "ak_lifecycle",
		Name:      "Lifecycle Test",
		CreatedAt: time.Now(),
	}
	store.Create(ctx, k)

	// Update last used
	store.UpdateLastUsed(ctx, "lifecycle-key", time.Now())

	// Verify
	keys, _ := store.ListByUser(ctx, "user1")
	if keys[0].LastUsed == nil {
		t.Error("expected LastUsed to be set")
	}

	// Revoke
	store.Revoke(ctx, "lifecycle-key", time.Now())

	// Verify revocation
	keys, _ = store.ListByUser(ctx, "user1")
	if keys[0].RevokedAt == nil {
		t.Error("expected RevokedAt to be set")
	}

	// Clear
	store.Clear()

	// Verify empty
	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 keys after Clear, got %d", len(all))
	}
}
