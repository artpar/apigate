# JSON:API Response Format

> Implementation: `pkg/jsonapi/`

APIGate implements the [JSON:API v1.1 specification](https://jsonapi.org/format/1.1/).

## Content Type

All API responses use the JSON:API media type:

```
Content-Type: application/vnd.api+json
```

## Document Structure

### Top-Level Members

Every JSON:API response is a **document** with these possible members:

| Member | Type | Description | Required |
|--------|------|-------------|----------|
| `data` | Resource, Resource[], null | Primary data | Mutually exclusive with `errors` |
| `errors` | Error[] | Error objects | Mutually exclusive with `data` |
| `meta` | object | Non-standard meta-information | Optional |
| `links` | Links | Pagination/navigation links | Optional |
| `included` | Resource[] | Related resources (compound documents) | Optional |
| `jsonapi` | object | JSON:API version info | Optional |

**Implementation**: `pkg/jsonapi/types.go:7-14`

```go
type Document struct {
    Data     any        `json:"data,omitempty"`
    Errors   []Error    `json:"errors,omitempty"`
    Meta     Meta       `json:"meta,omitempty"`
    Links    *Links     `json:"links,omitempty"`
    Included []Resource `json:"included,omitempty"`
    JSONAPI  *JSONAPI   `json:"jsonapi,omitempty"`
}
```

## Resource Objects

A resource object represents a single entity.

### Structure

| Member | Type | Description | Required |
|--------|------|-------------|----------|
| `type` | string | Resource type (plural form) | **Required** |
| `id` | string | Unique identifier | **Required** |
| `attributes` | object | Resource attributes | Optional |
| `relationships` | object | Related resources | Optional |
| `links` | object | Resource links | Optional |
| `meta` | object | Resource metadata | Optional |

**Implementation**: `pkg/jsonapi/types.go:17-24`

```go
type Resource struct {
    Type          string                  `json:"type"`
    ID            string                  `json:"id"`
    Attributes    map[string]any          `json:"attributes,omitempty"`
    Relationships map[string]Relationship `json:"relationships,omitempty"`
    Links         *ResourceLinks          `json:"links,omitempty"`
    Meta          Meta                    `json:"meta,omitempty"`
}
```

### Example: Single Resource Response

```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "attributes": {
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
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

### Example: Collection Response

```json
{
  "data": [
    {
      "type": "users",
      "id": "usr_abc123",
      "attributes": {
        "email": "user1@example.com",
        "name": "User One"
      }
    },
    {
      "type": "users",
      "id": "usr_def456",
      "attributes": {
        "email": "user2@example.com",
        "name": "User Two"
      }
    }
  ],
  "meta": {
    "total": 100,
    "page": 1,
    "per_page": 20,
    "pages": 5
  },
  "links": {
    "self": "/api/users?page[number]=1&page[size]=20",
    "first": "/api/users?page[number]=1&page[size]=20",
    "last": "/api/users?page[number]=5&page[size]=20",
    "next": "/api/users?page[number]=2&page[size]=20"
  }
}
```

## Relationships

Relationships describe connections between resources.

### To-One Relationship

```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "relationships": {
      "plan": {
        "data": { "type": "plans", "id": "plan_pro" }
      }
    }
  }
}
```

### To-Many Relationship

```json
{
  "data": {
    "type": "users",
    "id": "usr_abc123",
    "relationships": {
      "api_keys": {
        "data": [
          { "type": "keys", "id": "key_001" },
          { "type": "keys", "id": "key_002" }
        ]
      }
    }
  }
}
```

**Implementation**: `pkg/jsonapi/resource.go:52-77`

## Links Object

Links provide navigation URLs.

| Member | Description |
|--------|-------------|
| `self` | Link to current resource/page |
| `related` | Link to related resource |
| `first` | First page (pagination) |
| `last` | Last page (pagination) |
| `prev` | Previous page (pagination) |
| `next` | Next page (pagination) |

**Implementation**: `pkg/jsonapi/types.go:41-48`

## Meta Object

Meta contains non-standard information. Common uses:

| Key | Type | Description |
|-----|------|-------------|
| `total` | int | Total items in collection |
| `page` | int | Current page number |
| `per_page` | int | Items per page |
| `pages` | int | Total pages |

## Response Helpers

The `pkg/jsonapi/response.go` provides convenience functions:

| Function | HTTP Status | Use Case |
|----------|-------------|----------|
| `WriteResource` | Variable | Single resource |
| `WriteCollection` | 200 | Resource collection |
| `WriteCreated` | 201 | Resource created |
| `WriteNoContent` | 204 | Successful delete |
| `WriteAccepted` | 202 | Async operation started |
| `WriteMeta` | Variable | Metadata-only response |
| `WriteError` | Variable | Error response |

### Status Code Usage

| Status | Usage |
|--------|-------|
| 200 OK | Successful GET, PUT, PATCH |
| 201 Created | Successful POST creating a resource |
| 202 Accepted | Async operation accepted |
| 204 No Content | Successful DELETE |
| 400 Bad Request | Invalid request syntax |
| 401 Unauthorized | Missing/invalid authentication |
| 403 Forbidden | Authenticated but not authorized |
| 404 Not Found | Resource doesn't exist |
| 409 Conflict | Resource conflict (e.g., duplicate) |
| 422 Unprocessable Entity | Validation error |
| 429 Too Many Requests | Rate limit exceeded |
| 500 Internal Server Error | Server error |
| 501 Not Implemented | Feature not implemented |
| 503 Service Unavailable | Service temporarily unavailable |

## Request Format

### Creating Resources

POST requests should send resource data in the request body:

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "plan_id": "plan_pro"
}
```

> Note: APIGate currently accepts flat JSON for creation, not wrapped in `data.attributes`. This may change in future versions.

### Updating Resources

PUT/PATCH requests follow the same format as creation.

## Building Resources

Use the fluent builder API:

```go
import "github.com/artpar/apigate/pkg/jsonapi"

// Create a resource
resource := jsonapi.NewResource("users", "usr_123").
    Attr("email", "user@example.com").
    Attr("name", "John Doe").
    Attr("status", "active").
    BelongsTo("plan", "plans", "plan_pro").
    Build()

// Write response
jsonapi.WriteResource(w, http.StatusOK, resource)
```

## Building Documents

```go
// Collection with pagination
pagination := jsonapi.NewPagination(total, page, perPage, baseURL)
jsonapi.WriteCollection(w, http.StatusOK, resources, pagination)

// Error response
jsonapi.WriteError(w, jsonapi.ErrNotFound("user"))

// Meta-only response
jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
    "status": "healthy",
    "version": "1.0.0",
})
```
