# APIGate Technical Features

## Complete Feature Reference

This document provides a comprehensive technical overview of all features available in APIGate.

### API Specification

All API responses follow the **JSON:API v1.1 specification**. For detailed response format, error codes, and resource types, see:

| Document | Description |
|----------|-------------|
| [spec/json-api.md](spec/json-api.md) | Response format, document structure |
| [spec/error-codes.md](spec/error-codes.md) | All error codes and HTTP statuses |
| [spec/pagination.md](spec/pagination.md) | Pagination parameters and behavior |
| [spec/resource-types.md](spec/resource-types.md) | All API resource types |

---

## 1. Core Proxy Features

### 1.1 Request Proxying

| Feature | Description | Configuration |
|---------|-------------|---------------|
| HTTP Proxy | Forward HTTP/HTTPS requests to upstream APIs | Automatic |
| Method Support | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS | All supported |
| Header Forwarding | Pass-through and transformation of headers | Configurable per route |
| Body Forwarding | Stream request/response bodies efficiently | Automatic |
| Query Parameters | Full query string preservation | Automatic |
| Path Rewriting | Transform request paths before forwarding | `path_rewrite` field |

### 1.2 Protocol Support

| Protocol | Description | Use Case |
|----------|-------------|----------|
| `http` | Standard HTTP request/response | REST APIs |
| `http_stream` | HTTP streaming responses | Large file downloads |
| `sse` | Server-Sent Events | Real-time notifications |
| `websocket` | WebSocket connections | Bidirectional real-time |

### 1.3 Upstream Configuration

```yaml
upstream:
  name: "my-api"
  base_url: "https://api.example.com"
  timeout_ms: 30000
  max_idle_conns: 100
  idle_conn_timeout_ms: 90000
  auth_type: bearer  # none, header, bearer, basic
  auth_header: "X-Custom-Auth"
  auth_value_encrypted: <encrypted>
  enabled: true
```

### 1.4 Routing

| Feature | Description |
|---------|-------------|
| Path Matching | Exact, prefix, or regex matching |
| Method Filtering | Match specific HTTP methods |
| Header Conditions | Route based on header values |
| Priority | Control match order with priority scores |
| Priority Override | Routes with priority > 0 override built-in admin routes |
| Path Rewriting | Transform paths before forwarding |
| Method Override | Change HTTP method for upstream |

**Route Configuration:**
```yaml
route:
  name: "api-v2"
  path_pattern: "/v2/*"
  match_type: prefix  # exact, prefix, regex
  methods: ["GET", "POST"]
  upstream_id: "my-api"
  path_rewrite: "/api/v2$1"
  priority: 100
  enabled: true
```

**Priority-Based Route Override:**

Database routes with `priority > 0` take precedence over built-in admin routes (like `/login`, `/portal`, `/dashboard`). This allows you to serve custom applications at root paths:

```yaml
# Example: Serve custom frontend at root path
route:
  name: "custom-frontend"
  path_pattern: "/*"
  match_type: prefix
  upstream_id: "my-frontend"
  priority: 10        # > 0 enables override
  auth_required: false # Public access
  enabled: true
```

**How Priority Routing Works:**
1. For each request, APIGate checks database routes first
2. If a matching route has `priority > 0`, it's served immediately
3. Otherwise, the request falls through to built-in admin routes
4. This enables using APIGate as a full reverse proxy while maintaining admin access at `/admin/`

**Priority Levels:**
- `0` (default): Standard route, does not override built-in routes
- `1-99`: Low priority, suitable for general custom routes
- `100+`: High priority, for critical path overrides

---

## 2. Authentication & Security

### 2.1 API Key Authentication

| Feature | Description |
|---------|-------------|
| Key Format | `ak_` prefix + 64 hex characters |
| Storage | bcrypt hashed in database |
| Lookup | Fast prefix-based lookup (first 12 chars) |
| Headers | `X-API-Key` or `Authorization: Bearer` |
| Scopes | Optional endpoint-level restrictions |
| Expiration | Optional time-based expiry |
| Revocation | Immediate key invalidation |
| Usage Tracking | Last used timestamp |

**Key Lifecycle:**
- Create: Generates random key, stores hash, returns full key once
- Validate: Hash comparison with bcrypt
- Revoke: Sets `revoked_at` timestamp
- Expire: Automatic based on `expires_at`

### 2.2 Web Authentication

| Feature | Description |
|---------|-------------|
| Login | Email/password with bcrypt verification |
| Sessions | Cookie-based with configurable TTL |
| JWT Tokens | HS256/HS384/HS512 signed tokens |
| Refresh Tokens | Long-lived tokens for session renewal |
| Token Rotation | Optional refresh token rotation |
| Password Hashing | bcrypt with configurable cost factor |
| Account Lockout | After configurable failed attempts |
| Email Verification | Optional email confirmation flow |

### 2.3 Authorization

| Level | Description |
|-------|-------------|
| Public | No authentication required |
| Authenticated | Valid API key or session |
| Self or Admin | User can access own resources, admin all |
| Admin Only | Requires admin role |

**Route-Level Authentication:**

Each route can be configured with the `auth_required` field to control whether API key authentication is required:

```yaml
# Public route - no API key required
route:
  name: "public-docs"
  path_pattern: "/docs/*"
  auth_required: false  # Skip API key validation
  enabled: true

# Protected route - requires API key (default)
route:
  name: "api-endpoint"
  path_pattern: "/api/*"
  auth_required: true   # Require valid API key
  enabled: true
```

**Default Behavior:**
- `auth_required: true` (default): Route requires valid API key in `X-API-Key` header or `Authorization: Bearer` header
- `auth_required: false`: Route is publicly accessible without authentication
- Public routes skip rate limiting and quota enforcement
- Useful for serving static content, public frontends, or webhooks

### 2.4 Security Features

- HTTPS enforcement (recommended via reverse proxy)
- CORS configuration
- Rate limiting (see section 4)
- Request size limits
- Timeout protection
- Secret encryption at rest

---

## 3. User Management

### 3.1 User Model

```yaml
user:
  id: string (UUID)
  email: string (unique, required)
  password_hash: string (bcrypt, internal)
  name: string
  stripe_id: string (payment provider ID)
  plan_id: ref -> plan
  status: enum [pending, active, suspended, cancelled]
  created_at: timestamp
  updated_at: timestamp
```

### 3.2 User Statuses

| Status | Description | API Access |
|--------|-------------|------------|
| `pending` | Awaiting verification | No |
| `active` | Normal operation | Yes |
| `suspended` | Temporarily disabled | No |
| `cancelled` | Account closed | No |

### 3.3 User Actions

| Action | Description | Auth Required |
|--------|-------------|---------------|
| `create` | Register new user | None/Admin |
| `get` | View user details | Self/Admin |
| `update` | Modify profile | Self/Admin |
| `delete` | Remove account | Admin |
| `activate` | Enable account | Admin |
| `suspend` | Disable temporarily | Admin |
| `cancel` | Close account | Self/Admin |
| `set_password` | Change password | Self/Admin |

---

## 4. Rate Limiting

### 4.1 Algorithm

- **Type**: Token Bucket
- **Window**: Configurable (default 60 seconds)
- **Scope**: Per API key
- **Storage**: SQLite with atomic operations

### 4.2 Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rate_limit_per_minute` | Requests allowed per minute | 60 |
| Window size | Time window in seconds | 60 |
| Burst capacity | Max tokens in bucket | Same as rate |

### 4.3 Response Headers

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704067200
```

### 4.4 Exceeded Response

> See [docs/spec/error-codes.md](spec/error-codes.md) for full error format specification.

```
HTTP/1.1 429 Too Many Requests
Retry-After: 15
Content-Type: application/vnd.api+json

{
  "errors": [{
    "status": "429",
    "code": "rate_limit_exceeded",
    "title": "Too Many Requests",
    "detail": "Rate limit exceeded. Try again in 15 seconds."
  }]
}
```

---

## 5. Quota Management

### 5.1 Quota Types

| Type | Description | Unit |
|------|-------------|------|
| `requests_per_month` | Total API calls | Count |
| `bytes_per_month` | Data transfer | Bytes |

### 5.2 Enforcement Modes

| Mode | Behavior |
|------|----------|
| `hard` | Block requests when quota exceeded |
| `warn` | Allow but add warning headers |
| `soft` | Allow and bill overage |

### 5.3 Grace Period

- `quota_grace_pct`: Percentage buffer before hard block (default 5%)
- Allows slight overage during enforcement transition

### 5.4 Warning Levels

| Level | Threshold | Action |
|-------|-----------|--------|
| `none` | < 80% | Normal operation |
| `approaching` | 80-94% | Warning header |
| `critical` | 95-99% | Alert user |
| `exceeded` | >= 100% | Enforce based on mode |

### 5.5 Response Headers

```
X-Quota-Limit: 10000
X-Quota-Remaining: 2500
X-Quota-Reset: 2025-02-01T00:00:00Z
X-Quota-Warning: approaching
```

---

## 6. Usage Metering

### 6.1 Event Tracking

Each API request records:

| Field | Description |
|-------|-------------|
| `id` | Unique event ID |
| `key_id` | API key used |
| `user_id` | User reference |
| `method` | HTTP method |
| `path` | Request path |
| `status_code` | Response status |
| `latency_ms` | Processing time |
| `request_bytes` | Request size |
| `response_bytes` | Response size |
| `cost_multiplier` | Metering weight |
| `ip_address` | Client IP |
| `user_agent` | Client identifier |
| `timestamp` | Request time |

### 6.2 Metering Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `request` | Count each request as 1 | Standard APIs |
| `bytes` | Meter by data transfer | Data APIs |
| `response_field` | Extract count from response | Batch APIs |
| `custom` | Expression-based | Complex pricing |

### 6.3 Custom Metering

```yaml
route:
  metering_mode: custom
  metering_expr: "response.body.items.length * 0.01"
```

### 6.4 Aggregations

| Period | Description |
|--------|-------------|
| Hourly | Per-hour summaries |
| Daily | Per-day summaries |
| Monthly | Billing period totals |

---

## 7. Plans & Pricing

### 7.1 Plan Model

```yaml
plan:
  name: "Pro"
  description: "For production applications"
  rate_limit_per_minute: 600
  requests_per_month: 100000
  price_monthly: 2900  # cents
  overage_price: 1     # cents per request
  trial_days: 14
  stripe_price_id: "price_xxx"
  paddle_price_id: "pri_xxx"
  lemon_variant_id: "var_xxx"
  is_default: false
  enabled: true
```

### 7.2 Pricing Features

| Feature | Description |
|---------|-------------|
| Monthly subscription | Recurring billing |
| Overage pricing | Per-request beyond quota |
| Trial periods | Free trial before billing |
| Multiple providers | Stripe, Paddle, LemonSqueezy |
| Default plan | Auto-assign to new users |

### 7.3 Plan Actions

| Action | Description |
|--------|-------------|
| `create` | Add new plan |
| `update` | Modify plan |
| `delete` | Remove plan |
| `enable` | Activate plan |
| `disable` | Deactivate plan |
| `set_default` | Make default for new users |

---

## 8. Payment Integration

### 8.1 Supported Providers

| Provider | Features |
|----------|----------|
| **Stripe** | Full subscription management, webhooks |
| **Paddle** | Tax handling, merchant of record |
| **LemonSqueezy** | Simple setup, developer-friendly |
| **Dummy** | Testing without real payments |

### 8.2 Stripe Integration

| Feature | Description |
|---------|-------------|
| Customer sync | Auto-create Stripe customers |
| Price sync | Map plans to Stripe prices |
| Subscription management | Create/cancel subscriptions |
| Webhook handling | Payment events processing |
| Invoice access | Customer billing history |

### 8.3 Webhook Events

| Event | Action |
|-------|--------|
| `checkout.session.completed` | Activate subscription |
| `customer.subscription.updated` | Update plan |
| `customer.subscription.deleted` | Cancel access |
| `invoice.paid` | Confirm payment |
| `invoice.payment_failed` | Handle failure |

---

## 9. Request/Response Transformation

### 9.1 Request Transform

```yaml
request_transform:
  headers:
    add:
      X-Custom-Header: "value"
    remove:
      - "X-Unwanted-Header"
    rename:
      Old-Header: New-Header
  body:
    set:
      api_version: "2.0"
    remove:
      - "internal_field"
```

### 9.2 Response Transform

```yaml
response_transform:
  headers:
    add:
      X-Powered-By: "APIGate"
    remove:
      - "X-Internal-Header"
  body:
    wrap: "data"  # Wrap response in {data: ...}
    flatten: true  # Remove nesting
    rename:
      old_field: new_field
```

---

## 10. Module System

### 10.1 Module Structure

```yaml
module: module_name

meta:
  description: "Module description"
  icon: "icon-name"
  display_name: "Display Name"
  implements: [capability1, capability2]
  requires:
    dependency_name:
      capability: capability_type
      required: true
      default: "default_provider"

schema:
  field_name:
    type: string|int|bool|enum|ref|secret|json|timestamp|email|bytes
    required: true|false
    unique: true|false
    lookup: true|false
    default: value
    description: "Field description"

actions:
  action_name:
    description: "Action description"
    input:
      - { name: param, type: string, required: true }
    output:
      - { name: result, type: string }
    set: { field: value }  # For simple updates
    confirm: true  # Require confirmation

channels:
  http:
    serve:
      enabled: true
      base_path: /api/resource
      endpoints:
        - { action: list, method: GET, path: "/" }
    consume:
      external_api:
        base: https://api.external.com
        auth: { type: bearer, bearer: ${API_KEY} }
  cli:
    serve:
      enabled: true
      command: resource
      commands:
        - action: list
          columns: [id, name, status]

hooks:
  after_create:
    - emit: resource.created
    - call: sync_external
```

### 10.2 Field Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | "hello" |
| `int` | Integer | 42 |
| `bool` | Boolean | true |
| `enum` | Predefined values | active\|inactive |
| `ref` | Foreign key | user_id -> user |
| `secret` | Encrypted text | password |
| `json` | JSON object/array | {"key": "value"} |
| `timestamp` | Date/time | 2025-01-01T00:00:00Z |
| `email` | Email address | user@example.com |
| `bytes` | Binary data | encrypted blob |

### 10.3 Field Attributes

| Attribute | Description |
|-----------|-------------|
| `required` | Must be provided |
| `unique` | No duplicates allowed |
| `lookup` | Indexed for fast search |
| `immutable` | Cannot be changed after create |
| `internal` | Hidden from external APIs |
| `default` | Default value if not provided |

---

## 11. Capability System

### 11.1 Built-in Capabilities

| Capability | Purpose | Providers |
|------------|---------|-----------|
| `payment` | Payment processing | Stripe, Paddle, LemonSqueezy, Dummy |
| `email` | Email sending | SMTP, SendGrid, Log |
| `cache` | Data caching | Redis, Memory |
| `storage` | File storage | S3, Disk, Memory |
| `auth` | Authentication | Built-in JWT |
| `queue` | Message queuing | Redis, Memory |
| `notification` | Alerts | Slack, Webhook, Log |
| `data_source` | External data | Database, API, File |
| `sync` | Data synchronization | Table sync |
| `reconciliation` | Data validation | Default |

### 11.2 Provider Pattern

```yaml
# Capability definition
capability: payment

actions:
  create_customer:
    input: [{ name: email, type: email }]
    output: [{ name: customer_id, type: string }]
  create_subscription:
    input: [{ name: customer_id }, { name: price_id }]
    output: [{ name: subscription_id }]
  handle_webhook:
    input: [{ name: payload, type: json }]

# Provider implementation
module: payment_stripe

meta:
  implements: [payment]

schema:
  secret_key: { type: secret, required: true }
  webhook_secret: { type: secret }
```

### 11.3 Dependency Injection

```yaml
module: sync_table

meta:
  requires:
    source:
      capability: data_source
      required: true
    target:
      capability: data_source
      required: true
    notifier:
      capability: notification
      required: false
      default: notification_log
```

---

## 12. Channels

### 12.1 HTTP Channel

| Feature | Description |
|---------|-------------|
| Auto-CRUD | Automatic REST endpoints |
| Custom endpoints | Map actions to paths |
| Auth levels | Public, authenticated, admin |
| OpenAPI | Auto-generated spec |

### 12.2 CLI Channel

| Feature | Description |
|---------|-------------|
| Cobra commands | Auto-generated CLI |
| Subcommands | Nested command structure |
| Flags | Named parameters |
| Args | Positional arguments |
| Confirmation | Dangerous action prompts |
| Table output | Formatted columns |

### 12.3 WebSocket Channel

| Feature | Description |
|---------|-------------|
| Event streaming | Real-time updates |
| Subscriptions | Per-resource channels |
| Events | created, updated, deleted |

---

## 13. Hooks & Events

### 13.1 Hook Types

| Hook | Trigger |
|------|---------|
| `before_create` | Before record creation |
| `after_create` | After record creation |
| `before_update` | Before record update |
| `after_update` | After record update |
| `before_delete` | Before record deletion |
| `after_delete` | After record deletion |
| `before_{action}` | Before custom action |
| `after_{action}` | After custom action |

### 13.2 Hook Actions

```yaml
hooks:
  after_create:
    - emit: resource.created  # Emit event
    - call: external_sync     # Call function
    - set: { synced: true }   # Update field
```

### 13.3 Event System

| Event | Payload |
|-------|---------|
| `user.created` | User object |
| `key.created` | Key metadata (no secret) |
| `plan.updated` | Plan changes |
| `route.deleted` | Route ID |
| `setting.changed` | Key/value pair |

---

## 14. API Endpoints

### 14.1 Authentication Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | Login with email/password |
| POST | `/auth/register` | Create new account |
| POST | `/auth/logout` | End session |
| GET | `/auth/me` | Get current user |
| GET | `/auth/setup-required` | Check if setup needed |
| POST | `/auth/setup` | Complete initial setup |

### 14.2 Admin Endpoints

| Resource | Base Path | Actions |
|----------|-----------|---------|
| Users | `/api/users` | CRUD, activate, suspend, cancel |
| Plans | `/api/plans` | CRUD, enable, disable, set_default |
| Keys | `/api/keys` | CRUD, revoke |
| Routes | `/api/routes` | CRUD, enable, disable |
| Upstreams | `/api/upstreams` | CRUD, enable, disable, health |
| Settings | `/api/settings` | CRUD, batch |

### 14.3 Portal Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/portal/dashboard` | Customer dashboard |
| GET | `/portal/api-keys` | Key management |
| GET | `/portal/usage` | Usage statistics |
| GET | `/portal/plans` | Available plans |
| GET | `/portal/settings` | Account settings |

### 14.4 Documentation Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/docs` | Documentation home |
| GET | `/docs/quickstart` | Getting started |
| GET | `/docs/authentication` | Auth guide |
| GET | `/docs/api-reference` | API reference |
| GET | `/docs/examples` | Code examples |
| GET | `/docs/try-it` | Interactive tester |

### 14.5 System Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/version` | Version info |
| GET | `/metrics` | Prometheus metrics |
| GET | `/_schema` | Schema introspection |
| GET | `/_openapi` | OpenAPI spec |
| GET | `/swagger` | Swagger UI |

---

## 15. CLI Commands

### 15.1 Server Commands

```bash
# Start server
apigate serve

# Initialize setup
apigate init

# Validate configuration
apigate validate

# Show version
apigate version

# Run migrations
apigate migrate
```

### 15.2 Resource Commands

```bash
# Users
apigate users list
apigate users get <id>
apigate users create --email user@example.com
apigate users delete <id>
apigate users activate <id>
apigate users suspend <id>

# Plans
apigate plans list
apigate plans get <id>
apigate plans create --name "Pro" --rate_limit_per_minute 600
apigate plans enable <id>
apigate plans disable <id>

# API Keys
apigate keys list
apigate keys list-user --user <user_id>
apigate keys create --user <user_id> --name "Production"
apigate keys revoke <id>

# Routes
apigate routes list
apigate routes create --name "api" --path "/api/*" --upstream <id>
apigate routes enable <id>
apigate routes disable <id>

# Upstreams
apigate upstreams list
apigate upstreams create --name "api" --url "https://api.example.com"
apigate upstreams health <id>

# Settings
apigate settings list
apigate settings set --key "smtp.host" --value "smtp.example.com"
```

### 15.3 Module Commands

```bash
# Generic module operations
apigate mod <module> list
apigate mod <module> get <id>
apigate mod <module> create [flags]
apigate mod <module> update <id> [flags]
apigate mod <module> delete <id>
```

---

## 16. Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APIGATE_DATABASE_DSN` | SQLite database path | `apigate.db` |
| `APIGATE_SERVER_PORT` | HTTP port | `8080` |
| `APIGATE_SERVER_HOST` | Bind address | `0.0.0.0` |
| `APIGATE_LOG_LEVEL` | Log verbosity | `info` |
| `APIGATE_LOG_FORMAT` | Log format | `json` |
| `STRIPE_SECRET_KEY` | Stripe API key | - |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret | - |

---

## 17. Database

### 17.1 SQLite Configuration

| Setting | Value |
|---------|-------|
| Journal mode | WAL |
| Synchronous | NORMAL |
| Cache size | 64MB |
| Temp store | MEMORY |

### 17.2 Tables

| Table | Purpose |
|-------|---------|
| `users` | User accounts |
| `api_keys` | API keys |
| `plans` | Pricing plans |
| `routes` | Routing rules |
| `upstreams` | Backend services |
| `usage_events` | Request metrics |
| `usage_summaries` | Aggregated stats |
| `rate_limits` | Rate limit state |
| `settings` | Configuration |
| `sessions` | Web sessions |
| `schema_migrations` | Migration tracking |

### 17.3 Migrations

- Automatic on startup
- Version tracking
- Embedded in binary
- Rollback support

---

## 18. Monitoring

### 18.1 Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `apigate_requests_total` | Counter | Total requests |
| `apigate_request_duration_seconds` | Histogram | Latency |
| `apigate_request_size_bytes` | Histogram | Request size |
| `apigate_response_size_bytes` | Histogram | Response size |
| `apigate_active_keys` | Gauge | Active API keys |
| `apigate_active_users` | Gauge | Active users |

### 18.2 Health Check

```json
GET /health

{
  "status": "healthy",
  "database": "connected",
  "uptime": "24h15m30s"
}
```

### 18.3 Logging

- Structured JSON logs
- Request tracing
- Error tracking
- Configurable levels: debug, info, warn, error

---

## 19. Security Features

### 19.1 Data Protection

| Feature | Implementation |
|---------|----------------|
| Password hashing | bcrypt (cost 12) |
| API key hashing | bcrypt |
| Secret encryption | AES-256-GCM |
| Session security | Signed cookies |
| Token signing | HMAC-SHA256/384/512 |

### 19.2 Access Control

| Level | Capabilities |
|-------|--------------|
| Anonymous | Public endpoints only |
| API Key | Proxied API access |
| User Session | Portal access |
| Admin | Full management access |

### 19.3 Attack Prevention

| Attack | Mitigation |
|--------|------------|
| Brute force | Account lockout |
| Rate abuse | Token bucket limiting |
| Quota abuse | Hard/soft enforcement |
| Injection | Parameterized queries |
| XSS | Template escaping |

---

## 20. Extensibility

### 20.1 Custom Modules

Create new modules via YAML:
1. Define schema
2. Add actions
3. Configure channels
4. Implement hooks

### 20.2 Custom Capabilities

Register new capability types:
1. Define capability interface
2. Create provider implementations
3. Use dependency injection

### 20.3 Custom Providers

Implement capability interfaces:
1. Create Go implementation
2. Register with container
3. Configure via settings

### 20.4 Webhooks

Configure outbound webhooks:
1. Set webhook URL
2. Select events
3. Configure authentication
4. Handle retries
