# APIGate Development Plan

Self-hosted API monetization solution with authentication, rate limiting, usage metering, and multi-provider billing.

## Core Principle: Operator Self-Onboarding

> **Anyone can deploy, configure, and operate APIGate without reading source code or asking for help.**

Every feature ships with:
1. **CLI management** - Full control via command line
2. **Validation** - Pre-flight checks before deploy
3. **Introspection** - Diagnose issues without debugging
4. **Sensible defaults** - Works out of box

**Architecture:** Values-as-boundaries for 100% testability

---

## Progress: 13/28 tasks (46%)

---

## Phase 1: Foundation + CLI (IN PROGRESS)

Core proxy with full CLI management capability.

| Status | Task | Description |
|--------|------|-------------|
| ✅ | Core Domain | Pure functions: key, ratelimit, usage, billing |
| ✅ | SQLite Storage | KeyStore, UserStore, RateLimitStore, UsageStore adapters |
| ✅ | Proxy HTTP Handler | Auth → RateLimit → Forward → Meter |
| ✅ | Pluggable Components | Remote adapters for auth, usage, billing delegation |
| ✅ | Config & Bootstrap | YAML loader, graceful shutdown |
| ✅ | Hot Reload | Config file watching, SIGHUP, atomic updates |
| ✅ | Prometheus Metrics | `/metrics` endpoint with request stats |
| ✅ | OpenAPI/Swagger | Auto-generate spec at `/.well-known/openapi.json` |
| ✅ | CLI Foundation | Cobra subcommands: serve, init, validate |
| ✅ | User Management CLI | `apigate users list/create/delete` |
| ✅ | Key Management CLI | `apigate keys list/create/revoke` |
| ✅ | First-Run Bootstrap | Auto-detect empty DB, create admin |
| ⬜ | Env Var Config | `APIGATE_*` environment variables for Docker |

**CLI Structure:**
```
apigate serve               # Run proxy server (default)
apigate init                # Interactive setup wizard
apigate validate            # Validate config before deploy
apigate migrate             # Run database migrations
apigate version             # Show version info

apigate users list          # List all users
apigate users create        # Create user (interactive or flags)
apigate users delete <id>   # Delete user

apigate keys list           # List all keys
apigate keys create         # Create key for user
apigate keys revoke <id>    # Revoke key
```

**First-Run Experience:**
```bash
$ apigate serve
No configuration found. Run 'apigate init' to get started.

$ apigate init
Welcome to APIGate!

? Upstream API URL: https://api.myservice.com
? Database location [./apigate.db]:
? Create admin user? Yes
? Admin email: admin@example.com

✓ Generated apigate.yaml
✓ Created database
✓ Created admin user

Admin API Key (save this, shown once):
  ak_abc123...

Run 'apigate serve' to start.
```

**Deliverable:** Fully operable proxy via CLI only.

---

## Phase 2: Admin REST API

REST API for programmatic management (powers future UI).

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Admin Auth | Admin API key authentication |
| ⬜ | Users API | `POST/GET/DELETE /admin/users` |
| ⬜ | Keys API | `POST/GET/DELETE /admin/keys` |
| ⬜ | Plans API | `GET /admin/plans`, `GET /admin/usage` |
| ⬜ | Doctor Endpoint | `GET /admin/doctor` system health |

**API Structure:**
```
POST   /admin/users          - Create user
GET    /admin/users          - List users
GET    /admin/users/:id      - Get user
DELETE /admin/users/:id      - Delete user

POST   /admin/keys           - Create key
GET    /admin/keys           - List keys
DELETE /admin/keys/:id       - Revoke key

GET    /admin/plans          - List plans
GET    /admin/usage          - Usage statistics
GET    /admin/doctor         - System health check
```

**Deliverable:** Full REST API for management automation.

---

## Phase 3: Portal API (Self-Service)

Let developers manage their own keys.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | JWT Authentication | Login/register flow |
| ⬜ | Self-Service Keys | Create/revoke own keys |
| ⬜ | Usage Dashboard | View own usage |
| ⬜ | Plan Selection | Choose/upgrade plan |

**API Structure:**
```
POST   /portal/register      - Create account
POST   /portal/login         - Get JWT
GET    /portal/me            - Get profile
POST   /portal/keys          - Create key
GET    /portal/keys          - List own keys
DELETE /portal/keys/:id      - Revoke own key
GET    /portal/usage         - View usage
```

**Deliverable:** Developers can self-serve without admin.

---

## Phase 4: Billing Integration

Payment provider integration with CLI management.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Billing Interface | Abstract port for providers |
| ⬜ | Stripe Adapter | Subscriptions, usage billing |
| ⬜ | Paddle Adapter | EU/VAT compliant |
| ⬜ | LemonSqueezy | Indie-friendly option |
| ⬜ | Billing CLI | `apigate billing status/sync` |

**Deliverable:** Monetization with any major provider.

---

## Phase 5: Distribution & Operations

Production deployment tooling.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Docker Image | Multi-arch, minimal base |
| ⬜ | Docker Compose | One-command production setup |
| ⬜ | Helm Chart | Kubernetes deployment |
| ⬜ | Install Script | `curl \| sh` installer |
| ⬜ | Backup/Restore | `apigate backup/restore` |
| ⬜ | Upgrade CLI | `apigate upgrade` with rollback |

**Deliverable:** Deploy anywhere with confidence.

---

## Phase 6: Admin UI (Optional)

Web interface built on Admin API.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Admin Dashboard | User/key management |
| ⬜ | Analytics | Usage graphs, trends |
| ⬜ | Developer Portal | Self-service UI |

**Deliverable:** Visual management (optional, CLI is primary).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  serve │ init │ validate │ users │ keys │ backup │ doctor   │
├─────────────────────────────────────────────────────────────┤
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

## Environment Variables

All config can be set via `APIGATE_*` env vars:

```bash
APIGATE_UPSTREAM_URL=https://api.myservice.com
APIGATE_DATABASE_DSN=./data/apigate.db
APIGATE_SERVER_PORT=8080
APIGATE_AUTH_MODE=local
APIGATE_ADMIN_EMAIL=admin@example.com  # First-run only
APIGATE_LOG_LEVEL=info
APIGATE_LOG_FORMAT=json
```

---

## Docker Experience

```yaml
services:
  apigate:
    image: apigate/apigate
    environment:
      - APIGATE_UPSTREAM_URL=https://api.myservice.com
      - APIGATE_ADMIN_EMAIL=admin@example.com
    volumes:
      - ./data:/data
    ports:
      - "8080:8080"
```

First run auto-creates admin and prints API key to logs.

---

## Testing Status

| Level | Tests | Status |
|-------|-------|--------|
| Domain | 17 | ✅ All passing |
| App | 5 | ✅ All passing |
| Adapters (SQLite) | 16 | ✅ All passing |
| Adapters (HTTP) | 14 | ✅ All passing |
| Adapters (Metrics) | 9 | ✅ All passing |
| Config | 16 | ✅ All passing |
| Bootstrap | 3 | ✅ All passing |
| E2E | 7 | ✅ All passing |
| **Total** | **87** | ✅ All passing |

---

## Principles

1. **Operator Self-Onboarding** - Deploy without support
2. **CLI-First** - Full control via command line
3. **Values as Boundaries** - Pure domain, I/O at edges
4. **Dependency Injection** - All via ports interfaces
5. **Incremental Delivery** - Each phase is deployable
6. **Sensible Defaults** - Works out of box
7. **Transparency** - Validate, doctor, introspect
