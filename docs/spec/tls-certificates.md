# TLS Certificate Management Specification

> **This is the authoritative specification for TLS/ACME behavior.**
>
> - Implementation: `adapters/tls/`
> - Tests: `adapters/tls/acme_integration_test.go`

---

## Overview

APIGate provides automatic TLS certificate management using direct ACME (Let's Encrypt) client or manual certificate provisioning.

---

## ACME Configuration

### Required Configuration

| Setting | Environment Variable | Description | Required |
|---------|---------------------|-------------|----------|
| `tls.enabled` | `APIGATE_TLS_ENABLED` | Enable TLS | Yes |
| `tls.mode` | `APIGATE_TLS_MODE` | Must be `acme` | Yes |
| `tls.domain` | `APIGATE_TLS_DOMAIN` | Domain for certificate | Yes |
| `tls.acme_email` | `APIGATE_TLS_EMAIL` | Contact email for ACME | Yes |

### Optional Configuration

| Setting | Environment Variable | Description | Default |
|---------|---------------------|-------------|---------|
| `tls.acme_staging` | `APIGATE_TLS_ACME_STAGING` | Use Let's Encrypt staging | `false` |
| `tls.http_redirect` | `APIGATE_TLS_HTTP_REDIRECT` | Redirect HTTP to HTTPS | `true` |
| `tls.min_version` | `APIGATE_TLS_MIN_VERSION` | Minimum TLS version | `1.2` |

---

## ACME Provider Architecture

### Direct ACME Client

The implementation uses `golang.org/x/crypto/acme.Client` directly (not autocert.Manager) for full control over the ACME flow:

```go
provider.client = &acme.Client{
    DirectoryURL: directoryURL,  // Always set explicitly
    Key:          accountKey,
    HTTPClient:   httpClientWithTimeout,
}
```

### Why Direct ACME (not autocert)?

| Aspect | autocert.Manager | Direct acme.Client |
|--------|-----------------|-------------------|
| Error visibility | Errors swallowed silently | Full logging at each step |
| Cache timing | After internal callback (unreliable) | Immediately after issuance |
| Rate limit handling | Hangs for minutes | Fast-fail with retry-after tracking |
| Challenge types | HTTP-01 only | TLS-ALPN-01 (primary) + HTTP-01 (fallback) |
| Step timeouts | Overall only | Per-step (30s each) |
| Observability | None | Full flow tracking with [ACME:*] log prefixes |

### Directory URLs

| Staging | DirectoryURL |
|---------|--------------|
| `true` | `https://acme-staging-v02.api.letsencrypt.org/directory` |
| `false` | `https://acme-v02.api.letsencrypt.org/directory` |

---

## ACME Flow (5 Steps)

When a certificate is requested, the direct ACME flow executes:

```
Step 1: [ACME:ORDER]     → Create order with ACME server
Step 2: [ACME:AUTHZ]     → Get authorization for domain
Step 3: [ACME:CHALLENGE] → Solve TLS-ALPN-01 or HTTP-01 challenge
Step 4: [ACME:FINALIZE]  → Finalize order and receive certificate
Step 5: [ACME:CACHE]     → Store certificate IMMEDIATELY in database
```

Each step:
- Has its own 30-second timeout
- Logs start, success/failure, and duration
- Classifies errors (retryable, rate-limited, invalid, fatal)

### Challenge Selection Priority

1. **TLS-ALPN-01** (preferred) - Does not require port 80
2. **HTTP-01** (fallback) - Requires port 80 accessible

---

## Certificate Cache

### Storage Architecture

| Data Type | Storage Location | Key Format |
|-----------|-----------------|------------|
| TLS certificates | `certificates` table | Domain name |
| In-memory cache | `certCache` map | Domain name |

### Caching Layers

1. **Memory cache** - Fast lookup for repeated requests
2. **Database** - Persistent storage, survives restarts
3. **ACME issuance** - Triggered when no valid certificate exists

### Certificate Lookup Flow

```
1. Check TLS-ALPN-01 challenge cert (for ACME validation)
2. Check host policy (domain allowed?)
3. Check rate limit (fast-fail if rate limited)
4. Check memory cache
5. Check database
6. Obtain via ACME flow
```

---

## Rate Limit Fast-Fail

When a domain is rate limited:

1. Rate limit is recorded with retry-after time
2. Subsequent requests fail immediately with descriptive error
3. No ACME requests made until retry-after expires
4. Prevents hammering ACME servers during outages

```go
// Check in logs
[ACME:GETCERT] Domain is rate limited, fast-fail domain=example.com retry_after=2025-01-26T10:00:00Z
```

---

## Certificate Lifecycle

### Automatic Issuance

When ACME is enabled:

1. Server starts with TLS enabled
2. First TLS handshake triggers certificate request
3. TLS-ALPN-01 or HTTP-01 challenge solved
4. Certificate obtained and **immediately** stored in database
5. Certificate served for subsequent requests

### Automatic Renewal

| Timing | Action |
|--------|--------|
| 30 days before expiry | Attempt renewal |
| On renewal failure | Retry on next startup |
| On expiry | Continue with expired cert (warning logged) |

### Certificate Status Values

| Status | Description |
|--------|-------------|
| `active` | Valid, in use |
| `expired` | Past expiration date |
| `revoked` | Manually revoked |

---

## Error Classification

The provider classifies errors for appropriate handling:

| Error Type | Status Codes | Action |
|------------|-------------|--------|
| `ErrorRetryable` | 500, 502, 503, network errors | Retry with backoff |
| `ErrorRateLimited` | 429 | Fast-fail until retry-after |
| `ErrorInvalid` | 400, 403, 404 | Don't retry, log error |
| `ErrorFatal` | Other | Stop, requires investigation |

---

## Switching Between Staging and Production

### Process

1. Change configuration: `tls.acme_staging = false`
2. Restart server
3. System detects issuer mismatch (staging certs have "Fake LE" issuer)
4. New production certificates obtained automatically

### What Happens

| Component | Behavior |
|-----------|----------|
| Staging certificates | Ignored (issuer mismatch) |
| Production certificates | Obtained fresh |

### No Manual Cleanup Required

The system automatically handles the transition. Staging certificates remain in the database but are not used when in production mode.

---

## Logging

All ACME operations use structured logging with prefixes:

| Prefix | Description |
|--------|-------------|
| `[ACME:INIT]` | Provider initialization |
| `[ACME:GETCERT]` | Certificate request handling |
| `[ACME:ORDER]` | ACME order creation |
| `[ACME:AUTHZ]` | Authorization processing |
| `[ACME:CHALLENGE]` | Challenge preparation and validation |
| `[ACME:FINALIZE]` | Order finalization |
| `[ACME:CACHE]` | Certificate caching |
| `[ACME:RATELIMIT]` | Rate limit tracking |
| `[ACME:HTTP:REQ/RESP]` | HTTP requests to ACME server |

---

## Error Handling

### ACME Challenge Failed

**Cause**: DNS not pointing to server, port 80/443 blocked, or path blocked

**Requirements for TLS-ALPN-01**:
1. DNS A/AAAA record points to server
2. Port 443 accessible
3. ALPN negotiation supported

**Requirements for HTTP-01**:
1. DNS A/AAAA record points to server
2. Port 80 accessible
3. `/.well-known/acme-challenge/` path not blocked

### TLS Handshake Timeout

**Cause**: ACME provider not properly initialized or network issues

**Fix History**: Issue #48 - Replaced autocert.Manager with direct acme.Client for full control and visibility.

### Rate Limit Exceeded

**Cause**: Let's Encrypt rate limits (50 certs/week per domain)

**Prevention**:
1. Use staging for testing: `tls.acme_staging = true`
2. Rate limit fast-fail prevents hammering
3. Check logs for `[ACME:RATELIMIT]` entries

---

## Admin API Endpoints

### Settings API

Manage TLS/ACME settings via the Settings module API:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mod/api/settings/` | List all settings |
| GET | `/mod/api/settings/{key}` | Get setting by key |
| GET | `/mod/api/settings/prefix/tls.` | List TLS settings |
| PUT | `/mod/api/settings/{key}` | Update setting |

**Example: Get TLS staging mode**
```bash
curl http://localhost:8080/mod/api/settings/tls.acme_staging
```

**Example: Disable staging mode**
```bash
curl -X PUT http://localhost:8080/mod/api/settings/tls.acme_staging \
  -H "Content-Type: application/json" \
  -d '{"value": "false"}'
```

### Certificates API

Manage certificates via the Certificates module API:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mod/api/certificates/` | List all certificates |
| GET | `/mod/api/certificates/{id}` | Get certificate by ID |
| GET | `/mod/api/certificates/domain/{domain}` | Get certificate for domain |
| GET | `/mod/api/certificates/expiring?days=30` | List expiring certificates |
| GET | `/mod/api/certificates/expired` | List expired certificates |
| DELETE | `/mod/api/certificates/{id}` | Delete certificate |
| POST | `/mod/api/certificates/{id}/revoke` | Revoke certificate |

**Example: List certificates**
```bash
curl http://localhost:8080/mod/api/certificates/
```

**Example: Get certificate by domain**
```bash
curl http://localhost:8080/mod/api/certificates/domain/api.example.com
```

**Example: List certificates expiring in 30 days**
```bash
curl "http://localhost:8080/mod/api/certificates/expiring?days=30"
```

---

## Implementation Files

| File | Purpose |
|------|---------|
| `adapters/tls/acme.go` | Direct ACME provider implementation |
| `adapters/tls/certcache.go` | Certificate cache utilities |
| `adapters/tls/acme_test.go` | Unit tests |
| `adapters/tls/acme_integration_test.go` | Integration tests |
| `ports/tls.go` | TLS provider interface |
| `core/modules/setting.yaml` | Settings module definition |
| `core/modules/certificate.yaml` | Certificates module definition |
| `core/channel/http/http.go` | HTTP channel (generates endpoints from YAML) |

---

## Verification

```bash
# 1. Configure for staging first
sqlite3 apigate.db "UPDATE settings SET value='true' WHERE key='tls.acme_staging'"

# 2. Watch logs for ACME flow
./apigate serve 2>&1 | grep -E "ACME"

# 3. Make TLS request
curl -v --insecure https://yourdomain.com/

# Expected logs:
# [ACME:ORDER] Creating order domain=yourdomain.com
# [ACME:AUTHZ] Processing authorization domain=yourdomain.com
# [ACME:CHALLENGE] Preparing TLS-ALPN-01 challenge domain=yourdomain.com
# [ACME:FINALIZE] Finalizing order domain=yourdomain.com
# [ACME:CACHE:SUCCESS] Certificate cached domain=yourdomain.com cert_id=xxx

# 4. Verify in database
sqlite3 apigate.db "SELECT domain, status, expires_at FROM certificates"
```

---

## See Also

- [Resource Types](resource-types.md) - Settings and Certificates resource schemas
- [Certificates (Wiki)](wiki/Certificates.md) - User documentation
- [Configuration](wiki/Configuration.md) - All TLS settings
- [Troubleshooting](wiki/Troubleshooting.md) - Common issues
