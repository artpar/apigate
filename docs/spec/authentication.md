# Authentication Specification

**Status**: Authoritative
**Last Updated**: 2026-02-09
**Version**: 2.0

## Overview

APIGate uses **JWT (JSON Web Token) authentication** as the single auth mechanism across all components:

1. **JWT Bearer authentication** for the module WebUI (React SPA), admin API, and customer portal
2. **API key authentication** (header-based) for programmatic proxy access
3. **JWT cookie transport** for SSR Go template pages (same JWT, cookie transport)

A shared `TokenService` is injected at bootstrap into all components — proxy, admin handler, web handler, portal handler, and module HTTP channel. There is **no session store** and **no unsigned cookie sessions**.

---

## JWT Bearer Authentication

JWT Bearer authentication is the primary mechanism for all browser-based and SPA access. Tokens are returned in response bodies and sent via `Authorization: Bearer` headers.

### JWT Claims

```json
{
  "uid": "user_abc123",
  "email": "user@example.com",
  "role": "user",
  "pid": "free",
  "exp": 1738000000,
  "iat": 1737395200
}
```

| Claim | Type | Description |
|-------|------|-------------|
| `uid` | string | User ID |
| `email` | string | User email address |
| `role` | string | User role (`user`, `admin`) |
| `pid` | string | Plan ID (used by proxy for `X-Plan-ID` headers and rate limiting) |
| `exp` | int | Expiration timestamp (Unix) |
| `iat` | int | Issued-at timestamp (Unix) |

### Token Lifetime

- **Default**: 7 days (module HTTP channel), 24 hours (admin API)
- **Signing**: HMAC-SHA256 with shared `auth.jwt_secret` from settings
- **Refresh**: `POST /auth/refresh` returns a new token

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mod/auth/register` | Create user account, return JWT in response body |
| POST | `/mod/auth/login` | Authenticate user, return JWT in response body |
| POST | `/mod/auth/logout` | Stateless — returns `{"success": true}` |
| GET | `/mod/auth/me` | Get current user (requires `Authorization: Bearer` header) |
| GET | `/mod/auth/setup-required` | Check if first-time setup is needed |
| POST | `/mod/auth/setup` | Create first admin user (setup mode only) |

### Registration Flow

```http
POST /mod/auth/register HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123",
  "name": "John Doe"
}
```

**Success Response (201 Created)**:
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-02-16T12:00:00Z",
  "user": {
    "id": "user_abc123",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

**Error Response (409 Conflict - Email Exists)**:
```http
HTTP/1.1 409 Conflict
Content-Type: application/json

{
  "error": "email already registered"
}
```

### Login Flow

```http
POST /mod/auth/login HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123"
}
```

**Success Response (200 OK)**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-02-16T12:00:00Z",
  "user": {
    "id": "user_abc123",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

**Error Response (401 Unauthorized)**:
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "invalid email or password"
}
```

### Authenticated Requests

Include the JWT token in the `Authorization` header:

```http
GET /mod/auth/me HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Success Response (200 OK)**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "user": {
    "id": "user_abc123",
    "email": "user@example.com",
    "name": "John Doe",
    "status": "active"
  }
}
```

**Error Response (401 Unauthorized)**:
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "not authenticated"
}
```

### Logout Flow

Logout is stateless. The client discards the token.

```http
POST /mod/auth/logout HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response (200 OK)**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true
}
```

### React SPA Integration

The React frontend stores the JWT in `localStorage` and sends it via `Authorization: Bearer` header:

```typescript
// Store token after login/register
localStorage.setItem('auth_token', data.token);

// Send on every request
const token = localStorage.getItem('auth_token');
fetch('/mod/auth/me', {
  headers: { 'Authorization': `Bearer ${token}` }
});

// Clear on logout
localStorage.removeItem('auth_token');
```

---

## Admin API Authentication

The admin API (`/admin/*`) accepts two authentication methods:

### 1. JWT Bearer Token

```http
GET /admin/users HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### 2. API Key

```http
GET /admin/users HTTP/1.1
X-API-Key: ak_1234567890abcdef
```

Or using Bearer format:
```http
GET /admin/users HTTP/1.1
Authorization: Bearer ak_1234567890abcdef
```

The admin middleware checks in order:
1. `Authorization: Bearer <jwt>` — validate as JWT
2. `Authorization: Bearer <api-key>` — validate as API key (if JWT validation fails)
3. `X-API-Key` header — validate as API key

---

## API Key Authentication

API key authentication is used for programmatic access to proxy routes. API keys are passed via HTTP headers.

### Header Format

```http
X-API-Key: ak_1234567890abcdef
```

Or using Bearer token format:
```http
Authorization: Bearer ak_1234567890abcdef
```

### Key Prefix

All API keys MUST start with the configured prefix (default: `ak_`).

**Example**: `ak_1234567890abcdef`

### Key Scopes

API keys can optionally be restricted to specific scopes. If a key has **no scopes** (empty list), it has **full access** (no restrictions). If scopes are set, only the listed capabilities are allowed.

| Scope | Description |
|-------|-------------|
| `meter:write` | Submit usage events via metering API |
| `*` | Wildcard — matches any scope |

Creating a scoped service key:

```http
POST /admin/keys HTTP/1.1
Authorization: Bearer <admin-jwt>
Content-Type: application/json

{
  "user_id": "user_service_account",
  "name": "Hoster Metering Service",
  "scopes": ["meter:write"],
  "quota_bypass": true
}
```

The `quota_bypass` flag exempts the key from rate limiting and quota enforcement (for trusted service accounts).

### Usage

```http
GET /api/v1/resource HTTP/1.1
X-API-Key: ak_1234567890abcdef
```

**Success**: Request proceeds with user context loaded from API key
**Failure (401)**: Invalid or revoked API key
**Failure (403)**: Key lacks required scope for the endpoint

---

## Proxy Route Authentication

Proxy routes use the `Authorization: Bearer` header to authenticate requests. The proxy detects the token type by format: API keys start with the configured prefix (e.g., `ak_`), while JWTs do not.

### Auth-Required Routes (Billing Routes)

Routes with `auth_required=1` accept both API keys and JWT Bearer tokens:

```http
GET /api/billing HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**JWT authentication flow:**

1. Validate the JWT signature and expiration
2. Look up user in the local database by `uid` claim
3. If found and active: use database user record (authoritative for `PlanID`, `Status`)
4. If found and suspended: return `403 user_suspended`
5. **If not found**: construct user from JWT claims directly (external auth)

This allows users who exist in an external auth provider (e.g., a hosting platform) but not in APIGate's local database to access billing routes using their JWT. The JWT claims (`uid`, `email`, `role`, `pid`) provide all necessary auth context.

### Pass-Through Routes (Public)

Routes with `auth_required=0` perform opportunistic JWT validation: if a valid JWT is present, auth context is populated for transforms (e.g., `X-User-ID` headers). Invalid or missing tokens are silently ignored — the request proceeds anonymously.

### Legacy `apigate_session` Cookie

The `apigate_session` cookie is **no longer set** by any APIGate component (removed in v2.0.0, Issue #58). Browsers may still send stale cookies from prior versions; the server ignores them. Users should clear old cookies if observed.

---

## SSR Web Handlers (Go Templates)

The root web handlers (`web/handlers.go`) use a `token` JWT cookie for SSR Go template pages. This is **not** a dual auth path — it is the same JWT token, transported via cookie because HTML form submissions cannot set `Authorization` headers.

The `token` cookie:
- Contains a signed JWT (same format as Bearer tokens)
- Is set on login/register via Go template handlers
- Is read by middleware to extract user identity
- Is separate from the removed `apigate_session` unsigned cookie

---

## Security Considerations

### JWT Security

1. **Signing**: HMAC-SHA256 with server-side secret
2. **Expiration**: All tokens have an `exp` claim, validated on every request
3. **Stateless**: No server-side session storage, no session invalidation
4. **No cookies for SPA**: JWT is stored in `localStorage`, sent via header (no CSRF risk)

### Password Requirements

- **Minimum length**: 8 characters
- **Storage**: Passwords MUST be hashed using bcrypt (cost 10+)

### Production Deployment

The JWT secret (`auth.jwt_secret` setting) MUST be a strong random value in production. If not set, one is generated at startup and stored in the database.

---

## Testing Requirements

All implementations MUST be tested for:

1. **Valid JWT**: Middleware accepts valid Bearer token and sets user context
2. **Missing header**: Returns 401 when `Authorization` header is absent
3. **Invalid token**: Returns 401 for malformed or tampered tokens
4. **Expired token**: Returns 401 for expired JWT
5. **No TokenService**: Returns 401 when token service is not configured
6. **PlanID roundtrip**: JWT claims preserve planID through generate → validate cycle
7. **API key auth**: Admin middleware accepts API keys via `X-API-Key` and `Authorization: Bearer`

---

## Implementation Notes

### Architecture

A single `TokenService` instance is created in `bootstrap/bootstrap.go` and injected into all components. If no JWT secret exists in settings, one is auto-generated and persisted to the database on first startup:

```go
jwtSecret := s.Get(settings.KeyAuthJWTSecret)
if jwtSecret == "" {
    jwtSecret = auth.GenerateSecret() // 32-byte random hex
    a.Settings.Set(ctx, settings.KeyAuthJWTSecret, jwtSecret)
}
tokenService := auth.NewTokenService(jwtSecret, 7*24*time.Hour)
proxyService.SetTokenService(tokenService)
moduleRuntime.HTTP.SetTokenService(tokenService)
// Admin and web handlers receive JWTSecret string in their Deps
```

### File Locations

| File | Purpose |
|------|---------|
| `adapters/auth/jwt.go` | `TokenService`, `Claims`, `GenerateToken`, `ValidateToken` |
| `core/channel/http/auth.go` | Module WebUI auth (JWT Bearer only) |
| `adapters/http/admin/admin.go` | Admin API auth (JWT Bearer + API key) |
| `bootstrap/bootstrap.go` | Wires shared TokenService into all components |
| `web/handlers.go` | SSR web handlers (`token` JWT cookie for Go templates) |

### Key Functions

```go
// Generate a JWT token with user claims
func (s *TokenService) GenerateToken(userID, email, role, planID string) (string, time.Time, error)

// Validate a JWT token and return claims
func (s *TokenService) ValidateToken(tokenString string) (*Claims, error)
```

---

## Historical Issues

### Issue #54: Cookies Not Set on HTTPS (Module Runtime Handler)

**Problem**: Session cookies were being set but rejected by browsers on HTTPS.
**Root Cause**: `Secure` flag hardcoded to `false`.
**Resolution**: Superseded by Issue #58 — cookies removed entirely from module auth.

### Issue #55: Cookies Not Set on HTTPS (Admin Handler)

**Problem**: Session cookies not being set on HTTPS for admin auth.
**Resolution**: Superseded by Issue #58 — cookies and sessions removed from admin handler.

### Issue #58: Replace Cookie/Session Auth with JWT-Only Auth

**Problem**: The `apigate_session` cookie was unsigned base64 JSON, creating a broken dual-auth path. Browser users authenticated via cookie couldn't access proxy routes because the proxy's auth context (`userID`, `planID`) doesn't resolve from unsigned cookies. Upstream services received no `X-User-ID` header → 401 errors.

**Root Cause**: The module HTTP channel's `AuthHandler` created its own isolated random JWT secret, disconnected from the shared `TokenService`. It then used unsigned base64 JSON cookies instead of JWT.

**Fix**:
- Removed all cookie/session infrastructure from module auth, admin auth, and web handlers
- Injected shared `TokenService` into module HTTP channel via `SetTokenService()`
- Added `PlanID` to JWT claims for proxy auth context resolution
- React SPA stores JWT in `localStorage`, sends via `Authorization: Bearer` header
- Single auth path: JWT tokens everywhere

---

## References

- [RFC 7519: JSON Web Token](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 6750: Bearer Token Usage](https://datatracker.ietf.org/doc/html/rfc6750)
- [OWASP: JWT Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
