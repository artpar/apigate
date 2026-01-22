# Error Codes Specification

> Implementation: `pkg/jsonapi/errors.go`

All API errors follow the JSON:API error format.

## Error Object Structure

| Member | Type | Description | Required |
|--------|------|-------------|----------|
| `id` | string | Unique error identifier | Optional |
| `status` | string | HTTP status code as string | **Required** |
| `code` | string | Application-specific error code | **Required** |
| `title` | string | Short, human-readable summary | **Required** |
| `detail` | string | Human-readable explanation | Optional |
| `source` | object | Error source location | Optional |
| `links` | object | Links for more info | Optional |
| `meta` | object | Additional metadata | Optional |

**Implementation**: `pkg/jsonapi/types.go:56-65`

```go
type Error struct {
    ID     string       `json:"id,omitempty"`
    Links  *ErrorLinks  `json:"links,omitempty"`
    Status string       `json:"status"`
    Code   string       `json:"code"`
    Title  string       `json:"title"`
    Detail string       `json:"detail,omitempty"`
    Source *ErrorSource `json:"source,omitempty"`
    Meta   Meta         `json:"meta,omitempty"`
}
```

## Error Source Object

Points to the location of the error:

| Member | Description | Example |
|--------|-------------|---------|
| `pointer` | JSON pointer to offending field | `/data/attributes/email` |
| `parameter` | Query parameter that caused error | `page[number]` |
| `header` | Header that caused error | `Authorization` |

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

## Standard Error Codes

### Client Errors (4xx)

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `bad_request` | 400 | Bad Request | Malformed request syntax |
| `unauthorized` | 401 | Unauthorized | Missing or invalid authentication |
| `forbidden` | 403 | Forbidden | Authenticated but not authorized |
| `not_found` | 404 | Not Found | Resource doesn't exist |
| `method_not_allowed` | 405 | Method Not Allowed | HTTP method not supported |
| `conflict` | 409 | Conflict | Resource conflict (duplicate, etc.) |
| `validation_error` | 422 | Validation Failed | Request validation failed |
| `rate_limit_exceeded` | 429 | Too Many Requests | Rate limit exceeded |

### Metering Errors (4xx)

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

## Error Constructors

**Implementation**: `pkg/jsonapi/errors.go:99-203`

### Bad Request (400)

```go
jsonapi.ErrBadRequest("Invalid JSON body")
```

Response:
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

```go
jsonapi.ErrUnauthorized("Invalid API key")
```

Response:
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

```go
jsonapi.ErrForbidden("Admin access required")
```

Response:
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

```go
jsonapi.ErrNotFound("user")
jsonapi.ErrNotFoundWithID("user", "usr_123")
```

Response:
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

### Method Not Allowed (405)

The HTTP `Allow` header is set per RFC 7231 to indicate which methods are supported.

```go
jsonapi.WriteMethodNotAllowed(w, "PATCH", []string{"GET", "PUT", "DELETE"})
```

HTTP Headers:
```
HTTP/1.1 405 Method Not Allowed
Allow: GET, PUT, DELETE
Content-Type: application/vnd.api+json
```

Response:
```json
{
  "errors": [{
    "status": "405",
    "code": "method_not_allowed",
    "title": "Method Not Allowed",
    "detail": "PATCH is not supported. Use one of: GET, PUT, DELETE",
    "meta": {
      "requested_method": "PATCH",
      "allowed_methods": ["GET", "PUT", "DELETE"]
    }
  }]
}
```

**Self-Service**: The error response tells you exactly which methods ARE supported, not just what's wrong. Check the `Allow` header or `meta.allowed_methods` to see your options.

### Conflict (409)

```go
jsonapi.ErrConflict("User with this email already exists")
```

Response:
```json
{
  "errors": [{
    "status": "409",
    "code": "conflict",
    "title": "Conflict",
    "detail": "User with this email already exists"
  }]
}
```

### Validation Error (422)

```go
jsonapi.ErrValidation("email", "must be a valid email address")
jsonapi.ErrValidationRequired("email")
jsonapi.ErrValidationInvalid("email", "must be a valid email address")
```

Response:
```json
{
  "errors": [{
    "status": "422",
    "code": "validation_error",
    "title": "Validation Failed",
    "detail": "email must be a valid email address",
    "source": {
      "pointer": "/data/attributes/email"
    }
  }]
}
```

### Rate Limited (429)

```go
jsonapi.ErrRateLimited("Rate limit exceeded. Try again in 15 seconds.")
```

Response:
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

```go
jsonapi.ErrInternal("Database connection failed")
jsonapi.ErrFromError(err) // Wraps Go error
```

Response:
```json
{
  "errors": [{
    "status": "500",
    "code": "internal_error",
    "title": "Internal Server Error",
    "detail": "Database connection failed"
  }]
}
```

### Not Implemented (501)

```go
jsonapi.ErrNotImplemented("Bulk delete")
```

Response:
```json
{
  "errors": [{
    "status": "501",
    "code": "not_implemented",
    "title": "Not Implemented",
    "detail": "Bulk delete is not implemented"
  }]
}
```

### Service Unavailable (503)

```go
jsonapi.ErrServiceUnavailable("Database maintenance in progress")
```

Response:
```json
{
  "errors": [{
    "status": "503",
    "code": "service_unavailable",
    "title": "Service Unavailable",
    "detail": "Database maintenance in progress"
  }]
}
```

## Building Custom Errors

Use the fluent builder for complex errors:

```go
err := jsonapi.NewError(422, "invalid_format", "Invalid Format").
    Detail("The date must be in RFC3339 format").
    Pointer("/data/attributes/start_date").
    Meta("expected_format", "2006-01-02T15:04:05Z07:00").
    AboutLink("https://docs.example.com/errors/invalid-format").
    Build()

jsonapi.WriteError(w, err)
```

## Multiple Errors

Return multiple errors when appropriate:

```go
errors := []jsonapi.Error{
    jsonapi.ErrValidationRequired("email"),
    jsonapi.ErrValidationRequired("name"),
    jsonapi.ErrValidationInvalid("plan_id", "must be a valid plan ID"),
}

jsonapi.WriteError(w, errors...)
```

Response:
```json
{
  "errors": [
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "email is required",
      "source": { "pointer": "/data/attributes/email" }
    },
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "name is required",
      "source": { "pointer": "/data/attributes/name" }
    },
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "plan_id is invalid: must be a valid plan ID",
      "source": { "pointer": "/data/attributes/plan_id" }
    }
  ]
}
```

## Response Helper Functions

| Function | Use Case |
|----------|----------|
| `WriteBadRequest(w, detail)` | 400 errors |
| `WriteUnauthorized(w, detail)` | 401 errors |
| `WriteForbidden(w, detail)` | 403 errors |
| `WriteNotFound(w, resourceType)` | 404 errors |
| `WriteConflict(w, detail)` | 409 errors |
| `WriteValidationError(w, field, message)` | 422 errors |
| `WriteInternalError(w, detail)` | 500 errors |
| `WriteErrorFromGo(w, err)` | Convert Go error to 500 |
| `WriteError(w, ...errors)` | Custom errors |
