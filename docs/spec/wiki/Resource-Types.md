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
| `api_keys` | api_keys[] | User's API keys |
| `groups` | groups[] | Groups user belongs to |

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
      "created_at": "2025-01-19T10:00:00Z"
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
- `GET /admin/keys/:id` - Get key
- `POST /admin/keys` - Create key (returns full key once)
- `DELETE /admin/keys/:id` - Revoke key

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Key name |
| `prefix` | string | Key prefix for identification (immutable) |
| `key` | string | Full key (only on creation) |
| `scopes` | JSON | Allowed scopes |
| `expires_at` | timestamp | Expiration time |
| `last_used` | timestamp | Last usage time |
| `revoked_at` | timestamp | Revocation time |
| `created_at` | timestamp | Creation time |

**Relationships**:

| Relationship | Type | Description |
|--------------|------|-------------|
| `user` | users | Key owner (if user key) |
| `group` | groups | Key owner (if group key) |
| `created_by` | users | User who created the key |

**Example** (on creation):

```json
{
  "data": {
    "type": "api_keys",
    "id": "key_xyz789",
    "attributes": {
      "name": "Production Key",
      "prefix": "ak_abc123",
      "key": "ak_abc123def456789...",
      "scopes": ["read", "write"],
      "created_at": "2025-01-19T10:00:00Z"
    },
    "relationships": {
      "user": {
        "data": { "type": "users", "id": "usr_abc123" }
      }
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
| `base_url` | string | Backend URL |
| `timeout_ms` | int | Request timeout |
| `max_idle_conns` | int | Connection pool size |
| `auth_type` | enum | none, bearer, header, basic |
| `auth_header` | string | Custom auth header name |
| `health_check_path` | string | Health check endpoint |
| `enabled` | bool | Upstream active |
| `created_at` | timestamp | Creation time |

**Example**:

```json
{
  "data": {
    "type": "upstreams",
    "id": "ups_abc123",
    "attributes": {
      "name": "users-service",
      "base_url": "https://api.internal/users",
      "timeout_ms": 30000,
      "max_idle_conns": 100,
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
| `path_pattern` | string | URL pattern to match |
| `match_type` | enum | exact, prefix, regex |
| `methods` | []string | HTTP methods (empty = all) |
| `path_rewrite` | string | Path transformation |
| `protocol` | enum | http, http_stream, sse, websocket |
| `metering_mode` | enum | request, bytes, response_field, custom |
| `priority` | int | Matching priority |
| `enabled` | bool | Route active |
| `created_at` | timestamp | Creation time |

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
      "path_pattern": "/api/v1/users/*",
      "match_type": "prefix",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "path_rewrite": "/users/$1",
      "protocol": "http",
      "metering_mode": "request",
      "priority": 100,
      "enabled": true
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

## Webhooks

Represents webhook subscriptions.

**Type**: `webhooks`

**Endpoints**:
- `GET /admin/webhooks` - List webhooks
- `GET /admin/webhooks/:id` - Get webhook
- `POST /admin/webhooks` - Create webhook
- `PUT /admin/webhooks/:id` - Update webhook
- `DELETE /admin/webhooks/:id` - Delete webhook

**Attributes**:

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Webhook name |
| `url` | string | Delivery URL |
| `events` | []string | Subscribed events |
| `secret` | string | Signing secret |
| `enabled` | bool | Webhook active |
| `created_at` | timestamp | Creation time |

**Example**:

```json
{
  "data": {
    "type": "webhooks",
    "id": "whk_abc123",
    "attributes": {
      "name": "Slack Notifications",
      "url": "https://hooks.slack.com/services/xxx",
      "events": ["user.created", "quota.exceeded"],
      "enabled": true
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
