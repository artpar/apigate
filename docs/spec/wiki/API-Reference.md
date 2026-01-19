# API Reference

Complete reference for the APIGate Admin API.

---

## Overview

The Admin API follows JSON:API v1.1 specification. All endpoints return JSON:API formatted responses.

See [[JSON-API-Format]] for response format details.

---

## Authentication

Admin API requires authentication via session cookie or admin API key.

---

## Resource Endpoints

### Users

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | List users |
| GET | `/api/users/{id}` | Get user |
| POST | `/api/users` | Create user |
| PATCH | `/api/users/{id}` | Update user |
| DELETE | `/api/users/{id}` | Delete user |

See [[Users]] for schema.

### Plans

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/plans` | List plans |
| GET | `/api/plans/{id}` | Get plan |
| POST | `/api/plans` | Create plan |
| PATCH | `/api/plans/{id}` | Update plan |
| DELETE | `/api/plans/{id}` | Delete plan |

See [[Plans]] for schema.

### API Keys

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/api-keys` | List keys |
| GET | `/api/api-keys/{id}` | Get key |
| POST | `/api/api-keys` | Create key |
| DELETE | `/api/api-keys/{id}` | Delete key |
| POST | `/api/api-keys/{id}/revoke` | Revoke key |

See [[API-Keys]] for schema.

### Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/routes` | List routes |
| GET | `/api/routes/{id}` | Get route |
| POST | `/api/routes` | Create route |
| PATCH | `/api/routes/{id}` | Update route |
| DELETE | `/api/routes/{id}` | Delete route |

See [[Routes]] for schema.

### Upstreams

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/upstreams` | List upstreams |
| GET | `/api/upstreams/{id}` | Get upstream |
| POST | `/api/upstreams` | Create upstream |
| PATCH | `/api/upstreams/{id}` | Update upstream |
| DELETE | `/api/upstreams/{id}` | Delete upstream |

See [[Upstreams]] for schema.

### Groups

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/groups` | List groups |
| GET | `/api/groups/{id}` | Get group |
| POST | `/api/groups` | Create group |
| PATCH | `/api/groups/{id}` | Update group |
| DELETE | `/api/groups/{id}` | Delete group |

See [[Groups]] for schema.

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/webhooks` | List webhooks |
| GET | `/api/webhooks/{id}` | Get webhook |
| POST | `/api/webhooks` | Create webhook |
| PATCH | `/api/webhooks/{id}` | Update webhook |
| DELETE | `/api/webhooks/{id}` | Delete webhook |

See [[Webhooks]] for schema.

---

## Resource Types

See [[Resource-Types]] for complete schema documentation for all resources.

---

## Error Codes

See [[Error-Codes]] for all error codes and their meanings.

---

## Pagination

See [[Pagination]] for pagination parameters and response format.

---

## See Also

- [[JSON-API-Format]] - Response format
- [[Resource-Types]] - Resource schemas
- [[Error-Codes]] - Error reference
- [[Pagination]] - Pagination details
