# APIGate Development Plan

Self-hosted API monetization solution with authentication, rate limiting, usage metering, and multi-provider billing.

**Architecture:** Values-as-boundaries for 100% testability

## Progress: 7/24 tasks (29%) - MVP + Hot Reload COMPLETE

---

## Phase 1: MVP (COMPLETE)

| Status | Task | Description |
|--------|------|-------------|
| ✅ | Core Domain | Pure functions: key, ratelimit, usage, billing |
| ✅ | SQLite Storage | KeyStore, UserStore, RateLimitStore, UsageStore adapters |
| ✅ | Proxy HTTP Handler | Auth → RateLimit → Forward → Meter |
| ✅ | Pluggable Components | Remote adapters for auth, usage, billing delegation |
| ✅ | Config & Bootstrap | YAML loader, main.go, CLI, graceful shutdown |
| ✅ | E2E Test | Full flow verification with real HTTP |
| ✅ | Hot Reload | Config file watching, SIGHUP, atomic updates |

**Deliverable:** Working proxy that authenticates, rate limits, and forwards requests with hot reload.

---

## Phase 2: Transparency & APIs (NEXT)

Make the system self-documenting and add management APIs.

| Status | Task | Description | 3rd Party Libs |
|--------|------|-------------|----------------|
| ⬜ | 2.1 OpenAPI/Swagger | Auto-generate spec at `/.well-known/openapi.json` | `swaggo/swag` |
| ⬜ | 2.2 Admin REST API | CRUD users, keys, plans; view usage | `go-playground/validator` |
| ⬜ | 2.3 Portal REST API | Self-service: register, login, manage keys | `golang-jwt/jwt/v5` |
| ⬜ | 2.4 Prometheus Metrics | `/metrics` endpoint with request stats | `prometheus/client_golang` |

**Dependencies:**
```
2.1 OpenAPI → 2.2 Admin API → 2.3 Portal API → Phase 3
                            ↘ 2.4 Metrics (parallel)
```

**API Structure:**
```
/admin/*     - Protected by admin key, CRUD operations
/portal/*    - JWT auth, self-service for developers
/metrics     - Prometheus scrape endpoint
/.well-known/openapi.json - API specification
```

---

## Phase 3: Payment Integration

Integrate payment providers for monetization.

| Status | Task | Description | 3rd Party Libs |
|--------|------|-------------|----------------|
| ⬜ | 3.1 Provider Interface | Abstract port: CreateCustomer, CreateSubscription, RecordUsage | - |
| ⬜ | 3.2 Stripe Adapter | Subscriptions, usage billing, webhooks | `stripe/stripe-go` |
| ⬜ | 3.3 Paddle Adapter | EU/VAT compliant, international | Paddle API |
| ⬜ | 3.4 LemonSqueezy | Simple alternative for indie devs | LemonSqueezy API |

**Dependencies:**
```
3.1 Provider Interface → 3.2 Stripe → 3.3 Paddle → 3.4 LemonSqueezy
                                    ↘ (parallel)  ↗
```

---

## Phase 4: Self-Onboarding & DX

Make it trivial for new users to get started.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Quick Start Script | `curl -sSL https://apigate.dev/install.sh \| sh` |
| ⬜ | Interactive Wizard | `apigate init` with prompts |
| ⬜ | Example Configs | FastAPI, Express, Rails, Django templates |
| ⬜ | Docker Compose | One-command production setup |

---

## Phase 5: Admin UI (Future)

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Admin Dashboard | User/key management, analytics (React/htmx) |
| ⬜ | Developer Portal | Self-service, usage, billing |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Layer                              │
├─────────────┬─────────────┬─────────────┬─────────┬─────────┤
│   Proxy     │  Admin API  │ Portal API  │ Metrics │ OpenAPI │
│   /*        │  /admin/*   │ /portal/*   │/metrics │/.well-* │
├─────────────┴─────────────┴─────────────┴─────────┴─────────┤
│                  Application Services                        │
│         ProxyService  │  AdminService  │  PortalService      │
├─────────────────────────────────────────────────────────────┤
│                     Domain Layer (Pure)                      │
│    key  │  ratelimit  │  usage  │  billing  │  user  │ plan │
├─────────────────────────────────────────────────────────────┤
│                        Ports (Interfaces)                    │
│  KeyStore │ UserStore │ UsageRecorder │ BillingProvider │... │
├─────────────────────────────────────────────────────────────┤
│                         Adapters                             │
│  SQLite │ Memory │ Remote HTTP │ Stripe │ Paddle │ Prometheus│
└─────────────────────────────────────────────────────────────┘
```

---

## 3rd Party Libraries

| Category | Library | Purpose |
|----------|---------|---------|
| Router | `go-chi/chi` | HTTP routing (already using) |
| Logging | `rs/zerolog` | Structured logging (already using) |
| Crypto | `golang.org/x/crypto/bcrypt` | Password hashing (already using) |
| File Watcher | `fsnotify/fsnotify` | Hot reload config (already using) |
| OpenAPI | `swaggo/swag` | Auto-generate OpenAPI from annotations |
| Validation | `go-playground/validator` | Request validation |
| JWT | `golang-jwt/jwt/v5` | Portal authentication |
| Metrics | `prometheus/client_golang` | Prometheus metrics |
| Stripe | `stripe/stripe-go` | Payment processing |

---

## Pluggable Architecture

Components can run locally or delegate to external services:

| Component | Local Mode | Remote Mode |
|-----------|------------|-------------|
| Auth | SQLite KeyStore | HTTP call to your auth API |
| Usage | SQLite UsageStore | HTTP POST to your analytics |
| Billing | Stripe/Paddle/Lemon | HTTP call to your billing API |

Configuration:
```yaml
auth:
  mode: "remote"
  remote:
    url: "https://your-auth.com/api"
    api_key: "${AUTH_API_KEY}"

usage:
  mode: "remote"
  remote:
    url: "https://your-analytics.com/api"

billing:
  mode: "remote"
  remote:
    url: "https://your-billing.com/api"
```

---

## Testing Status

| Level | Tests | Status |
|-------|-------|--------|
| Domain | 25 | ✅ All passing |
| Adapters (SQLite) | 16 | ✅ All passing |
| Adapters (HTTP) | 10 | ✅ All passing |
| Config | 16 | ✅ All passing |
| Bootstrap | 3 | ✅ All passing |
| E2E | 7 | ✅ All passing |
| **Total** | **77** | ✅ All passing |

---

## Principles

1. **Values as Boundaries** - Pure domain, I/O at edges
2. **Dependency Injection** - All via ports interfaces
3. **No Mocks in Domain** - Pure functions don't need them
4. **Incremental Delivery** - Each phase is deployable
5. **Self-Documenting** - OpenAPI, CLI help, embedded docs
6. **Pluggable by Default** - Local or remote for each component
7. **3rd-Party Libraries** - Use battle-tested libs, maintain interfaces
8. **Transparency** - Debug mode, metrics, request tracing
