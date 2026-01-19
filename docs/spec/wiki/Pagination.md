# Pagination

APIGate uses JSON:API style pagination for collection endpoints.

---

## Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page[number]` | int | 1 | Page number (1-indexed) |
| `page[size]` | int | 20 | Items per page (max: 100) |

### Example Request

```bash
curl -H "X-API-Key: ak_xxx" \
  "http://localhost:8080/api/users?page[number]=2&page[size]=50"
```

---

## Response Format

### Pagination Metadata

Collection responses include pagination in `meta` and `links`:

```json
{
  "data": [...],
  "meta": {
    "total": 100,
    "page": 2,
    "per_page": 20,
    "pages": 5
  },
  "links": {
    "self": "/api/users?page[number]=2&page[size]=20",
    "first": "/api/users?page[number]=1&page[size]=20",
    "prev": "/api/users?page[number]=1&page[size]=20",
    "next": "/api/users?page[number]=3&page[size]=20",
    "last": "/api/users?page[number]=5&page[size]=20"
  }
}
```

### Meta Fields

| Field | Type | Description |
|-------|------|-------------|
| `total` | int | Total items across all pages |
| `page` | int | Current page number |
| `per_page` | int | Items per page |
| `pages` | int | Total number of pages |

### Link Fields

| Field | Description |
|-------|-------------|
| `self` | Current page URL |
| `first` | First page URL |
| `prev` | Previous page URL (omitted on first page) |
| `next` | Next page URL (omitted on last page) |
| `last` | Last page URL |

---

## Default Values

| Setting | Value |
|---------|-------|
| Default page size | 20 |
| Maximum page size | 100 |
| Starting page | 1 |

---

## Examples

### First Page

```bash
curl "http://localhost:8080/api/users"
# Equivalent to: ?page[number]=1&page[size]=20
```

Response:
```json
{
  "data": [...],
  "meta": {
    "total": 100,
    "page": 1,
    "per_page": 20,
    "pages": 5
  },
  "links": {
    "self": "/api/users?page[number]=1&page[size]=20",
    "first": "/api/users?page[number]=1&page[size]=20",
    "next": "/api/users?page[number]=2&page[size]=20",
    "last": "/api/users?page[number]=5&page[size]=20"
  }
}
```

### Custom Page Size

```bash
curl "http://localhost:8080/api/users?page[size]=50"
```

Response:
```json
{
  "meta": {
    "total": 100,
    "page": 1,
    "per_page": 50,
    "pages": 2
  }
}
```

### Last Page

```bash
curl "http://localhost:8080/api/users?page[number]=5"
```

Response:
```json
{
  "meta": {
    "total": 100,
    "page": 5,
    "per_page": 20,
    "pages": 5
  },
  "links": {
    "self": "/api/users?page[number]=5&page[size]=20",
    "first": "/api/users?page[number]=1&page[size]=20",
    "prev": "/api/users?page[number]=4&page[size]=20",
    "last": "/api/users?page[number]=5&page[size]=20"
  }
}
```

Note: `next` link is omitted on the last page.

---

## Edge Cases

### Empty Collection

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
    "self": "/api/users?page[number]=1&page[size]=20",
    "first": "/api/users?page[number]=1&page[size]=20",
    "last": "/api/users?page[number]=1&page[size]=20"
  }
}
```

### Page Beyond Range

Request for page 10 when only 5 pages exist:

```json
{
  "data": [],
  "meta": {
    "total": 100,
    "page": 10,
    "per_page": 20,
    "pages": 5
  }
}
```

### Invalid Parameters

| Invalid Input | Behavior |
|---------------|----------|
| `page[number]=0` | Defaults to 1 |
| `page[number]=-1` | Defaults to 1 |
| `page[size]=0` | Defaults to 20 |
| `page[size]=200` | Capped at 100 |
| `page[size]=-1` | Defaults to 20 |

---

## Client Implementation

### Iterate All Pages

```python
def get_all_items(base_url, api_key):
    items = []
    page = 1

    while True:
        response = requests.get(
            f"{base_url}?page[number]={page}&page[size]=100",
            headers={'X-API-Key': api_key}
        )
        data = response.json()

        items.extend(data['data'])

        if 'next' not in data.get('links', {}):
            break

        page += 1

    return items
```

### Using Links

```python
def get_all_items_with_links(url, api_key):
    items = []

    while url:
        response = requests.get(url, headers={'X-API-Key': api_key})
        data = response.json()

        items.extend(data['data'])

        # Follow 'next' link
        url = data.get('links', {}).get('next')

    return items
```

---

## See Also

- [[JSON-API-Format]] - Response format
- [[Error-Codes]] - Error responses
