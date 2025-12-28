# APIGate Development Plan

Self-hosted API monetization solution with authentication, rate limiting, usage metering, and multi-provider billing.

## Core Principle: Operator Self-Onboarding

> **Anyone can deploy, configure, and operate APIGate without reading source code or asking for help.**

**Two Interfaces, One Goal:**
- **Web UI (Primary)** - Visual management for all operators, zero technical knowledge required
- **CLI (Power Users)** - Automation, scripting, AI agents, advanced operators

Every feature ships with:
1. **Web UI** - Self-explanatory interface with contextual help
2. **CLI equivalent** - Full control via command line
3. **Validation** - Pre-flight checks before deploy
4. **Introspection** - Diagnose issues without debugging
5. **Sensible defaults** - Works out of box

**Architecture:** Values-as-boundaries for 100% testability

---

## Progress: 21/42 tasks (50%)

---

## Phase 1: Foundation + CLI (COMPLETE)

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
| ✅ | Env Var Config | `APIGATE_*` environment variables for Docker |

**Deliverable:** Fully operable proxy via CLI.

---

## Phase 2: Admin REST API (COMPLETE)

REST API powering the Admin Web UI.

| Status | Task | Description |
|--------|------|-------------|
| ✅ | Admin Auth | Admin API key + session authentication |
| ✅ | Users API | `POST/GET/PUT/DELETE /admin/users` |
| ✅ | Keys API | `POST/GET/DELETE /admin/keys` |
| ✅ | Plans API | `GET /admin/plans` (dynamic creation TBD) |
| ✅ | Usage API | `GET /admin/usage` with filters |
| ✅ | Settings API | `GET /admin/settings` (updates TBD) |
| ✅ | Doctor API | `GET /admin/doctor` system health |

**API Structure:**
```
POST   /admin/login          - Admin login (returns session)
POST   /admin/logout         - End session

GET    /admin/users          - List users
POST   /admin/users          - Create user
GET    /admin/users/:id      - Get user
PUT    /admin/users/:id      - Update user
DELETE /admin/users/:id      - Delete user

GET    /admin/keys           - List keys
POST   /admin/keys           - Create key
DELETE /admin/keys/:id       - Revoke key

GET    /admin/plans          - List plans
POST   /admin/plans          - Create plan
PUT    /admin/plans/:id      - Update plan

GET    /admin/usage          - Usage statistics
GET    /admin/settings       - Get settings
PUT    /admin/settings       - Update settings
GET    /admin/doctor         - System health check
```

**Deliverable:** Complete REST API for admin operations.

---

## Phase 3: Admin Web UI (SSR)

Server-side rendered admin interface. **Primary interface for operators.**

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | SSR Framework | Go templates + htmx for interactivity |
| ⬜ | First-Run Wizard | Web-based setup for fresh instances |
| ⬜ | Admin Login | Session-based authentication |
| ⬜ | Dashboard | Overview: users, keys, usage stats |
| ⬜ | User Management | List, create, edit, delete users |
| ⬜ | Key Management | List, create, revoke keys |
| ⬜ | Plan Management | Create and edit pricing plans |
| ⬜ | Usage Analytics | Charts, trends, export |
| ⬜ | Settings Page | Configure upstream, auth, rate limits |
| ⬜ | System Health | Visual doctor/diagnostics |
| ⬜ | Contextual Help | Tooltips, inline docs, guided tours |
| ⬜ | Responsive Design | Mobile-friendly admin |

**First-Run Web Experience:**
```
1. Deploy fresh instance (Docker/binary)
2. Open http://localhost:8080 in browser
3. Redirected to /setup (first-run wizard)

┌─────────────────────────────────────────┐
│  Welcome to APIGate                     │
│                                         │
│  Let's get you set up in 2 minutes.     │
│                                         │
│  Step 1 of 4: Upstream API              │
│  ─────────────────────────              │
│                                         │
│  What API are you monetizing?           │
│                                         │
│  URL: [https://api.myservice.com    ]   │
│                                         │
│  [Test Connection]                      │
│                                         │
│  ℹ️ This is the API your customers      │
│     will access through APIGate.        │
│                                         │
│                        [Next →]         │
└─────────────────────────────────────────┘

Step 2: Create Admin Account
Step 3: Create First Plan
Step 4: Done! Here's your admin key
```

**Admin Dashboard:**
```
┌─────────────────────────────────────────────────────────────┐
│  APIGate Admin                           [? Help] [Logout]  │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                  │
│ Dashboard│  Overview                                        │
│ Users    │  ┌─────────┐ ┌─────────┐ ┌─────────┐            │
│ API Keys │  │ 24      │ │ 156     │ │ 45.2K   │            │
│ Plans    │  │ Users   │ │ Keys    │ │ Requests│            │
│ Usage    │  │ +3 new  │ │ Active  │ │ Today   │            │
│ Settings │  └─────────┘ └─────────┘ └─────────┘            │
│ Health   │                                                  │
│          │  Recent Activity                                 │
│ ──────── │  ─────────────────                              │
│ ? Help   │  • User dev@startup.com created key             │
│          │  • Plan "Enterprise" updated                     │
│          │  • Rate limit triggered for user_123             │
│          │                                                  │
└──────────┴──────────────────────────────────────────────────┘
```

**Self-Help Features:**
- Contextual tooltips on every field
- Inline documentation (expandable)
- "What's this?" links to detailed help
- Guided setup wizards for complex tasks
- Error messages with fix suggestions
- System health with actionable remediation

**Deliverable:** Non-technical operators can fully manage APIGate via browser.

---

## Phase 4: Developer Portal (Self-Service)

Let API consumers manage their own accounts.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Portal API | JWT auth, self-service endpoints |
| ⬜ | Registration Flow | Email verification, onboarding |
| ⬜ | Developer Login | JWT-based sessions |
| ⬜ | Key Management UI | Create, view, revoke own keys |
| ⬜ | Usage Dashboard | View own usage, limits |
| ⬜ | Plan Selection | View plans, request upgrade |
| ⬜ | API Documentation | Interactive API explorer |
| ⬜ | Billing History | View invoices, payment status |

**Developer Portal:**
```
┌─────────────────────────────────────────────────────────────┐
│  MyAPI Developer Portal                    [Account] [Docs] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Welcome back, developer@startup.com                        │
│                                                             │
│  Your Plan: Pro ($49/mo)                                    │
│  ───────────────────────                                    │
│  10,000 requests/month │ 5,234 used │ 4,766 remaining      │
│  ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  52%         │
│                                                             │
│  Your API Keys                              [+ Create Key]  │
│  ─────────────                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Production Key      ak_prod_****1234    [Revoke]   │    │
│  │ Created: Dec 1      Last used: 2 min ago           │    │
│  ├────────────────────────────────────────────────────┤    │
│  │ Development Key     ak_dev_****5678     [Revoke]   │    │
│  │ Created: Nov 15     Last used: 1 hour ago          │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│  Quick Start                                                │
│  ───────────                                                │
│  curl -H "X-API-Key: ak_prod_****1234" \                   │
│       https://api.example.com/v1/endpoint                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Deliverable:** Developers self-serve without contacting admin.

---

## Phase 5: Billing Integration

Payment provider integration.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Billing Interface | Abstract port for providers |
| ⬜ | Stripe Adapter | Subscriptions, usage billing, webhooks |
| ⬜ | Paddle Adapter | EU/VAT compliant |
| ⬜ | LemonSqueezy | Indie-friendly option |
| ⬜ | Billing UI | Admin: revenue dashboard |
| ⬜ | Customer Billing | Portal: invoices, payment methods |
| ⬜ | Billing CLI | `apigate billing status/sync` |

**Admin Billing Dashboard:**
```
┌─────────────────────────────────────────┐
│  Revenue Overview                       │
│                                         │
│  MRR: $4,250    Customers: 24           │
│                                         │
│  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁ Revenue Trend          │
│                                         │
│  By Plan:                               │
│  • Free:       156 users    $0          │
│  • Pro:         18 users    $882        │
│  • Enterprise:   6 users    $3,368      │
└─────────────────────────────────────────┘
```

**Deliverable:** Monetization with any major provider.

---

## Phase 6: Distribution & Operations

Production deployment tooling.

| Status | Task | Description |
|--------|------|-------------|
| ⬜ | Docker Image | Multi-arch, minimal base, web UI included |
| ⬜ | Docker Compose | One-command production setup |
| ⬜ | Helm Chart | Kubernetes deployment |
| ⬜ | Install Script | `curl \| sh` installer |
| ⬜ | Backup/Restore | CLI + UI for data backup |
| ⬜ | Upgrade Flow | In-app upgrade notifications, CLI upgrade |

**Deliverable:** Deploy anywhere with confidence.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  serve │ init │ validate │ users │ keys │ backup │ doctor   │
├─────────────────────────────────────────────────────────────┤
│                      HTTP Layer                              │
├────────┬────────────┬────────────┬──────────┬───────────────┤
│ Proxy  │ Admin API  │ Portal API │ Admin UI │ Developer     │
│ /*     │ /admin/api │ /portal/api│ /admin/* │ Portal /portal│
├────────┴────────────┴────────────┴──────────┴───────────────┤
│                  Application Services                        │
│    ProxyService  │  AdminService  │  PortalService          │
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

## Web UI Technology Stack

**Why SSR (Server-Side Rendering):**
- Single binary deployment (no separate frontend build)
- No Node.js/npm dependency
- Fast initial load
- SEO-friendly (for portal)
- Works without JavaScript (progressive enhancement)

**Stack:**
- **Go html/template** - Server-side rendering
- **htmx** - Dynamic updates without full page reload
- **Tailwind CSS** - Utility-first styling (pre-compiled)
- **Alpine.js** - Minimal JS for interactions (optional)

**Embedded Assets:**
```go
//go:embed templates/* static/*
var webAssets embed.FS
```

All UI assets compiled into single binary. No external dependencies.

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
    volumes:
      - ./data:/data
    ports:
      - "8080:8080"
```

First run: Open browser → Redirected to setup wizard → Done in 2 minutes.

---

## CLI Structure

```
apigate serve               # Run proxy server (default)
apigate init                # Interactive CLI setup wizard
apigate validate            # Validate config before deploy
apigate migrate             # Run database migrations
apigate version             # Show version info

apigate users list          # List all users
apigate users create        # Create user (interactive or flags)
apigate users delete <id>   # Delete user

apigate keys list           # List all keys
apigate keys create         # Create key for user
apigate keys revoke <id>    # Revoke key

apigate billing status      # Show billing status
apigate billing sync        # Sync with payment provider

apigate backup              # Backup database
apigate restore <file>      # Restore from backup
```

---

## Testing Status

| Level | Tests | Status |
|-------|-------|--------|
| Domain | 17 | ✅ All passing |
| App | 5 | ✅ All passing |
| Adapters (SQLite) | 16 | ✅ All passing |
| Adapters (HTTP) | 14 | ✅ All passing |
| Adapters (Admin API) | 16 | ✅ All passing |
| Adapters (Metrics) | 9 | ✅ All passing |
| Config | 24 | ✅ All passing |
| Bootstrap | 3 | ✅ All passing |
| E2E | 7 | ✅ All passing |
| **Total** | **111** | ✅ All passing |

---

## Principles

1. **Web UI Primary** - Visual interface for all operators
2. **CLI for Automation** - Power users, scripts, AI agents
3. **Self-Onboarding** - Deploy and configure without support
4. **Self-Help** - Contextual docs, guided wizards, clear errors
5. **Values as Boundaries** - Pure domain, I/O at edges
6. **Single Binary** - No external dependencies
7. **Sensible Defaults** - Works out of box
8. **Transparency** - Validate, doctor, introspect
