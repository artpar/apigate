# APIGate Development Guide

## Architecture Overview

APIGate is a **generic API gateway** that provides authentication, rate limiting, usage metering, and request/response transformation for any upstream API.

### Core Principles

1. **No Provider-Specific Code**: Everything is configurable per-route via Expr expressions
2. **Orthogonal Modules**: Components use values as boundaries, not types
3. **Pure Functions + I/O Separation**: Domain logic is pure, adapters handle I/O
4. **Hot-Reloadable Configuration**: Routes, upstreams, and plans reload without restart

### Directory Structure

```
apigate/
├── cmd/apigate/          # CLI entry point (Cobra commands)
├── bootstrap/            # Application wiring and initialization
├── config/               # Configuration loading and validation
├── domain/               # Pure domain logic (no I/O)
│   ├── key/              # API key validation
│   ├── plan/             # Plan/tier definitions
│   ├── proxy/            # Request/response types
│   ├── ratelimit/        # Rate limiting algorithm
│   ├── route/            # Route and upstream entities
│   ├── streaming/        # SSE/NDJSON parsing utilities
│   └── usage/            # Usage event types
├── app/                  # Application services (orchestration)
│   ├── proxy.go          # Main proxy service
│   ├── route.go          # Route matching service
│   └── transform.go      # Expr-based transformations
├── adapters/             # I/O implementations
│   ├── http/             # HTTP handlers and upstream client
│   ├── sqlite/           # Database storage
│   ├── memory/           # In-memory stores (testing)
│   └── ...
├── ports/                # Interfaces (dependency inversion)
└── web/                  # Admin UI templates
```

## Key Abstractions

### Route

A route defines how requests are matched, transformed, forwarded, and metered.

```go
type Route struct {
    ID              string
    Name            string
    PathPattern     string      // "/api/v1/*", "/users/{id}"
    MatchType       MatchType   // exact, prefix, regex
    Methods         []string    // empty = all
    UpstreamID      string      // target upstream
    PathRewrite     string      // Expr: "\"/v2\" + path"
    RequestTransform  *Transform
    ResponseTransform *Transform
    MeteringExpr    string      // Expr: "respBody.usage.tokens"
    Protocol        Protocol    // http, http_stream, sse, websocket
    Priority        int
    Enabled         bool
}
```

### Upstream

An upstream represents a backend API with its connection and auth configuration.

```go
type Upstream struct {
    ID          string
    Name        string
    BaseURL     string        // https://api.example.com
    Timeout     time.Duration
    AuthType    string        // none, header, bearer, basic, query
    AuthHeader  string        // Header name for auth
    AuthValue   string        // Can use env vars: "${API_KEY}"
    Enabled     bool
}
```

### Transform

Transforms modify requests or responses using Expr expressions.

```go
type Transform struct {
    SetHeaders    map[string]string `json:"set_headers,omitempty"`
    DeleteHeaders []string          `json:"delete_headers,omitempty"`
    BodyExpr      string            `json:"body_expr,omitempty"`
    SetQuery      map[string]string `json:"set_query,omitempty"`
    DeleteQuery   []string          `json:"delete_query,omitempty"`
}
```

**Important**: JSON tags use snake_case for database serialization.

## Request Flow

```
1. HTTP Request arrives
2. Extract API key (header, bearer, query param)
3. Validate key format and lookup in DB
4. Verify key hash (bcrypt)
5. Check key expiration/revocation
6. Load user and verify status
7. Check rate limit (token bucket)
8. Match route by path/method/headers
9. Apply request transform (if configured)
10. Rewrite path (if configured)
11. Apply upstream auth headers
12. Forward to upstream (buffered or streaming)
13. Apply response transform (if configured)
14. Evaluate metering expression
15. Record usage event
16. Return response with rate limit headers
```

### Streaming vs Buffered

The gateway automatically detects streaming based on:
- Route protocol: `sse`, `http_stream`, `websocket`
- Accept header: `text/event-stream`

For streaming:
- Response is forwarded chunk-by-chunk
- Data is accumulated for metering expression evaluation
- Metering happens after stream completes

## Expr Expressions

All dynamic values use [Expr](https://github.com/expr-lang/expr) expressions.

### Available Context Variables

**Request Context** (for transforms):
```
method      string            - HTTP method
path        string            - Request path
query       map[string]string - Query parameters
headers     map[string]string - Request headers
body        any               - Parsed JSON body
rawBody     []byte            - Raw body bytes
userID      string            - Authenticated user ID
planID      string            - User's plan ID
keyID       string            - API key ID
```

**Response Context** (for metering):
```
status        int               - HTTP status code
responseBytes int64             - Response size in bytes
requestBytes  int64             - Request size in bytes
respBody      any               - Parsed JSON response
path          string            - Original request path
method        string            - HTTP method
```

**Streaming Context** (for SSE/NDJSON metering):
```
status        int     - HTTP status code
responseBytes int64   - Total bytes streamed
lastChunk     []byte  - Last chunk received
allData       []byte  - All accumulated data
userID        string  - User ID
planID        string  - Plan ID
keyID         string  - Key ID
```

### Built-in Functions

**String Functions**:
- `lower(s)`, `upper(s)`, `trim(s)`
- `trimPrefix(s, prefix)`, `trimSuffix(s, suffix)`
- `replace(s, old, new)`, `split(s, sep)`, `join(arr, sep)`

**Encoding Functions**:
- `base64Encode(s)`, `base64Decode(s)`
- `urlEncode(s)`, `urlDecode(s)`
- `jsonEncode(obj)`, `jsonDecode(s)`

**Data Parsing Functions** (for metering):
- `json(data)` - Parse JSON from bytes/string
- `lines(data)` - Split into lines
- `linesNonEmpty(data)` - Split into non-empty lines
- `sseEvents(data)` - Parse SSE into array of {event, data, id}
- `sseLastData(data)` - Get last SSE event's data field
- `sseAllData(data)` - Concatenate all SSE data fields

**Array Functions**:
- `len(arr)`, `count(arr)` - Array length
- `first(arr)`, `last(arr)` - First/last element
- `sum(arr)`, `sum(arr, field)` - Sum values
- `avg(arr)`, `min(arr)`, `max(arr)` - Aggregations

**Object Functions**:
- `get(obj, "path.to.field")` - Safe nested access with dot notation

**Utility Functions**:
- `env(name)` - Environment variable
- `now()` - Unix timestamp
- `nowRFC3339()` - RFC3339 timestamp
- `coalesce(a, b, c)` - First non-nil value
- `default(val, defaultVal)` - Default if nil/empty

### Metering Expression Examples

```yaml
# Simple request counting
metering_expr: "1"

# Token-based (LLM APIs)
metering_expr: "json(sseLastData(allData)).usage.total_tokens ?? 1"

# Anthropic tokens (input + output from different events)
metering_expr: >
  (json(sseEvents(allData)[0].data).message.usage.input_tokens ?? 0) +
  (json(sseEvents(allData)[count(sseEvents(allData))-2].data).usage.output_tokens ?? 0)

# Gemini tokens
metering_expr: "json(sseLastData(allData)).usageMetadata.totalTokenCount ?? 1"

# Byte-based (KB)
metering_expr: "responseBytes / 1024"

# Array item count
metering_expr: "len(respBody)"

# Nested field extraction
metering_expr: "len(respBody[\"json\"][\"images\"])"

# Safe nested access
metering_expr: "get(respBody, \"json.audio_duration_seconds\") ?? 1"

# NDJSON line count
metering_expr: "count(linesNonEmpty(allData))"

# Conditional
metering_expr: "status < 400 ? respBody.units : 0"
```

## Database Schema

### Core Tables

```sql
-- API Keys
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    hash BLOB NOT NULL,           -- bcrypt hash
    prefix TEXT NOT NULL,         -- first 12 chars for lookup
    name TEXT,
    scopes TEXT,                  -- JSON array
    expires_at DATETIME,
    revoked_at DATETIME,
    last_used_at DATETIME,
    created_at DATETIME
);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix);

-- Users
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    plan_id TEXT DEFAULT 'free',
    status TEXT DEFAULT 'active',  -- active, suspended
    created_at DATETIME
);

-- Upstreams
CREATE TABLE upstreams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    base_url TEXT NOT NULL,
    timeout_ms INTEGER DEFAULT 30000,
    auth_type TEXT DEFAULT 'none',
    auth_header TEXT,
    auth_value_encrypted BLOB,
    enabled INTEGER DEFAULT 1
);

-- Routes
CREATE TABLE routes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path_pattern TEXT NOT NULL,
    match_type TEXT DEFAULT 'prefix',
    methods TEXT,                  -- JSON array
    upstream_id TEXT REFERENCES upstreams(id),
    path_rewrite TEXT,
    request_transform TEXT,        -- JSON Transform
    response_transform TEXT,       -- JSON Transform
    metering_expr TEXT DEFAULT '1',
    protocol TEXT DEFAULT 'http',
    priority INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1
);
CREATE INDEX idx_routes_enabled ON routes(enabled, priority DESC);

-- Usage Events
CREATE TABLE usage_events (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    latency_ms INTEGER,
    request_bytes INTEGER,
    response_bytes INTEGER,
    cost_multiplier REAL DEFAULT 1.0,  -- metering value
    ip_address TEXT,
    user_agent TEXT,
    timestamp DATETIME
);
CREATE INDEX idx_usage_events_user_time ON usage_events(user_id, timestamp);
```

## Testing Patterns

### Unit Tests

Domain logic uses pure functions that are easy to test:

```go
func TestKeyValidation(t *testing.T) {
    result := key.Validate(testKey, time.Now())
    if !result.Valid {
        t.Errorf("expected valid key")
    }
}
```

### Integration Tests

Use in-memory stores for fast integration tests:

```go
func setupTestHandler() (*apihttp.ProxyHandler, *testStores) {
    stores := &testStores{
        keys:      memory.NewKeyStore(),
        users:     memory.NewUserStore(),
        rateLimit: memory.NewRateLimitStore(),
        usage:     &testUsageRecorder{},
    }
    // ... wire up service
}
```

### Test Upstream

Implement `ports.Upstream` interface for testing:

```go
type testUpstream struct {
    healthy bool
}

func (u *testUpstream) Forward(ctx context.Context, req proxy.Request) (proxy.Response, error) {
    return proxy.Response{Status: 200, Body: []byte(`{"ok":true}`)}, nil
}

func (u *testUpstream) ForwardTo(ctx context.Context, req proxy.Request, upstream *route.Upstream) (proxy.Response, error) {
    return u.Forward(ctx, req)
}

func (u *testUpstream) HealthCheck(ctx context.Context) error {
    if !u.healthy {
        return context.DeadlineExceeded
    }
    return nil
}
```

## Configuration

### YAML Config Structure

```yaml
server:
  addr: "0.0.0.0:8080"

database:
  dsn: "./data/apigate.db"

upstream:
  url: "https://api.example.com"  # default upstream
  timeout: 30s

auth:
  mode: local           # local or remote
  key_prefix: "ak_"

rate_limit:
  burst: 10
  window: 60            # seconds

plans:
  - id: free
    name: Free
    rate_limit_per_minute: 60
    requests_per_month: 1000
  - id: pro
    name: Pro
    rate_limit_per_minute: 600
    requests_per_month: 100000
```

### Hot Reload

Configuration is automatically reloaded:
- File watcher detects config changes
- SIGHUP triggers manual reload
- Routes/upstreams reload every 30 seconds from DB

## Adding New Features

### Adding a New Expr Function

1. Add to `app/transform.go` in `NewTransformService()`:

```go
expr.Function("myFunc", func(params ...any) (any, error) {
    if len(params) != 1 {
        return nil, fmt.Errorf("myFunc requires 1 argument")
    }
    // Implementation
    return result, nil
}),
```

### Adding a New Protocol

1. Add protocol constant to `domain/route/route.go`:
```go
const ProtocolMyProtocol Protocol = "my_protocol"
```

2. Update `ShouldStream()` in `app/proxy.go` if streaming
3. Add handling in `adapters/http/handler.go`

### Adding a New Auth Type

1. Add to `domain/route/route.go` upstream auth types
2. Update `ApplyUpstreamAuth()` in `app/route.go`
3. Handle in request transform if query-based

## Common Patterns

### Accessing Nested JSON Safely

Use `get()` function for safe access:
```
get(respBody, "deeply.nested.field") ?? defaultValue
```

### Metering from SSE Streams

1. Parse SSE events: `sseEvents(allData)`
2. Get specific event: `sseEvents(allData)[0]` or `last(sseEvents(allData))`
3. Parse data field: `json(event.data)`
4. Extract value: `json(sseLastData(allData)).usage.tokens`

### Query Parameter Auth

Use request transform with SetQuery:
```json
{
  "set_query": {
    "key": "\"your-api-key\"",
    "alt": "\"sse\""
  }
}
```

Note: Values are Expr expressions, so string literals need quotes.

### Path Rewriting

Use Expr for dynamic rewrites:
```
"/v2" + trimPrefix(path, "/v1")
```

## Debugging

### Check Route Matching

```bash
# View loaded routes
sqlite3 ./data/test.db "SELECT id, path_pattern, upstream_id FROM routes WHERE enabled=1;"

# Check logs for route matching
tail -f /tmp/apigate.log | grep -E "(route|match)"
```

### Check Metering Values

```bash
# View recent usage events
sqlite3 ./data/test.db "SELECT path, response_bytes, cost_multiplier FROM usage_events ORDER BY timestamp DESC LIMIT 10;"
```

### Expression Cache

Expressions are compiled and cached. Restart server to clear cache after changing expressions in DB.

## Performance Considerations

1. **Expression Compilation**: Expressions are compiled once and cached
2. **Route Matching**: Routes sorted by priority, first match wins
3. **Streaming**: Data accumulated only when metering expression needs `allData`
4. **Connection Pooling**: Upstream client reuses connections
5. **Async Usage Recording**: Usage events recorded asynchronously

## Security Notes

1. API keys stored as bcrypt hashes
2. Upstream auth values can be encrypted in DB
3. Environment variables supported for secrets: `${ENV_VAR}`
4. Rate limiting prevents abuse
5. Request size limits on upstream responses (50MB default)
