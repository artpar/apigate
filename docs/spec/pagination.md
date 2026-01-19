# Pagination Specification

> Implementation: `pkg/jsonapi/pagination.go`

All collection endpoints support pagination using JSON:API style parameters.

## Query Parameters

### JSON:API Style (Preferred)

| Parameter | Description | Example |
|-----------|-------------|---------|
| `page[number]` | Page number (1-based) | `?page[number]=2` |
| `page[size]` | Items per page | `?page[size]=50` |

### Legacy Style (Supported)

| Parameter | Description | Example |
|-----------|-------------|---------|
| `page` | Page number (1-based) | `?page=2` |
| `per_page` | Items per page | `?per_page=50` |
| `limit` | Alias for per_page | `?limit=50` |

**Implementation**: `pkg/jsonapi/pagination.go:113-157`

## Default Values

| Setting | Value |
|---------|-------|
| Default page | 1 |
| Default per_page | 20 |
| Maximum per_page | 100 |

## Response Structure

Paginated responses include both `meta` and `links` sections:

```json
{
  "data": [...],
  "meta": {
    "total": 150,
    "page": 2,
    "per_page": 20,
    "pages": 8
  },
  "links": {
    "self": "/admin/users?page[number]=2&page[size]=20",
    "first": "/admin/users?page[number]=1&page[size]=20",
    "last": "/admin/users?page[number]=8&page[size]=20",
    "prev": "/admin/users?page[number]=1&page[size]=20",
    "next": "/admin/users?page[number]=3&page[size]=20"
  }
}
```

## Meta Object

| Key | Type | Description |
|-----|------|-------------|
| `total` | int64 | Total number of items across all pages |
| `page` | int | Current page number (1-based) |
| `per_page` | int | Number of items per page |
| `pages` | int | Total number of pages |

**Implementation**: `pkg/jsonapi/pagination.go:104-111`

## Links Object

| Key | Type | Description | Presence |
|-----|------|-------------|----------|
| `self` | string | Current page URL | Always |
| `first` | string | First page URL | Always |
| `last` | string | Last page URL | Always |
| `prev` | string | Previous page URL | When page > 1 |
| `next` | string | Next page URL | When page < total pages |

**Implementation**: `pkg/jsonapi/pagination.go:65-82`

## Behavior Rules

### Page Number

1. Page numbers are 1-based (first page is 1, not 0)
2. Page number < 1 defaults to 1
3. Page number > total pages returns empty data array

### Per Page

1. Per page < 1 defaults to 20
2. Per page > 100 is capped at 100
3. Per page = 0 defaults to 20

### Empty Collections

When a collection has no items:

```json
{
  "data": [],
  "meta": {
    "total": 0,
    "page": 1,
    "per_page": 20,
    "pages": 1
  },
  "links": {
    "self": "/admin/users?page[number]=1&page[size]=20",
    "first": "/admin/users?page[number]=1&page[size]=20",
    "last": "/admin/users?page[number]=1&page[size]=20"
  }
}
```

## Implementation Usage

### Parsing Parameters

```go
import "github.com/artpar/apigate/pkg/jsonapi"

func ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse pagination params with default per_page of 20
    page, perPage := jsonapi.ParsePaginationParams(r.URL.Query(), 20)

    // Use for database query
    offset := (page - 1) * perPage
    limit := perPage

    // Query database
    users, total := db.ListUsers(offset, limit)

    // Create pagination object
    pagination := jsonapi.NewPagination(total, page, perPage, r.URL.String())

    // Write response
    jsonapi.WriteCollection(w, http.StatusOK, resources, pagination)
}
```

### Pagination Object Methods

| Method | Return | Description |
|--------|--------|-------------|
| `TotalPages()` | int | Calculate total pages |
| `HasPrev()` | bool | Check if previous page exists |
| `HasNext()` | bool | Check if next page exists |
| `Offset()` | int | Get database offset |
| `Limit()` | int | Get database limit |
| `Links()` | *Links | Generate pagination links |
| `Meta()` | Meta | Generate pagination metadata |

**Implementation**: `pkg/jsonapi/pagination.go:33-111`

## Example Requests

### First Page (Default)

```
GET /admin/users
```

Response includes:
- `page`: 1
- `per_page`: 20
- Links: self, first, last, next (if more pages)

### Specific Page

```
GET /admin/users?page[number]=3&page[size]=50
```

Response includes:
- `page`: 3
- `per_page`: 50
- Links: self, first, last, prev, next (if applicable)

### Last Page

```
GET /admin/users?page[number]=10&page[size]=20
```

Response includes:
- `page`: 10
- `per_page`: 20
- Links: self, first, last, prev (no next)

## Endpoints Supporting Pagination

| Endpoint | Default per_page |
|----------|------------------|
| `GET /admin/users` | 20 |
| `GET /admin/keys` | 20 |
| `GET /admin/plans` | 20 |
| `GET /admin/routes` | 20 |
| `GET /admin/upstreams` | 20 |
| Module list endpoints | 20 |
