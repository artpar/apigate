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

## ACME (Let's Encrypt)

### Enable ACME

```bash
# Environment variables
TLS_ACME_ENABLED=true
TLS_ACME_EMAIL=admin@example.com
TLS_ACME_DIRECTORY=https://acme-v02.api.letsencrypt.org/directory

# For testing, use staging
TLS_ACME_DIRECTORY=https://acme-staging-v02.api.letsencrypt.org/directory
```

Or via CLI:

```bash
apigate settings set tls.acme.enabled true
apigate settings set tls.acme.email "admin@example.com"
```

### Automatic Certificate Issuance

When ACME is enabled, APIGate automatically:

1. Issues certificates for configured domains
2. Handles HTTP-01 challenges at `/.well-known/acme-challenge/`
3. Renews certificates 30 days before expiration
4. Stores certificates in the database

### Configure Domains

```bash
# Add domain for automatic certificate
apigate certificates obtain --domain "api.example.com"
```

---

## Manual Certificates

### Upload Certificate

```bash
# CLI
apigate certificates create \
  --domain "api.example.com" \
  --cert-pem /path/to/cert.pem \
  --key-pem /path/to/key.pem \
  --chain-pem /path/to/chain.pem

# API
curl -X POST http://localhost:8080/admin/certificates \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "api.example.com",
    "cert_pem": "-----BEGIN CERTIFICATE-----...",
    "key_pem": "-----BEGIN PRIVATE KEY-----...",
    "chain_pem": "-----BEGIN CERTIFICATE-----...",
    "expires_at": "2026-01-19T00:00:00Z"
  }'
```

### Update Certificate

```bash
apigate certificates renew <id> \
  --cert-pem /path/to/new-cert.pem \
  --key-pem /path/to/new-key.pem
```

---

## Certificate Management

### List Certificates

```bash
# CLI
apigate certificates list

# API
curl http://localhost:8080/admin/certificates
```

### Get Certificate by Domain

```bash
# CLI
apigate certificates get-domain "api.example.com"

# API
curl http://localhost:8080/admin/certificates/domain/api.example.com
```

### Check Expiring Certificates

```bash
# Certificates expiring within 30 days
apigate certificates expiring --days 30

# API
curl http://localhost:8080/admin/certificates/expiring?days=30
```

### List Expired Certificates

```bash
apigate certificates expired

# API
curl http://localhost:8080/admin/certificates/expired
```

### Revoke Certificate

```bash
# CLI
apigate certificates revoke <id> --reason "Key compromised"

# API
curl -X POST http://localhost:8080/admin/certificates/<id>/revoke \
  -H "Content-Type: application/json" \
  -d '{"reason": "Key compromised"}'
```

### Delete Certificate

```bash
# CLI
apigate certificates delete <id>

# API
curl -X DELETE http://localhost:8080/admin/certificates/<id>
```

---

## Automatic Renewal

When using ACME, certificates are automatically renewed:

1. **30 days before expiration**: Renewal attempt
2. **If renewal fails**: Retry daily
3. **7 days before expiration**: Alert notification
4. **On expiration**: Service continues with expired cert (warning logged)

### Manual Renewal

```bash
# Trigger renewal for a domain
apigate certificates renew --domain "api.example.com"
```

---

## TLS Configuration

### HTTPS Listener

```bash
# Enable HTTPS
HTTPS_ENABLED=true
HTTPS_PORT=443

# Certificate source
TLS_CERT_SOURCE=database  # Use database-stored certs
# TLS_CERT_SOURCE=file    # Use file-based certs
```

### File-Based Certificates

For simple deployments without ACME:

```bash
TLS_CERT_SOURCE=file
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

### SNI Support

APIGate supports Server Name Indication (SNI) for multiple domains:

```
api.example.com     → Certificate A
staging.example.com → Certificate B
```

---

## Certificate Events

Webhooks are triggered for certificate events:

| Event | Description |
|-------|-------------|
| `certificate.created` | New certificate added |
| `certificate.renewed` | Certificate successfully renewed |
| `certificate.revoked` | Certificate revoked |
| `certificate.expiring` | Certificate expires within 7 days |

### Example Webhook Payload

```json
{
  "event": "certificate.renewed",
  "data": {
    "domain": "api.example.com",
    "issuer": "Let's Encrypt",
    "expires_at": "2026-04-19T00:00:00Z",
    "previous_expires_at": "2026-01-19T00:00:00Z"
  }
}
```

---

## Security Best Practices

### 1. Use ACME for Production

Let's Encrypt provides free, automated certificates:

```bash
TLS_ACME_ENABLED=true
TLS_ACME_DIRECTORY=https://acme-v02.api.letsencrypt.org/directory
```

### 2. Monitor Expiration

Set up alerts for expiring certificates:

```bash
# List certificates expiring in 14 days
apigate certificates expiring --days 14
```

### 3. Use Staging for Testing

Test certificate issuance with Let's Encrypt staging:

```bash
TLS_ACME_DIRECTORY=https://acme-staging-v02.api.letsencrypt.org/directory
```

### 4. Backup Private Keys

While keys are stored encrypted in the database, maintain backups:

```bash
# Export certificate (for backup)
apigate certificates export <id> --output /backup/cert-backup.json
```

### 5. Revoke Compromised Certificates

If a private key is compromised:

```bash
apigate certificates revoke <id> --reason "Key compromised"
apigate certificates obtain --domain "api.example.com"  # Get new cert
```

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

# Obtain new certificate
apigate certificates obtain --domain "api.example.com"
```

### Expired Certificate

**Error**: `certificate has expired`

**Solution**:
```bash
# Force renewal
apigate certificates renew --domain "api.example.com" --force
```

### Rate Limit Hit

**Error**: `rate limit exceeded`

**Cause**: Let's Encrypt rate limits (50 certs/week per domain)

**Solution**:
- Wait for rate limit reset
- Use staging for testing
- Request rate limit increase (for high volume)

---

## CLI Reference

```bash
# List all certificates
apigate certificates list

# Get certificate by ID
apigate certificates get <id>

# Get certificate by domain
apigate certificates get-domain <domain>

# Create/upload certificate
apigate certificates create --domain <domain> --cert-pem <path> --key-pem <path>

# Obtain certificate via ACME
apigate certificates obtain --domain <domain>

# Renew certificate
apigate certificates renew <id>
apigate certificates renew --domain <domain>

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
- [[Architecture]] - How TLS termination works
