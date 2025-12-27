# Developer Knowledge Transfer

## Quick Start for New Developers

```bash
# Clone and setup
git clone <repo>
cd apigate

# Run tests (no external dependencies needed)
go test ./domain/...           # Pure unit tests - instant
go test ./app/...              # App tests with in-memory adapters
go test -short ./...           # All non-integration tests

# Run with integration tests
go test ./adapters/sqlite/...  # Requires SQLite
go test ./...                  # Everything

# Run locally
cp configs/apigate.example.yaml apigate.yaml
make dev
```

---

## Core Concepts You Must Understand

### 1. Values as Boundaries

**The #1 rule**: Business logic lives in `domain/` and is **pure**.

```go
// GOOD: Pure function in domain/
func CalculateOverage(usage, included, price int64) int64 {
    if usage <= included {
        return 0
    }
    return (usage - included) * price
}

// BAD: Side effect in domain/ (DON'T DO THIS)
func CalculateOverage(userID string, db *sql.DB) int64 {
    usage := db.Query("SELECT ...") // NO! Side effect!
    // ...
}
```

**Why?**
- Pure functions are trivially testable (no mocks)
- Pure functions are predictable (same input → same output)
- Pure functions are composable
- Bugs are easier to find and fix

### 2. Dependency Injection via Ports

External systems are accessed through interfaces defined in `ports/`:

```go
// ports/clock.go
type Clock interface {
    Now() time.Time
}

// adapters/clock/real.go
type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now() }

// adapters/clock/fake.go
type FakeClock struct{ current time.Time }
func (f *FakeClock) Now() time.Time { return f.current }
func (f *FakeClock) Advance(d time.Duration) { f.current = f.current.Add(d) }
```

In tests, inject `FakeClock`. In production, inject `RealClock`.

### 3. The Composition Root

All wiring happens in `main.go`:

```go
func main() {
    cfg := config.MustLoad("apigate.yaml")

    // Create adapters
    clock := clock.Real{}
    random := random.Real{}
    db := sqlite.MustOpen(cfg.Database.DSN)
    keyStore := sqlite.NewKeyStore(db)
    userStore := sqlite.NewUserStore(db)
    usageStore := sqlite.NewUsageStore(db)
    billing := stripe.NewClient(cfg.Billing.StripeKey)
    upstream := http.NewUpstreamClient(cfg.Upstream.URL)

    // Create application services
    proxySvc := app.NewProxyService(keyStore, userStore, usageStore, clock, upstream, cfg)
    adminSvc := app.NewAdminService(keyStore, userStore, usageStore, clock, random, cfg)
    portalSvc := app.NewPortalService(keyStore, userStore, billing, clock, random, cfg)

    // Create HTTP handlers
    proxyHandler := httpx.NewProxyHandler(proxySvc)
    adminHandler := httpx.NewAdminHandler(adminSvc, cfg.Auth.AdminSecret)
    portalHandler := httpx.NewPortalHandler(portalSvc)

    // Start servers...
}
```

**Never** create dependencies inside services. Always inject them.

---

## Directory Guide

### `domain/` - The Functional Core

```
domain/
├── key/           # API key value type and validation
├── plan/          # Plan value type and quota logic
├── usage/         # Usage events and aggregation
├── ratelimit/     # Rate limiting algorithms
├── billing/       # Invoice and pricing calculations
└── proxy/         # Request/response value types
```

**Rules for domain/**:
- ✅ Value types (structs with data)
- ✅ Pure functions (no side effects)
- ✅ No imports from `adapters/` or `app/`
- ✅ No `context.Context` (no cancellation needed for pure functions)
- ✅ `time.Time` as parameter, never `time.Now()`
- ❌ No database, HTTP, file I/O
- ❌ No logging (pure functions don't need it)
- ❌ No global state

### `ports/` - The Contracts

```
ports/
├── clock.go       # Time abstraction
├── random.go      # Randomness abstraction
├── keystore.go    # Key persistence interface
├── userstore.go   # User persistence interface
├── usagestore.go  # Usage persistence interface
├── billing.go     # Payment provider interface
└── upstream.go    # HTTP upstream interface
```

**Rules for ports/**:
- ✅ Interface definitions only
- ✅ Types used in interfaces
- ❌ No implementations
- ❌ No business logic

### `adapters/` - The I/O Layer

```
adapters/
├── clock/         # Real and fake time
├── random/        # Real and fake randomness
├── sqlite/        # SQLite store implementations
├── postgres/      # PostgreSQL store implementations
├── memory/        # In-memory implementations (testing!)
├── stripe/        # Stripe billing implementation
└── http/          # HTTP handlers and clients
```

**Rules for adapters/**:
- ✅ Implement interfaces from `ports/`
- ✅ Handle I/O operations
- ✅ Use `context.Context` for cancellation
- ✅ Logging allowed here
- ❌ No business logic (delegate to domain/)

### `app/` - The Orchestration Layer

```
app/
├── proxy.go       # Proxy service (orchestrates key→ratelimit→forward→meter)
├── admin.go       # Admin operations
├── portal.go      # Portal operations
└── billing.go     # Billing orchestration
```

**Rules for app/**:
- ✅ Coordinates domain functions and adapters
- ✅ Depends on ports/ interfaces (not concrete adapters)
- ✅ Contains "glue" logic
- ✅ Transaction boundaries defined here
- ❌ No HTTP-specific code (that's in adapters/http/)

---

## Testing Guide

### Test File Naming

```
thing.go           # Implementation
thing_test.go      # Tests (same package - white box)
thing_ext_test.go  # Tests (different package - black box)
```

### Test Categories

```bash
# Unit tests (domain/) - no external deps, instant
go test ./domain/...

# Application tests (app/) - uses memory adapters
go test ./app/...

# Integration tests - uses real DB
go test -tags=integration ./adapters/sqlite/...

# E2E tests - full system
go test -tags=e2e ./test/e2e/...

# All fast tests
go test -short ./...

# All tests
go test ./...
```

### Writing Domain Tests

```go
// domain/ratelimit/window_test.go
package ratelimit_test  // External test package for black-box testing

import (
    "testing"
    "time"

    "github.com/artpar/apigate/domain/ratelimit"
    "github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
    tests := []struct {
        name      string
        state     ratelimit.WindowState
        limit     int
        now       time.Time
        wantAllow bool
        wantRem   int
    }{
        {
            name:      "allows when under limit",
            state:     ratelimit.WindowState{Count: 5, WindowEnd: futureTime},
            limit:     10,
            now:       baseTime,
            wantAllow: true,
            wantRem:   4,
        },
        {
            name:      "denies when at limit",
            state:     ratelimit.WindowState{Count: 10, WindowEnd: futureTime},
            limit:     10,
            now:       baseTime,
            wantAllow: false,
            wantRem:   0,
        },
        {
            name:      "resets expired window",
            state:     ratelimit.WindowState{Count: 100, WindowEnd: pastTime},
            limit:     10,
            now:       baseTime,
            wantAllow: true,
            wantRem:   9,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ratelimit.Check(tt.state, tt.limit, tt.now)
            assert.Equal(t, tt.wantAllow, result.Allowed)
            assert.Equal(t, tt.wantRem, result.Remaining)
        })
    }
}

var (
    baseTime   = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
    pastTime   = baseTime.Add(-time.Hour)
    futureTime = baseTime.Add(time.Hour)
)
```

### Writing Application Tests

```go
// app/proxy_test.go
package app_test

import (
    "context"
    "testing"

    "github.com/artpar/apigate/adapters/clock"
    "github.com/artpar/apigate/adapters/memory"
    "github.com/artpar/apigate/app"
    "github.com/artpar/apigate/domain/key"
    "github.com/artpar/apigate/domain/proxy"
    "github.com/stretchr/testify/assert"
)

func TestProxyService_Handle(t *testing.T) {
    // Arrange
    ctx := context.Background()
    clk := clock.NewFake(baseTime)
    keys := memory.NewKeyStore()
    users := memory.NewUserStore()
    usage := memory.NewUsageStore()
    upstream := &fakeUpstream{response: proxy.Response{Status: 200, Body: []byte("ok")}}

    svc := app.NewProxyService(app.ProxyDeps{
        Keys:     keys,
        Users:    users,
        Usage:    usage,
        Clock:    clk,
        Upstream: upstream,
        Config:   testConfig,
    })

    // Seed data
    testKey := key.Key{
        ID:     "key-1",
        UserID: "user-1",
        Prefix: "ak_testtest",
        Hash:   hashKey("ak_testtest1234567890"),
    }
    keys.Create(ctx, testKey)
    users.Create(ctx, user.User{ID: "user-1", PlanID: "free", Status: "active"})

    // Act
    req := proxy.Request{
        APIKey: "ak_testtest1234567890",
        Method: "GET",
        Path:   "/api/resource",
    }
    resp, err := svc.Handle(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.Status)
    assert.Equal(t, "ok", string(resp.Body))

    // Verify side effects
    events := usage.Drain()
    assert.Len(t, events, 1)
    assert.Equal(t, "key-1", events[0].KeyID)
}

type fakeUpstream struct {
    response proxy.Response
    err      error
}

func (f *fakeUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
    return f.response, f.err
}
```

---

## Common Patterns

### Pattern: Result Types

Instead of returning errors for expected outcomes, use result types:

```go
// domain/key/validate.go
type ValidationResult struct {
    Valid  bool
    Reason string  // Only set if Valid=false
    Key    Key     // Only set if Valid=true
}

func Validate(k Key, now time.Time) ValidationResult {
    if k.RevokedAt != nil {
        return ValidationResult{Valid: false, Reason: "key_revoked"}
    }
    if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
        return ValidationResult{Valid: false, Reason: "key_expired"}
    }
    return ValidationResult{Valid: true, Key: k}
}
```

### Pattern: Functional Options

For configurable components:

```go
// app/proxy.go
type ProxyOption func(*ProxyService)

func WithRateLimitBurst(n int) ProxyOption {
    return func(s *ProxyService) {
        s.rateLimitBurst = n
    }
}

func NewProxyService(deps ProxyDeps, opts ...ProxyOption) *ProxyService {
    s := &ProxyService{
        deps:           deps,
        rateLimitBurst: 10, // default
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### Pattern: Event Sourcing for Usage

Usage is append-only, never mutated:

```go
// domain/usage/event.go
type Event struct {
    ID             string
    KeyID          string
    UserID         string
    Method         string
    Path           string
    StatusCode     int
    LatencyMs      int64
    RequestBytes   int64
    ResponseBytes  int64
    CostMultiplier float64
    Timestamp      time.Time
}

// Events are immutable - create new, never modify
func NewEvent(keyID, userID, method, path string, status int, latency time.Duration, reqSize, respSize int64, cost float64, ts time.Time) Event {
    return Event{
        ID:             uuid.New().String(),
        KeyID:          keyID,
        UserID:         userID,
        Method:         method,
        Path:           path,
        StatusCode:     status,
        LatencyMs:      latency.Milliseconds(),
        RequestBytes:   reqSize,
        ResponseBytes:  respSize,
        CostMultiplier: cost,
        Timestamp:      ts,
    }
}
```

---

## Debugging Guide

### Logging Levels

```
DEBUG  Detailed tracing (request/response bodies, SQL queries)
INFO   Normal operations (server started, request handled)
WARN   Recoverable issues (rate limit hit, retrying)
ERROR  Failures requiring attention (DB connection lost, Stripe error)
```

### Request Tracing

Each request gets a trace ID:

```go
// adapters/http/middleware.go
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.New().String()[:8]
        }
        ctx := context.WithValue(r.Context(), traceIDKey, traceID)
        w.Header().Set("X-Trace-ID", traceID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Common Issues

| Symptom | Likely Cause | Solution |
|---------|--------------|----------|
| "key not found" but key exists | Key prefix mismatch | Check first 12 chars match |
| Rate limit not resetting | Clock not advancing | Check `Clock.Now()` |
| Usage not recorded | Channel full | Check buffer size, increase |
| Stripe webhook failing | Wrong secret | Check `STRIPE_WEBHOOK_SECRET` |

---

## Making Changes

### Adding a New Domain Concept

1. Create `domain/newconcept/types.go` - value types
2. Create `domain/newconcept/logic.go` - pure functions
3. Create `domain/newconcept/logic_test.go` - tests
4. Run `go test ./domain/newconcept/...`

### Adding a New Port

1. Define interface in `ports/newport.go`
2. Create `adapters/memory/newport.go` for testing
3. Create `adapters/real/newport.go` for production
4. Inject in `main.go`

### Adding a New API Endpoint

1. Add to HTTP handler in `adapters/http/`
2. Add service method in `app/`
3. Wire domain logic
4. Add tests at each level

### Database Migrations

1. Create `migrations/XXX_description.sql`
2. Update migration runner
3. Test with fresh DB
4. Test upgrade from previous version

---

## Performance Considerations

### Hot Path (Proxy)

The proxy path must be fast. Current design:

```
API Key Lookup     ~1ms  (Redis cache in future)
Rate Limit Check   ~0μs  (in-memory)
Upstream Forward   var   (network)
Usage Record       ~0μs  (async channel send)
─────────────────────────
Overhead           ~1ms
```

### Optimization Rules

1. **Never block on usage recording** - always async
2. **Cache API keys** - short TTL (1 min) is fine
3. **Batch DB writes** - usage events buffered
4. **No allocations in hot path** - reuse buffers

### Benchmarking

```bash
# Run benchmarks
go test -bench=. -benchmem ./domain/ratelimit/

# Profile CPU
go test -cpuprofile=cpu.prof -bench=. ./app/
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=. ./app/
go tool pprof mem.prof
```

---

## Release Checklist

- [ ] All tests pass: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] Linting passes: `golangci-lint run`
- [ ] Docs updated
- [ ] Migration tested (fresh + upgrade)
- [ ] Config example updated
- [ ] CHANGELOG updated
- [ ] Version bumped
