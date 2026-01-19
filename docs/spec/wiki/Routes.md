# Routes

A **route** defines how incoming requests are matched and forwarded to upstreams.

---

## Overview

Routes are the traffic rules of APIGate. Each route specifies:
- **What to match**: Path pattern, HTTP methods, headers
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

## See Also

- [[Upstreams]] - Configure backend services
- [[Transformations]] - Modify requests/responses
- [[Rate-Limiting]] - Protect routes
- [[Protocols]] - HTTP, SSE, WebSocket
