# Proxying

APIGate acts as a reverse proxy, forwarding authenticated requests to upstream services.

---

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Proxy Architecture                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Client                                                    │
│      │                                                      │
│      │ Request + API Key                                    │
│      ▼                                                      │
│   ┌─────────────┐                                           │
│   │  APIGate    │                                           │
│   │             │  1. Authenticate                          │
│   │  - Auth     │  2. Rate limit                            │
│   │  - Rate     │  3. Route match                           │
│   │  - Route    │  4. Transform                             │
│   │  - Forward  │  5. Forward                               │
│   └──────┬──────┘                                           │
│          │                                                  │
│          │ Transformed request                              │
│          ▼                                                  │
│   ┌─────────────┐                                           │
│   │  Upstream   │  Your backend API                        │
│   │  Service    │                                           │
│   └─────────────┘                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Default Upstream

Configure a default upstream for simple deployments:

```bash
APIGATE_UPSTREAM_URL=https://api.backend.com
```

All requests forward to this URL after authentication.

---

## Route-Based Proxying

For multiple backends, use routes:

```bash
# Create upstreams
apigate upstreams create --name "Users API" --url "https://users.internal:8080"
apigate upstreams create --name "Orders API" --url "https://orders.internal:8080"

# Create routes
apigate routes create --path "/users/*" --upstream users-api
apigate routes create --path "/orders/*" --upstream orders-api
```

See [[Routes]] and [[Upstreams]] for details.

---

## Request Handling

### Headers Added

APIGate adds these headers to upstream requests:

| Header | Description |
|--------|-------------|
| `X-User-ID` | Authenticated user ID |
| `X-Key-ID` | API key ID used |
| `X-Plan-ID` | User's plan |
| `X-Request-ID` | Trace ID for logging |
| `X-Entitlement-*` | Feature flag headers |

### Headers Removed

For security, these are stripped:
- `X-API-Key` (after auth)
- Internal headers

---

## Path Rewriting

Transform paths before forwarding:

```bash
# /api/v1/users/123 → /users/123
apigate routes update <id> --path-rewrite "'/users/' + pathParams.id"
```

See [[Transformations]] for expressions.

---

## Protocols

| Protocol | Use Case |
|----------|----------|
| `http` | Standard request/response |
| `http_stream` | Streaming responses |
| `sse` | Server-Sent Events |
| `websocket` | WebSocket connections |

```bash
apigate routes update <id> --protocol sse
```

---

## Timeouts

```bash
# Default timeout
APIGATE_UPSTREAM_TIMEOUT=30s

# Per-upstream timeout (via Admin UI or API)
```

---

## Load Balancing

Coming soon: Support for multiple upstream targets with load balancing.

---

## See Also

- [[Routes]] - Route configuration
- [[Upstreams]] - Upstream configuration
- [[Transformations]] - Request/response transforms
- [[Request-Lifecycle]] - Full request flow
