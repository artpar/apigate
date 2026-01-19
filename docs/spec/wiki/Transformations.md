# Transformations

**Transformations** modify requests and responses as they pass through APIGate.

---

## Overview

Transform requests before sending to upstream, and responses before returning to clients:

```
┌────────────────────────────────────────────────────────────────┐
│                    Transformation Pipeline                      │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Client Request                                                 │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────┐                   │
│  │         REQUEST TRANSFORMATION          │                   │
│  │  • Add/Remove/Modify headers            │                   │
│  │  • Rewrite path                         │                   │
│  │  • Transform body                       │                   │
│  │  • Add authentication                   │                   │
│  └─────────────────────────────────────────┘                   │
│       │                                                         │
│       ▼                                                         │
│    Upstream                                                     │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────┐                   │
│  │        RESPONSE TRANSFORMATION          │                   │
│  │  • Add/Remove/Modify headers            │                   │
│  │  • Transform body                       │                   │
│  │  • Sanitize sensitive data              │                   │
│  └─────────────────────────────────────────┘                   │
│       │                                                         │
│       ▼                                                         │
│  Client Response                                                │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Request Transformations

### Header Modifications

#### Add Headers

```yaml
request_transform:
  headers:
    add:
      X-Custom-Header: "value"
      X-Request-ID: "${uuid}"
      X-Forwarded-For: "${client_ip}"
```

#### Remove Headers

```yaml
request_transform:
  headers:
    remove:
      - Cookie
      - Authorization  # Let APIGate handle auth
```

#### Modify Headers

```yaml
request_transform:
  headers:
    set:
      Host: "internal-api.local"
      Accept: "application/json"
```

### Path Rewriting

#### Prefix Stripping

```yaml
path_pattern: /api/v1/*
path_rewrite: /$1

# /api/v1/users → /users
# /api/v1/orders/123 → /orders/123
```

#### Prefix Adding

```yaml
path_pattern: /public/*
path_rewrite: /api/v2/$1

# /public/users → /api/v2/users
```

#### Version Transformation

```yaml
path_pattern: /v1/*
path_rewrite: /2024-01/$1

# /v1/users → /2024-01/users
```

### Query Parameter Transformations

#### Add Parameters

```yaml
request_transform:
  query:
    add:
      format: json
      api_version: "2024-01"
```

#### Remove Parameters

```yaml
request_transform:
  query:
    remove:
      - debug
      - internal_flag
```

### Body Transformations

#### JSON Field Injection

```yaml
request_transform:
  body:
    json:
      inject:
        metadata:
          source: "apigate"
          user_id: "${user_id}"
```

#### JSON Field Removal

```yaml
request_transform:
  body:
    json:
      remove:
        - internal_field
        - debug_info
```

---

## Response Transformations

### Header Modifications

#### Add Headers

```yaml
response_transform:
  headers:
    add:
      X-Powered-By: "APIGate"
      X-Request-ID: "${request_id}"
```

#### Remove Headers

```yaml
response_transform:
  headers:
    remove:
      - Server
      - X-Internal-Version
      - X-Debug-Info
```

### Body Transformations

#### JSON Field Removal

```yaml
response_transform:
  body:
    json:
      remove:
        - internal_id
        - _links.internal
```

#### JSON Field Masking

```yaml
response_transform:
  body:
    json:
      mask:
        - email  # user@domain.com → u***@d***.com
        - phone  # +1234567890 → +1***890
```

#### JSON Field Renaming

```yaml
response_transform:
  body:
    json:
      rename:
        _id: id
        created_at: createdAt
```

---

## Variable Substitution

Use variables in transformations:

| Variable | Description |
|----------|-------------|
| `${uuid}` | Random UUID |
| `${timestamp}` | Current Unix timestamp |
| `${iso_time}` | Current ISO 8601 time |
| `${client_ip}` | Client IP address |
| `${user_id}` | Authenticated user ID |
| `${user_email}` | Authenticated user email |
| `${plan_name}` | User's plan name |
| `${api_key_id}` | API key ID |
| `${api_key_prefix}` | API key prefix |
| `${request_id}` | Unique request ID |
| `${path}` | Request path |
| `${method}` | HTTP method |
| `${host}` | Request host |
| `${header.X-Custom}` | Request header value |
| `${query.param}` | Query parameter value |

### Example

```yaml
request_transform:
  headers:
    add:
      X-Request-ID: "${uuid}"
      X-User-ID: "${user_id}"
      X-Timestamp: "${timestamp}"
      X-Original-Path: "${path}"
```

---

## Conditional Transformations

Apply transformations based on conditions:

### By Method

```yaml
request_transform:
  conditions:
    - when:
        method: POST
      headers:
        add:
          X-Idempotency-Key: "${uuid}"
```

### By Path

```yaml
request_transform:
  conditions:
    - when:
        path_matches: "/admin/*"
      headers:
        add:
          X-Admin-Request: "true"
```

### By Header

```yaml
request_transform:
  conditions:
    - when:
        header_exists: X-Debug
      headers:
        add:
          X-Debug-Enabled: "true"
```

---

## Common Use Cases

### 1. Add Upstream Authentication

```yaml
# Route uses bearer token for upstream
upstream_auth_type: bearer
upstream_auth_value: "${UPSTREAM_API_KEY}"

# Or via transformation
request_transform:
  headers:
    add:
      Authorization: "Bearer ${env.UPSTREAM_API_KEY}"
```

### 2. CORS Headers

```yaml
response_transform:
  headers:
    add:
      Access-Control-Allow-Origin: "*"
      Access-Control-Allow-Methods: "GET, POST, PUT, DELETE"
      Access-Control-Allow-Headers: "Content-Type, X-API-Key"
```

### 3. Remove Internal Headers

```yaml
response_transform:
  headers:
    remove:
      - X-Internal-Request-ID
      - X-Backend-Server
      - X-Debug-Timing
```

### 4. Add Request Metadata

```yaml
request_transform:
  headers:
    add:
      X-Forwarded-For: "${client_ip}"
      X-Forwarded-Proto: "https"
      X-User-ID: "${user_id}"
      X-Plan-Name: "${plan_name}"
```

### 5. Version Header to Path

```yaml
# Client sends: GET /users, X-API-Version: 2
# Transform to: GET /v2/users

request_transform:
  path_rewrite: "/v${header.X-API-Version}${path}"
```

### 6. Sanitize Response

```yaml
response_transform:
  body:
    json:
      remove:
        - password_hash
        - internal_notes
        - admin_only_field
      mask:
        - ssn
        - credit_card
```

---

## Configuration via CLI

```bash
# Create route with path rewrite
apigate routes create \
  --name "api-v1" \
  --path "/api/v1/*" \
  --upstream "backend" \
  --rewrite "/$1"
```

> **Note**: Complex transformations (header modifications, body transforms) must be configured via the Admin UI or API. The CLI supports basic route creation with path rewriting only.

---

## Configuration via API

```bash
curl -X POST http://localhost:8080/admin/routes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-v1",
    "path_pattern": "/api/v1/*",
    "upstream_id": "upstream-id",
    "path_rewrite": "/$1",
    "request_transform": {
      "headers": {
        "add": {
          "X-Source": "apigate",
          "X-Request-ID": "${uuid}"
        },
        "remove": ["Cookie"]
      }
    },
    "response_transform": {
      "headers": {
        "remove": ["Server", "X-Powered-By"]
      },
      "body": {
        "json": {
          "remove": ["internal_id"]
        }
      }
    }
  }'
```

---

## Configuration via Admin UI

1. Go to **Routes** → Select route
2. Click **Transformations** tab
3. Configure:
   - **Request Headers**: Add/Remove/Set
   - **Response Headers**: Add/Remove/Set
   - **Path Rewrite**: Pattern and replacement
   - **Body Transforms**: JSON operations
4. Click **Save**

---

## Performance Considerations

### Fast Operations

- Header add/remove: Negligible overhead
- Path rewrite: Negligible overhead
- Query parameter changes: Minimal overhead

### Slower Operations

- Body JSON parsing: ~1-5ms per request
- Body field injection: ~1-2ms per operation
- Body masking: ~2-5ms depending on field count

### Best Practices

1. **Minimize body transformations** for high-throughput routes
2. **Use header-only transforms** when possible
3. **Cache compiled patterns** (handled automatically)

---

## Debugging Transformations

### Enable Debug Headers

```yaml
response_transform:
  headers:
    add:
      X-Transform-Applied: "true"
      X-Original-Path: "${path}"
```

### Check Server Logs

View transformation activity in the server logs:

```bash
# If running in foreground, watch server output
# If using systemd: journalctl -u apigate -f
```

---

## See Also

- [[Routes]] - Configure routes with transformations
- [[Upstreams]] - Upstream authentication
- [[Protocols]] - Protocol-specific transformations
