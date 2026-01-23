# Routes

A **route** defines how incoming requests are matched and forwarded to upstreams.

---

## Overview

Routes are the traffic rules of APIGate. Each route specifies:
- **What to match**: Host, path pattern, HTTP methods, headers
- **Where to send**: Which upstream to forward to
- **How to transform**: Modify request/response

```
Request: GET /api/v1/users/123
                │
                ▼
┌───────────────────────────────────────┐
│           Route Matching               │
│                                        │
│  Route 1: /api/v1/*  → users-service  │ ✓ Match!
│  Route 2: /api/v2/*  → users-v2       │
│  Route 3: /health    → health-check   │
└───────────────────────────────────────┘
                │
                ▼
        Forward to users-service
```

---

## Route Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Human-readable name (required) |
| `description` | string | Route purpose |
| `host_pattern` | string | Host/domain pattern to match |
| `host_match_type` | enum | How to match host: exact, wildcard, regex |
| `path_pattern` | string | URL pattern to match (required) |
| `match_type` | enum | How to match: exact, prefix, regex |
| `methods` | []string | HTTP methods (empty = all) |
| `headers` | object | Header conditions |
| `upstream_id` | string | Target upstream (required) |
| `path_rewrite` | string | Transform path before forwarding |
| `method_override` | string | Change HTTP method |
| `request_transform` | object | Request modifications |
| `response_transform` | object | Response modifications |
| `metering_mode` | enum | How to count usage |
| `metering_expr` | string | Custom metering expression |
| `protocol` | enum | http, http_stream, sse, websocket |
| `auth_required` | bool | Require API key authentication (default: true) |
| `priority` | int | Match priority (higher = first) |
| `enabled` | bool | Route active state |

---

## Match Types

### Exact Match

Matches the path exactly.

```yaml
path_pattern: /users
match_type: exact
```

| Request | Match? |
|---------|--------|
| `/users` | Yes |
| `/users/` | No |
| `/users/123` | No |

### Prefix Match (Default)

Matches paths starting with the pattern.

```yaml
path_pattern: /api/v1
match_type: prefix
```

| Request | Match? |
|---------|--------|
| `/api/v1` | Yes |
| `/api/v1/users` | Yes |
| `/api/v1/users/123` | Yes |
| `/api/v2/users` | No |

### Regex Match

Matches using regular expressions.

```yaml
path_pattern: ^/users/[0-9]+$
match_type: regex
```

| Request | Match? |
|---------|--------|
| `/users/123` | Yes |
| `/users/456` | Yes |
| `/users/abc` | No |
| `/users/123/orders` | No |

---

## Path Rewriting

Transform the path before forwarding to upstream.

### Simple Substitution

```yaml
path_pattern: /api/v1/*
path_rewrite: /internal/$1
```

| Request | Forwarded |
|---------|-----------|
| `/api/v1/users` | `/internal/users` |
| `/api/v1/orders/123` | `/internal/orders/123` |

### Prefix Stripping

```yaml
path_pattern: /api/v1/*
path_rewrite: /$1
```

| Request | Forwarded |
|---------|-----------|
| `/api/v1/users` | `/users` |
| `/api/v1/orders` | `/orders` |

### Version Transformation

```yaml
path_pattern: /v1/*
path_rewrite: /api/2024-01/$1
```

---

## Method Filtering

Match only specific HTTP methods:

```yaml
methods: ["GET", "POST"]
```

| Request | Match? |
|---------|--------|
| `GET /api/users` | Yes |
| `POST /api/users` | Yes |
| `DELETE /api/users` | No |

Empty array or omitted = match all methods.

---

## Header Conditions

Match based on request headers:

```yaml
headers:
  X-API-Version: "2"
  Content-Type: "application/json"
```

All specified headers must match for the route to match.

---

## Host-Based Routing

Route requests by hostname for multi-tenant or subdomain-based APIs.

### Matching Order

When a request arrives, routes are matched in this order:

```
host → method → path → headers
```

Host matching is checked first for optimal multi-tenant routing performance.

### Host Match Types

| Type | Pattern | Matches | Does Not Match |
|------|---------|---------|----------------|
| `exact` | `api.example.com` | `api.example.com` | `www.example.com`, `API.example.com` |
| `wildcard` | `*.example.com` | `api.example.com`, `www.example.com` | `a.b.example.com` (multiple levels) |
| `regex` | `^v[0-9]+\.api\.example\.com$` | `v1.api.example.com`, `v2.api.example.com` | `api.example.com` |
| (empty) | - | Any host | - |

**Notes:**
- All host matching is **case-insensitive**
- Port numbers are stripped (`api.example.com:8080` matches `api.example.com`)
- Trailing dots are removed (`api.example.com.` matches `api.example.com`)
- Wildcard (`*`) matches exactly **one** subdomain level

### Usage Scenarios

#### Multi-Tenant SaaS

Route different customers to their dedicated backends:

```yaml
# Customer 1
host_pattern: api.customer1.example.com
host_match_type: exact
path_pattern: /v1/*
upstream_id: customer1-backend

# Customer 2
host_pattern: api.customer2.example.com
host_match_type: exact
path_pattern: /v1/*
upstream_id: customer2-backend
```

#### Subdomain-Based Versioning

```yaml
# Legacy API
host_pattern: v1.api.example.com
host_match_type: exact
path_pattern: /*
upstream_id: legacy-api

# Modern API
host_pattern: v2.api.example.com
host_match_type: exact
path_pattern: /*
upstream_id: modern-api
```

#### Regional Routing

```yaml
# EU region
host_pattern: eu.api.example.com
host_match_type: exact
upstream_id: eu-cluster

# US region
host_pattern: us.api.example.com
host_match_type: exact
upstream_id: us-cluster
```

#### Dynamic Tenant Routing

Use wildcard for dynamic subdomain routing:

```yaml
host_pattern: "*.api.example.com"
host_match_type: wildcard
path_pattern: /*
upstream_id: tenant-router
```

This matches `tenant1.api.example.com`, `tenant2.api.example.com`, etc.

### Creating Host-Based Routes

#### CLI

```bash
# Exact host match
apigate routes create \
  --name "customer1-api" \
  --host-pattern "api.customer1.example.com" \
  --host-match-type exact \
  --path "/v1/*" \
  --upstream customer1-backend

# Wildcard subdomain
apigate routes create \
  --name "tenant-apis" \
  --host-pattern "*.api.example.com" \
  --host-match-type wildcard \
  --path "/*" \
  --upstream tenant-router
```

#### REST API

```bash
# Create host-based route
curl -X POST http://localhost:8080/admin/routes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "customer1-api",
    "host_pattern": "api.customer1.example.com",
    "host_match_type": "exact",
    "path_pattern": "/v1/*",
    "match_type": "prefix",
    "upstream_id": "customer1-backend-id",
    "enabled": true
  }'
```

### Priority with Host Matching

Routes with host patterns take precedence over routes without:

```
1. Priority (higher = first)
2. Host specificity (exact > wildcard > regex > none)
3. Path match type (exact > prefix > regex)
4. Pattern length (longer = first)
```

Example:

```yaml
# Route 1: Catches all traffic to api.example.com (priority 100)
host_pattern: api.example.com
host_match_type: exact
path_pattern: /*
priority: 100

# Route 2: Specific path on same host (priority 100, but exact path wins)
host_pattern: api.example.com
host_match_type: exact
path_pattern: /admin
match_type: exact
priority: 100

# Route 3: Wildcard catch-all (lower host specificity)
host_pattern: "*.example.com"
host_match_type: wildcard
path_pattern: /*
priority: 100
```

Request to `api.example.com/admin` matches Route 2 (exact path wins).
Request to `api.example.com/users` matches Route 1.
Request to `other.example.com/anything` matches Route 3.

### Backward Compatibility

Routes without `host_pattern` (or with empty `host_pattern`) match **any host**. This ensures existing routes continue to work unchanged.

### Match Type Inference

When `host_pattern` is set but `host_match_type` is empty, the match type is **inferred from the pattern**:

| Pattern | Inferred Match Type |
|---------|---------------------|
| `*.example.com` | `wildcard` |
| `api.example.com` | `exact` |

This ensures host patterns are always respected, even if the match type wasn't explicitly configured.

---

## Route Priority

When multiple routes match, priority determines which wins:

```yaml
# Route 1
path_pattern: /api/*
priority: 0

# Route 2 (higher priority, matches first)
path_pattern: /api/admin/*
priority: 100
```

Higher priority values match first. Same priority = more specific path wins.

---

## Creating Routes

### Admin UI

1. Go to **Routes** in sidebar
2. Click **Add Route**
3. Configure:
   - Name: `user-api`
   - Path Pattern: `/api/users/*`
   - Match Type: `prefix`
   - Upstream: Select your upstream
4. Click **Save**

### CLI

```bash
# Basic route
apigate routes create \
  --name "user-api" \
  --path "/api/users/*" \
  --upstream "users-service"

# With options
apigate routes create \
  --name "admin-api" \
  --path "/admin/*" \
  --upstream "admin-service" \
  --methods "GET,POST,PUT,DELETE" \
  --priority 100
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/routes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-api",
    "path_pattern": "/api/users/*",
    "match_type": "prefix",
    "upstream_id": "upstream-id-here",
    "methods": ["GET", "POST"],
    "enabled": true
  }'
```

---

## Protocol Support

### HTTP (Default)

Standard request/response.

```yaml
protocol: http
```

### HTTP Stream

For large responses, streams without buffering.

```yaml
protocol: http_stream
```

### Server-Sent Events (SSE)

For real-time event streams.

```yaml
protocol: sse
```

### WebSocket

For bidirectional real-time communication.

```yaml
protocol: websocket
```

---

## Metering Modes

How API usage is counted for this route.

### Per Request (Default)

Each request counts as 1.

```yaml
metering_mode: request
```

### Per Byte

Count bytes transferred.

```yaml
metering_mode: bytes
```

### Response Field

Extract count from response.

```yaml
metering_mode: response_field
metering_expr: "response.body.items.length"
```

### Custom Expression

Calculate custom cost.

```yaml
metering_mode: custom
metering_expr: "request.body.batch_size * 0.1"
```

---

## Enable/Disable Routes

### Via CLI

```bash
# Update route to enable
apigate routes update <id> --enabled true

# Update route to disable
apigate routes update <id> --enabled false
```

### Via API

```bash
# Enable a route
curl -X PUT http://localhost:8080/admin/routes/<id> \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Disable a route
curl -X PUT http://localhost:8080/admin/routes/<id> \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

---

## Best Practices

### 1. Use Clear Naming

```bash
# Good
--name "public-users-api"
--name "internal-billing-webhook"

# Bad
--name "route1"
--name "my-route"
```

### 2. Set Explicit Methods

```bash
# Good - explicit
--methods "GET,POST"

# Risky - allows everything
# (no methods specified)
```

### 3. Use Priority for Specificity

```bash
# Catch-all (low priority)
--path "/*" --priority 0

# Specific routes (higher priority)
--path "/api/admin/*" --priority 100
--path "/api/users/*" --priority 50
```

### 4. Group Related Routes

Create logical groupings:

```bash
# User service routes
apigate routes create --name "users-list" --path "/api/users" --methods "GET"
apigate routes create --name "users-create" --path "/api/users" --methods "POST"
apigate routes create --name "users-detail" --path "/api/users/*" --methods "GET,PUT,DELETE"
```

---

## Public Routes (No Authentication)

By default, all routes require API key authentication. Set `auth_required: false` to create public routes that skip authentication, rate limiting, and quota checks.

### Use Cases

- **Reverse proxy**: Forward traffic to deployed applications that handle their own auth
- **Health checks**: Public `/health` endpoints
- **Webhooks**: Receive callbacks from external services
- **Static content**: Serve public assets without auth overhead

### Creating Public Routes

#### CLI

```bash
# Create a public route for a deployed app
apigate routes create \
  --name "deployed-app" \
  --host "myapp.apps.localhost" \
  --host-match exact \
  --path "/*" \
  --upstream myapp-service \
  --auth=false
```

#### REST API

```bash
curl -X POST http://localhost:8080/admin/routes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deployed-app",
    "host_pattern": "myapp.apps.localhost",
    "host_match_type": "exact",
    "path_pattern": "/*",
    "match_type": "prefix",
    "upstream_id": "myapp-service-id",
    "auth_required": false,
    "enabled": true
  }'
```

### Behavior

When a request hits a public route (`auth_required: false`):

1. **No API key required** - requests without API keys are accepted
2. **No rate limiting** - requests are not rate limited
3. **No quota tracking** - requests don't count against user quotas
4. **Anonymous usage** - usage is logged with `anonymous` user/key IDs
5. **Transforms still apply** - request/response transformations work normally
6. **Upstream auth works** - backend authentication headers are still injected

### Security Considerations

- Public routes expose your upstreams without authentication
- The upstream service is responsible for its own security
- Consider using host-based routing to isolate public routes
- Monitor anonymous usage for abuse patterns

---

## Design Notes

### Field Coupling: host_pattern and host_match_type

These two fields have **implicit coupling** - they must be considered together:

| `host_pattern` | `host_match_type` | Behavior |
|----------------|-------------------|----------|
| empty | any | Match any host |
| set | `exact` | Exact host match |
| set | `wildcard` | Wildcard match (e.g., `*.example.com`) |
| set | `regex` | Regex match |
| set | empty | **Inferred** from pattern (see below) |

**Match Type Inference**: When `host_pattern` is set but `host_match_type` is empty:
- Patterns starting with `*.` → wildcard match
- Other patterns → exact match

This defensive behavior ensures host patterns are always respected.

### Why This Matters

A previous bug occurred when:
1. Route A had `host_pattern = *.apps.example.com` but empty `host_match_type`
2. The empty match type caused the host pattern to be **ignored**
3. Route A matched **all hosts** instead of just `*.apps.example.com`
4. This caused higher-priority routes to intercept requests meant for lower-priority routes

**Lesson**: When two fields work together (like `host_pattern` + `host_match_type`), the spec must define behavior for ALL combinations, including edge cases where one is set and the other isn't.

### Validation Rules

The API enforces:
- If `host_pattern` is set and starts with `*.`, `host_match_type` should be `wildcard` (or empty for inference)
- If `host_match_type` is `wildcard`, `host_pattern` must start with `*.`
- If `host_match_type` is `regex`, `host_pattern` must be valid regex

---

## See Also

- [[Upstreams]] - Configure backend services
- [[Transformations]] - Modify requests/responses
- [[Rate-Limiting]] - Protect routes
- [[Protocols]] - HTTP, SSE, WebSocket
