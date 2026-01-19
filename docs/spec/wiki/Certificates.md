# TLS Certificates

APIGate supports automatic TLS certificate management using ACME (Let's Encrypt) or manual certificate provisioning.

---

## Overview

TLS certificates enable HTTPS for your API gateway:

```
┌────────────────────────────────────────────────────────────────┐
│                    Certificate Management                       │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐     ┌──────────────────┐     ┌─────────────┐  │
│  │   Domain    │────▶│   Certificate    │◀────│    ACME     │  │
│  │             │     │                  │     │   Provider  │  │
│  │ api.example │     │ • cert_pem      │     │             │  │
│  │    .com     │     │ • key_pem       │     │ Let's       │  │
│  │             │     │ • expires_at    │     │ Encrypt     │  │
│  └─────────────┘     └──────────────────┘     └─────────────┘  │
│                              │                                  │
│                              ▼                                  │
│                      ┌──────────────┐                          │
│                      │    HTTPS     │                          │
│                      │   Listener   │                          │
│                      └──────────────┘                          │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Certificate Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `domain` | string | Domain name (required, unique) |
| `cert_pem` | string | PEM-encoded certificate (encrypted) |
| `chain_pem` | string | PEM-encoded certificate chain |
| `key_pem` | string | PEM-encoded private key (encrypted) |
| `issued_at` | timestamp | Certificate issue date |
| `expires_at` | timestamp | Certificate expiration date |
| `issuer` | string | Certificate issuer (e.g., Let's Encrypt) |
| `serial_number` | string | Certificate serial number |
| `acme_account_url` | string | ACME account URL (if using ACME) |
| `status` | enum | active, expired, revoked |
| `revoked_at` | timestamp | Revocation time |
| `revoke_reason` | string | Reason for revocation |

---

## TLS Configuration

TLS is configured through settings, not environment variables.

### Enable ACME (Let's Encrypt)

```bash
# Enable TLS with ACME
apigate settings set tls.enabled true
apigate settings set tls.mode acme
apigate settings set tls.domain "api.example.com"
apigate settings set tls.acme_email "admin@example.com"

# For testing, use staging server
apigate settings set tls.acme_staging true
```

### Automatic Certificate Issuance

When ACME mode is enabled, APIGate automatically:

1. Issues certificates for configured domains
2. Handles HTTP-01 challenges at `/.well-known/acme-challenge/`
3. Renews certificates 30 days before expiration
4. Stores certificates in the database

**Note**: ACME certificates are obtained automatically when TLS is enabled. There is no separate CLI command to "obtain" a certificate - the system handles this on startup.

---

## Manual Certificates

### Configure Manual TLS

```bash
# Enable TLS with manual certificates
apigate settings set tls.enabled true
apigate settings set tls.mode manual
apigate settings set tls.cert_path "/path/to/cert.pem"
apigate settings set tls.key_path "/path/to/key.pem"
```

### Upload Certificate to Database

For database-backed certificate storage:

#### Via Admin UI

1. Go to **Certificates** in the sidebar
2. Click **Add Certificate**
3. Enter the domain and upload certificate files
4. Click **Save**

#### Via CLI

```bash
apigate certificates create \
  --domain "api.example.com" \
  --cert-pem /path/to/cert.pem \
  --key-pem /path/to/key.pem \
  --chain-pem /path/to/chain.pem \
  --expires-at "2026-01-19T00:00:00Z"
```

#### Via API

```bash
curl -X POST http://localhost:8080/api/certificates \
  -H "Content-Type: application/vnd.api+json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "data": {
      "type": "certificates",
      "attributes": {
        "domain": "api.example.com",
        "cert_pem": "-----BEGIN CERTIFICATE-----...",
        "key_pem": "-----BEGIN PRIVATE KEY-----...",
        "chain_pem": "-----BEGIN CERTIFICATE-----...",
        "expires_at": "2026-01-19T00:00:00Z"
      }
    }
  }'
```

---

## Certificate Management

### List Certificates

```bash
# CLI
apigate certificates list

# API
curl http://localhost:8080/api/certificates \
  -H "Cookie: session=YOUR_SESSION"
```

### Get Certificate by Domain

```bash
# CLI
apigate certificates get-domain "api.example.com"

# API
curl http://localhost:8080/api/certificates/domain/api.example.com \
  -H "Cookie: session=YOUR_SESSION"
```

### Check Expiring Certificates

```bash
# Certificates expiring within 30 days
apigate certificates expiring --days 30

# API
curl http://localhost:8080/api/certificates/expiring?days=30 \
  -H "Cookie: session=YOUR_SESSION"
```

### List Expired Certificates

```bash
# CLI
apigate certificates expired

# API
curl http://localhost:8080/api/certificates/expired \
  -H "Cookie: session=YOUR_SESSION"
```

### Revoke Certificate

```bash
# CLI
apigate certificates revoke <id> --reason "Key compromised"

# API
curl -X POST http://localhost:8080/api/certificates/<id>/revoke \
  -H "Content-Type: application/vnd.api+json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{"reason": "Key compromised"}'
```

### Delete Certificate

```bash
# CLI
apigate certificates delete <id>

# API
curl -X DELETE http://localhost:8080/api/certificates/<id> \
  -H "Cookie: session=YOUR_SESSION"
```

---

## Automatic Renewal

When using ACME, certificates are automatically renewed:

1. **30 days before expiration**: Renewal attempt
2. **If renewal fails**: Retry on next startup
3. **On expiration**: Service continues with expired cert (warning logged)

**Note**: ACME renewal is automatic. There is no CLI command to manually trigger renewal - the system handles this automatically.

---

## TLS Settings Reference

| Setting | Description | Default |
|---------|-------------|---------|
| `tls.enabled` | Enable TLS/HTTPS | `false` |
| `tls.mode` | Certificate mode: `acme`, `manual`, or `none` | `none` |
| `tls.domain` | Domain for ACME certificates | - |
| `tls.acme_email` | Contact email for ACME | - |
| `tls.cert_path` | Certificate file path (manual mode) | - |
| `tls.key_path` | Private key file path (manual mode) | - |
| `tls.http_redirect` | Redirect HTTP to HTTPS | `true` |
| `tls.min_version` | Minimum TLS version: `1.2` or `1.3` | `1.2` |
| `tls.acme_staging` | Use Let's Encrypt staging server | `false` |

### SNI Support

APIGate supports Server Name Indication (SNI) for multiple domains when using database-stored certificates.

---

## Certificate Events

Webhooks are triggered for certificate events:

| Event | Description |
|-------|-------------|
| `certificate.created` | New certificate added |
| `certificate.renewed` | Certificate successfully renewed |
| `certificate.revoked` | Certificate revoked |

### Example Webhook Payload

```json
{
  "event": "certificate.renewed",
  "data": {
    "domain": "api.example.com",
    "issuer": "Let's Encrypt",
    "expires_at": "2026-04-19T00:00:00Z"
  }
}
```

---

## Security Best Practices

### 1. Use ACME for Production

Let's Encrypt provides free, automated certificates:

```bash
apigate settings set tls.enabled true
apigate settings set tls.mode acme
apigate settings set tls.domain "api.example.com"
apigate settings set tls.acme_email "admin@example.com"
```

### 2. Monitor Expiration

Set up alerts for expiring certificates via the Admin UI or by checking periodically:

```bash
apigate certificates expiring --days 14
```

### 3. Use Staging for Testing

Test certificate issuance with Let's Encrypt staging to avoid rate limits:

```bash
apigate settings set tls.acme_staging true
```

### 4. Revoke Compromised Certificates

If a private key is compromised:

```bash
apigate certificates revoke <id> --reason "Key compromised"
```

Then obtain a new certificate by restarting with ACME enabled, or upload a new manual certificate.

---

## Troubleshooting

### Challenge Failed

**Error**: `ACME challenge failed for domain`

**Causes**:
- DNS not pointing to APIGate server
- Port 80 blocked (HTTP-01 challenge)
- Domain validation failed

**Solution**:
1. Verify DNS: `dig api.example.com`
2. Check port 80 is accessible
3. Ensure `/.well-known/acme-challenge/` is not blocked

### Certificate Not Found

**Error**: `no certificate found for domain`

**Solution**:
```bash
# Check if certificate exists
apigate certificates get-domain "api.example.com"

# For ACME mode, ensure domain is configured
apigate settings get tls.domain
```

### Rate Limit Hit

**Error**: `rate limit exceeded`

**Cause**: Let's Encrypt rate limits (50 certs/week per domain)

**Solution**:
- Wait for rate limit reset
- Use staging for testing: `apigate settings set tls.acme_staging true`

---

## CLI Reference

```bash
# List all certificates
apigate certificates list

# Get certificate by ID
apigate certificates get <id>

# Get certificate by domain
apigate certificates get-domain <domain>

# Create/upload certificate (manual)
apigate certificates create \
  --domain <domain> \
  --cert-pem <path> \
  --key-pem <path> \
  --chain-pem <path> \
  --expires-at <datetime>

# List expiring certificates
apigate certificates expiring --days 30

# List expired certificates
apigate certificates expired

# Revoke certificate
apigate certificates revoke <id> --reason "reason"

# Delete certificate
apigate certificates delete <id>
```

---

## See Also

- [[Configuration]] - TLS settings
- [[Webhooks]] - Certificate event notifications
- [[Security]] - Security best practices
