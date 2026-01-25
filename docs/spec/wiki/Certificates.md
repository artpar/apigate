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

## Technical Specification

For the authoritative technical specification including implementation details, test requirements, and initialization behavior, see [TLS Certificates Spec](../tls-certificates.md).

---

## TLS Configuration

TLS can be configured via environment variables, CLI settings, or the Admin UI.

### Enable ACME (Let's Encrypt)

**Via Environment Variables (Recommended for Docker/Systemd):**
```bash
# Set these in your .env file or systemd unit
APIGATE_TLS_ENABLED=true
APIGATE_TLS_MODE=acme
APIGATE_TLS_DOMAIN=api.example.com
APIGATE_TLS_EMAIL=admin@example.com
APIGATE_SERVER_PORT=443

# For testing, use staging server
APIGATE_TLS_ACME_STAGING=true
```

**Via CLI Settings:**
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

**Via Environment Variables:**
```bash
APIGATE_TLS_ENABLED=true
APIGATE_TLS_MODE=manual
APIGATE_TLS_CERT=/etc/letsencrypt/live/example.com/fullchain.pem
APIGATE_TLS_KEY=/etc/letsencrypt/live/example.com/privkey.pem
APIGATE_SERVER_PORT=443
```

**Via CLI Settings:**
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

| Setting | Environment Variable | Description | Default |
|---------|---------------------|-------------|---------|
| `tls.enabled` | `APIGATE_TLS_ENABLED` | Enable TLS/HTTPS | `false` |
| `tls.mode` | `APIGATE_TLS_MODE` | Certificate mode: `acme`, `manual`, or `none` | `none` |
| `tls.domain` | `APIGATE_TLS_DOMAIN` | Domain for ACME certificates | - |
| `tls.acme_email` | `APIGATE_TLS_EMAIL` | Contact email for ACME | - |
| `tls.cert_path` | `APIGATE_TLS_CERT` | Certificate file path (manual mode) | - |
| `tls.key_path` | `APIGATE_TLS_KEY` | Private key file path (manual mode) | - |
| `tls.http_redirect` | `APIGATE_TLS_HTTP_REDIRECT` | Redirect HTTP to HTTPS | `true` |
| `tls.min_version` | `APIGATE_TLS_MIN_VERSION` | Minimum TLS version: `1.2` or `1.3` | `1.2` |
| `tls.acme_staging` | `APIGATE_TLS_ACME_STAGING` | Use Let's Encrypt staging server | `false` |

### SNI Support

APIGate supports Server Name Indication (SNI) for multiple domains when using database-stored certificates.

### ACME Certificate Cache

The ACME certificate cache stores two types of data:

| Data Type | Storage | Description |
|-----------|---------|-------------|
| **ACME account keys** | `acme_cache` table | Private key for ACME account registration |
| **TLS certificates** | `certificates` table | Domain certificates with metadata |

**Cache key format**: The autocert library uses specific key formats:
- `domain` - Request for ECDSA certificate (preferred)
- `domain+rsa` - Request for RSA certificate
- `domain+ecdsa` - Explicit ECDSA request
- `+acme_account+<url>` - ACME account key

**Key type matching**: If a cached certificate's key type doesn't match the request (e.g., RSA cert for ECDSA request), the cache returns a miss and autocert tries the appropriate key format.

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

### Verifying Binary Version

Before troubleshooting, verify you're running the correct binary version. APIGate logs version info at startup:

```
apigate v1.2.3 (commit: 1f021b9, built: 2026-01-19T17:20:21Z)
```

You can also check the version explicitly:

```bash
apigate version
```

**If version info shows `dev` or `none`**: You may be running a development build without proper ldflags. Use official releases or build with:

```bash
make build  # Sets version, commit, and build date
```

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

### ACME Account Key Errors

**Error**: `expected at least 2 PEM blocks, got 1`

**Cause**: This error occurred in versions before v0.x.x when the certificate cache incorrectly tried to parse ACME account keys as certificates.

**Background**: The ACME protocol uses two types of data stored in the certificate cache:
- **ACME account keys**: Single PEM block (just the private key)
- **TLS certificates**: Multiple PEM blocks (certificate + private key + optional chain)

**Solution**:
1. **Verify binary version** - Check startup logs show recent commit hash
2. **Upgrade to latest version** - This was fixed in commit `1f021b9`
3. **Clear stale data** - If upgrading from an old version:
   ```bash
   # The certificates table can be safely cleared if using ACME
   # (certificates will be re-obtained automatically)
   sqlite3 apigate.db "DELETE FROM certificates;"
   ```

**Debug logging**: Enable debug logging to see ACME certificate operations:
```bash
APIGATE_LOG_LEVEL=debug apigate serve
```

You'll see messages like:
- `ACME account key stored in database` - Account key persisted
- `ACME account key retrieved from database` - Account key loaded after restart
- `certificate retrieved from database` - Certificate loaded from DB
- `new certificate obtained and stored` - Fresh certificate from Let's Encrypt

**Note**: As of v0.x.x, ACME account keys are persisted to the database (`acme_cache` table), ensuring they survive restarts and preventing Let's Encrypt rate limiting.

### Rate Limit Hit

**Error**: `rate limit exceeded`

**Cause**: Let's Encrypt rate limits (50 certs/week per domain)

**Solution**:
- Wait for rate limit reset
- Use staging for testing: `apigate settings set tls.acme_staging true`

### Switching from Staging to Production

When switching from Let's Encrypt staging to production:

1. **Certificates are re-obtained automatically**: Staging certificates (issued by "Fake LE Intermediate X1") won't be used in production mode. The system detects the key type mismatch and obtains new production certificates.

2. **ACME account keys persist**: The ACME account key is stored in the `acme_cache` table and reused, preventing unnecessary account creation.

3. **No manual cleanup needed**: Simply change the setting and restart:
   ```bash
   apigate settings set tls.acme_staging false
   # Restart the server
   ```

**Note**: If you experience issues after switching, you can clear the certificate cache:
```bash
sqlite3 apigate.db "DELETE FROM certificates WHERE issuer LIKE '%Fake%';"
```

### TLS Handshake Timeout

**Error**: TLS handshake hangs or times out

**Causes**:
- ACME challenge cannot complete (firewall blocking port 80)
- DNS not properly configured
- Rate limits exceeded
- ACME provider not properly initialized (Issue #48)

**Solution**:
1. Verify port 80 is accessible for HTTP-01 challenges
2. Check DNS resolution: `dig api.example.com`
3. Check logs for ACME errors: `APIGATE_LOG_LEVEL=debug apigate serve`
4. If rate limited, wait or use staging mode for testing
5. Verify you're running a recent version (Issue #48 was fixed in v0.x.x)

**Technical Detail (Issue #48 Fix):**

The ACME provider's `autocert.Manager.Client` must be explicitly set for both staging and production modes. Earlier versions only set this for staging, causing production mode to fail during lazy initialization.

The fix ensures `Manager.Client` is always configured with the correct ACME directory URL:
- Production: `https://acme-v02.api.letsencrypt.org/directory`
- Staging: `https://acme-staging-v02.api.letsencrypt.org/directory`

See [TLS Certificates Spec](../tls-certificates.md) for implementation details

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
