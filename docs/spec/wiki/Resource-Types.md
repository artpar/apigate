# Resource Types

APIGate exposes these resource types via its JSON:API endpoints.

---

## Users

Represents API customers.

**Type**: `users`

**Endpoints**:
- `GET /admin/users` - List users
- `GET /admin/users/:id` - Get user
- `POST /admin/users` - Create user
- `PUT /admin/users/:id` - Update user
- `DELETE /admin/users/:id` - Delete user

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `email` | string | Email address (unique, required) |
| `name` | string | Display name |
| `status` | enum | active, pending, suspended, cancelled |
| `plan_id` | string | Assigned plan ID |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

**Relationships**:

| Relationship | Type | Description |
|--------------|------|-------------|
| `plan` | plans | User's subscription plan |

**Example**:

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

---

## Plans

Represents subscription/pricing plans.

**Type**: `plans`

**Endpoints**:
- `GET /admin/plans` - List plans
- `GET /admin/plans/:id` - Get plan
- `POST /admin/plans` - Create plan
- `PUT /admin/plans/:id` - Update plan
- `DELETE /admin/plans/:id` - Delete plan

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Plan name (required, unique) |
| `description` | string | Plan description |
| `price_monthly` | int | Monthly price in cents |
| `overage_price` | int | Overage price per unit in cents |
| `requests_per_month` | int | Monthly quota (0 = unlimited) |
| `rate_limit_per_minute` | int | Rate limit (default: 60) |
| `trial_days` | int | Free trial period in days |
| `stripe_price_id` | string | Stripe price ID |
| `paddle_price_id` | string | Paddle price ID |
| `lemon_variant_id` | string | LemonSqueezy variant ID |
| `is_default` | bool | Default plan for new users |
| `enabled` | bool | Plan available for selection |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

**Example**:

```json
{
  "data": {
    "type": "plans",
    "id": "plan_pro",
    "attributes": {
      "name": "Pro",
      "description": "For production workloads",
      "price_monthly": 9900,
      "requests_per_month": 100000,
      "rate_limit_per_minute": 600,
      "trial_days": 14,
      "is_default": false,
      "enabled": true
    }
  }
}
```

---

## API Keys

Represents authentication tokens for API access.

**Type**: `api_keys`

**Endpoints**:
- `GET /admin/keys` - List keys
- `GET /admin/keys?user_id={id}` - List keys for user
- `POST /admin/keys` - Create key (returns full key in meta, shown once)
- `DELETE /admin/keys/:id` - Revoke key

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Key name |
| `prefix` | string | Key prefix for identification (immutable) |
| `expires_at` | timestamp | Expiration time |
| `last_used` | timestamp | Last usage time |
| `revoked_at` | timestamp | Revocation time |
| `created_at` | timestamp | Creation time |

**Relationships**:

| Relationship | Type | Description |
|--------------|------|-------------|
| `user` | users | Key owner |

**Example** (on creation - key in meta):

```json
{
  "data": {
    "type": "api_keys",
    "id": "key_xyz789",
    "attributes": {
      "name": "Production Key",
      "prefix": "ak_abc123",
      "created_at": "2025-01-19T10:00:00Z"
    },
    "relationships": {
      "user": {
        "data": { "type": "users", "id": "usr_abc123" }
      }
    },
    "meta": {
      "key": "ak_abc123def456789...",
      "note": "Save this key securely. It will not be shown again."
    }
  }
}
```

---

## Upstreams

Represents backend services.

**Type**: `upstreams`

**Endpoints**:
- `GET /admin/upstreams` - List upstreams
- `GET /admin/upstreams/:id` - Get upstream
- `POST /admin/upstreams` - Create upstream
- `PUT /admin/upstreams/:id` - Update upstream
- `DELETE /admin/upstreams/:id` - Delete upstream

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Upstream name |
| `description` | string | Upstream description |
| `base_url` | string | Backend URL |
| `timeout_ms` | int | Request timeout (default: 30000) |
| `max_idle_conns` | int | Connection pool size (default: 100) |
| `idle_conn_timeout_ms` | int | Idle connection timeout (default: 90000) |
| `auth_type` | enum | none, bearer, header, basic |
| `auth_header` | string | Custom auth header name |
| `enabled` | bool | Upstream active |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

**Example**:

```json
{
  "data": {
    "type": "upstreams",
    "id": "ups_abc123",
    "attributes": {
      "name": "users-service",
      "description": "User management backend",
      "base_url": "https://api.internal/users",
      "timeout_ms": 30000,
      "max_idle_conns": 100,
      "idle_conn_timeout_ms": 90000,
      "auth_type": "bearer",
      "enabled": true
    }
  }
}
```

---

## Routes

Represents URL routing rules.

**Type**: `routes`

**Endpoints**:
- `GET /admin/routes` - List routes
- `GET /admin/routes/:id` - Get route
- `POST /admin/routes` - Create route
- `PUT /admin/routes/:id` - Update route
- `DELETE /admin/routes/:id` - Delete route

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Route name |
| `description` | string | Route description |
| `host_pattern` | string | Host/domain pattern to match |
| `host_match_type` | enum | exact, wildcard, regex, or empty (any host) |
| `path_pattern` | string | URL pattern to match |
| `match_type` | enum | exact, prefix, regex |
| `methods` | []string | HTTP methods (empty = all) |
| `headers` | []object | Header match conditions |
| `path_rewrite` | string | Path transformation |
| `method_override` | string | Override HTTP method for upstream |
| `metering_expr` | string | Expression to calculate request cost |
| `metering_mode` | enum | request, bytes, response_field, custom |
| `protocol` | enum | http, http_stream, sse, websocket |
| `priority` | int | Matching priority |
| `enabled` | bool | Route active |
| `request_transform` | object | Request header/body transformations |
| `response_transform` | object | Response header/body transformations |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

**Relationships**:

| Relationship | Type | Description |
|--------------|------|-------------|
| `upstream` | upstreams | Target backend |

**Example**:

```json
{
  "data": {
    "type": "routes",
    "id": "rte_abc123",
    "attributes": {
      "name": "users-api",
      "description": "User management API routes",
      "host_pattern": "api.example.com",
      "host_match_type": "exact",
      "path_pattern": "/api/v1/users/*",
      "match_type": "prefix",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "path_rewrite": "/users/$1",
      "metering_mode": "request",
      "protocol": "http",
      "priority": 100,
      "enabled": true,
      "created_at": "2025-01-19T10:00:00Z",
      "updated_at": "2025-01-19T10:00:00Z"
    },
    "relationships": {
      "upstream": {
        "data": { "type": "upstreams", "id": "ups_abc123" }
      }
    }
  }
}
```

---

## Usage Events

Represents API usage records (read-only).

**Type**: `usage_events`

**Endpoints**:
- `GET /admin/usage` - List usage events
- `GET /admin/users/:id/usage` - User's usage

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `method` | string | HTTP method |
| `path` | string | Request path |
| `status_code` | int | Response status |
| `latency_ms` | int | Response time |
| `request_bytes` | int | Request size |
| `response_bytes` | int | Response size |
| `metered_units` | int | Usage units |
| `created_at` | timestamp | Request time |

**Relationships**:

| Relationship | Type | Description |
|--------------|------|-------------|
| `user` | users | Request user |
| `api_key` | api_keys | API key used |
| `route` | routes | Matched route |

---

## See Also

- [[JSON-API-Format]] - Response format
- [[Pagination]] - Collection pagination
- [[Error-Codes]] - Error responses
