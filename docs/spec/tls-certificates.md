# TLS Certificate Management Specification

> **This is the authoritative specification for TLS/ACME behavior.**
>
> - Implementation: `adapters/tls/`
> - Tests: `adapters/tls/acme_integration_test.go`

---

## Overview

APIGate provides automatic TLS certificate management using ACME (Let's Encrypt) or manual certificate provisioning.

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

## ACME Provider Initialization

### Manager Configuration

The `autocert.Manager` is initialized with explicit configuration:

```go
provider.manager = &autocert.Manager{
    Cache:      cache,
    Prompt:     autocert.AcceptTOS,
    Email:      cfg.Email,
    HostPolicy: provider.hostPolicy,
    Client: &acme.Client{
        DirectoryURL: directoryURL,  // MUST be set explicitly
    },
}
```

### Critical Requirement: Manager.Client

**The `Manager.Client` MUST always be set explicitly.**

| Staging | DirectoryURL |
|---------|--------------|
| `true` | `https://acme-staging-v02.api.letsencrypt.org/directory` |
| `false` | `https://acme-v02.api.letsencrypt.org/directory` |

**Why this matters:**
- When `Client` is `nil`, autocert uses lazy initialization
- Lazy initialization can fail silently in production mode
- This causes TLS handshake timeouts (Issue #48)

### Test Coverage Requirement

Every code path through `NewACMEProvider` MUST be tested:

```go
func TestNewACMEProvider(t *testing.T) {
    tests := []struct {
        name       string
        staging    bool
        wantDirURL string
    }{
        {"production mode", false, letsEncryptProduction},
        {"staging mode", true, letsEncryptStaging},
    }
    // Verify Manager.Client is non-nil
    // Verify DirectoryURL matches expected value
}
```

---

## Certificate Cache

### Storage Architecture

| Data Type | Storage Location | Key Format |
|-----------|-----------------|------------|
| ACME account keys | `acme_cache` table | `+acme_account+<directory_url>` |
| TLS certificates | `certificates` table | Domain name |

### Cache Key Formats

The `autocert` library uses specific key formats:

| Key Format | Description |
|------------|-------------|
| `domain` | ECDSA certificate (preferred) |
| `domain+rsa` | RSA certificate request |
| `domain+ecdsa` | Explicit ECDSA request |
| `+acme_account+<url>` | ACME account private key |

### Key Type Matching

If a cached certificate's key type doesn't match the request:
1. Cache returns `ErrCacheMiss`
2. autocert requests certificate with correct key type
3. New certificate is stored with appropriate suffix

---

## Certificate Lifecycle

### Automatic Issuance

When ACME is enabled:

1. Server starts with TLS enabled
2. First TLS handshake triggers certificate request
3. HTTP-01 challenge served at `/.well-known/acme-challenge/`
4. Certificate obtained and stored in database
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

## Switching Between Staging and Production

### Process

1. Change configuration: `tls.acme_staging = false`
2. Restart server
3. System detects issuer mismatch (staging certs have "Fake LE" issuer)
4. New production certificates obtained automatically

### What Happens

| Component | Behavior |
|-----------|----------|
| ACME account key | Persisted in `acme_cache`, reused |
| Staging certificates | Ignored (issuer mismatch) |
| Production certificates | Obtained fresh |

### No Manual Cleanup Required

The system automatically handles the transition. Staging certificates remain in the database but are not used when in production mode.

---

## Error Handling

### ACME Challenge Failed

**Cause**: DNS not pointing to server, port 80 blocked, or path blocked

**Requirements**:
1. DNS A/AAAA record points to server
2. Port 80 accessible (HTTP-01 challenge)
3. `/.well-known/acme-challenge/` path not blocked

### TLS Handshake Timeout

**Cause**: ACME provider not properly initialized

**Fix History**: Issue #48 - Manager.Client was only set for staging mode. Fixed by always setting Client explicitly with correct DirectoryURL.

**Prevention**: Test coverage for all `ACMEConfig.Staging` values.

### Rate Limit Exceeded

**Cause**: Let's Encrypt rate limits (50 certs/week per domain)

**Prevention**:
1. Use staging for testing: `tls.acme_staging = true`
2. ACME account keys persist across restarts (no duplicate registrations)

---

## Implementation Files

| File | Purpose |
|------|---------|
| `adapters/tls/acme.go` | ACME provider implementation |
| `adapters/tls/cache.go` | Certificate cache implementation |
| `adapters/tls/acme_integration_test.go` | Integration tests |
| `ports/tls.go` | TLS provider interface |

---

## See Also

- [Certificates (Wiki)](wiki/Certificates.md) - User documentation
- [Configuration](wiki/Configuration.md) - All TLS settings
- [Troubleshooting](wiki/Troubleshooting.md) - Common issues
