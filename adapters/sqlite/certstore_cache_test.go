package sqlite_test

import (
	"context"
	"os"
	"testing"

	"github.com/artpar/apigate/adapters/sqlite"
)

func setupCacheTestDB(t *testing.T) (*sqlite.DB, func()) {
	t.Helper()

	f, err := os.CreateTemp("", "apigate-cache-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	f.Close()

	db, err := sqlite.Open(path)
	if err != nil {
		os.Remove(path)
		t.Fatalf("open database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		os.Remove(path)
		t.Fatalf("migrate: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(path)
	}

	return db, cleanup
}

func TestCertStoreCache_GetPutDelete(t *testing.T) {
	db, cleanup := setupCacheTestDB(t)
	defer cleanup()

	store := sqlite.NewCertificateStore(db)
	ctx := context.Background()

	key := "+acme_account+https://acme-v02.api.letsencrypt.org/directory"
	data := []byte("test-account-key-data")

	// Test GetCache when key doesn't exist
	_, err := store.GetCache(ctx, key)
	if err == nil {
		t.Error("expected error for non-existent key")
	}

	// Test PutCache
	err = store.PutCache(ctx, key, data)
	if err != nil {
		t.Fatalf("PutCache failed: %v", err)
	}

	// Test GetCache after Put
	retrieved, err := store.GetCache(ctx, key)
	if err != nil {
		t.Fatalf("GetCache failed: %v", err)
	}
	if string(retrieved) != string(data) {
		t.Errorf("data mismatch: got %s, want %s", retrieved, data)
	}

	// Test PutCache update (upsert)
	newData := []byte("updated-account-key-data")
	err = store.PutCache(ctx, key, newData)
	if err != nil {
		t.Fatalf("PutCache update failed: %v", err)
	}

	retrieved, err = store.GetCache(ctx, key)
	if err != nil {
		t.Fatalf("GetCache after update failed: %v", err)
	}
	if string(retrieved) != string(newData) {
		t.Errorf("data mismatch after update: got %s, want %s", retrieved, newData)
	}

	// Test DeleteCache
	err = store.DeleteCache(ctx, key)
	if err != nil {
		t.Fatalf("DeleteCache failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetCache(ctx, key)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestCertStoreCache_MultipleKeys(t *testing.T) {
	db, cleanup := setupCacheTestDB(t)
	defer cleanup()

	store := sqlite.NewCertificateStore(db)
	ctx := context.Background()

	keys := []string{
		"+acme_account+https://acme-v02.api.letsencrypt.org/directory",
		"+acme_account+https://acme-staging-v02.api.letsencrypt.org/directory",
		"+http-01-token-domain.example.com",
	}

	// Store multiple keys
	for i, key := range keys {
		data := []byte("data-" + key)
		if err := store.PutCache(ctx, key, data); err != nil {
			t.Fatalf("PutCache failed for key %d: %v", i, err)
		}
	}

	// Verify all keys can be retrieved
	for i, key := range keys {
		expected := []byte("data-" + key)
		retrieved, err := store.GetCache(ctx, key)
		if err != nil {
			t.Fatalf("GetCache failed for key %d: %v", i, err)
		}
		if string(retrieved) != string(expected) {
			t.Errorf("data mismatch for key %d: got %s, want %s", i, retrieved, expected)
		}
	}

	// Delete one key and verify others still exist
	if err := store.DeleteCache(ctx, keys[1]); err != nil {
		t.Fatalf("DeleteCache failed: %v", err)
	}

	// Verify deleted key is gone
	_, err := store.GetCache(ctx, keys[1])
	if err == nil {
		t.Error("expected error for deleted key")
	}

	// Verify other keys still exist
	for i, key := range []string{keys[0], keys[2]} {
		if _, err := store.GetCache(ctx, key); err != nil {
			t.Errorf("key %d should still exist: %v", i, err)
		}
	}
}
