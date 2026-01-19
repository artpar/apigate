# Authentication

APIGate supports multiple authentication methods for different use cases.

---

## API Authentication

For API requests, use API keys:

```http
GET /api/v1/users HTTP/1.1
Host: api.example.com
X-API-Key: ak_abc123...
```

Or Bearer token:

```http
Authorization: Bearer ak_abc123...
```

See [[API-Keys]] for details on creating and managing API keys.

---

## User Authentication

For the customer portal and admin UI:

### Email/Password

Traditional login with email and password.

### OAuth/Social Login

Sign in with external providers:
- Google
- GitHub
- Generic OIDC

See [[OAuth]] for configuration.

---

## Admin Authentication

Admin access is controlled via the invite system:

1. First user during setup becomes admin
2. Existing admins can invite new admins
3. Invites are single-use and expire

See [[Admin-Invites]] for details.

---

## See Also

- [[API-Keys]] - API key management
- [[OAuth]] - OAuth provider configuration
- [[Security]] - Security overview
- [[Admin-Invites]] - Admin invite system
