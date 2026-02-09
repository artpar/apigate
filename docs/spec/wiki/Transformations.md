# Transformations

**Transformations** modify requests and responses as they pass through APIGate using the [Expr expression language](https://expr-lang.org/).

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

## Transform JSON Structure

Transforms are configured per-route as JSON objects:

```json
{
  "request_transform": {
    "set_headers": {
      "X-User-ID": "userID",
      "X-Custom": "\"static-value\""
    },
    "delete_headers": ["Cookie", "X-Internal"],
    "set_query": {
      "format": "\"json\""
    },
    "delete_query": ["debug"],
    "body_expr": "{\"wrapped\": body, \"user\": userID}"
  },
  "response_transform": {
    "set_headers": {
      "X-Powered-By": "\"APIGate\""
    },
    "delete_headers": ["Server", "X-Debug"],
    "body_expr": "{\"success\": true, \"result\": respBody}"
  }
}
```

### Transform Fields

| Field | Type | Description |
|-------|------|-------------|
| `set_headers` | `map[string]string` | Header name → Expr expression. Sets or adds headers. |
| `delete_headers` | `string[]` | Header names to remove. |
| `set_query` | `map[string]string` | Query param name → Expr expression. (Request only.) |
| `delete_query` | `string[]` | Query param names to remove. (Request only.) |
| `body_expr` | `string` | Expr expression that produces the new body (JSON). |

---

## Expr Expression Language

All transform values use the **Expr** expression language. Values are expressions, not template strings.

### Syntax Rules

- **Bare identifiers** resolve to context variables: `userID`, `email`, `path`
- **Quoted strings** are literal values: `"static-value"`, `"Bearer " + env("API_KEY")`
- **String concatenation**: `"prefix_" + userID`
- **Map access**: `headers["Content-Type"]`, `query["page"]`
- **Ternary**: `role == "admin" ? "true" : "false"`
- **Nil coalescing**: `email ?? "anonymous"`
- **Function calls**: `lower(email)`, `sha256(userID)`

### Examples

```
"userID"                         → resolves to user ID (e.g., "user-42")
"\"static\""                     → literal string "static"
"\"Bearer \" + env(\"API_KEY\")" → "Bearer sk-abc123"
"\"user_\" + userID"             → "user_user-42"
"lower(email)"                   → "alice@example.com"
"nowRFC3339()"                   → "2024-01-15T10:30:00Z"
```

---

## Available Variables

### Request Context

Available in `request_transform.set_headers`, `request_transform.set_query`, and `request_transform.body_expr`:

| Variable | Type | Description |
|----------|------|-------------|
| `userID` | string | Authenticated user ID (empty if public route) |
| `email` | string | User email from JWT claims |
| `role` | string | User role from JWT claims (e.g., "admin", "user") |
| `planID` | string | User's plan ID |
| `keyID` | string | API key ID, or "session:{uid}" for JWT auth |
| `clientIP` | string | Client IP address |
| `host` | string | Request Host header |
| `userAgent` | string | User-Agent header |
| `method` | string | HTTP method (GET, POST, etc.) |
| `path` | string | Request path |
| `query` | map | Query parameters (`query["page"]`) |
| `headers` | map | Request headers (`headers["Content-Type"]`) |
| `body` | any | Parsed JSON request body |
| `rawBody` | bytes | Raw request body bytes |

### Response Context

Available in `response_transform.set_headers` and `response_transform.body_expr`:

| Variable | Type | Description |
|----------|------|-------------|
| `userID` | string | Authenticated user ID |
| `email` | string | User email from JWT claims |
| `role` | string | User role from JWT claims |
| `planID` | string | User's plan ID |
| `keyID` | string | API key ID |
| `status` | int | HTTP status code |
| `respHeaders` | map | Response headers (`respHeaders["Content-Type"]`) |
| `respBody` | any | Parsed JSON response body |
| `responseBytes` | int64 | Response body size in bytes |

### Streaming/Metering Context

Available in `metering_expr`:

| Variable | Type | Description |
|----------|------|-------------|
| `userID` | string | Authenticated user ID |
| `email` | string | User email |
| `role` | string | User role |
| `planID` | string | User's plan ID |
| `keyID` | string | API key ID |
| `status` | int | HTTP status code |
| `responseBytes` | int64 | Total response bytes |
| `requestBytes` | int64 | Request body bytes |
| `allData` | bytes | All streamed data |
| `lastChunk` | bytes | Last data chunk |

---

## Available Functions

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `lower(s)` | Lowercase | `lower("HELLO")` → `"hello"` |
| `upper(s)` | Uppercase | `upper("hello")` → `"HELLO"` |
| `trim(s)` | Trim whitespace | `trim("  hi  ")` → `"hi"` |
| `trimPrefix(s, prefix)` | Remove prefix | `trimPrefix("/v1/users", "/v1")` → `"/users"` |
| `trimSuffix(s, suffix)` | Remove suffix | `trimSuffix("/api/", "/")` → `"/api"` |
| `replace(s, old, new)` | Replace all | `replace("a-b", "-", "_")` → `"a_b"` |
| `split(s, sep)` | Split string | `split("a,b,c", ",")` → `["a","b","c"]` |
| `join(arr, sep)` | Join array | `join(split("a,b", ","), "-")` → `"a-b"` |

### Encoding Functions

| Function | Description | Example |
|----------|-------------|---------|
| `base64Encode(s)` | Base64 encode | `base64Encode("hello")` → `"aGVsbG8="` |
| `base64Decode(s)` | Base64 decode | `base64Decode("aGVsbG8=")` → `"hello"` |
| `urlEncode(s)` | URL encode | `urlEncode("a b")` → `"a+b"` |
| `urlDecode(s)` | URL decode | `urlDecode("a%20b")` → `"a b"` |
| `jsonEncode(v)` | JSON encode | `jsonEncode(body)` → `"{...}"` |
| `jsonDecode(s)` | JSON decode | `jsonDecode(rawStr).key` |

### Crypto Functions

| Function | Description | Example |
|----------|-------------|---------|
| `sha256(s)` | SHA-256 hash | `sha256("test")` → `"9f86d0..."` |
| `hmacSha256(data, key)` | HMAC-SHA256 | `hmacSha256("data", "secret")` |

### Utility Functions

| Function | Description | Example |
|----------|-------------|---------|
| `env(name)` | Environment variable | `env("API_KEY")` |
| `now()` | Unix timestamp | `now()` → `1705312200` |
| `nowRFC3339()` | RFC3339 time | `nowRFC3339()` → `"2024-01-15T..."` |
| `coalesce(a, b, ...)` | First non-nil/empty | `coalesce(email, "anon")` |
| `default(val, fallback)` | Default if empty | `default(role, "user")` |
| `toString(v)` | Convert to string | `toString(123)` → `"123"` |
| `toInt(v)` | Convert to int | `toInt("42")` → `42` |
| `toFloat(v)` | Convert to float | `toFloat("3.14")` → `3.14` |
| `get(obj, path)` | Nested field access | `get(body, "a.b.c")` |

### Data Functions (Metering/Streaming)

| Function | Description |
|----------|-------------|
| `json(data)` | Parse JSON from bytes/string |
| `lines(data)` | Split into lines |
| `linesNonEmpty(data)` | Split into non-empty lines |
| `sseEvents(data)` | Parse SSE events |
| `sseLastData(data)` | Last SSE event data |
| `sseAllData(data)` | All SSE data concatenated |
| `first(arr)` | First array element |
| `last(arr)` | Last array element |
| `count(arr)` | Array length |
| `sum(arr)` / `sum(arr, field)` | Sum values |
| `avg(arr)` / `avg(arr, field)` | Average values |
| `max(arr)` / `max(arr, field)` | Maximum value |
| `min(arr)` / `min(arr, field)` | Minimum value |

---

## Request Transformations

### Header Modifications

#### Set Headers

```json
{
  "request_transform": {
    "set_headers": {
      "X-User-ID": "userID",
      "X-Email": "email",
      "X-Request-Time": "nowRFC3339()",
      "X-Forwarded-For": "clientIP",
      "Authorization": "\"Bearer \" + env(\"UPSTREAM_KEY\")"
    }
  }
}
```

#### Remove Headers

```json
{
  "request_transform": {
    "delete_headers": ["Cookie", "Authorization"]
  }
}
```

### Path Rewriting

Path rewriting uses the `path_rewrite` field (Expr expression):

```json
{
  "path_pattern": "/api/v1/**",
  "path_rewrite": "trimPrefix(path, \"/api/v1\")"
}
```

Examples:
- Strip prefix: `trimPrefix(path, "/api/v1")` — `/api/v1/users` → `/users`
- Add prefix: `"/api/v2" + path` — `/users` → `/api/v2/users`
- Version swap: `replace(path, "/v1/", "/v2/")`

### Query Parameter Transformations

```json
{
  "request_transform": {
    "set_query": {
      "format": "\"json\"",
      "api_version": "\"2024-01\""
    },
    "delete_query": ["debug", "internal_flag"]
  }
}
```

### Body Transformation

```json
{
  "request_transform": {
    "body_expr": "{\"wrapped\": body, \"metadata\": {\"source\": \"apigate\", \"user\": userID}}"
  }
}
```

---

## Response Transformations

### Header Modifications

```json
{
  "response_transform": {
    "set_headers": {
      "X-Powered-By": "\"APIGate\"",
      "X-Request-User": "userID"
    },
    "delete_headers": ["Server", "X-Internal-Version", "X-Debug-Info"]
  }
}
```

### Body Transformation

```json
{
  "response_transform": {
    "body_expr": "{\"success\": status < 400, \"data\": respBody}"
  }
}
```

---

## Common Use Cases

### 1. Pass Authenticated User to Upstream

```json
{
  "request_transform": {
    "set_headers": {
      "X-User-ID": "userID",
      "X-User-Email": "email",
      "X-User-Role": "role",
      "X-Plan-ID": "planID"
    }
  }
}
```

### 2. Add Upstream Authentication

```json
{
  "request_transform": {
    "set_headers": {
      "Authorization": "\"Bearer \" + env(\"UPSTREAM_API_KEY\")"
    }
  }
}
```

### 3. CORS Headers

```json
{
  "response_transform": {
    "set_headers": {
      "Access-Control-Allow-Origin": "\"*\"",
      "Access-Control-Allow-Methods": "\"GET, POST, PUT, DELETE\"",
      "Access-Control-Allow-Headers": "\"Content-Type, X-API-Key\""
    }
  }
}
```

### 4. Remove Internal Headers

```json
{
  "response_transform": {
    "delete_headers": ["X-Internal-Request-ID", "X-Backend-Server", "X-Debug-Timing"]
  }
}
```

### 5. Add Request Metadata

```json
{
  "request_transform": {
    "set_headers": {
      "X-Forwarded-For": "clientIP",
      "X-Forwarded-Proto": "\"https\"",
      "X-User-ID": "userID",
      "X-User-Agent": "userAgent"
    }
  }
}
```

### 6. Metering by Token Usage

```json
{
  "metering_expr": "get(json(sseLastData(allData)), \"usage.total_tokens\") ?? 1"
}
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
    "path_rewrite": "trimPrefix(path, \"/api/v1\")",
    "request_transform": {
      "set_headers": {
        "X-Source": "\"apigate\"",
        "X-User-ID": "userID"
      },
      "delete_headers": ["Cookie"]
    },
    "response_transform": {
      "delete_headers": ["Server", "X-Powered-By"]
    }
  }'
```

---

## Configuration via Admin UI

1. Go to **Routes** → Select route
2. Click **Transformations** tab
3. Configure:
   - **Request Headers**: Set/Delete
   - **Response Headers**: Set/Delete
   - **Path Rewrite**: Expr expression
   - **Body Expression**: Expr expression
4. Click **Save**

---

## Expression Validation

Use the validation API to check expressions before saving:

```bash
curl -X POST http://localhost:8080/admin/validate-expr \
  -H "Content-Type: application/json" \
  -d '{
    "expression": "\"user_\" + userID",
    "context": "request"
  }'
```

Response:
```json
{
  "valid": true,
  "message": "Expression is valid"
}
```

---

## Performance Considerations

### Fast Operations

- Header set/delete: Negligible overhead
- Path rewrite: Negligible overhead
- Query parameter changes: Minimal overhead

### Slower Operations

- Body JSON parsing: ~1-5ms per request
- Body expression evaluation: ~1-2ms per operation

### Best Practices

1. **Minimize body transformations** for high-throughput routes
2. **Use header-only transforms** when possible
3. **Compiled expressions are cached** automatically

---

## Debugging Transformations

### Enable Debug Headers

```json
{
  "response_transform": {
    "set_headers": {
      "X-Transform-Applied": "\"true\"",
      "X-Original-User": "userID"
    }
  }
}
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
