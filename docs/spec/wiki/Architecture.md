# Architecture

How APIGate is designed and how requests flow through the system.

---

## System Overview

APIGate is built with a clean architecture separating concerns into distinct layers:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  Web Browser          │  API Clients        │  CLI                          │
│  (Portal/Admin UI)    │  (HTTP/WS/SSE)     │  (Commands)                   │
└──────────┬────────────┴─────────┬───────────┴──────────────┬────────────────┘
           │                      │                          │
           ▼                      ▼                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PRESENTATION LAYER                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  Web Portal  │  │  Admin UI    │  │  REST API    │  │  CLI Handler │     │
│  │  /portal/*   │  │  /ui/*       │  │  /api/*      │  │  apigate ... │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  Docs Portal │  │  Proxy       │  │  WebSocket   │                       │
│  │  /docs/*     │  │  Handler     │  │  Handler     │                       │
│  └──────────────┘  └──────────────┘  └──────────────┘                       │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           APPLICATION LAYER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ ProxyService │  │ RouteService │  │ AuthService  │  │ UsageService │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  Transform   │  │  Settings    │  │  Module      │                       │
│  │  Service     │  │  Service     │  │  Runtime     │                       │
│  └──────────────┘  └──────────────┘  └──────────────┘                       │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            DOMAIN LAYER                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │  User   │  │   Key   │  │  Plan   │  │  Route  │  │ Upstream│            │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘            │
│                                                                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐                         │
│  │  Usage  │  │  Quota  │  │RateLimit│  │Settings │                         │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘                         │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          INFRASTRUCTURE LAYER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────┐  ┌─────────────────────────────────────────┐   │
│  │      PORTS              │  │              ADAPTERS                    │   │
│  │  ┌─────────────────┐    │  │  ┌─────────────────┐  ┌──────────────┐  │   │
│  │  │   UserStore     │◄───┼──┼──│  SQLiteStore    │  │   HTTPClient │  │   │
│  │  │   KeyStore      │    │  │  │                 │  │   SMTPClient │  │   │
│  │  │   PlanStore     │    │  │  │                 │  │StripeClient  │  │   │
│  │  │   RouteStore    │    │  │  │                 │  │              │  │   │
│  │  │   UsageStore    │    │  │  │                 │  │              │  │   │
│  │  └─────────────────┘    │  │  └─────────────────┘  └──────────────┘  │   │
│  └─────────────────────────┘  └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Request Lifecycle

When a client makes an API request through APIGate:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         REQUEST LIFECYCLE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. REQUEST ARRIVES                                                          │
│     └─▶ HTTP Server receives request                                        │
│                                                                              │
│  2. ROUTE MATCHING                                                           │
│     └─▶ Find matching route by path/method/headers                          │
│     └─▶ No match? Return 404                                                │
│                                                                              │
│  3. AUTHENTICATION                                                           │
│     └─▶ Extract API key from header (X-API-Key or Authorization)            │
│     └─▶ Validate key (not expired, not revoked)                             │
│     └─▶ Invalid? Return 401                                                 │
│                                                                              │
│  4. AUTHORIZATION                                                            │
│     └─▶ Check key scopes against route requirements                         │
│     └─▶ Forbidden? Return 403                                               │
│                                                                              │
│  5. RATE LIMITING                                                            │
│     └─▶ Check token bucket for this key                                     │
│     └─▶ Exceeded? Return 429 with Retry-After                               │
│                                                                              │
│  6. QUOTA CHECK                                                              │
│     └─▶ Check monthly quota for user's plan                                 │
│     └─▶ Exceeded? Return 429 or allow with warning                          │
│                                                                              │
│  7. REQUEST TRANSFORMATION                                                   │
│     └─▶ Apply header modifications                                          │
│     └─▶ Apply body transformations                                          │
│     └─▶ Rewrite path if configured                                          │
│                                                                              │
│  8. PROXY TO UPSTREAM                                                        │
│     └─▶ Forward request to backend service                                  │
│     └─▶ Add upstream authentication if configured                           │
│                                                                              │
│  9. RESPONSE TRANSFORMATION                                                  │
│     └─▶ Apply response header modifications                                 │
│     └─▶ Apply response body transformations                                 │
│                                                                              │
│  10. USAGE RECORDING                                                         │
│      └─▶ Record request details (async)                                     │
│      └─▶ Update quota counters                                              │
│                                                                              │
│  11. RESPONSE                                                                │
│      └─▶ Return response to client                                          │
│      └─▶ Include rate limit headers                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### Proxy Handler

The heart of APIGate - handles all proxied API requests.

- Matches routes based on path, method, headers
- Authenticates API keys
- Enforces rate limits and quotas
- Forwards requests to upstreams
- Records usage metrics

### Route Matcher

Efficient route matching with priority support.

- **Exact match**: `/users/123` matches exactly
- **Prefix match**: `/api/*` matches `/api/anything`
- **Regex match**: `/users/[0-9]+` matches `/users/456`

Routes are sorted by priority (highest first), then by specificity.

### Rate Limiter

Token bucket algorithm for rate limiting.

- Per-key rate limits
- Configurable window (default: 1 minute)
- Atomic operations with SQLite
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Quota Manager

Monthly quota tracking and enforcement.

- Request count and byte quotas
- Configurable grace percentage
- Warning levels (80%, 95%, 100%)
- Hard/soft/warn enforcement modes

### Usage Tracker

Records every API request for analytics and billing.

- Request metadata (method, path, status)
- Timing (latency)
- Size (request/response bytes)
- Metering mode (per-request, per-byte, custom)

---

## Data Model

### Entity Relationships

```
┌──────────┐       ┌──────────┐       ┌──────────┐
│   Plan   │◄──────│   User   │──────▶│  API Key │
│          │ 1   n │          │ 1   n │          │
│ - limits │       │ - email  │       │ - prefix │
│ - quota  │       │ - status │       │ - scopes │
│ - price  │       │          │       │ - expiry │
└──────────┘       └──────────┘       └──────────┘
                        │
                        │ 1
                        ▼ n
                   ┌──────────┐
                   │  Usage   │
                   │  Event   │
                   │          │
                   │ - method │
                   │ - path   │
                   │ - bytes  │
                   └──────────┘

┌──────────┐       ┌──────────┐
│ Upstream │◄──────│  Route   │
│          │ 1   n │          │
│ - url    │       │ - path   │
│ - auth   │       │ - match  │
│ - health │       │ - proto  │
└──────────┘       └──────────┘
```

---

## Module System

APIGate uses a YAML-based module system for extensibility.

```yaml
module: route

schema:
  name:         { type: string, required: true }
  path_pattern: { type: string, required: true }
  upstream_id:  { type: ref, to: upstream }
  enabled:      { type: bool, default: true }

actions:
  enable:  { set: { enabled: true } }
  disable: { set: { enabled: false } }

channels:
  http:
    serve:
      base_path: /api/routes
      endpoints:
        - { action: list, method: GET, path: "/" }
        - { action: create, method: POST, path: "/" }
```

---

## Capability System

Pluggable implementations for external integrations:

| Capability | Providers |
|------------|-----------|
| **payment** | Stripe, Paddle, LemonSqueezy, Dummy |
| **email** | SMTP, SendGrid, Log |
| **cache** | Redis, Memory |
| **storage** | S3, Disk, Memory |
| **queue** | Redis, Memory |
| **notification** | Slack, Webhook, Log |

---

## Security Model

### Authentication

1. **API Keys** - For programmatic API access
   - bcrypt hashed storage
   - Prefix-based lookup
   - Optional scopes and expiration

2. **Sessions** - For web UI access
   - Cookie-based
   - CSRF protection
   - Configurable TTL

### Authorization

| Level | Description |
|-------|-------------|
| **public** | No authentication required |
| **authenticated** | Valid API key or session |
| **self** | Own resources only |
| **admin** | Full access |

---

## See Also

- [[Request-Lifecycle]] - Detailed request flow
- [[Module-System]] - Creating custom modules
- [[Configuration]] - System configuration
