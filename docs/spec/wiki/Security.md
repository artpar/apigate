# Security

APIGate implements multiple security layers to protect your API and data.

---

## Authentication

### API Key Security

API keys use bcrypt hashing:

```
┌─────────────────────────────────────────────────────────────┐
│                    API Key Security                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Key Generation:                                            │
│  ak_abc123...xyz789  →  bcrypt($key)  →  stored hash       │
│       ↑                                                     │
│       │                                                     │
│  prefix (lookup)                                            │
│                                                             │
│  Key Validation:                                            │
│  Request key  →  extract prefix  →  lookup by prefix       │
│                                   →  bcrypt compare hash   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Keys shown only once at creation
- Only bcrypt hash stored in database
- Prefix enables indexed lookup without full hash comparison
- Keys can have expiration dates
- Keys can be revoked instantly

### OAuth Security

OAuth implementation includes:

- **PKCE** - Proof Key for Code Exchange prevents code interception
- **State tokens** - CSRF protection with 10-minute expiry
- **Token encryption** - Access/refresh tokens encrypted at rest
- **Nonce validation** - OIDC ID token verification

```bash
# Enable PKCE (recommended)
apigate settings set oauth.use_pkce true
```

### Password Security

User passwords are:
- Hashed with bcrypt (cost factor 10)
- Never stored in plaintext
- Never logged or exposed in API

---

## TLS/HTTPS

### Automatic Certificates

ACME (Let's Encrypt) integration:

```bash
TLS_ACME_ENABLED=true
TLS_ACME_EMAIL=admin@example.com
```

**Features:**
- Automatic certificate issuance
- Automatic renewal (30 days before expiry)
- HTTP-01 challenge support
- SNI for multiple domains

### Manual Certificates

Upload certificates via the admin UI:

1. Go to **Certificates** in the sidebar
2. Click **Add Certificate**
3. Upload your certificate and private key files
4. Click **Save**

Private keys are encrypted at rest.

---

## Rate Limiting

Protection against abuse:

```
┌─────────────────────────────────────────────────────────────┐
│                    Token Bucket Algorithm                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Bucket fills at: rate_limit_per_minute / 60 tokens/sec    │
│  Burst capacity: configurable (default: rate_limit / 10)   │
│                                                             │
│  Request arrives:                                           │
│  - If tokens available: consume 1, allow request           │
│  - If no tokens: reject with 429, return Retry-After       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Headers returned:**
- `X-RateLimit-Remaining` - Tokens remaining
- `X-RateLimit-Reset` - When bucket refills
- `Retry-After` - Seconds until retry (on 429)

---

## Quotas

Monthly usage limits:

- Configurable per plan
- Grace period support (default 5%)
- Multiple enforcement modes:
  - **Hard** - Block at limit
  - **Soft** - Block at limit + grace
  - **Warn** - Allow but send notifications

---

## Data Protection

### Encryption at Rest

Sensitive fields encrypted in database:
- `password_hash` - User passwords
- `api_key.hash` - API key hashes
- `certificate.key_pem` - TLS private keys
- `oauth_identity.access_token` - OAuth tokens
- `oauth_identity.refresh_token` - OAuth refresh tokens
- Provider credentials (Stripe keys, etc.)

### Secrets in Configuration

Never commit secrets:

```bash
# Good - environment variables
STRIPE_API_KEY=${STRIPE_API_KEY}

# Bad - hardcoded
STRIPE_API_KEY=sk_live_xxx
```

Use secret management:
- Environment variables
- Docker secrets
- Vault integration
- Cloud secret managers

---

## Input Validation

### Request Body Limits

```go
// 10MB default limit
body, err = io.ReadAll(io.LimitReader(r.Body, 10<<20))
```

### Schema Validation

Module schemas enforce:
- Required fields
- Type validation
- Enum constraints
- Email format validation
- Reference integrity

---

## Admin Access

### Admin Invite System

Admin access is controlled via invites:

1. First user is auto-admin during setup
2. Existing admins invite new admins
3. Invites expire after configurable time
4. Invites are single-use

Invitations are managed via the admin UI:

1. Go to **Admin Invites** in the sidebar
2. Click **Create Invite**
3. Enter the email address
4. Click **Send Invitation**

### Admin Endpoints

Admin endpoints require authentication:
- `/admin/*` - Admin UI and API
- Requires valid admin session
- Actions are logged

---

## Webhook Security

### Signature Verification

Outgoing webhooks include signatures:

```http
X-Webhook-Signature: sha256=abc123...
```

Compute signature:
```
HMAC-SHA256(secret, timestamp + "." + payload)
```

### Replay Protection

Include timestamp header:
```http
X-Webhook-Timestamp: 1642000000
```

Reject requests older than 5 minutes.

---

## Network Security

### Recommended Setup

```
┌─────────────────────────────────────────────────────────────┐
│                    Production Architecture                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Internet                                                  │
│      │                                                      │
│      ▼                                                      │
│   ┌─────────────┐                                           │
│   │   CDN/WAF   │  DDoS protection, WAF rules              │
│   └──────┬──────┘                                           │
│          │                                                  │
│          ▼                                                  │
│   ┌─────────────┐                                           │
│   │ Load Balancer│  SSL termination (optional)             │
│   └──────┬──────┘                                           │
│          │                                                  │
│          ▼                                                  │
│   ┌─────────────┐                                           │
│   │  APIGate    │  API gateway                             │
│   └──────┬──────┘                                           │
│          │                                                  │
│          ▼                                                  │
│   ┌─────────────┐                                           │
│   │  Upstream   │  Private network only                    │
│   │  Services   │                                           │
│   └─────────────┘                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Firewall Rules

- Allow 80/443 inbound (public)
- Allow admin port only from trusted IPs
- Restrict upstream access to APIGate only
- Block direct database access

---

## Security Headers

APIGate sets security headers on responses:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
```

For admin UI:
```http
Content-Security-Policy: default-src 'self'
```

---

## Audit Logging

All admin actions are logged:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "action": "user.create",
  "actor": "admin@example.com",
  "target": "usr_abc123",
  "ip": "192.168.1.1"
}
```

View audit logs via the admin UI in **Analytics > Audit Log**.

---

## Security Checklist

### Production Deployment

- [ ] Enable HTTPS (ACME or manual certs)
- [ ] Use strong database password
- [ ] Set secure admin password
- [ ] Configure rate limits appropriately
- [ ] Enable webhook signatures
- [ ] Restrict admin port access
- [ ] Set up monitoring/alerting
- [ ] Regular security updates
- [ ] Backup encryption keys

### API Key Best Practices

- [ ] Rotate keys regularly
- [ ] Monitor for unusual usage
- [ ] Revoke compromised keys immediately
- [ ] Use separate keys for different environments

---

## See Also

- [[OAuth]] - OAuth configuration
- [[Certificates]] - TLS setup
- [[Rate-Limiting]] - Rate limit configuration
- [[API-Keys]] - API key management
