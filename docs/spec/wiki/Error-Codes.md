# Error Codes

APIGate returns errors in JSON:API format with consistent error codes.

---

## Error Response Format

```json
{
  "errors": [
    {
      "status": "400",
      "code": "validation_error",
      "title": "Validation Error",
      "detail": "Email is required",
      "source": {
        "pointer": "/data/attributes/email"
      }
    }
  ]
}
```

### Error Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | HTTP status code (as string) |
| `code` | string | Machine-readable error code |
| `title` | string | Human-readable error title |
| `detail` | string | Specific error description |
| `source` | object | Location of error in request |
| `meta` | object | Additional error metadata |

---

## Error Codes by Category

### Authentication Errors (401)

| Code | Title | Description |
|------|-------|-------------|
| `unauthorized` | Unauthorized | No valid authentication provided |
| `invalid_api_key` | Invalid API Key | API key is malformed or doesn't exist |
| `expired_api_key` | API Key Expired | API key has passed its expiration date |
| `revoked_api_key` | API Key Revoked | API key has been revoked |
| `invalid_token` | Invalid Token | JWT or session token is invalid |
| `expired_token` | Token Expired | Authentication token has expired |

### Authorization Errors (403)

| Code | Title | Description |
|------|-------|-------------|
| `forbidden` | Forbidden | Action not permitted for this user |
| `insufficient_scope` | Insufficient Scope | API key lacks required scope |
| `account_suspended` | Account Suspended | User account is suspended |
| `plan_required` | Plan Required | Action requires a specific plan |

### Not Found Errors (404)

| Code | Title | Description |
|------|-------|-------------|
| `not_found` | Not Found | Requested resource doesn't exist |
| `user_not_found` | User Not Found | Specified user doesn't exist |
| `route_not_found` | Route Not Found | No route matches the request |
| `upstream_not_found` | Upstream Not Found | Specified upstream doesn't exist |
| `plan_not_found` | Plan Not Found | Specified plan doesn't exist |

### Validation Errors (400/422)

| Code | Title | Description |
|------|-------|-------------|
| `validation_error` | Validation Error | Request failed validation |
| `invalid_json` | Invalid JSON | Request body is not valid JSON |
| `missing_field` | Missing Field | Required field not provided |
| `invalid_field` | Invalid Field | Field value is invalid |
| `invalid_email` | Invalid Email | Email format is invalid |

### Rate Limiting Errors (429)

| Code | Title | Description |
|------|-------|-------------|
| `rate_limited` | Rate Limit Exceeded | Too many requests in time window |
| `quota_exceeded` | Quota Exceeded | Monthly quota has been exceeded |

### Conflict Errors (409)

| Code | Title | Description |
|------|-------|-------------|
| `conflict` | Conflict | Resource state conflict |
| `duplicate_email` | Duplicate Email | Email already registered |
| `duplicate_name` | Duplicate Name | Name already exists |

### Server Errors (5xx)

| Code | Title | Description |
|------|-------|-------------|
| `internal_error` | Internal Error | Unexpected server error |
| `upstream_error` | Upstream Error | Error from upstream service |
| `upstream_timeout` | Upstream Timeout | Upstream service timed out |
| `service_unavailable` | Service Unavailable | Service temporarily unavailable |

---

## Error Response Examples

### Invalid API Key

```json
{
  "errors": [{
    "status": "401",
    "code": "invalid_api_key",
    "title": "Invalid API Key",
    "detail": "The provided API key is invalid or has been revoked"
  }]
}
```

### Rate Limited

```json
{
  "errors": [{
    "status": "429",
    "code": "rate_limited",
    "title": "Rate Limit Exceeded",
    "detail": "You have exceeded the rate limit of 60 requests per minute. Please retry after 45 seconds.",
    "meta": {
      "limit": 60,
      "remaining": 0,
      "reset_at": "2025-01-19T10:01:00Z",
      "retry_after": 45
    }
  }]
}
```

### Validation Error

```json
{
  "errors": [
    {
      "status": "422",
      "code": "missing_field",
      "title": "Missing Field",
      "detail": "Email is required",
      "source": {
        "pointer": "/email"
      }
    },
    {
      "status": "422",
      "code": "invalid_field",
      "title": "Invalid Field",
      "detail": "Plan ID does not exist",
      "source": {
        "pointer": "/plan_id"
      }
    }
  ]
}
```

### Quota Exceeded

```json
{
  "errors": [{
    "status": "429",
    "code": "quota_exceeded",
    "title": "Quota Exceeded",
    "detail": "Your monthly quota of 10000 requests has been exceeded",
    "meta": {
      "quota_limit": 10000,
      "quota_used": 10001,
      "resets_at": "2025-02-01T00:00:00Z"
    }
  }]
}
```

### Upstream Timeout

```json
{
  "errors": [{
    "status": "504",
    "code": "upstream_timeout",
    "title": "Upstream Timeout",
    "detail": "The upstream service did not respond within 30 seconds",
    "meta": {
      "upstream": "api-service",
      "timeout_ms": 30000
    }
  }]
}
```

---

## HTTP Status Code Mapping

| Status Code | Category | Common Codes |
|-------------|----------|--------------|
| 400 | Bad Request | `invalid_json`, `validation_error` |
| 401 | Unauthorized | `unauthorized`, `invalid_api_key`, `expired_api_key` |
| 403 | Forbidden | `forbidden`, `insufficient_scope`, `account_suspended` |
| 404 | Not Found | `not_found`, `user_not_found`, `route_not_found` |
| 409 | Conflict | `conflict`, `duplicate_email` |
| 422 | Unprocessable Entity | `validation_error`, `missing_field`, `invalid_field` |
| 429 | Too Many Requests | `rate_limited`, `quota_exceeded` |
| 500 | Internal Server Error | `internal_error` |
| 502 | Bad Gateway | `upstream_error` |
| 503 | Service Unavailable | `service_unavailable` |
| 504 | Gateway Timeout | `upstream_timeout` |

---

## Handling Errors

### Client Best Practices

```python
import requests

def api_call(url, api_key):
    response = requests.get(url, headers={'X-API-Key': api_key})

    if response.status_code == 200:
        return response.json()['data']

    # Handle error
    error = response.json()['errors'][0]
    code = error['code']

    if code == 'rate_limited':
        retry_after = error.get('meta', {}).get('retry_after', 60)
        time.sleep(retry_after)
        return api_call(url, api_key)  # Retry

    elif code == 'quota_exceeded':
        raise QuotaExceededError(error['detail'])

    elif code in ['invalid_api_key', 'expired_api_key']:
        raise AuthenticationError(error['detail'])

    else:
        raise APIError(code, error['detail'])
```

---

## See Also

- [[JSON-API-Format]] - Response format specification
- [[Rate-Limiting]] - Rate limit behavior
- [[Quotas]] - Quota management
