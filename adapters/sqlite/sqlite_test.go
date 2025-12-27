package sqlite_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

func setupTestDB(t *testing.T) (*sqlite.DB, func()) {
	t.Helper()

	// Create temp file for test database
	f, err := os.CreateTemp("", "apigate-test-*.db")
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

// -----------------------------------------------------------------------------
// UserStore Tests
// -----------------------------------------------------------------------------

func TestUserStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		PlanID: "free",
		Status: "active",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %s, want %s", got.ID, user.ID)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %s, want %s", got.Email, user.Email)
	}
	if got.Status != user.Status {
		t.Errorf("Status = %s, want %s", got.Status, user.Status)
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "lookup@example.com",
		PlanID: "free",
		Status: "active",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := store.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %s, want %s", got.ID, user.ID)
	}
}

func TestUserStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "update@example.com",
		PlanID: "free",
		Status: "active",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	user.PlanID = "pro"
	user.Status = "suspended"

	if err := store.Update(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if got.PlanID != "pro" {
		t.Errorf("PlanID = %s, want pro", got.PlanID)
	}
	if got.Status != "suspended" {
		t.Errorf("Status = %s, want suspended", got.Status)
	}
}

func TestUserStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 5; i++ {
		user := ports.User{
			ID:     "user-" + itoa(i),
			Email:  "user" + itoa(i) + "@example.com",
			PlanID: "free",
			Status: "active",
		}
		if err := store.Create(ctx, user); err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
	}

	// List with pagination
	users, err := store.List(ctx, 3, 0)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("len = %d, want 3", len(users))
	}

	// Count
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}

	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestUserStore_DuplicateEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user1 := ports.User{
		ID:     "user-1",
		Email:  "dupe@example.com",
		PlanID: "free",
		Status: "active",
	}

	if err := store.Create(ctx, user1); err != nil {
		t.Fatalf("create user1: %v", err)
	}

	user2 := ports.User{
		ID:     "user-2",
		Email:  "dupe@example.com", // Same email
		PlanID: "free",
		Status: "active",
	}

	err := store.Create(ctx, user2)
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestUserStore_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// KeyStore Tests
// -----------------------------------------------------------------------------

func TestKeyStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	// Create user first (foreign key)
	user := ports.User{
		ID:     "user-1",
		Email:  "key@example.com",
		PlanID: "free",
		Status: "active",
	}
	if err := userStore.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	k := key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      []byte("hash123"),
		Prefix:    "ak_test12345",
		Name:      "Test Key",
		Scopes:    []string{"/api/v1/*"},
		CreatedAt: time.Now().UTC(),
	}

	if err := keyStore.Create(ctx, k); err != nil {
		t.Fatalf("create key: %v", err)
	}

	keys, err := keyStore.Get(ctx, k.Prefix)
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("len = %d, want 1", len(keys))
	}

	got := keys[0]
	if got.ID != k.ID {
		t.Errorf("ID = %s, want %s", got.ID, k.ID)
	}
	if got.UserID != k.UserID {
		t.Errorf("UserID = %s, want %s", got.UserID, k.UserID)
	}
	if len(got.Scopes) != 1 || got.Scopes[0] != "/api/v1/*" {
		t.Errorf("Scopes = %v, want [/api/v1/*]", got.Scopes)
	}
}

func TestKeyStore_Revoke(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "revoke@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	k := key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      []byte("hash"),
		Prefix:    "ak_revoke123",
		CreatedAt: time.Now().UTC(),
	}
	keyStore.Create(ctx, k)

	revokeTime := time.Now().UTC()
	if err := keyStore.Revoke(ctx, k.ID, revokeTime); err != nil {
		t.Fatalf("revoke key: %v", err)
	}

	got, err := keyStore.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}

	if got.RevokedAt == nil {
		t.Fatal("RevokedAt should not be nil")
	}
}

func TestKeyStore_ListByUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "list@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	// Create multiple keys
	for i := 0; i < 3; i++ {
		k := key.Key{
			ID:        "key-" + itoa(i),
			UserID:    "user-1",
			Hash:      []byte("hash"),
			Prefix:    "ak_list" + itoa(i) + "1234",
			CreatedAt: time.Now().UTC(),
		}
		keyStore.Create(ctx, k)
	}

	keys, err := keyStore.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("len = %d, want 3", len(keys))
	}
}

func TestKeyStore_UpdateLastUsed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-1",
		Email:  "lastused@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	k := key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      []byte("hash"),
		Prefix:    "ak_lastused1",
		CreatedAt: time.Now().UTC(),
	}
	keyStore.Create(ctx, k)

	lastUsed := time.Now().UTC()
	if err := keyStore.UpdateLastUsed(ctx, k.ID, lastUsed); err != nil {
		t.Fatalf("update last used: %v", err)
	}

	got, err := keyStore.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}

	if got.LastUsed == nil {
		t.Fatal("LastUsed should not be nil")
	}
}

// -----------------------------------------------------------------------------
// RateLimitStore Tests
// -----------------------------------------------------------------------------

func TestRateLimitStore_GetAndSet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRateLimitStore(db)
	ctx := context.Background()

	state := ratelimit.WindowState{
		Count:     5,
		WindowEnd: time.Now().UTC().Add(time.Minute),
		BurstUsed: 2,
	}

	if err := store.Set(ctx, "key-1", state); err != nil {
		t.Fatalf("set state: %v", err)
	}

	got, err := store.Get(ctx, "key-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	if got.Count != state.Count {
		t.Errorf("Count = %d, want %d", got.Count, state.Count)
	}
	if got.BurstUsed != state.BurstUsed {
		t.Errorf("BurstUsed = %d, want %d", got.BurstUsed, state.BurstUsed)
	}
}

func TestRateLimitStore_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRateLimitStore(db)
	ctx := context.Background()

	// Should return empty state, not error
	got, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	if got.Count != 0 {
		t.Errorf("Count = %d, want 0", got.Count)
	}
}

func TestRateLimitStore_Upsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRateLimitStore(db)
	ctx := context.Background()

	// Initial set
	state1 := ratelimit.WindowState{
		Count:     1,
		WindowEnd: time.Now().UTC().Add(time.Minute),
		BurstUsed: 0,
	}
	store.Set(ctx, "key-1", state1)

	// Update
	state2 := ratelimit.WindowState{
		Count:     5,
		WindowEnd: time.Now().UTC().Add(time.Minute),
		BurstUsed: 2,
	}
	if err := store.Set(ctx, "key-1", state2); err != nil {
		t.Fatalf("upsert state: %v", err)
	}

	got, _ := store.Get(ctx, "key-1")
	if got.Count != 5 {
		t.Errorf("Count = %d, want 5", got.Count)
	}
}

// -----------------------------------------------------------------------------
// UsageStore Tests
// -----------------------------------------------------------------------------

func TestUsageStore_RecordBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	events := []usage.Event{
		{
			ID:            "evt-1",
			KeyID:         "key-1",
			UserID:        "user-1",
			Method:        "GET",
			Path:          "/api/data",
			StatusCode:    200,
			LatencyMs:     50,
			RequestBytes:  100,
			ResponseBytes: 500,
			CostMultiplier: 1.0,
			Timestamp:     time.Now().UTC(),
		},
		{
			ID:            "evt-2",
			KeyID:         "key-1",
			UserID:        "user-1",
			Method:        "POST",
			Path:          "/api/data",
			StatusCode:    201,
			LatencyMs:     100,
			RequestBytes:  500,
			ResponseBytes: 200,
			CostMultiplier: 2.0,
			Timestamp:     time.Now().UTC(),
		},
	}

	if err := store.RecordBatch(ctx, events); err != nil {
		t.Fatalf("record batch: %v", err)
	}

	// Verify by getting recent requests
	recent, err := store.GetRecentRequests(ctx, "user-1", 10)
	if err != nil {
		t.Fatalf("get recent: %v", err)
	}

	if len(recent) != 2 {
		t.Errorf("len = %d, want 2", len(recent))
	}
}

func TestUsageStore_GetSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	now := time.Now().UTC()
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	events := []usage.Event{
		{
			ID:             "evt-1",
			KeyID:          "key-1",
			UserID:         "user-1",
			Method:         "GET",
			Path:           "/api/data",
			StatusCode:     200,
			LatencyMs:      50,
			RequestBytes:   100,
			ResponseBytes:  500,
			CostMultiplier: 1.0,
			Timestamp:      now,
		},
		{
			ID:             "evt-2",
			KeyID:          "key-1",
			UserID:         "user-1",
			Method:         "GET",
			Path:           "/api/data",
			StatusCode:     500, // Error
			LatencyMs:      150,
			RequestBytes:   100,
			ResponseBytes:  50,
			CostMultiplier: 1.0,
			Timestamp:      now,
		},
	}

	if err := store.RecordBatch(ctx, events); err != nil {
		t.Fatalf("record batch: %v", err)
	}

	summary, err := store.GetSummary(ctx, "user-1", start, end)
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}

	if summary.RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", summary.RequestCount)
	}
	if summary.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", summary.ErrorCount)
	}
	if summary.BytesIn != 200 {
		t.Errorf("BytesIn = %d, want 200", summary.BytesIn)
	}
	if summary.BytesOut != 550 {
		t.Errorf("BytesOut = %d, want 550", summary.BytesOut)
	}
}

// -----------------------------------------------------------------------------
// Migration Tests
// -----------------------------------------------------------------------------

func TestMigration_Idempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Run migrations again - should be idempotent
	if err := db.Migrate(); err != nil {
		t.Fatalf("second migration: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
