# Error Codes

APIGate returns errors in JSON:API format with consistent error codes.

---

## Error Response Format

```json
{
  "errors": [
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "email is required",
      "source": {
        "pointer": "/data/attributes/email"
      }
    }
  ]
}
```

### Error Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | HTTP status code (as string) |
| `code` | string | Yes | Machine-readable error code |
| `title` | string | Yes | Human-readable error title |
| `detail` | string | No | Specific error description |
| `source` | object | No | Location of error in request |
| `meta` | object | No | Additional error metadata |

### Source Object Fields

| Field | Description | Example |
|-------|-------------|---------|
| `pointer` | JSON pointer to offending field | `/data/attributes/email` |
| `parameter` | Query parameter that caused error | `page[number]` |
| `header` | Header that caused error | `Authorization` |

---

## Standard Error Codes

### Client Errors (4xx)

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `bad_request` | 400 | Bad Request | Malformed request syntax, invalid JSON |
| `unauthorized` | 401 | Unauthorized | Missing or invalid authentication |
| `forbidden` | 403 | Forbidden | Authenticated but not authorized |
| `not_found` | 404 | Not Found | Resource doesn't exist |
| `method_not_allowed` | 405 | Method Not Allowed | HTTP method not supported |
| `conflict` | 409 | Conflict | Resource conflict (duplicate, etc.) |
| `validation_error` | 422 | Validation Failed | Request validation failed |
| `rate_limit_exceeded` | 429 | Too Many Requests | Rate limit exceeded |

### Metering API Errors (4xx)

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `invalid_event_type` | 422 | Invalid Event Type | Unknown event type submitted |
| `duplicate_event` | 409 | Duplicate Event | Event ID already processed |
| `user_not_found` | 422 | User Not Found | user_id doesn't exist |
| `invalid_quantity` | 422 | Invalid Quantity | Quantity <= 0 |
| `invalid_timestamp` | 422 | Invalid Timestamp | Timestamp in future or too old |
| `insufficient_scope` | 403 | Insufficient Scope | API key lacks required scope |

### Server Errors (5xx)

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `internal_error` | 500 | Internal Server Error | Unexpected server error |
| `not_implemented` | 501 | Not Implemented | Feature not implemented |
| `service_unavailable` | 503 | Service Unavailable | Service temporarily down |

---

## Error Response Examples

### Bad Request (400)

```json
{
  "errors": [{
    "status": "400",
    "code": "bad_request",
    "title": "Bad Request",
    "detail": "Invalid JSON body"
  }]
}
```

### Unauthorized (401)

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

### Forbidden (403)

```json
{
  "errors": [{
    "status": "403",
    "code": "forbidden",
    "title": "Forbidden",
    "detail": "Admin access required"
  }]
}
```

### Not Found (404)

```json
{
  "errors": [{
    "status": "404",
    "code": "not_found",
    "title": "Not Found",
    "detail": "The requested user was not found"
  }]
}
```

### Validation Error (422)

```json
{
  "errors": [
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "email is required",
      "source": {
        "pointer": "/data/attributes/email"
      }
    },
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "name is required",
      "source": {
        "pointer": "/data/attributes/name"
      }
    }
  ]
}
```

### Rate Limited (429)

```json
{
  "errors": [{
    "status": "429",
    "code": "rate_limit_exceeded",
    "title": "Too Many Requests",
    "detail": "Rate limit exceeded. Try again in 15 seconds."
  }]
}
```

### Internal Error (500)

```json
{
  "errors": [{
    "status": "500",
    "code": "internal_error",
    "title": "Internal Server Error",
    "detail": "An internal error occurred"
  }]
}
```

---

## Handling Errors

### Client Best Practices

```python
import requests
import time

def api_call(url, api_key):
    response = requests.get(url, headers={'X-API-Key': api_key})

    if response.status_code == 200:
        return response.json()['data']

    # Handle error
    error = response.json()['errors'][0]
    code = error['code']

    if code == 'rate_limit_exceeded':
        # Retry after delay
        time.sleep(60)
        return api_call(url, api_key)

    elif code == 'unauthorized':
        raise AuthenticationError(error['detail'])

    elif code == 'validation_error':
        raise ValidationError(error['detail'], error.get('source'))

    else:
        raise APIError(code, error['detail'])
```

---

## See Also

- [[JSON-API-Format]] - Response format specification
- [[Pagination]] - Collection pagination
- [[Resource-Types]] - API resource types
