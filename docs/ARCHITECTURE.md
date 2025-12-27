# APIGate Architecture

## Design Philosophy

### Values as Boundaries

This codebase follows the "Functional Core, Imperative Shell" pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                    IMPERATIVE SHELL                         │
│  (HTTP handlers, database, external APIs, file I/O)         │
│                                                             │
│    ┌─────────────────────────────────────────────────┐     │
│    │              FUNCTIONAL CORE                     │     │
│    │  (Pure functions, value types, decision logic)  │     │
│    │                                                  │     │
│    │  • No side effects                              │     │
│    │  • Deterministic (same input → same output)     │     │
│    │  • Easy to test (no mocks needed)               │     │
│    └─────────────────────────────────────────────────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key Principle**: Push I/O to the edges. Keep business logic pure.

---

## Layer Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         main.go                              │
│                    (Composition Root)                        │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                      app/ (Application)                      │
│              Orchestrates domain + adapters                  │
│                                                              │
│  ProxyService, AdminService, PortalService, BillingService  │
└──────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
┌─────────────────────────┐     ┌─────────────────────────────┐
│    domain/ (Core)       │     │    adapters/ (I/O)          │
│                         │     │                             │
│  Pure types & functions │     │  Implementations of ports   │
│  No dependencies        │     │  HTTP, DB, Stripe, etc.     │
│  100% unit testable     │     │                             │
└─────────────────────────┘     └─────────────────────────────┘
              │                               │
              └───────────────┬───────────────┘
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                     ports/ (Interfaces)                      │
│                                                              │
│  Abstract boundaries between domain and adapters             │
│  Enables dependency injection and testing                    │
└──────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
apigate/
├── cmd/
│   └── apigate/
│       └── main.go              # Composition root - wires everything
│
├── domain/                      # FUNCTIONAL CORE (no I/O, no deps)
│   ├── key/
│   │   ├── key.go              # APIKey value type
│   │   ├── validate.go         # Pure validation functions
│   │   └── validate_test.go    # Unit tests (no mocks!)
│   │
│   ├── plan/
│   │   ├── plan.go             # Plan value type
│   │   ├── quota.go            # Quota checking (pure)
│   │   └── quota_test.go
│   │
│   ├── usage/
│   │   ├── event.go            # UsageEvent value type
│   │   ├── aggregate.go        # Aggregation logic (pure)
│   │   └── aggregate_test.go
│   │
│   ├── ratelimit/
│   │   ├── bucket.go           # Token bucket algorithm (pure)
│   │   ├── window.go           # Sliding window algorithm (pure)
│   │   └── window_test.go
│   │
│   ├── billing/
│   │   ├── invoice.go          # Invoice calculation (pure)
│   │   ├── pricing.go          # Price calculation (pure)
│   │   └── pricing_test.go
│   │
│   └── proxy/
│       ├── request.go          # Request context value
│       ├── response.go         # Response value
│       ├── decision.go         # Proxy decisions (pure)
│       └── decision_test.go
│
├── ports/                       # INTERFACES (contracts)
│   ├── clock.go                # Time abstraction
│   ├── random.go               # Randomness abstraction
│   ├── keystore.go             # Key persistence
│   ├── userstore.go            # User persistence
│   ├── usagestore.go           # Usage persistence
│   ├── billing.go              # Payment provider
│   └── upstream.go             # Upstream HTTP client
│
├── adapters/                    # IMPERATIVE SHELL (I/O)
│   ├── clock/
│   │   ├── real.go             # time.Now()
│   │   └── fake.go             # Controllable for tests
│   │
│   ├── random/
│   │   ├── real.go             # crypto/rand
│   │   └── fake.go             # Deterministic for tests
│   │
│   ├── sqlite/
│   │   ├── store.go            # SQLite connection
│   │   ├── keystore.go         # KeyStore implementation
│   │   ├── userstore.go        # UserStore implementation
│   │   └── usagestore.go       # UsageStore implementation
│   │
│   ├── postgres/
│   │   └── ...                 # Same structure as sqlite
│   │
│   ├── memory/                 # For testing!
│   │   ├── keystore.go
│   │   ├── userstore.go
│   │   └── usagestore.go
│   │
│   ├── stripe/
│   │   ├── client.go           # Stripe API client
│   │   └── billing.go          # Billing port implementation
│   │
│   └── http/
│       ├── proxy.go            # Proxy HTTP handler
│       ├── admin.go            # Admin HTTP handlers
│       ├── portal.go           # Portal HTTP handlers
│       └── middleware.go       # Common middleware
│
├── app/                         # APPLICATION SERVICES
│   ├── proxy.go                # Proxy orchestration
│   ├── admin.go                # Admin operations
│   ├── portal.go               # Portal operations
│   └── billing.go              # Billing orchestration
│
├── config/
│   ├── config.go               # Configuration types
│   ├── loader.go               # YAML loading
│   └── validate.go             # Config validation (pure)
│
├── migrations/
│   └── 001_initial.sql
│
└── test/
    ├── integration/            # Integration tests
    └── e2e/                    # End-to-end tests
```

---

## Data Flow

### Request Flow (Proxy)

```
                    HTTP Request
                         │
                         ▼
┌────────────────────────────────────────────────────────────┐
│                  adapters/http/proxy.go                    │
│                                                            │
│  1. Extract headers, path, method                          │
│  2. Create domain/proxy.RequestContext (VALUE)             │
└────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────┐
│                    app/proxy.go                            │
│                                                            │
│  3. Call KeyStore.Get(keyHash) → Key (VALUE)              │
│  4. Call domain/key.Validate(key, now) → ValidationResult │
│  5. Call domain/ratelimit.Check(state, limit) → Decision  │
│  6. If allowed: call Upstream.Forward(req) → Response     │
│  7. Create domain/usage.Event (VALUE)                      │
│  8. Call UsageStore.Record(event) [async]                  │
└────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────┐
│                  adapters/http/proxy.go                    │
│                                                            │
│  9. Convert domain/proxy.Response → HTTP Response          │
└────────────────────────────────────────────────────────────┘
                         │
                         ▼
                   HTTP Response
```

### Value Types (Immutable Data)

All domain types are **values** - immutable data with no behavior that causes side effects:

```go
// domain/key/key.go
type Key struct {
    ID        string
    UserID    string
    Hash      []byte
    Prefix    string
    Scopes    []string
    ExpiresAt *time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
}

// domain/ratelimit/window.go
type WindowState struct {
    Count     int
    WindowEnd time.Time
}

type CheckResult struct {
    Allowed   bool
    Remaining int
    ResetAt   time.Time
    Reason    string  // If not allowed
}

// domain/usage/event.go
type Event struct {
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
```

### Pure Functions (No Side Effects)

Domain functions are **pure** - same input always produces same output:

```go
// domain/ratelimit/window.go
func Check(state WindowState, limit int, now time.Time) CheckResult {
    // If window expired, reset
    if now.After(state.WindowEnd) {
        return CheckResult{
            Allowed:   true,
            Remaining: limit - 1,
            ResetAt:   now.Truncate(time.Minute).Add(time.Minute),
        }
    }

    // Check limit
    if state.Count >= limit {
        return CheckResult{
            Allowed:   false,
            Remaining: 0,
            ResetAt:   state.WindowEnd,
            Reason:    "rate_limit_exceeded",
        }
    }

    return CheckResult{
        Allowed:   true,
        Remaining: limit - state.Count - 1,
        ResetAt:   state.WindowEnd,
    }
}

// domain/key/validate.go
func Validate(key Key, now time.Time) ValidationResult {
    if key.RevokedAt != nil {
        return ValidationResult{Valid: false, Reason: "key_revoked"}
    }
    if key.ExpiresAt != nil && now.After(*key.ExpiresAt) {
        return ValidationResult{Valid: false, Reason: "key_expired"}
    }
    return ValidationResult{Valid: true}
}

// domain/billing/pricing.go
func CalculateOverage(usage int64, included int64, pricePerUnit int64) int64 {
    if usage <= included {
        return 0
    }
    return (usage - included) * pricePerUnit
}
```

---

## Port Interfaces

Ports define the **contracts** between domain logic and external systems:

```go
// ports/clock.go
type Clock interface {
    Now() time.Time
}

// ports/random.go
type Random interface {
    Bytes(n int) ([]byte, error)
}

// ports/keystore.go
type KeyStore interface {
    Get(ctx context.Context, prefix string) ([]key.Key, error)
    Create(ctx context.Context, k key.Key) error
    Revoke(ctx context.Context, id string, at time.Time) error
    ListByUser(ctx context.Context, userID string) ([]key.Key, error)
    UpdateLastUsed(ctx context.Context, id string, at time.Time) error
}

// ports/userstore.go
type UserStore interface {
    Get(ctx context.Context, id string) (user.User, error)
    GetByEmail(ctx context.Context, email string) (user.User, error)
    Create(ctx context.Context, u user.User) error
    Update(ctx context.Context, u user.User) error
    List(ctx context.Context, limit, offset int) ([]user.User, error)
}

// ports/usagestore.go
type UsageStore interface {
    Record(ctx context.Context, events []usage.Event) error
    GetPeriod(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error)
    GetHistory(ctx context.Context, userID string, months int) ([]usage.Summary, error)
}

// ports/billing.go
type BillingProvider interface {
    CreateCustomer(ctx context.Context, user user.User) (string, error)
    CreateSubscription(ctx context.Context, customerID, priceID string) (Subscription, error)
    ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, ts time.Time) error
    CreateInvoice(ctx context.Context, customerID string, items []InvoiceItem) (Invoice, error)
}

// ports/upstream.go
type Upstream interface {
    Forward(ctx context.Context, req proxy.Request) (proxy.Response, error)
    HealthCheck(ctx context.Context) error
}
```

---

## Testability Strategy

### Level 1: Domain Unit Tests (No Mocks!)

Pure functions can be tested with simple input/output:

```go
// domain/ratelimit/window_test.go
func TestCheck_AllowsWithinLimit(t *testing.T) {
    state := WindowState{Count: 5, WindowEnd: time.Now().Add(time.Minute)}
    result := Check(state, 10, time.Now())

    assert.True(t, result.Allowed)
    assert.Equal(t, 4, result.Remaining)
}

func TestCheck_DeniesOverLimit(t *testing.T) {
    state := WindowState{Count: 10, WindowEnd: time.Now().Add(time.Minute)}
    result := Check(state, 10, time.Now())

    assert.False(t, result.Allowed)
    assert.Equal(t, "rate_limit_exceeded", result.Reason)
}

func TestCheck_ResetsExpiredWindow(t *testing.T) {
    past := time.Now().Add(-time.Hour)
    state := WindowState{Count: 100, WindowEnd: past}
    result := Check(state, 10, time.Now())

    assert.True(t, result.Allowed)
    assert.Equal(t, 9, result.Remaining)
}
```

### Level 2: Application Tests (In-Memory Adapters)

Test orchestration with fake implementations:

```go
// app/proxy_test.go
func TestProxyService_ValidRequest(t *testing.T) {
    // Arrange: wire up with in-memory adapters
    keys := memory.NewKeyStore()
    users := memory.NewUserStore()
    usage := memory.NewUsageStore()
    clock := clock.NewFake(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
    upstream := &fakeUpstream{response: proxy.Response{Status: 200}}

    svc := app.NewProxyService(keys, users, usage, clock, upstream, config)

    // Seed test data
    keys.Create(ctx, key.Key{ID: "k1", UserID: "u1", Prefix: "ak_test1234"})
    users.Create(ctx, user.User{ID: "u1", PlanID: "free"})

    // Act
    req := proxy.Request{
        APIKey: "ak_test1234xxxx",
        Method: "GET",
        Path:   "/api/data",
    }
    resp, err := svc.Handle(ctx, req)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.Status)

    // Verify usage was recorded
    events := usage.GetAll()
    assert.Len(t, events, 1)
    assert.Equal(t, "k1", events[0].KeyID)
}
```

### Level 3: Integration Tests (Real Database)

Test adapters against real infrastructure:

```go
// adapters/sqlite/keystore_test.go
func TestKeyStore_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := setupTestDB(t)
    store := sqlite.NewKeyStore(db)

    // Test create
    k := key.Key{ID: "test-1", UserID: "user-1", Prefix: "ak_abcd"}
    err := store.Create(ctx, k)
    assert.NoError(t, err)

    // Test get
    keys, err := store.Get(ctx, "ak_abcd")
    assert.NoError(t, err)
    assert.Len(t, keys, 1)
}
```

### Level 4: E2E Tests (Full System)

Test the complete system via HTTP:

```go
// test/e2e/proxy_test.go
func TestE2E_ProxyFlow(t *testing.T) {
    // Start test server
    srv := startTestServer(t)
    defer srv.Close()

    // Create user and get key
    resp := httpPost(srv.AdminURL+"/api/users", `{"email":"test@example.com"}`)
    apiKey := resp["api_key"].(string)

    // Make proxied request
    client := &http.Client{}
    req, _ := http.NewRequest("GET", srv.ProxyURL+"/test", nil)
    req.Header.Set("X-API-Key", apiKey)

    resp, err := client.Do(req)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

---

## Configuration Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     apigate.yaml                            │
│                    (YAML on disk)                           │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼ config.Load()
┌─────────────────────────────────────────────────────────────┐
│                  config.Raw                                 │
│           (Unvalidated parsed YAML)                         │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼ config.Validate() [PURE]
┌─────────────────────────────────────────────────────────────┐
│                  config.Config                              │
│            (Validated, typed config)                        │
│                                                             │
│  Contains:                                                  │
│  - Server ports                                             │
│  - Upstream URL                                             │
│  - []Plan (value types)                                     │
│  - []EndpointRule (value types)                            │
│  - BillingConfig                                            │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼ main.go (Composition Root)
┌─────────────────────────────────────────────────────────────┐
│              Wire up all dependencies                       │
│                                                             │
│  adapters := createAdapters(config)                        │
│  services := createServices(config, adapters)              │
│  handlers := createHandlers(services)                      │
│  servers  := createServers(config, handlers)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Error Handling

### Domain Errors (Values, not exceptions)

```go
// domain/key/validate.go
type ValidationResult struct {
    Valid  bool
    Reason string  // "key_expired", "key_revoked", "user_suspended"
}

// domain/ratelimit/window.go
type CheckResult struct {
    Allowed bool
    Reason  string  // "rate_limit_exceeded"
    // ... other fields
}
```

### Application Errors (Typed)

```go
// app/errors.go
type Error struct {
    Code    string // Machine-readable: "key_not_found"
    Message string // Human-readable
    Status  int    // HTTP status code
}

var (
    ErrKeyNotFound     = Error{Code: "key_not_found", Message: "API key not found", Status: 401}
    ErrKeyExpired      = Error{Code: "key_expired", Message: "API key has expired", Status: 401}
    ErrRateLimited     = Error{Code: "rate_limited", Message: "Rate limit exceeded", Status: 429}
    ErrUpstreamTimeout = Error{Code: "upstream_timeout", Message: "Upstream service timeout", Status: 504}
)
```

---

## Concurrency Model

```
┌─────────────────────────────────────────────────────────────┐
│                    Request Goroutine                        │
│                                                             │
│  1. Handle request synchronously                            │
│  2. Create UsageEvent (value)                              │
│  3. Send to buffered channel (non-blocking)                │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼ chan usage.Event (buffered)
┌─────────────────────────────────────────────────────────────┐
│                   Usage Worker Goroutine                    │
│                                                             │
│  • Batches events (100 or 5 seconds)                       │
│  • Writes to database in transaction                        │
│  • Aggregates into period summaries                        │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**
- Request handling is synchronous (predictable latency)
- Usage recording is async (no impact on response time)
- Backpressure: if channel full, drop event (log warning)
- Graceful shutdown: drain channel before exit

---

## Security Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                      UNTRUSTED                              │
│                   (Internet/Client)                         │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Proxy Port :8080                          │
│                                                             │
│  Validates:                                                 │
│  • API key format (before DB lookup)                       │
│  • Request size limits                                      │
│  • Path allowlist (optional)                               │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      TRUSTED                                │
│               (Internal services)                           │
│                                                             │
│  • Admin Port :8081 (requires admin secret)                │
│  • Upstream API (internal network)                         │
│  • Database                                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## Extension Points

The architecture supports extension without modification:

1. **New storage backend**: Implement `ports.KeyStore`, etc.
2. **New billing provider**: Implement `ports.BillingProvider`
3. **New rate limit algorithm**: Add to `domain/ratelimit/`
4. **Custom authentication**: Implement `ports.Authenticator`
5. **Webhooks**: Add `ports.WebhookSender`, implement adapter
