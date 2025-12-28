package sqlite_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/ratelimit"
	"github.com/artpar/apigate/domain/route"
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
// Helpers
// -----------------------------------------------------------------------------

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
