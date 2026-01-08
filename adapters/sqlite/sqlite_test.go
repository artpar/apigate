package sqlite_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/entitlement"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/domain/webhook"
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
// RouteStore Tests
// -----------------------------------------------------------------------------

func TestRouteStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First create an upstream (foreign key reference)
	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Test Upstream", "https://api.example.com")
	if err := upstreamStore.Create(ctx, upstream); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-1", "Test Route", "/api/*", "up-1")
	r.Description = "A test route"
	r.MatchType = route.MatchPrefix
	r.Methods = []string{"GET", "POST"}
	r.Priority = 10
	r.MeteringExpr = `respBody.usage.tokens ?? 1`

	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	got, err := store.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	if got.ID != r.ID {
		t.Errorf("ID = %s, want %s", got.ID, r.ID)
	}
	if got.Name != r.Name {
		t.Errorf("Name = %s, want %s", got.Name, r.Name)
	}
	if got.PathPattern != r.PathPattern {
		t.Errorf("PathPattern = %s, want %s", got.PathPattern, r.PathPattern)
	}
	if got.MatchType != route.MatchPrefix {
		t.Errorf("MatchType = %s, want prefix", got.MatchType)
	}
	if len(got.Methods) != 2 {
		t.Errorf("Methods len = %d, want 2", len(got.Methods))
	}
	if got.Priority != 10 {
		t.Errorf("Priority = %d, want 10", got.Priority)
	}
	if got.MeteringExpr != r.MeteringExpr {
		t.Errorf("MeteringExpr = %s, want %s", got.MeteringExpr, r.MeteringExpr)
	}
}

func TestRouteStore_CreateWithTransforms(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-1", "Transform Route", "/api/*", "up-1")
	r.RequestTransform = &route.Transform{
		SetHeaders:    map[string]string{"X-Custom": `"value"`},
		DeleteHeaders: []string{"X-Remove"},
		SetQuery:      map[string]string{"added": `"param"`},
		BodyExpr:      `{"wrapped": body}`,
	}
	r.ResponseTransform = &route.Transform{
		SetHeaders:    map[string]string{"X-Response": `"resp-value"`},
		DeleteHeaders: []string{"X-Internal"},
	}

	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	got, err := store.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	if got.RequestTransform == nil {
		t.Fatal("RequestTransform should not be nil")
	}
	if got.RequestTransform.SetHeaders["X-Custom"] != `"value"` {
		t.Errorf("SetHeaders = %v", got.RequestTransform.SetHeaders)
	}
	if got.RequestTransform.BodyExpr != `{"wrapped": body}` {
		t.Errorf("BodyExpr = %s", got.RequestTransform.BodyExpr)
	}
	if got.ResponseTransform == nil {
		t.Fatal("ResponseTransform should not be nil")
	}
}

func TestRouteStore_CreateWithHeaders(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-1", "Header Route", "/api/*", "up-1")
	r.Headers = []route.HeaderMatch{
		{Name: "Content-Type", Value: "application/json", Required: true},
		{Name: "X-Version", Value: `^v[0-9]+$`, IsRegex: true, Required: false},
	}

	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	got, err := store.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	if len(got.Headers) != 2 {
		t.Fatalf("Headers len = %d, want 2", len(got.Headers))
	}
	if got.Headers[0].Name != "Content-Type" {
		t.Errorf("Headers[0].Name = %s, want Content-Type", got.Headers[0].Name)
	}
	if !got.Headers[0].Required {
		t.Error("Headers[0].Required should be true")
	}
	if !got.Headers[1].IsRegex {
		t.Error("Headers[1].IsRegex should be true")
	}
}

func TestRouteStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	// Create routes with different priorities
	for i := 0; i < 5; i++ {
		r := route.NewRoute("route-"+itoa(i), "Route "+itoa(i), "/api/"+itoa(i)+"/*", "up-1")
		r.Priority = i * 10
		if err := store.Create(ctx, r); err != nil {
			t.Fatalf("create route %d: %v", i, err)
		}
	}

	routes, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list routes: %v", err)
	}

	if len(routes) != 5 {
		t.Errorf("len = %d, want 5", len(routes))
	}

	// Should be ordered by priority descending
	if routes[0].Priority != 40 {
		t.Errorf("first route priority = %d, want 40", routes[0].Priority)
	}
}

func TestRouteStore_ListEnabled(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	// Create enabled and disabled routes
	for i := 0; i < 5; i++ {
		r := route.NewRoute("route-"+itoa(i), "Route "+itoa(i), "/api/"+itoa(i)+"/*", "up-1")
		r.Enabled = i%2 == 0 // 0, 2, 4 are enabled
		if err := store.Create(ctx, r); err != nil {
			t.Fatalf("create route %d: %v", i, err)
		}
	}

	routes, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled routes: %v", err)
	}

	if len(routes) != 3 {
		t.Errorf("len = %d, want 3", len(routes))
	}

	for _, r := range routes {
		if !r.Enabled {
			t.Errorf("route %s should be enabled", r.ID)
		}
	}
}

func TestRouteStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-1", "Original", "/api/*", "up-1")
	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	// Update
	r.Name = "Updated"
	r.Priority = 100
	r.MeteringExpr = "2"
	r.Enabled = false

	if err := store.Update(ctx, r); err != nil {
		t.Fatalf("update route: %v", err)
	}

	got, err := store.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	if got.Name != "Updated" {
		t.Errorf("Name = %s, want Updated", got.Name)
	}
	if got.Priority != 100 {
		t.Errorf("Priority = %d, want 100", got.Priority)
	}
	if got.Enabled {
		t.Error("Enabled should be false")
	}
}

func TestRouteStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-1", "To Delete", "/api/*", "up-1")
	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	if err := store.Delete(ctx, r.ID); err != nil {
		t.Fatalf("delete route: %v", err)
	}

	_, err := store.Get(ctx, r.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRouteStore_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRouteStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRouteStore_DeleteNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRouteStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// UpstreamStore Tests
// -----------------------------------------------------------------------------

func TestUpstreamStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "API Backend", "https://api.example.com")
	u.Description = "Backend API server"
	u.Timeout = 30 * time.Second
	u.MaxIdleConns = 50
	u.IdleConnTimeout = 60 * time.Second

	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	got, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get upstream: %v", err)
	}

	if got.ID != u.ID {
		t.Errorf("ID = %s, want %s", got.ID, u.ID)
	}
	if got.Name != u.Name {
		t.Errorf("Name = %s, want %s", got.Name, u.Name)
	}
	if got.BaseURL != u.BaseURL {
		t.Errorf("BaseURL = %s, want %s", got.BaseURL, u.BaseURL)
	}
	if got.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", got.Timeout)
	}
	if got.MaxIdleConns != 50 {
		t.Errorf("MaxIdleConns = %d, want 50", got.MaxIdleConns)
	}
}

func TestUpstreamStore_CreateWithAuth(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "Auth Upstream", "https://api.example.com")
	u = u.WithAuth(route.AuthBearer, "", "secret-token")

	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	got, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get upstream: %v", err)
	}

	if got.AuthType != route.AuthBearer {
		t.Errorf("AuthType = %s, want bearer", got.AuthType)
	}
	if got.AuthValue != "secret-token" {
		t.Errorf("AuthValue = %s, want secret-token", got.AuthValue)
	}
}

func TestUpstreamStore_CreateWithHeaderAuth(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "Header Auth", "https://api.example.com")
	u = u.WithAuth(route.AuthHeader, "X-API-Key", "api-key-123")

	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	got, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get upstream: %v", err)
	}

	if got.AuthType != route.AuthHeader {
		t.Errorf("AuthType = %s, want header", got.AuthType)
	}
	if got.AuthHeader != "X-API-Key" {
		t.Errorf("AuthHeader = %s, want X-API-Key", got.AuthHeader)
	}
	if got.AuthValue != "api-key-123" {
		t.Errorf("AuthValue = %s, want api-key-123", got.AuthValue)
	}
}

func TestUpstreamStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	// Create multiple upstreams
	names := []string{"Alpha", "Beta", "Gamma"}
	for i, name := range names {
		u := route.NewUpstream("up-"+itoa(i), name, "https://"+name+".example.com")
		if err := store.Create(ctx, u); err != nil {
			t.Fatalf("create upstream %s: %v", name, err)
		}
	}

	upstreams, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list upstreams: %v", err)
	}

	if len(upstreams) != 3 {
		t.Errorf("len = %d, want 3", len(upstreams))
	}

	// Should be ordered by name
	if upstreams[0].Name != "Alpha" {
		t.Errorf("first upstream = %s, want Alpha", upstreams[0].Name)
	}
}

func TestUpstreamStore_ListEnabled(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	// Create enabled and disabled upstreams
	for i := 0; i < 4; i++ {
		u := route.NewUpstream("up-"+itoa(i), "Upstream "+itoa(i), "https://up"+itoa(i)+".example.com")
		u.Enabled = i%2 == 0
		if err := store.Create(ctx, u); err != nil {
			t.Fatalf("create upstream %d: %v", i, err)
		}
	}

	upstreams, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled upstreams: %v", err)
	}

	if len(upstreams) != 2 {
		t.Errorf("len = %d, want 2", len(upstreams))
	}

	for _, u := range upstreams {
		if !u.Enabled {
			t.Errorf("upstream %s should be enabled", u.ID)
		}
	}
}

func TestUpstreamStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "Original", "https://original.example.com")
	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	// Update
	u.Name = "Updated"
	u.BaseURL = "https://updated.example.com"
	u.Timeout = 60 * time.Second
	u.Enabled = false

	if err := store.Update(ctx, u); err != nil {
		t.Fatalf("update upstream: %v", err)
	}

	got, err := store.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get upstream: %v", err)
	}

	if got.Name != "Updated" {
		t.Errorf("Name = %s, want Updated", got.Name)
	}
	if got.BaseURL != "https://updated.example.com" {
		t.Errorf("BaseURL = %s, want https://updated.example.com", got.BaseURL)
	}
	if got.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", got.Timeout)
	}
	if got.Enabled {
		t.Error("Enabled should be false")
	}
}

func TestUpstreamStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "To Delete", "https://delete.example.com")
	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	if err := store.Delete(ctx, u.ID); err != nil {
		t.Fatalf("delete upstream: %v", err)
	}

	_, err := store.Get(ctx, u.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpstreamStore_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpstreamStore_DeleteNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpstreamStore_DuplicateID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("up-1", "First", "https://first.example.com")
	if err := store.Create(ctx, u); err != nil {
		t.Fatalf("create first: %v", err)
	}

	u2 := route.NewUpstream("up-1", "Second", "https://second.example.com")
	err := store.Create(ctx, u2)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
}

// -----------------------------------------------------------------------------
// UserStore Additional Tests
// -----------------------------------------------------------------------------

func TestUserStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-delete-1",
		Email:  "delete@example.com",
		PlanID: "free",
		Status: "active",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := store.Delete(ctx, user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	_, err := store.Get(ctx, user.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestUserStore_DeleteNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_UpdateNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "nonexistent",
		Email:  "nonexistent@example.com",
		PlanID: "free",
		Status: "active",
	}

	err := store.Update(ctx, user)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_CreateWithTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	now := time.Now().UTC()
	user := ports.User{
		ID:        "user-timestamps",
		Email:     "timestamps@example.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestUserStore_UpdateDuplicateEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	// Create first user
	user1 := ports.User{
		ID:     "user-1",
		Email:  "first@example.com",
		PlanID: "free",
		Status: "active",
	}
	if err := store.Create(ctx, user1); err != nil {
		t.Fatalf("create user1: %v", err)
	}

	// Create second user
	user2 := ports.User{
		ID:     "user-2",
		Email:  "second@example.com",
		PlanID: "free",
		Status: "active",
	}
	if err := store.Create(ctx, user2); err != nil {
		t.Fatalf("create user2: %v", err)
	}

	// Try to update second user with first user's email
	user2.Email = "first@example.com"
	err := store.Update(ctx, user2)
	if err == nil {
		t.Fatal("expected error for duplicate email on update")
	}
}

func TestUserStore_WithStripeID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUserStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:       "user-stripe",
		Email:    "stripe@example.com",
		PlanID:   "pro",
		Status:   "active",
		StripeID: "cus_12345",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if got.StripeID != "cus_12345" {
		t.Errorf("StripeID = %s, want cus_12345", got.StripeID)
	}
}

// -----------------------------------------------------------------------------
// KeyStore Additional Tests
// -----------------------------------------------------------------------------

func TestKeyStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-list",
		Email:  "list@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	// Create multiple keys
	for i := 0; i < 3; i++ {
		k := key.Key{
			ID:        "key-list-" + itoa(i),
			UserID:    "user-list",
			Hash:      []byte("hash"),
			Prefix:    "ak_list" + itoa(i) + "abcd",
			CreatedAt: time.Now().UTC(),
		}
		keyStore.Create(ctx, k)
	}

	keys, err := keyStore.List(ctx)
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("len = %d, want 3", len(keys))
	}
}

func TestKeyStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-update-key",
		Email:  "updatekey@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	k := key.Key{
		ID:        "key-update-1",
		UserID:    "user-update-key",
		Hash:      []byte("hash"),
		Prefix:    "ak_update1234",
		Name:      "Original Name",
		Scopes:    []string{"/api/*"},
		CreatedAt: time.Now().UTC(),
	}
	keyStore.Create(ctx, k)

	// Update the key
	k.Name = "Updated Name"
	k.Scopes = []string{"/api/v1/*", "/api/v2/*"}
	expiresAt := time.Now().UTC().Add(time.Hour * 24)
	k.ExpiresAt = &expiresAt

	if err := keyStore.Update(ctx, k); err != nil {
		t.Fatalf("update key: %v", err)
	}

	got, err := keyStore.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", got.Name)
	}
	if len(got.Scopes) != 2 {
		t.Errorf("Scopes len = %d, want 2", len(got.Scopes))
	}
	if got.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}
}

func TestKeyStore_UpdateNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	k := key.Key{
		ID:        "nonexistent-key",
		UserID:    "user-1",
		Hash:      []byte("hash"),
		Prefix:    "ak_nonexist12",
		CreatedAt: time.Now().UTC(),
	}

	err := keyStore.Update(ctx, k)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKeyStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-del-key",
		Email:  "delkey@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	k := key.Key{
		ID:        "key-del-1",
		UserID:    "user-del-key",
		Hash:      []byte("hash"),
		Prefix:    "ak_delkey1234",
		CreatedAt: time.Now().UTC(),
	}
	keyStore.Create(ctx, k)

	if err := keyStore.Delete(ctx, k.ID); err != nil {
		t.Fatalf("delete key: %v", err)
	}

	_, err := keyStore.GetByID(ctx, k.ID)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKeyStore_DeleteNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	err := keyStore.Delete(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKeyStore_RevokeNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	err := keyStore.Revoke(ctx, "nonexistent", time.Now())
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKeyStore_GetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	_, err := keyStore.GetByID(ctx, "nonexistent")
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestKeyStore_GetByPrefixNoResults(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	keys, err := keyStore.Get(ctx, "nonexistent_prefix")
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected empty slice, got %d keys", len(keys))
	}
}

func TestKeyStore_WithNullScopes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)
	ctx := context.Background()

	user := ports.User{
		ID:     "user-null-scopes",
		Email:  "nullscopes@example.com",
		PlanID: "free",
		Status: "active",
	}
	userStore.Create(ctx, user)

	k := key.Key{
		ID:        "key-null-scopes",
		UserID:    "user-null-scopes",
		Hash:      []byte("hash"),
		Prefix:    "ak_nullscope1",
		Name:      "No Scopes",
		Scopes:    nil, // No scopes
		CreatedAt: time.Now().UTC(),
	}
	keyStore.Create(ctx, k)

	got, err := keyStore.GetByID(ctx, k.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}

	if len(got.Scopes) != 0 {
		t.Errorf("expected empty scopes, got %v", got.Scopes)
	}
}

// -----------------------------------------------------------------------------
// RateLimitStore Additional Tests
// -----------------------------------------------------------------------------

func TestRateLimitStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRateLimitStore(db)
	ctx := context.Background()

	state := ratelimit.WindowState{
		Count:     10,
		WindowEnd: time.Now().UTC().Add(time.Minute),
		BurstUsed: 5,
	}

	if err := store.Set(ctx, "key-delete", state); err != nil {
		t.Fatalf("set state: %v", err)
	}

	if err := store.Delete(ctx, "key-delete"); err != nil {
		t.Fatalf("delete state: %v", err)
	}

	// Verify deleted - should return empty state
	got, err := store.Get(ctx, "key-delete")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	if got.Count != 0 {
		t.Errorf("Count = %d, want 0", got.Count)
	}
}

func TestRateLimitStore_Cleanup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRateLimitStore(db)
	ctx := context.Background()

	// Create expired state (window_end is more than 1 hour ago)
	expiredState := ratelimit.WindowState{
		Count:     10,
		WindowEnd: time.Now().UTC().Add(-2 * time.Hour), // 2 hours ago
		BurstUsed: 5,
	}
	store.Set(ctx, "key-expired", expiredState)

	// Create valid state
	validState := ratelimit.WindowState{
		Count:     20,
		WindowEnd: time.Now().UTC().Add(time.Minute),
		BurstUsed: 10,
	}
	store.Set(ctx, "key-valid", validState)

	// Cleanup
	deleted, err := store.Cleanup(ctx)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify expired is gone
	got, err := store.Get(ctx, "key-expired")
	if err != nil {
		t.Fatalf("get expired: %v", err)
	}
	if got.Count != 0 {
		t.Errorf("expired Count = %d, want 0", got.Count)
	}

	// Verify valid still exists
	got, err = store.Get(ctx, "key-valid")
	if err != nil {
		t.Fatalf("get valid: %v", err)
	}
	if got.Count != 20 {
		t.Errorf("valid Count = %d, want 20", got.Count)
	}
}

// -----------------------------------------------------------------------------
// UsageStore Additional Tests
// -----------------------------------------------------------------------------

func TestUsageStore_RecordBatchEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	// Empty batch should succeed
	if err := store.RecordBatch(ctx, []usage.Event{}); err != nil {
		t.Fatalf("record empty batch: %v", err)
	}
}

func TestUsageStore_GetHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create events in different months
	events := []usage.Event{
		{
			ID:             "evt-hist-1",
			KeyID:          "key-1",
			UserID:         "user-hist",
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
			ID:             "evt-hist-2",
			KeyID:          "key-1",
			UserID:         "user-hist",
			Method:         "POST",
			Path:           "/api/data",
			StatusCode:     500, // Error
			LatencyMs:      100,
			RequestBytes:   200,
			ResponseBytes:  50,
			CostMultiplier: 2.0,
			Timestamp:      now,
		},
	}

	if err := store.RecordBatch(ctx, events); err != nil {
		t.Fatalf("record batch: %v", err)
	}

	history, err := store.GetHistory(ctx, "user-hist", 6)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("len = %d, want 1", len(history))
	}

	if history[0].RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2", history[0].RequestCount)
	}
	if history[0].ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", history[0].ErrorCount)
	}
}

func TestUsageStore_SaveSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	now := time.Now().UTC()
	summary := usage.Summary{
		UserID:       "user-summary",
		PeriodStart:  now.Truncate(time.Hour * 24),
		PeriodEnd:    now.Truncate(time.Hour * 24).Add(time.Hour * 24),
		RequestCount: 100,
		ComputeUnits: 500.0,
		BytesIn:      10000,
		BytesOut:     50000,
		ErrorCount:   5,
		AvgLatencyMs: 45,
	}

	if err := store.SaveSummary(ctx, summary); err != nil {
		t.Fatalf("save summary: %v", err)
	}

	// Save another summary for the same period (should update)
	summary2 := usage.Summary{
		UserID:       "user-summary",
		PeriodStart:  now.Truncate(time.Hour * 24),
		PeriodEnd:    now.Truncate(time.Hour * 24).Add(time.Hour * 24),
		RequestCount: 50,
		ComputeUnits: 250.0,
		BytesIn:      5000,
		BytesOut:     25000,
		ErrorCount:   2,
		AvgLatencyMs: 35,
	}

	if err := store.SaveSummary(ctx, summary2); err != nil {
		t.Fatalf("save summary2: %v", err)
	}
}

func TestUsageStore_Cleanup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create old and new events
	events := []usage.Event{
		{
			ID:             "evt-old",
			KeyID:          "key-1",
			UserID:         "user-cleanup",
			Method:         "GET",
			Path:           "/api/data",
			StatusCode:     200,
			LatencyMs:      50,
			CostMultiplier: 1.0,
			Timestamp:      now.Add(-48 * time.Hour), // 2 days ago
		},
		{
			ID:             "evt-new",
			KeyID:          "key-1",
			UserID:         "user-cleanup",
			Method:         "GET",
			Path:           "/api/data",
			StatusCode:     200,
			LatencyMs:      50,
			CostMultiplier: 1.0,
			Timestamp:      now,
		},
	}

	if err := store.RecordBatch(ctx, events); err != nil {
		t.Fatalf("record batch: %v", err)
	}

	// Cleanup events older than 1 day ago
	deleted, err := store.Cleanup(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only new event remains
	recent, err := store.GetRecentRequests(ctx, "user-cleanup", 10)
	if err != nil {
		t.Fatalf("get recent: %v", err)
	}

	if len(recent) != 1 {
		t.Errorf("len = %d, want 1", len(recent))
	}
	if recent[0].ID != "evt-new" {
		t.Errorf("ID = %s, want evt-new", recent[0].ID)
	}
}

func TestUsageStore_GetRecentRequestsWithOptionalFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUsageStore(db)
	ctx := context.Background()

	events := []usage.Event{
		{
			ID:             "evt-optional",
			KeyID:          "key-1",
			UserID:         "user-optional",
			Method:         "GET",
			Path:           "/api/data",
			StatusCode:     200,
			LatencyMs:      50,
			CostMultiplier: 1.0,
			IPAddress:      "192.168.1.1",
			UserAgent:      "TestAgent/1.0",
			Timestamp:      time.Now().UTC(),
		},
	}

	if err := store.RecordBatch(ctx, events); err != nil {
		t.Fatalf("record batch: %v", err)
	}

	recent, err := store.GetRecentRequests(ctx, "user-optional", 10)
	if err != nil {
		t.Fatalf("get recent: %v", err)
	}

	if len(recent) != 1 {
		t.Fatalf("len = %d, want 1", len(recent))
	}

	if recent[0].IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %s, want 192.168.1.1", recent[0].IPAddress)
	}
	if recent[0].UserAgent != "TestAgent/1.0" {
		t.Errorf("UserAgent = %s, want TestAgent/1.0", recent[0].UserAgent)
	}
}

// -----------------------------------------------------------------------------
// PlanStore Tests
// -----------------------------------------------------------------------------

func TestPlanStore_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewPlanStore(db)
	ctx := context.Background()

	plan := ports.Plan{
		ID:                 "plan-1",
		Name:               "Pro Plan",
		Description:        "Professional tier",
		RateLimitPerMinute: 1000,
		RequestsPerMonth:   100000,
		PriceMonthly:       4999, // cents
		OveragePrice:       1,    // cents
		IsDefault:          false,
		Enabled:            true,
	}

	if err := store.Create(ctx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	got, err := store.Get(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}

	if got.ID != plan.ID {
		t.Errorf("ID = %s, want %s", got.ID, plan.ID)
	}
	if got.Name != plan.Name {
		t.Errorf("Name = %s, want %s", got.Name, plan.Name)
	}
	if got.RateLimitPerMinute != plan.RateLimitPerMinute {
		t.Errorf("RateLimitPerMinute = %d, want %d", got.RateLimitPerMinute, plan.RateLimitPerMinute)
	}
	if got.PriceMonthly != plan.PriceMonthly {
		t.Errorf("PriceMonthly = %d, want %d", got.PriceMonthly, plan.PriceMonthly)
	}
}

func TestPlanStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewPlanStore(db)
	ctx := context.Background()

	// Create multiple plans with different prices (in cents)
	// Note: there's already a default "free" plan from migrations
	plans := []ports.Plan{
		{ID: "plan-pro", Name: "Pro", PriceMonthly: 4999, Enabled: true},
		{ID: "plan-ent", Name: "Enterprise", PriceMonthly: 19999, Enabled: true},
		{ID: "plan-disabled", Name: "Old Plan", PriceMonthly: 2500, Enabled: false},
	}

	for _, p := range plans {
		if err := store.Create(ctx, p); err != nil {
			t.Fatalf("create plan %s: %v", p.ID, err)
		}
	}

	// List returns only enabled plans (default free + 2 we added)
	result, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("len = %d, want 3 (enabled only: free + pro + enterprise)", len(result))
	}

	// Should be ordered by price ascending (free plan from seed is first)
	if result[0].Name != "Free" {
		t.Errorf("first plan = %s, want Free", result[0].Name)
	}
}

func TestPlanStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewPlanStore(db)
	ctx := context.Background()

	plan := ports.Plan{
		ID:           "plan-update",
		Name:         "Original",
		PriceMonthly: 1000, // cents
		Enabled:      true,
	}
	store.Create(ctx, plan)

	// Update
	plan.Name = "Updated"
	plan.PriceMonthly = 2000 // cents
	plan.Enabled = false

	if err := store.Update(ctx, plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	got, err := store.Get(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}

	if got.Name != "Updated" {
		t.Errorf("Name = %s, want Updated", got.Name)
	}
	if got.PriceMonthly != 2000 {
		t.Errorf("PriceMonthly = %d, want 2000", got.PriceMonthly)
	}
}

func TestPlanStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewPlanStore(db)
	ctx := context.Background()

	plan := ports.Plan{
		ID:      "plan-delete",
		Name:    "To Delete",
		Enabled: true,
	}
	store.Create(ctx, plan)

	if err := store.Delete(ctx, plan.ID); err != nil {
		t.Fatalf("delete plan: %v", err)
	}

	_, err := store.Get(ctx, plan.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestPlanStore_GetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewPlanStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

// -----------------------------------------------------------------------------
// SettingsStore Tests
// -----------------------------------------------------------------------------

func TestSettingsStore_SetAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	if err := store.Set(ctx, "app.name", "MyApp", false); err != nil {
		t.Fatalf("set setting: %v", err)
	}

	got, err := store.Get(ctx, "app.name")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}

	if got.Key != "app.name" {
		t.Errorf("Key = %s, want app.name", got.Key)
	}
	if got.Value != "MyApp" {
		t.Errorf("Value = %s, want MyApp", got.Value)
	}
	if got.Encrypted {
		t.Error("Encrypted should be false")
	}
}

func TestSettingsStore_SetEncrypted(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	if err := store.Set(ctx, "api.secret", "secret123", true); err != nil {
		t.Fatalf("set encrypted setting: %v", err)
	}

	got, err := store.Get(ctx, "api.secret")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}

	if !got.Encrypted {
		t.Error("Encrypted should be true")
	}
}

func TestSettingsStore_SetUpsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	// Initial set
	store.Set(ctx, "app.version", "1.0.0", false)

	// Upsert
	if err := store.Set(ctx, "app.version", "2.0.0", false); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}

	got, err := store.Get(ctx, "app.version")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}

	if got.Value != "2.0.0" {
		t.Errorf("Value = %s, want 2.0.0", got.Value)
	}
}

func TestSettingsStore_GetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent setting")
	}
}

func TestSettingsStore_GetAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	// Set multiple settings
	store.Set(ctx, "app.name", "MyApp", false)
	store.Set(ctx, "app.version", "1.0.0", false)
	store.Set(ctx, "api.url", "https://api.example.com", false)

	all, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all settings: %v", err)
	}

	// Note: there are default settings seeded by migrations, so count includes those
	// We verify our custom settings are present
	if len(all) < 3 {
		t.Errorf("len = %d, want at least 3", len(all))
	}

	if all["app.name"] != "MyApp" {
		t.Errorf("app.name = %s, want MyApp", all["app.name"])
	}
	if all["app.version"] != "1.0.0" {
		t.Errorf("app.version = %s, want 1.0.0", all["app.version"])
	}
	if all["api.url"] != "https://api.example.com" {
		t.Errorf("api.url = %s, want https://api.example.com", all["api.url"])
	}
}

func TestSettingsStore_GetByPrefix(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	// Set settings with different prefixes
	store.Set(ctx, "app.name", "MyApp", false)
	store.Set(ctx, "app.version", "1.0.0", false)
	store.Set(ctx, "api.url", "https://api.example.com", false)
	store.Set(ctx, "api.key", "secret", false)

	// Get by prefix
	appSettings, err := store.GetByPrefix(ctx, "app.")
	if err != nil {
		t.Fatalf("get by prefix: %v", err)
	}

	if len(appSettings) != 2 {
		t.Errorf("len = %d, want 2", len(appSettings))
	}

	if appSettings["app.name"] != "MyApp" {
		t.Errorf("app.name = %s, want MyApp", appSettings["app.name"])
	}
}

func TestSettingsStore_SetBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	batch := map[string]string{
		"batch.setting1": "value1",
		"batch.setting2": "value2",
		"batch.setting3": "value3",
	}

	if err := store.SetBatch(ctx, batch); err != nil {
		t.Fatalf("set batch: %v", err)
	}

	// Verify all were set
	for key, want := range batch {
		got, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get %s: %v", key, err)
		}
		if got.Value != want {
			t.Errorf("%s = %s, want %s", key, got.Value, want)
		}
	}
}

func TestSettingsStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	store.Set(ctx, "to.delete", "value", false)

	if err := store.Delete(ctx, "to.delete"); err != nil {
		t.Fatalf("delete setting: %v", err)
	}

	_, err := store.Get(ctx, "to.delete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

// -----------------------------------------------------------------------------
// Route and Upstream Additional Tests
// -----------------------------------------------------------------------------

func TestRouteStore_UpdateNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewRouteStore(db)
	ctx := context.Background()

	r := route.NewRoute("nonexistent", "Nonexistent", "/api/*", "up-1")
	err := store.Update(ctx, r)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpstreamStore_UpdateNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	u := route.NewUpstream("nonexistent", "Nonexistent", "https://example.com")
	err := store.Update(ctx, u)
	if err != sqlite.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRouteStore_CreateWithPathRewriteAndMethodOverride(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-rewrite", "Rewrite Route", "/api/v1/*", "up-1")
	r.PathRewrite = "/v2/$1"
	r.MethodOverride = "POST"

	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	got, err := store.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	if got.PathRewrite != "/v2/$1" {
		t.Errorf("PathRewrite = %s, want /v2/$1", got.PathRewrite)
	}
	if got.MethodOverride != "POST" {
		t.Errorf("MethodOverride = %s, want POST", got.MethodOverride)
	}
}

func TestRouteStore_CreateDuplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	upstreamStore := sqlite.NewUpstreamStore(db)
	ctx := context.Background()

	upstream := route.NewUpstream("up-1", "Upstream", "https://api.example.com")
	upstreamStore.Create(ctx, upstream)

	store := sqlite.NewRouteStore(db)

	r := route.NewRoute("route-dup", "Route", "/api/*", "up-1")
	if err := store.Create(ctx, r); err != nil {
		t.Fatalf("create route: %v", err)
	}

	// Try to create duplicate
	err := store.Create(ctx, r)
	if err == nil {
		t.Fatal("expected error for duplicate route")
	}
}

// -----------------------------------------------------------------------------
// EntitlementStore Tests
// -----------------------------------------------------------------------------

func TestEntitlementStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewEntitlementStore(db)
	ctx := context.Background()

	// Create - use unique name to avoid conflict with seed data
	e := entitlement.Entitlement{
		ID:           "ent-test-1",
		Name:         "test.custom.feature",
		DisplayName:  "Custom Test Feature",
		Description:  "A test entitlement for unit testing",
		Category:     entitlement.CategoryAPI,
		ValueType:    entitlement.ValueTypeBoolean,
		DefaultValue: "true",
		HeaderName:   "X-Test-Feature",
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := store.Create(ctx, e); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}

	// Get
	got, err := store.Get(ctx, e.ID)
	if err != nil {
		t.Fatalf("get entitlement: %v", err)
	}
	if got.Name != e.Name {
		t.Errorf("Name = %s, want %s", got.Name, e.Name)
	}
	if got.DisplayName != e.DisplayName {
		t.Errorf("DisplayName = %s, want %s", got.DisplayName, e.DisplayName)
	}

	// GetByName
	gotByName, err := store.GetByName(ctx, e.Name)
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if gotByName.ID != e.ID {
		t.Errorf("GetByName ID = %s, want %s", gotByName.ID, e.ID)
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one entitlement")
	}

	// ListEnabled
	enabled, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) == 0 {
		t.Error("expected at least one enabled entitlement")
	}

	// Update
	e.DisplayName = "Updated Display Name"
	if err := store.Update(ctx, e); err != nil {
		t.Fatalf("update entitlement: %v", err)
	}
	updated, _ := store.Get(ctx, e.ID)
	if updated.DisplayName != "Updated Display Name" {
		t.Errorf("DisplayName = %s, want Updated Display Name", updated.DisplayName)
	}

	// Delete
	if err := store.Delete(ctx, e.ID); err != nil {
		t.Fatalf("delete entitlement: %v", err)
	}
	_, err = store.Get(ctx, e.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// -----------------------------------------------------------------------------
// PlanEntitlementStore Tests
// -----------------------------------------------------------------------------

func TestPlanEntitlementStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First create an entitlement
	entStore := sqlite.NewEntitlementStore(db)
	ctx := context.Background()

	e := entitlement.Entitlement{
		ID:           "ent-pe-1",
		Name:         "api.ratelimit",
		DisplayName:  "Rate Limit",
		Category:     entitlement.CategoryAPI,
		ValueType:    entitlement.ValueTypeNumber,
		DefaultValue: "1000",
		Enabled:      true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	entStore.Create(ctx, e)

	store := sqlite.NewPlanEntitlementStore(db)

	// Create
	pe := entitlement.PlanEntitlement{
		ID:            "pe-1",
		PlanID:        "plan-1",
		EntitlementID: "ent-pe-1",
		Value:         "5000",
		Notes:         "Premium plan rate limit",
		Enabled:       true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := store.Create(ctx, pe); err != nil {
		t.Fatalf("create plan entitlement: %v", err)
	}

	// Get
	got, err := store.Get(ctx, pe.ID)
	if err != nil {
		t.Fatalf("get plan entitlement: %v", err)
	}
	if got.Value != pe.Value {
		t.Errorf("Value = %s, want %s", got.Value, pe.Value)
	}

	// GetByPlanAndEntitlement
	gotByPE, err := store.GetByPlanAndEntitlement(ctx, pe.PlanID, pe.EntitlementID)
	if err != nil {
		t.Fatalf("get by plan and entitlement: %v", err)
	}
	if gotByPE.ID != pe.ID {
		t.Errorf("ID = %s, want %s", gotByPE.ID, pe.ID)
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one plan entitlement")
	}

	// ListByPlan
	byPlan, err := store.ListByPlan(ctx, pe.PlanID)
	if err != nil {
		t.Fatalf("list by plan: %v", err)
	}
	if len(byPlan) == 0 {
		t.Error("expected at least one plan entitlement for plan")
	}

	// ListByEntitlement
	byEnt, err := store.ListByEntitlement(ctx, pe.EntitlementID)
	if err != nil {
		t.Fatalf("list by entitlement: %v", err)
	}
	if len(byEnt) == 0 {
		t.Error("expected at least one plan entitlement for entitlement")
	}

	// Update
	pe.Value = "10000"
	if err := store.Update(ctx, pe); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := store.Get(ctx, pe.ID)
	if updated.Value != "10000" {
		t.Errorf("Value = %s, want 10000", updated.Value)
	}

	// Delete
	if err := store.Delete(ctx, pe.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = store.Get(ctx, pe.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// -----------------------------------------------------------------------------
// WebhookStore Tests
// -----------------------------------------------------------------------------

func TestWebhookStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewWebhookStore(db.DB)
	ctx := context.Background()

	// Create
	wh := webhook.Webhook{
		ID:          "wh-1",
		UserID:      "user-1",
		Name:        "Test Webhook",
		Description: "Test webhook description",
		URL:         "https://example.com/webhook",
		Secret:      "secret123",
		Events:      []webhook.EventType{webhook.EventUsageThreshold, webhook.EventKeyCreated},
		RetryCount:  3,
		TimeoutMS:   5000,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := store.Create(ctx, wh); err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	// Get
	got, err := store.Get(ctx, wh.ID)
	if err != nil {
		t.Fatalf("get webhook: %v", err)
	}
	if got.Name != wh.Name {
		t.Errorf("Name = %s, want %s", got.Name, wh.Name)
	}
	if len(got.Events) != 2 {
		t.Errorf("Events len = %d, want 2", len(got.Events))
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list webhooks: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one webhook")
	}

	// ListByUser
	byUser, err := store.ListByUser(ctx, wh.UserID)
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(byUser) == 0 {
		t.Error("expected at least one webhook for user")
	}

	// ListEnabled
	enabled, err := store.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) == 0 {
		t.Error("expected at least one enabled webhook")
	}

	// ListForEvent
	forEvent, err := store.ListForEvent(ctx, webhook.EventUsageThreshold)
	if err != nil {
		t.Fatalf("list for event: %v", err)
	}
	if len(forEvent) == 0 {
		t.Error("expected at least one webhook for event")
	}

	// Update
	wh.Name = "Updated Webhook"
	if err := store.Update(ctx, wh); err != nil {
		t.Fatalf("update webhook: %v", err)
	}
	updated, _ := store.Get(ctx, wh.ID)
	if updated.Name != "Updated Webhook" {
		t.Errorf("Name = %s, want Updated Webhook", updated.Name)
	}

	// Delete
	if err := store.Delete(ctx, wh.ID); err != nil {
		t.Fatalf("delete webhook: %v", err)
	}
	got, err = store.Get(ctx, wh.ID)
	if got.ID != "" {
		t.Error("expected empty webhook after delete")
	}
}

// -----------------------------------------------------------------------------
// DeliveryStore Tests
// -----------------------------------------------------------------------------

func TestDeliveryStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First create a webhook
	whStore := sqlite.NewWebhookStore(db.DB)
	ctx := context.Background()

	wh := webhook.Webhook{
		ID:      "wh-del-1",
		UserID:  "user-1",
		Name:    "Delivery Test",
		URL:     "https://example.com/webhook",
		Events:  []webhook.EventType{webhook.EventTest},
		Enabled: true,
	}
	whStore.Create(ctx, wh)

	store := sqlite.NewDeliveryStore(db.DB)

	// Create
	now := time.Now().UTC()
	nextRetry := now.Add(time.Hour)
	del := webhook.Delivery{
		ID:         "del-1",
		WebhookID:  "wh-del-1",
		EventID:    "evt-1",
		Payload:    `{"test": true}`,
		Status:     webhook.DeliveryPending,
		Attempt:    0,
		NextRetry:  &nextRetry,
		CreatedAt:  now,
	}

	if err := store.Create(ctx, del); err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	// Get
	got, err := store.Get(ctx, del.ID)
	if err != nil {
		t.Fatalf("get delivery: %v", err)
	}
	if got.WebhookID != del.WebhookID {
		t.Errorf("WebhookID = %s, want %s", got.WebhookID, del.WebhookID)
	}
	if got.Status != webhook.DeliveryPending {
		t.Errorf("Status = %s, want %s", got.Status, webhook.DeliveryPending)
	}

	// List
	list, err := store.List(ctx, del.WebhookID, 10)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one delivery")
	}

	// ListPending
	pending, err := store.ListPending(ctx, now.Add(2*time.Hour), 10)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) == 0 {
		t.Error("expected at least one pending delivery")
	}

	// Update
	del.Status = webhook.DeliverySuccess
	del.StatusCode = 200
	del.ResponseBody = `{"ok": true}`
	if err := store.Update(ctx, del); err != nil {
		t.Fatalf("update delivery: %v", err)
	}
	updated, _ := store.Get(ctx, del.ID)
	if updated.Status != webhook.DeliverySuccess {
		t.Errorf("Status = %s, want %s", updated.Status, webhook.DeliverySuccess)
	}
}

// -----------------------------------------------------------------------------
// InvoiceStore Tests
// -----------------------------------------------------------------------------

func TestInvoiceStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewInvoiceStore(db)
	ctx := context.Background()

	// Create
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)
	inv := billing.Invoice{
		ID:          "inv-1",
		UserID:      "user-1",
		ProviderID:  "pi_123",
		Provider:    "stripe",
		PeriodStart: now,
		PeriodEnd:   periodEnd,
		Total:       9900,
		Currency:    "USD",
		Status:      billing.InvoiceStatusOpen,
		CreatedAt:   now,
	}

	if err := store.Create(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	// ListByUser
	list, err := store.ListByUser(ctx, inv.UserID, 10)
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one invoice")
	}
	if list[0].Total != inv.Total {
		t.Errorf("Total = %d, want %d", list[0].Total, inv.Total)
	}

	// UpdateStatus
	paidAt := time.Now()
	if err := store.UpdateStatus(ctx, inv.ID, billing.InvoiceStatusPaid, &paidAt); err != nil {
		t.Fatalf("update status: %v", err)
	}
	updated, _ := store.ListByUser(ctx, inv.UserID, 10)
	if len(updated) == 0 || updated[0].Status != billing.InvoiceStatusPaid {
		t.Error("expected status to be paid")
	}
}

// -----------------------------------------------------------------------------
// SubscriptionStore Tests
// -----------------------------------------------------------------------------

func TestSubscriptionStore_CRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSubscriptionStore(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	periodEnd := now.Add(30 * 24 * time.Hour)

	// Create
	sub := billing.Subscription{
		ID:                 "sub-test-1",
		UserID:             "user-1",
		PlanID:             "plan-1",
		Provider:           "stripe",
		ProviderID:         "sub_stripe123",
		ProviderItemID:     "si_item123",
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CancelAtPeriodEnd:  false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := store.Create(ctx, sub); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	// Get
	got, err := store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if got.UserID != sub.UserID {
		t.Errorf("UserID = %s, want %s", got.UserID, sub.UserID)
	}
	if got.PlanID != sub.PlanID {
		t.Errorf("PlanID = %s, want %s", got.PlanID, sub.PlanID)
	}
	if got.Status != billing.SubscriptionStatusActive {
		t.Errorf("Status = %s, want %s", got.Status, billing.SubscriptionStatusActive)
	}
	if got.Provider != "stripe" {
		t.Errorf("Provider = %s, want stripe", got.Provider)
	}
	if got.ProviderID != "sub_stripe123" {
		t.Errorf("ProviderID = %s, want sub_stripe123", got.ProviderID)
	}

	// GetByUser
	gotByUser, err := store.GetByUser(ctx, sub.UserID)
	if err != nil {
		t.Fatalf("get by user: %v", err)
	}
	if gotByUser.ID != sub.ID {
		t.Errorf("GetByUser ID = %s, want %s", gotByUser.ID, sub.ID)
	}

	// Update
	sub.Status = billing.SubscriptionStatusCancelled
	cancelledAt := time.Now().UTC()
	sub.CancelledAt = &cancelledAt
	sub.CancelAtPeriodEnd = true
	if err := store.Update(ctx, sub); err != nil {
		t.Fatalf("update subscription: %v", err)
	}

	updatedSub, _ := store.Get(ctx, sub.ID)
	if updatedSub.Status != billing.SubscriptionStatusCancelled {
		t.Errorf("Status = %s, want %s", updatedSub.Status, billing.SubscriptionStatusCancelled)
	}
	if !updatedSub.CancelAtPeriodEnd {
		t.Error("expected CancelAtPeriodEnd to be true")
	}
	if updatedSub.CancelledAt == nil {
		t.Error("expected CancelledAt to be set")
	}
}

func TestSubscriptionStore_GetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSubscriptionStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent subscription")
	}
}

func TestSubscriptionStore_GetByUserNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSubscriptionStore(db)
	ctx := context.Background()

	_, err := store.GetByUser(ctx, "nonexistent-user")
	if err == nil {
		t.Error("expected error for nonexistent user subscription")
	}
}

func TestSubscriptionStore_UpdateNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := sqlite.NewSubscriptionStore(db)
	ctx := context.Background()

	sub := billing.Subscription{
		ID:     "nonexistent",
		UserID: "user-1",
		PlanID: "plan-1",
		Status: billing.SubscriptionStatusActive,
	}

	err := store.Update(ctx, sub)
	if err == nil {
		t.Error("expected error updating nonexistent subscription")
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
