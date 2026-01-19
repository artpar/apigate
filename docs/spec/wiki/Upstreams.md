# Upstreams

An **upstream** is a backend service that APIGate proxies requests to.

---

## Overview

Upstreams define where your actual API lives. When a request matches a route, APIGate forwards it to the route's configured upstream.

```
┌──────────┐     ┌──────────┐     ┌──────────────────┐
│  Client  │────▶│ APIGate  │────▶│    Upstream      │
│          │     │          │     │                  │
│          │     │  Route   │     │ api.example.com  │
│          │     │    ▼     │     │                  │
│          │     │ Upstream │────▶│  /api/v1/...     │
└──────────┘     └──────────┘     └──────────────────┘
```

---

## Upstream Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier (auto-generated) |
| `name` | string | Human-readable name (required) |
| `description` | string | Upstream description |
| `base_url` | string | Backend URL (required) |
| `timeout_ms` | int | Request timeout in milliseconds (default: 30000) |
| `max_idle_conns` | int | Connection pool size (default: 100) |
| `idle_conn_timeout_ms` | int | Idle connection timeout (default: 90000) |
| `auth_type` | enum | Authentication type |
| `auth_header` | string | Custom auth header name |
| `auth_value` | string | Auth value (write-only, set via API but not returned) |
| `enabled` | bool | Whether upstream is active |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

---

## Authentication Types

### None

No authentication added to upstream requests.

```yaml
auth_type: none
```

### Bearer Token

Adds `Authorization: Bearer <token>` header.

```yaml
auth_type: bearer
auth_value_encrypted: <your-token>
```

### Custom Header

Adds a custom header with your value.

```yaml
auth_type: header
auth_header: X-API-Key
auth_value_encrypted: <your-key>
```

### Basic Auth

Adds `Authorization: Basic <base64>` header.

```yaml
auth_type: basic
auth_value_encrypted: <username:password>
```

---

## Creating Upstreams

### Admin UI

1. Go to **Upstreams** in the sidebar
2. Click **Add Upstream**
3. Fill in the details
4. Click **Save**

### CLI

```bash
# Basic upstream
apigate upstreams create \
  --name "my-api" \
  --url "https://api.example.com"

# With authentication
apigate upstreams create \
  --name "my-api" \
  --url "https://api.example.com" \
  --auth-type bearer \
  --auth-value "secret-token"

# With timeouts
apigate upstreams create \
  --name "my-api" \
  --url "https://api.example.com" \
  --timeout 30000 \
  --max-idle-conns 100
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/upstreams \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "name": "my-api",
    "base_url": "https://api.example.com",
    "timeout_ms": 30000,
    "auth_type": "bearer",
    "auth_value": "secret-token",
    "enabled": true
  }'
```

---

## Managing Upstreams

### List All

```bash
# CLI
apigate upstreams list

# API
curl http://localhost:8080/admin/upstreams
```

### Get One

```bash
# CLI
apigate upstreams get <id>

# API
curl http://localhost:8080/admin/upstreams/<id>
```

### Update

```bash
# CLI
apigate upstreams update <id> --timeout 60000

# API
curl -X PUT http://localhost:8080/admin/upstreams/<id> \
  -H "Content-Type: application/json" \
  -d '{"timeout_ms": 60000}'
```

### Delete

```bash
# CLI
apigate upstreams delete <id>

# API
curl -X DELETE http://localhost:8080/admin/upstreams/<id>
```

---

## Connection Pooling

APIGate maintains connection pools to upstreams for efficiency.

| Setting | Default | Description |
|---------|---------|-------------|
| `max_idle_conns` | 100 | Maximum idle connections |
| `idle_conn_timeout_ms` | 90000 | How long idle connections stay open |

### Tuning Tips

- **High traffic**: Increase `max_idle_conns` to reduce connection overhead
- **Low traffic**: Decrease `idle_conn_timeout_ms` to free resources
- **Flaky backend**: Decrease `timeout_ms` to fail fast

---

## Best Practices

### 1. Use Descriptive Names

```bash
# Good
apigate upstreams create --name "payments-service" ...
apigate upstreams create --name "user-api-v2" ...

# Bad
apigate upstreams create --name "api1" ...
apigate upstreams create --name "backend" ...
```

### 2. Set Appropriate Timeouts

```bash
# Fast endpoints (< 1s expected)
--timeout 5000

# Standard endpoints (1-5s expected)
--timeout 30000

# Long-running operations
--timeout 120000
```

### 3. Secure Credentials

- Never log auth values
- Use environment variables for sensitive data
- Rotate credentials regularly

### 4. One Upstream Per Service

Create separate upstreams for different services:

```bash
# Separate services
apigate upstreams create --name "auth-service" --url "https://auth.internal"
apigate upstreams create --name "user-service" --url "https://users.internal"
apigate upstreams create --name "order-service" --url "https://orders.internal"
```

---

## Troubleshooting

### Connection Refused

```
Error: dial tcp: connection refused
```

- Check the upstream URL is correct
- Verify the service is running
- Check firewall rules

### Timeout

```
Error: context deadline exceeded
```

- Increase `timeout_ms`
- Check network latency
- Verify backend performance

### SSL Certificate Error

```
Error: x509: certificate signed by unknown authority
```

- Use a valid SSL certificate
- Or configure the system to trust internal CAs

---

## See Also

- [[Routes]] - Map requests to upstreams
- [[Transformations]] - Modify requests before forwarding
- [[Protocols]] - HTTP, SSE, WebSocket support
