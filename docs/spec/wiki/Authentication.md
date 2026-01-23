# Authentication

APIGate supports multiple authentication methods for different use cases.

---

## API Authentication (Proxy Requests)

For proxied API requests, use API keys or JWT session tokens:

### API Key via Header

```http
GET /v1/users HTTP/1.1
Host: api.example.com
X-API-Key: ak_abc123...
```

### API Key via Bearer Token

```http
Authorization: Bearer ak_abc123...
```

### JWT Session Token

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

APIGate automatically detects the token type by format:
- Tokens starting with `ak_` (or configured prefix) → API key authentication
- Other tokens → JWT session token validation

See [[API-Keys]] for details on creating and managing API keys.

---

## User Authentication Endpoints

APIGate provides authentication endpoints at `/auth/*` (and `/admin/*` as aliases):

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "data": {
    "type": "sessions",
    "id": "sess_abc123",
    "attributes": {
      "user_id": "usr_xyz789",
      "user_email": "user@example.com",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expires_at": "2025-01-20T10:00:00Z"
    }
  }
}
```

### Register

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "attributes": {
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
      "plan_id": "free",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "created_at": "2025-01-19T10:00:00Z"
    }
  }
}
```

### Get Current User

```http
GET /auth/me
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "attributes": {
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
      "plan_id": "free"
    }
  }
}
```

### Logout

```http
POST /auth/logout
Authorization: Bearer <jwt_token>
```

### Endpoint Summary

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| POST | `/auth/login` | No | User login |
| POST | `/auth/register` | No | Create new account |
| GET | `/auth/me` | Yes | Get current user |
| POST | `/auth/logout` | Yes | End session |

> **Note**: `/admin/login`, `/admin/register`, `/admin/me`, and `/admin/logout` are equivalent endpoints.

---

## OAuth/Social Login

Sign in with external providers:
- Google
- GitHub
- Generic OIDC

See [[OAuth]] for configuration.

---

## Admin Authentication

Admin access is controlled via the invite system:

1. First user during setup becomes admin
2. Existing admins can invite new admins
3. Invites are single-use and expire

See [[Admin-Invites]] for details.

---

## See Also

- [[API-Keys]] - API key management
- [[OAuth]] - OAuth provider configuration
- [[Security]] - Security overview
- [[Admin-Invites]] - Admin invite system
- [[Resource-Types]] - Full API resource documentation
