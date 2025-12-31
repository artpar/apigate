# E1: Authentication Errors

> **The first line of defense - clear errors that guide to resolution.**

---

## Overview

Authentication errors occur when API requests fail due to invalid, missing, or expired credentials. Clear error messages help users self-resolve quickly.

---

## Error Types

### Missing API Key (401)

**Trigger:** Request without `X-API-Key` header or `Authorization: Bearer` token.

**Response:**
```json
{
  "error": {
    "code": "missing_api_key",
    "message": "API key is required. Include it in the X-API-Key header.",
    "docs": "/docs/authentication"
  }
}
```

**Headers:**
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="api"
```

**User Experience:**
- Clear indication of what's missing
- Link to authentication docs
- Example of correct header format

**Screenshot:** `errors/e1-01-missing-key.png`

---

### Invalid API Key (401)

**Trigger:** Key doesn't match expected format or doesn't exist.

**Response:**
```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "The API key provided is invalid.",
    "docs": "/docs/authentication"
  }
}
```

**Common Causes:**
- Key copied incorrectly (truncated)
- Key from different environment
- Key was revoked
- Typo in key

**User Recovery:**
1. Check key is complete (67 characters)
2. Verify key starts with `ak_`
3. Check if key was revoked in portal
4. Generate new key if needed

**Screenshot:** `errors/e1-02-invalid-key.png`

---

### Expired API Key (401)

**Trigger:** Key has passed its expiration date.

**Response:**
```json
{
  "error": {
    "code": "key_expired",
    "message": "This API key has expired. Please generate a new key.",
    "expired_at": "2024-01-01T00:00:00Z",
    "docs": "/portal/keys"
  }
}
```

**User Recovery:**
1. Log in to portal
2. Revoke expired key
3. Generate new key
4. Update application

**Screenshot:** `errors/e1-03-expired-key.png`

---

### Account Suspended (403)

**Trigger:** User account has been suspended by admin.

**Response:**
```json
{
  "error": {
    "code": "account_suspended",
    "message": "Your account has been suspended. Please contact support.",
    "support": "support@example.com"
  }
}
```

**User Recovery:**
1. Check email for suspension notice
2. Contact support
3. Resolve issue (payment, ToS violation, etc.)
4. Account reactivated by admin

**Screenshot:** `errors/e1-04-suspended.png`

---

## UX Guidelines

### Error Message Quality

| Principle | Good | Bad |
|-----------|------|-----|
| Specific | "API key is required" | "Unauthorized" |
| Actionable | "Include it in X-API-Key header" | "Please authenticate" |
| Helpful | Link to docs | No guidance |

### Error Response Format

All auth errors should include:
```json
{
  "error": {
    "code": "machine_readable_code",
    "message": "Human readable message",
    "docs": "/link/to/relevant/docs"
  }
}
```

### Don't Reveal Too Much

For security, don't distinguish between:
- User not found
- Wrong password
- Account locked

Use generic: "Invalid credentials"

---

## Screenshot Automation

```yaml
journey: e1-authentication-errors
viewport: 1280x720

steps:
  - name: missing-key
    action: api_call
    request:
      method: GET
      url: /api/data
      # No API key
    capture: response

  - name: invalid-key
    action: api_call
    request:
      method: GET
      url: /api/data
      headers:
        X-API-Key: "ak_invalid"
    capture: response

  - name: expired-key
    setup: create_expired_key
    action: api_call
    request:
      method: GET
      url: /api/data
      headers:
        X-API-Key: "${EXPIRED_KEY}"
    capture: response
```

---

## Related

- [J6: API Access](../customer/j6-api-access.md) - Creating valid keys
- [J9: Documentation](../customer/j9-documentation.md) - Auth docs
