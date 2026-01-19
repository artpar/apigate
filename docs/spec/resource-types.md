# Resource Types Specification

> Implementation: `adapters/http/admin/`

This document defines all JSON:API resource types used in the APIGate API.

## Resource Type Constants

| Type Constant | Resource Type | Implementation |
|---------------|---------------|----------------|
| `TypeUser` | `users` | `adapters/http/admin/admin.go:23` |
| `TypeKey` | `api_keys` | `adapters/http/admin/admin.go:24` |
| `TypeSession` | `sessions` | `adapters/http/admin/admin.go:25` |
| `TypePlan` | `plans` | `adapters/http/admin/plans.go:14` |
| `TypeRoute` | `routes` | `adapters/http/admin/routes.go:19` |
| `TypeUpstream` | `upstreams` | `adapters/http/admin/routes.go:20` |
| `TypeUsageEvent` | `usage_events` | `adapters/http/admin/meter.go:20` |

## Users Resource

**Type**: `users`

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `email` | string | User's email address | Yes |
| `name` | string | User's display name | Yes |
| `status` | enum | Account status | Yes |
| `plan_id` | string | Associated plan ID | Yes |
| `stripe_id` | string | Stripe customer ID | Yes |
| `created_at` | timestamp | Creation time | No |
| `updated_at` | timestamp | Last update time | No |

### Status Values

| Status | Description |
|--------|-------------|
| `pending` | Account awaiting verification |
| `active` | Account active and operational |
| `suspended` | Account temporarily disabled |
| `cancelled` | Account closed |

### Relationships

| Relationship | Type | Description |
|--------------|------|-------------|
| `plan` | to-one | User's pricing plan |

### Example

```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "attributes": {
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
      "plan_id": "plan_pro",
      "created_at": "2025-01-19T10:00:00Z",
      "updated_at": "2025-01-19T10:00:00Z"
    },
    "relationships": {
      "plan": {
        "data": { "type": "plans", "id": "plan_pro" }
      }
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/users` | List users (paginated) |
| POST | `/admin/users` | Create user |
| GET | `/admin/users/{id}` | Get user |
| PUT | `/admin/users/{id}` | Update user |
| DELETE | `/admin/users/{id}` | Delete user |

**Implementation**: `adapters/http/admin/admin.go:646-660`

---

## API Keys Resource

**Type**: `api_keys`

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `name` | string | Key name/description | Yes |
| `prefix` | string | Key prefix (for identification) | No |
| `key` | string | Full key (only on create) | No |
| `user_id` | string | Owner user ID | No |
| `scopes` | []string | Allowed scopes | Yes |
| `expires_at` | timestamp | Expiration time | Yes |
| `last_used_at` | timestamp | Last usage time | No |
| `revoked_at` | timestamp | Revocation time | No |
| `created_at` | timestamp | Creation time | No |

### Example: Create Response

The full key is only returned once, at creation:

```json
{
  "data": {
    "type": "api_keys",
    "id": "key_abc123",
    "attributes": {
      "name": "Production Key",
      "prefix": "ak_abc123def456",
      "key": "ak_abc123def456789...(full key)",
      "user_id": "usr_xyz789",
      "scopes": ["read", "write"],
      "created_at": "2025-01-19T10:00:00Z"
    }
  }
}
```

### Example: List Response

Keys in list don't include the full key:

```json
{
  "data": {
    "type": "api_keys",
    "id": "key_abc123",
    "attributes": {
      "name": "Production Key",
      "prefix": "ak_abc123def456",
      "user_id": "usr_xyz789",
      "scopes": ["read", "write"],
      "last_used_at": "2025-01-19T09:00:00Z",
      "created_at": "2025-01-19T08:00:00Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/keys` | List all keys |
| GET | `/admin/keys?user_id={id}` | List keys for user |
| POST | `/admin/keys` | Create key |
| DELETE | `/admin/keys/{id}` | Revoke key |

**Implementation**: `adapters/http/admin/admin.go:827-846`

---

## Sessions Resource

**Type**: `sessions`

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `user_id` | string | Authenticated user ID |
| `user_email` | string | User's email |
| `token` | string | Session token |
| `expires_at` | timestamp | Session expiration |

### Example

```json
{
  "data": {
    "type": "sessions",
    "id": "sess_abc123",
    "attributes": {
      "user_id": "usr_xyz789",
      "user_email": "user@example.com",
      "token": "session_token_value",
      "expires_at": "2025-01-20T10:00:00Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/admin/login` | Create session |
| POST | `/admin/logout` | End session |

**Implementation**: `adapters/http/admin/admin.go:847-856`

---

## Plans Resource

**Type**: `plans`

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `name` | string | Plan name | Yes |
| `description` | string | Plan description | Yes |
| `rate_limit_per_minute` | int | Requests per minute | Yes |
| `requests_per_month` | int | Monthly request quota | Yes |
| `price_monthly` | int | Monthly price in cents | Yes |
| `overage_price` | int | Per-request overage price | Yes |
| `trial_days` | int | Trial period length | Yes |
| `stripe_price_id` | string | Stripe price ID | Yes |
| `is_default` | bool | Default plan flag | Yes |
| `enabled` | bool | Plan availability | Yes |
| `created_at` | timestamp | Creation time | No |
| `updated_at` | timestamp | Last update time | No |

### Example

```json
{
  "data": {
    "type": "plans",
    "id": "plan_pro",
    "attributes": {
      "name": "Pro",
      "description": "For production applications",
      "rate_limit_per_minute": 600,
      "requests_per_month": 100000,
      "price_monthly": 2900,
      "overage_price": 1,
      "trial_days": 14,
      "stripe_price_id": "price_xxx",
      "is_default": false,
      "enabled": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-15T00:00:00Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/plans` | List plans |
| POST | `/admin/plans` | Create plan |
| GET | `/admin/plans/{id}` | Get plan |
| PUT | `/admin/plans/{id}` | Update plan |
| DELETE | `/admin/plans/{id}` | Delete plan |

**Implementation**: `adapters/http/admin/plans.go:324-343`

---

## Routes Resource

**Type**: `routes`

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `name` | string | Route name | Yes |
| `path_pattern` | string | URL pattern to match | Yes |
| `match_type` | enum | Pattern match type | Yes |
| `methods` | []string | HTTP methods | Yes |
| `headers` | object | Header conditions to match | Yes |
| `upstream_id` | string | Target upstream | Yes |
| `path_rewrite` | string | Path transformation | Yes |
| `method_override` | string | Override HTTP method for upstream | Yes |
| `priority` | int | Match priority | Yes |
| `protocol` | enum | Protocol type | Yes |
| `description` | string | Route description | Yes |
| `enabled` | bool | Route active state | Yes |
| `metering_expr` | string | Expression to calculate request cost | Yes |
| `metering_mode` | enum | How usage is measured | Yes |
| `request_transform` | object | Request transformation | Yes |
| `response_transform` | object | Response transformation | Yes |
| `created_at` | timestamp | Creation time | No |
| `updated_at` | timestamp | Last update time | No |

### Match Types

| Value | Description |
|-------|-------------|
| `exact` | Exact path match |
| `prefix` | Prefix match (default) |
| `regex` | Regular expression match |

### Protocol Types

| Value | Description |
|-------|-------------|
| `http` | Standard HTTP |
| `http_stream` | Streaming HTTP |
| `sse` | Server-Sent Events |
| `websocket` | WebSocket |

### Metering Modes

| Value | Description |
|-------|-------------|
| `request` | Count each request as 1 |
| `response_field` | Extract count from response |
| `bytes` | Count bytes transferred |
| `custom` | Use metering_expr for custom calculation |

### Example

```json
{
  "data": {
    "type": "routes",
    "id": "rt_abc123",
    "attributes": {
      "name": "API v2",
      "path_pattern": "/v2/*",
      "match_type": "prefix",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "upstream_id": "up_xyz789",
      "path_rewrite": "/api/v2$1",
      "priority": 100,
      "protocol": "http",
      "description": "Version 2 API routes",
      "enabled": true,
      "metering_mode": "request",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-15T00:00:00Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/routes` | List routes |
| POST | `/admin/routes` | Create route |
| GET | `/admin/routes/{id}` | Get route |
| PUT | `/admin/routes/{id}` | Update route |
| DELETE | `/admin/routes/{id}` | Delete route |

**Implementation**: `adapters/http/admin/routes.go:699-729`

---

## Upstreams Resource

**Type**: `upstreams`

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `name` | string | Upstream name | Yes |
| `description` | string | Upstream description | Yes |
| `base_url` | string | Upstream base URL | Yes |
| `timeout_ms` | int | Request timeout in ms (default: 30000) | Yes |
| `max_idle_conns` | int | Connection pool size (default: 100) | Yes |
| `idle_conn_timeout_ms` | int | Idle connection timeout in ms (default: 90000) | Yes |
| `auth_type` | enum | Authentication type | Yes |
| `auth_header` | string | Custom auth header name | Yes |
| `auth_value_encrypted` | bytes | Encrypted auth credentials | Yes |
| `enabled` | bool | Upstream active state | Yes |
| `created_at` | timestamp | Creation time | No |
| `updated_at` | timestamp | Last update time | No |

### Auth Types

| Value | Description |
|-------|-------------|
| `none` | No authentication |
| `header` | Custom header |
| `bearer` | Bearer token |
| `basic` | Basic authentication |

### Example

```json
{
  "data": {
    "type": "upstreams",
    "id": "up_xyz789",
    "attributes": {
      "name": "Main API",
      "description": "Primary backend service",
      "base_url": "https://api.example.com",
      "timeout_ms": 30000,
      "max_idle_conns": 100,
      "idle_conn_timeout_ms": 90000,
      "auth_type": "bearer",
      "auth_header": "Authorization",
      "enabled": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-15T00:00:00Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/upstreams` | List upstreams |
| POST | `/admin/upstreams` | Create upstream |
| GET | `/admin/upstreams/{id}` | Get upstream |
| PUT | `/admin/upstreams/{id}` | Update upstream |
| DELETE | `/admin/upstreams/{id}` | Delete upstream |

**Implementation**: `adapters/http/admin/routes.go:731-744`

---

## Dynamic Module Resources

Modules defined in `core/modules/` automatically get CRUD endpoints with resource types based on their plural name.

### URL Pattern

```
GET    /{module_plural}           # List
POST   /{module_plural}           # Create
GET    /{module_plural}/{id}      # Get
PUT    /{module_plural}/{id}      # Update
DELETE /{module_plural}/{id}      # Delete
POST   /{module_plural}/{id}/{action}  # Custom action
```

### Example: Custom Module

For a module with `plural: widgets`:

```json
{
  "data": {
    "type": "widgets",
    "id": "wgt_abc123",
    "attributes": {
      "name": "My Widget",
      "config": {"key": "value"}
    }
  }
}
```

**Implementation**: `core/channel/http/http.go:236-405`

---

## Usage Events Resource

**Type**: `usage_events`

External usage events submitted via the Metering API.

### Attributes

| Attribute | Type | Description | Mutable |
|-----------|------|-------------|---------|
| `user_id` | string | User to attribute usage to | No |
| `event_type` | string | Event category (e.g., `deployment.started`) | No |
| `resource_id` | string | Identifier of resource used | No |
| `resource_type` | string | Type of resource | No |
| `quantity` | float64 | Units consumed (default: 1.0) | No |
| `metadata` | object | Arbitrary key-value context | No |
| `timestamp` | timestamp | When event occurred | No |
| `source` | string | Service that submitted event | No |
| `created_at` | timestamp | When event was recorded | No |

### Event Types

| Value | Description |
|-------|-------------|
| `api.request` | API request from external service |
| `deployment.created` | Deployment created |
| `deployment.started` | Deployment started running |
| `deployment.stopped` | Deployment stopped |
| `deployment.deleted` | Deployment removed |
| `compute.minutes` | Compute time in minutes |
| `storage.gb_hours` | Storage in GB-hours |
| `bandwidth.gb` | Data transfer in GB |
| `custom.*` | Custom event types |

### Example: Submit Events

```json
{
  "data": [
    {
      "type": "usage_events",
      "attributes": {
        "id": "evt_abc123",
        "user_id": "usr_xyz789",
        "event_type": "deployment.started",
        "resource_id": "depl_456",
        "resource_type": "deployment",
        "quantity": 1,
        "metadata": {
          "template_id": "tmpl_789",
          "region": "us-east-1"
        },
        "timestamp": "2026-01-19T12:00:00Z"
      }
    }
  ]
}
```

### Example: Query Response

```json
{
  "data": {
    "type": "usage_events",
    "id": "evt_abc123",
    "attributes": {
      "user_id": "usr_xyz789",
      "event_type": "deployment.started",
      "resource_id": "depl_456",
      "resource_type": "deployment",
      "quantity": 1,
      "metadata": {
        "template_id": "tmpl_789"
      },
      "timestamp": "2026-01-19T12:00:00Z",
      "source": "hoster-service",
      "created_at": "2026-01-19T12:00:01Z"
    }
  }
}
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/meter` | Submit usage events |
| GET | `/api/v1/meter` | Query usage events (admin) |

**Implementation**: `adapters/http/admin/meter.go`

See [Metering API Specification](metering-api.md) for full details.
