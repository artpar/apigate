# Authentication Specification

**Status**: Authoritative
**Last Updated**: 2026-01-27
**Version**: 1.0

## Overview

APIGate provides two authentication mechanisms:
1. **Session-based authentication** (cookie-based) for web UI and customer portal
2. **API key authentication** (header-based) for programmatic API access

This document specifies the behavior, security requirements, and implementation details for both mechanisms.

---

## Session-Based Authentication

Session-based authentication is used for browser-based access to the Web UI and customer portal. It uses HTTP cookies to maintain state across requests.

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Create user account and return session cookie |
| POST | `/auth/login` | Authenticate user and return session cookie |
| POST | `/auth/logout` | Invalidate session |
| GET | `/auth/me` | Get current authenticated user |
| GET | `/auth/setup-required` | Check if first-time setup is needed |
| POST | `/auth/setup` | Create first admin user (setup mode only) |

For SPA frontends, these are also available at:
- `/api/portal/auth/*` (same routes, same behavior)

### Session Cookie Specification

#### Cookie Name
```
apigate_session
```

#### Cookie Attributes

| Attribute | Value | Requirement | Rationale |
|-----------|-------|-------------|-----------|
| **Name** | `apigate_session` | MUST | Identifies the session cookie |
| **Value** | Base64-encoded JSON | MUST | Contains serialized `Session` object |
| **Path** | `/` | MUST | Cookie valid for entire application |
| **HttpOnly** | `true` | MUST | Prevents XSS attacks (no JavaScript access) |
| **SameSite** | `Lax` | MUST | Prevents CSRF attacks while allowing navigation |
| **Secure** | Dynamic | MUST | `true` for HTTPS, `false` for HTTP |
| **Expires** | 7 days from creation | MUST | Automatic session expiration |

#### Secure Flag Behavior

The `Secure` flag MUST be set dynamically based on the request protocol:

**HTTPS Requests**:
```
Secure: true
```
- Direct HTTPS: Detected via `r.TLS != nil`
- Proxied HTTPS: Detected via `X-Forwarded-Proto: https` header

**HTTP Requests**:
```
Secure: false
```

**Rationale**: Modern browsers **reject cookies with `Secure: false` on HTTPS connections**. This is a security measure to prevent downgrade attacks. The server MUST detect the protocol and set the flag appropriately.

**Critical**: Failure to set `Secure: true` on HTTPS connections causes browsers to **silently discard** session cookies, resulting in authentication failure.

### Session Object Structure

The cookie value is a Base64-encoded JSON object:

```json
{
  "user_id": "user_abc123",
  "email": "user@example.com",
  "name": "John Doe",
  "expires_at": "2026-02-03T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | string | Unique user identifier |
| `email` | string | User email address |
| `name` | string | User display name |
| `expires_at` | timestamp | Session expiration (ISO 8601) |

### Registration Flow

```http
POST /auth/register HTTP/1.1
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
Set-Cookie: apigate_session=eyJ1c2VyX2lkIjoi...; Path=/; HttpOnly; SameSite=Lax; Secure; Expires=Mon, 03 Feb 2026 12:00:00 GMT
Content-Type: application/json

{
  "success": true,
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
POST /auth/login HTTP/1.1
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123"
}
```

**Success Response (200 OK)**:
```http
HTTP/1.1 200 OK
Set-Cookie: apigate_session=eyJ1c2VyX2lkIjoi...; Path=/; HttpOnly; SameSite=Lax; Secure; Expires=Mon, 03 Feb 2026 12:00:00 GMT
Content-Type: application/json

{
  "success": true,
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

### Session Validation

To use the session for authenticated requests, the browser automatically includes the cookie:

```http
GET /auth/me HTTP/1.1
Cookie: apigate_session=eyJ1c2VyX2lkIjoi...
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

```http
POST /auth/logout HTTP/1.1
Cookie: apigate_session=eyJ1c2VyX2lkIjoi...
```

**Response (200 OK)**:
```http
HTTP/1.1 200 OK
Set-Cookie: apigate_session=; Path=/; HttpOnly; SameSite=Lax; MaxAge=-1
Content-Type: application/json

{
  "success": true
}
```

The `MaxAge=-1` directive instructs the browser to delete the cookie immediately.

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

### Usage

```http
GET /api/v1/resource HTTP/1.1
X-API-Key: ak_1234567890abcdef
```

**Success**: Request proceeds with user context loaded from API key
**Failure (401)**: Invalid or revoked API key

---

## Security Considerations

### Cookie Security

1. **XSS Protection**: `HttpOnly` flag prevents JavaScript from accessing session cookies
2. **CSRF Protection**: `SameSite=Lax` prevents cross-site request forgery
3. **Transport Security**: `Secure` flag (on HTTPS) prevents cookie transmission over insecure connections
4. **Path Scoping**: `Path=/` limits cookie to application scope

### Password Requirements

- **Minimum length**: 8 characters
- **Complexity**: Must contain uppercase, lowercase, and digit
- **Storage**: Passwords MUST be hashed using bcrypt (cost 10)

### Session Expiration

- **Default lifetime**: 7 days
- **Expiration checking**: Server validates `expires_at` on each request
- **Expired sessions**: Return 401 Unauthorized

### Production Deployment

**Critical HTTPS Requirements**:
1. The `Secure` cookie flag MUST be `true` on HTTPS connections
2. Server MUST detect HTTPS via `r.TLS != nil` or `X-Forwarded-Proto` header
3. Browsers **will reject** cookies with `Secure: false` on HTTPS

**Reverse Proxy Configuration**:
If APIGate runs behind a reverse proxy (nginx, Caddy, etc.), the proxy MUST set:
```
X-Forwarded-Proto: https
```

This allows APIGate to detect the original protocol and set the `Secure` flag correctly.

---

## Testing Requirements

All implementations MUST be tested for:

1. **HTTP cookie behavior**: `Secure=false` on HTTP requests
2. **HTTPS cookie behavior**: `Secure=true` on HTTPS requests
3. **Proxied HTTPS behavior**: `Secure=true` when `X-Forwarded-Proto: https`
4. **Cookie attributes**: HttpOnly, SameSite, Path, Expires all set correctly
5. **Session expiration**: Cookies expire after 7 days
6. **Cookie encoding**: Value is valid Base64-encoded JSON

---

## Implementation Notes

### File Location

Implementation: `core/channel/http/auth.go`

### Key Functions

```go
// setSessionCookie sets the session cookie with protocol-aware Secure flag
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, session Session)
```

**Protocol Detection**:
```go
isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
```

### Testing

Tests: `core/channel/http/auth_secure_cookie_test.go`

Required test coverage:
- HTTP request → `Secure=false`
- HTTPS request → `Secure=true`
- Proxied HTTPS → `Secure=true`
- All cookie attributes validated
- Cookie expiration validated

---

## Historical Issues

### Issue #54: Cookies Not Set on HTTPS

**Problem**: Session cookies were being set by the server but rejected by browsers on HTTPS deployments.

**Root Cause**: The `Secure` flag was hardcoded to `false`, causing browsers to silently reject cookies on HTTPS connections.

**Fix**: Implemented dynamic `Secure` flag based on protocol detection (commit `aa1ffe4`).

**Lesson**: Cookie behavior differs significantly between HTTP and HTTPS. Always test authentication in production-like environments (HTTPS).

---

## References

- [MDN: Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie)
- [MDN: Using HTTP cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies)
- [OWASP: Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [RFC 6265: HTTP State Management Mechanism](https://datatracker.ietf.org/doc/html/rfc6265)
