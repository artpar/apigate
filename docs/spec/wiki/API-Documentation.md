# API Documentation

APIGate provides built-in API documentation via Swagger/OpenAPI.

---

## Built-in Documentation

### Swagger UI

When OpenAPI is enabled, access Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

Features:
- Interactive API explorer
- Try-it-out functionality
- Authentication testing

### OpenAPI Specification

The OpenAPI JSON specification is available at:

```
http://localhost:8080/.well-known/openapi.json
```

---

## Enabling Documentation

### Environment Variables

```bash
# Enable OpenAPI/Swagger endpoints
APIGATE_OPENAPI_ENABLED=true
```

### Configuration File

```yaml
openapi:
  enabled: true
```

---

## Customer Documentation Portal

If a docs handler is configured, customer-facing documentation is served at `/docs`.

The docs portal is generated from route configuration and includes:
- Endpoint descriptions
- Code examples
- Authentication guide

---

## Admin API Documentation

The admin API is documented with OpenAPI annotations in the source code. Key endpoints include:

| Endpoint | Description |
|----------|-------------|
| `POST /admin/login` | Authenticate to admin API |
| `GET /admin/users` | List users |
| `GET /admin/keys` | List API keys |
| `GET /admin/plans` | List subscription plans |
| `GET /admin/routes` | List routes |
| `GET /admin/upstreams` | List upstreams |
| `GET /admin/usage` | Get usage statistics |
| `GET /admin/settings` | Get current settings |
| `GET /admin/doctor` | System health check |

---

## See Also

- [[JSON-API-Format]] - Response format documentation
- [[API-Reference]] - Complete admin API reference
- [[Configuration]] - OpenAPI configuration options
