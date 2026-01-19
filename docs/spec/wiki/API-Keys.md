# API Keys

**API keys** authenticate requests to your API through APIGate.

---

## Overview

API keys are the primary authentication mechanism for API access:

```
┌──────────────────────────────────────────────────────────────┐
│                      Request Flow                             │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Client Request                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ GET /api/users                                          │ │
│  │ X-API-Key: ak_abc123def456789...                        │ │
│  └─────────────────────────────────────────────────────────┘ │
│                          │                                    │
│                          ▼                                    │
│  APIGate validates:                                           │
│  ✓ Key exists                                                 │
│  ✓ Key not revoked                                            │
│  ✓ Key not expired                                            │
│  ✓ Key scopes allow this endpoint                             │
│                          │                                    │
│                          ▼                                    │
│  Request forwarded to upstream                                │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

---

## Key Format

API keys follow this format:

```
ak_<random-64-hex-characters>

Example: ak_a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef12345678
```

- **Prefix**: `ak_` identifies it as an API key
- **Body**: 64 hexadecimal characters (256 bits of entropy)
- **Storage**: Only the bcrypt hash is stored (key cannot be recovered)

---

## Key Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Human-readable name |
| `prefix` | string | First chars for identification (immutable) |
| `hash` | string | Bcrypt hash of key (internal, never exposed) |
| `user_id` | string | Owner user |
| `group_id` | string | Owner group (for team keys) |
| `created_by` | string | User who created the key |
| `scopes` | JSON | Allowed endpoints/operations |
| `expires_at` | timestamp | Expiration time (optional) |
| `last_used` | timestamp | Last usage time (internal) |
| `revoked_at` | timestamp | Revocation time (internal) |
| `created_at` | timestamp | Creation time |

> **Note**: Keys can belong to either a user OR a group, enabling team-based API access. See [[Groups]].

---

## Creating API Keys

### Admin UI

1. Go to **API Keys** in sidebar
2. Click **Add Key**
3. Select:
   - **User**: Key owner
   - **Name**: Descriptive name
   - **Scopes**: Optional restrictions
   - **Expires**: Optional expiration
4. Click **Save**
5. **Copy the key immediately** - it's only shown once!

### CLI

```bash
# Create key for user
apigate keys create \
  --user "user-id-here" \
  --name "Production Key"

# With expiration
apigate keys create \
  --user "user-id-here" \
  --name "Temp Key" \
  --expires "2025-12-31T23:59:59Z"

# With scopes
apigate keys create \
  --user "user-id-here" \
  --name "Read-Only Key" \
  --scopes "read"
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/keys \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-id-here",
    "name": "Production Key",
    "scopes": ["read", "write"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

**Response** (key shown only once):

```json
{
  "data": {
    "type": "api_keys",
    "id": "key-id",
    "attributes": {
      "name": "Production Key",
      "prefix": "ak_abc123def4",
      "key": "ak_abc123def456789...",
      "user_id": "user-id",
      "created_at": "2025-01-19T10:00:00Z"
    }
  }
}
```

---

## Using API Keys

### Header: X-API-Key

```bash
curl -H "X-API-Key: ak_your_key_here" \
  https://api.example.com/v1/users
```

### Header: Authorization Bearer

```bash
curl -H "Authorization: Bearer ak_your_key_here" \
  https://api.example.com/v1/users
```

---

## Key Scopes

Restrict what a key can access:

```bash
# Read-only key
apigate keys create --scopes "read"

# Specific endpoints
apigate keys create --scopes "users:read,orders:write"
```

### Scope Patterns

| Scope | Allows |
|-------|--------|
| `read` | All GET requests |
| `write` | All POST/PUT/PATCH/DELETE |
| `users:read` | GET /api/users/* |
| `users:write` | POST/PUT/DELETE /api/users/* |
| `*` | Everything (default) |

---

## Key Expiration

Set automatic expiration:

```bash
# Expires in 30 days
apigate keys create \
  --user "user-id" \
  --name "Temp Key" \
  --expires "2025-02-18T00:00:00Z"
```

Expired keys return `401 Unauthorized`.

---

## Revoking Keys

Immediately invalidate a key:

### CLI

```bash
apigate keys revoke <key-id>
```

### API

```bash
curl -X DELETE http://localhost:8080/admin/keys/<key-id>
```

Revoked keys cannot be un-revoked. Create a new key instead.

---

## Key Lifecycle

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Created │────▶│ Active  │────▶│ Revoked │
└─────────┘     └────┬────┘     └─────────┘
     │               │
     │               ▼
     │          ┌─────────┐
     │          │ Expired │
     │          └─────────┘
     │               │
     └───────────────┘
        (if expires_at set)
```

---

## Security Best Practices

### 1. One Key Per Use Case

```bash
# Good - separate keys
apigate keys create --name "Production Backend"
apigate keys create --name "Mobile App"
apigate keys create --name "Partner Integration"

# Bad - shared key
apigate keys create --name "My Key"
```

### 2. Use Scopes

```bash
# Good - minimal permissions
apigate keys create --name "Analytics" --scopes "read"

# Risky - full access
apigate keys create --name "Analytics"
```

### 3. Set Expiration for Temporary Access

```bash
# Good - expires
apigate keys create --name "Contractor" --expires "2025-03-01"

# Risky - never expires
apigate keys create --name "Contractor"
```

### 4. Rotate Keys Regularly

```bash
# 1. Create new key
apigate keys create --user "user-id" --name "Production v2"

# 2. Update applications to use new key

# 3. Revoke old key
apigate keys revoke <old-key-id>
```

### 5. Never Log Full Keys

The full key is only shown at creation. APIGate stores only:
- Prefix (for identification)
- bcrypt hash (for validation)

---

## Customer Self-Service

Customers can manage their own keys via the portal:

1. Customer logs into portal
2. Goes to **API Keys**
3. Can:
   - View existing keys (prefix only)
   - Create new keys
   - Revoke their own keys

---

## Rate Limits & Quotas

Each key inherits limits from its user's plan:

| Metric | Source |
|--------|--------|
| Rate limit | Plan's `rate_limit_per_minute` |
| Monthly quota | Plan's `requests_per_month` |

Multiple keys for the same user share quota but have separate rate limit buckets.

---

## Troubleshooting

### Invalid API Key (401)

```json
{
  "errors": [{
    "status": "401",
    "code": "unauthorized",
    "title": "Unauthorized",
    "detail": "Invalid API key"
  }]
}
```

**Causes**:
- Key doesn't exist
- Key was revoked
- Key has expired
- Typo in key

### Forbidden (403)

```json
{
  "errors": [{
    "status": "403",
    "code": "forbidden",
    "title": "Forbidden",
    "detail": "Key scope does not allow this operation"
  }]
}
```

**Causes**:
- Key scopes don't include this endpoint
- User account is suspended

---

## See Also

- [[Users]] - User management
- [[Plans]] - Rate limits and quotas
- [[Rate-Limiting]] - How rate limiting works
- [[Authentication]] - Authentication overview
