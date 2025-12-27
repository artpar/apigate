package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/clock"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/plan"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/bcrypt"
)

var baseTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func TestProxyService_Handle_ValidRequest(t *testing.T) {
	// Arrange
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Seed test data
	// Key must be: prefix (3) + 64 hex chars = 67 chars minimum
	rawKey := "ak_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12], // "ak_012345678"
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Email:  "test@example.com",
		PlanID: "free",
		Status: "active",
	})

	// Act
	req := proxy.Request{
		APIKey:    rawKey,
		Method:    "GET",
		Path:      "/api/data",
		RemoteIP:  "1.2.3.4",
		UserAgent: "test-agent",
	}
	result := svc.Handle(ctx, req)

	// Assert
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
	if result.Response.Status != 200 {
		t.Errorf("status = %d, want 200", result.Response.Status)
	}
	if result.Auth == nil {
		t.Fatal("expected auth context")
	}
	if result.Auth.UserID != "user-1" {
		t.Errorf("userID = %s, want user-1", result.Auth.UserID)
	}

	// Verify usage was recorded
	events := stores.usage.Drain()
	if len(events) != 1 {
		t.Fatalf("expected 1 usage event, got %d", len(events))
	}
	if events[0].KeyID != "key-1" {
		t.Errorf("event keyID = %s, want key-1", events[0].KeyID)
	}
}

func TestProxyService_Handle_InvalidKey(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProxyService()

	req := proxy.Request{
		APIKey: "ak_invalid0123456789abcdef0123456789abcdef0123456789abcdef01234",
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for invalid key")
	}
	if result.Error.Status != 401 {
		t.Errorf("status = %d, want 401", result.Error.Status)
	}
}

func TestProxyService_Handle_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create expired key (prefix + 64 hex chars = 67 total)
	rawKey := "ak_1111111111111111111111111111111111111111111111111111111111111111"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	expiredAt := baseTime.Add(-time.Hour)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		ExpiresAt: &expiredAt,
		CreatedAt: baseTime.Add(-24 * time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Status: "active",
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for expired key")
	}
	if result.Error.Code != key.ReasonExpired {
		t.Errorf("code = %s, want %s", result.Error.Code, key.ReasonExpired)
	}
}

func TestProxyService_Handle_RateLimited(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// Create key with tight rate limit (prefix + 64 hex chars = 67 total)
	rawKey := "ak_2222222222222222222222222222222222222222222222222222222222222222"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		PlanID: "limited",
		Status: "active",
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}

	// Make requests until rate limited
	// Plan "limited" has rate limit of 2 per minute + 2 burst = 4 total
	for i := 0; i < 10; i++ {
		result := svc.Handle(ctx, req)
		if result.Error != nil && result.Error.Code == "rate_limit_exceeded" {
			// Expected - rate limited after a few requests
			return
		}
	}

	t.Error("expected to be rate limited after exceeding limit")
}

func TestProxyService_Handle_SuspendedUser(t *testing.T) {
	ctx := context.Background()
	svc, stores := newTestProxyService()

	// prefix + 64 hex chars = 67 total
	rawKey := "ak_3333333333333333333333333333333333333333333333333333333333333333"
	keyHash, _ := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)

	stores.keys.Create(ctx, key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		Hash:      keyHash,
		Prefix:    rawKey[:12],
		CreatedAt: baseTime.Add(-time.Hour),
	})

	stores.users.Create(ctx, ports.User{
		ID:     "user-1",
		Status: "suspended", // Suspended user
	})

	req := proxy.Request{
		APIKey: rawKey,
		Method: "GET",
		Path:   "/api/data",
	}
	result := svc.Handle(ctx, req)

	if result.Error == nil {
		t.Fatal("expected error for suspended user")
	}
	if result.Error.Code != "user_suspended" {
		t.Errorf("code = %s, want user_suspended", result.Error.Code)
	}
}

// Test helpers

type testStores struct {
	keys      *memory.KeyStore
	users     *memory.UserStore
	rateLimit *memory.RateLimitStore
	usage     *testUsageRecorder
}

type testUsageRecorder struct {
	events []usage.Event
}

func (r *testUsageRecorder) Record(e usage.Event) {
	r.events = append(r.events, e)
}

func (r *testUsageRecorder) Flush(ctx context.Context) error {
	return nil
}

func (r *testUsageRecorder) Close() error {
	return nil
}

func (r *testUsageRecorder) Drain() []usage.Event {
	events := r.events
	r.events = nil
	return events
}

type testUpstream struct{}

func (u *testUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
	return proxy.Response{
		Status:    200,
		Body:      []byte(`{"ok":true}`),
		LatencyMs: 50,
	}, nil
}

func (u *testUpstream) HealthCheck(ctx context.Context) error {
	return nil
}

type testIDGen struct {
	counter int
}

func (g *testIDGen) New() string {
	g.counter++
	return "id-" + itoa(g.counter)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func newTestProxyService() (*app.ProxyService, *testStores) {
	stores := &testStores{
		keys:      memory.NewKeyStore(),
		users:     memory.NewUserStore(),
		rateLimit: memory.NewRateLimitStore(),
		usage:     &testUsageRecorder{},
	}

	deps := app.ProxyDeps{
		Keys:      stores.keys,
		Users:     stores.users,
		RateLimit: stores.rateLimit,
		Usage:     stores.usage,
		Upstream:  &testUpstream{},
		Clock:     clock.NewFake(baseTime),
		IDGen:     &testIDGen{},
	}

	cfg := app.ProxyConfig{
		KeyPrefix:  "ak_",
		RateBurst:  2,
		RateWindow: 60,
		Plans: []plan.Plan{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60, RequestsPerMonth: 1000},
			{ID: "limited", Name: "Limited", RateLimitPerMinute: 2, RequestsPerMonth: 100},
		},
	}

	return app.NewProxyService(deps, cfg), stores
}
