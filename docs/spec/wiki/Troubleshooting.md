# Troubleshooting

Common issues and solutions for APIGate.

---

## Authentication Issues

### Invalid API Key

**Error**: `invalid_api_key` (401)

**Causes**:
- Key doesn't exist
- Key typo (check for extra spaces)
- Wrong key prefix

**Solutions**:
```bash
# Verify keys for a user
apigate keys list

# Check key format starts with prefix
echo $API_KEY | grep "^ak_"
```

### Key Expired

**Error**: `key_expired` (401)

**Solution**:
```bash
# Create new key (old keys cannot be recovered)
apigate keys create --user <user-id> --name "New Key"
```

### Key Revoked

**Error**: `key_revoked` (401)

**Solution**: Create a new key. Revoked keys cannot be un-revoked.

```bash
apigate keys create --user <user-id> --name "Replacement Key"
```

### User Suspended

**Error**: `user_suspended` (403)

**Solution**:
```bash
# Check user status
apigate users get <user-id>

# Reactivate if appropriate
apigate users activate <user-id>
```

---

## Rate Limiting Issues

### Rate Limited

**Error**: `rate_limited` (429)

**Check headers**:
```
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 2024-01-15T10:30:00Z
Retry-After: 30
```

**Solutions**:
1. Wait for `Retry-After` seconds
2. Upgrade to higher plan
3. Implement exponential backoff

```bash
# Check user's plan rate limit
apigate plans get <plan-id>
```

### Quota Exceeded

**Error**: `quota_exceeded` (429)

**Check headers**:
```
X-Quota-Used: 10000
X-Quota-Limit: 10000
X-Quota-Reset: 2024-02-01T00:00:00Z
```

**Solutions**:
1. Wait for monthly reset
2. Upgrade plan
3. Contact for quota increase

```bash
# Check usage for a user
apigate usage --user <user-id>
```

---

## Upstream Issues

### Upstream Error

**Error**: `upstream_error` (502)

**Causes**:
- Upstream service down
- Network connectivity issue
- Timeout

**Solutions**:
```bash
# Check upstream configuration
apigate routes list

# Test upstream directly
curl -v https://upstream.example.com/health

# Check APIGate logs (in terminal output or log file)
```

### Timeout

**Error**: Connection timeout to upstream

**Solutions**:

1. Check upstream response times manually:
```bash
curl -w "@curl-format.txt" -o /dev/null -s https://upstream.example.com/health
```

2. Verify upstream URL in route configuration:
```bash
apigate routes get <route-id>
```

3. Consider network connectivity between APIGate and upstream.

---

## TLS/Certificate Issues

### Certificate Not Found

**Error**: `no certificate found for domain`

**Solutions**:
```bash
# List certificates
apigate certificates list

# Check if domain certificate exists
apigate certificates get-domain api.example.com

# For ACME mode, ensure TLS is properly configured
apigate settings get tls.enabled
apigate settings get tls.domain
```

### ACME Challenge Failed

**Error**: `ACME challenge failed`

**Causes**:
- DNS not pointing to server
- Port 80 blocked
- `.well-known/acme-challenge/` blocked

**Solutions**:
1. Verify DNS: `dig api.example.com`
2. Check port 80 is open
3. Ensure no proxy blocks `/.well-known/`

### Certificate Expired

**Error**: `certificate has expired`

**Solutions**:

For ACME mode, restart APIGate to trigger renewal:
```bash
# Check certificate expiration
apigate certificates expiring --days 7

# Restart to trigger ACME renewal
systemctl restart apigate
```

For manual certificates, upload a new one:
```bash
apigate certificates create \
  --domain api.example.com \
  --cert-pem new-cert.pem \
  --key-pem new-key.pem \
  --expires-at "2027-01-01T00:00:00Z"
```

---

## OAuth Issues

### Redirect URI Mismatch

**Error**: `redirect_uri_mismatch`

**Solution**: Ensure callback URL matches exactly:
```
https://your-domain.com/auth/oauth/{provider}/callback
```

Must match in:
- OAuth provider settings (Google/GitHub console)
- APIGate configuration

### State Mismatch

**Error**: `invalid_state`

**Causes**:
- Flow took longer than 10 minutes
- Cookies disabled
- Multiple tabs/sessions

**Solutions**:
- Complete OAuth flow within 10 minutes
- Enable cookies in browser
- Use single tab for login

### Missing Email

**Problem**: User created without email

**Solution**: Ensure OAuth app requests email scope:
- Google: `openid email profile`
- GitHub: `user:email`

---

## Database Issues

### Database Locked (SQLite)

**Error**: `database is locked`

**Solutions**:

APIGate uses SQLite by default. If you encounter locking:

1. Check for multiple processes accessing the database:
```bash
lsof | grep apigate.db
```

2. Ensure WAL mode is enabled (default):
```bash
sqlite3 apigate.db "PRAGMA journal_mode;"
```

3. Only run one instance of APIGate per database file.

### Migration Failed

**Error**: `migration failed` on startup

**Solutions**:

Migrations run automatically on startup. If they fail:

1. Check the error message in logs for specific issue
2. Ensure database file is writable
3. Check disk space
4. Back up your database and try again

```bash
# Check database integrity
sqlite3 apigate.db "PRAGMA integrity_check;"
```

---

## Performance Issues

### Slow Responses

**Causes**:
- Slow upstream
- Database queries
- Rate limit checks

**Solutions**:

1. Test upstream latency directly:
```bash
curl -w "Total time: %{time_total}s\n" -o /dev/null -s https://upstream.example.com/endpoint
```

2. Enable debug logging:
```bash
APIGATE_LOG_LEVEL=debug ./apigate serve
```

3. Check database size and consider maintenance:
```bash
ls -lh apigate.db
sqlite3 apigate.db "VACUUM;"
```

### High Memory Usage

**Solutions**:
- Reduce connection pool size
- Enable request body limits
- Check for memory leaks in transforms

### High CPU

**Solutions**:
- Check transform expressions for complexity
- Reduce regex-based route matching
- Scale horizontally

---

## Webhook Issues

### Webhook Not Delivered

**Check delivery status** via Admin UI:
1. Go to **Analytics** > **Webhooks**
2. View delivery attempts and errors

**Common causes**:
- Endpoint unreachable
- SSL certificate issues on endpoint
- Signature verification failing on receiver
- Endpoint returning non-2xx status

### Signature Verification Failed

Ensure you're verifying the webhook signature correctly:

```python
import hmac
import hashlib

def verify_webhook(payload, timestamp, signature, secret):
    expected = hmac.new(
        secret.encode(),
        (timestamp + "." + payload).encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

---

## Admin UI Issues

### Can't Access Admin UI

**Causes**:
- Admin port not exposed
- Firewall blocking
- No admin users exist

**Solutions**:
```bash
# Check admin port is running
curl http://localhost:9090/admin/health

# Create admin user if needed
apigate admin create --email admin@example.com
```

### Session Expired

**Solution**: Login again. Sessions expire after configured timeout.

---

## Common CLI Commands

```bash
# Check users
apigate users list
apigate users get <user-id>
apigate users activate <user-id>
apigate users deactivate <user-id>

# Check keys
apigate keys list
apigate keys create --user <user-id> --name "Key Name"
apigate keys revoke <key-id>

# Check plans
apigate plans list
apigate plans get <plan-id>

# Check routes
apigate routes list
apigate routes get <route-id>

# Check settings
apigate settings list
apigate settings get <key>

# Check usage
apigate usage --user <user-id>

# Check certificates
apigate certificates list
apigate certificates expiring --days 30
```

---

## Getting Help

1. Check logs (terminal output or log file)
2. Review this page for common issues
3. Check [GitHub Issues](https://github.com/artpar/apigate/issues)
4. Open new issue with:
   - APIGate version (`apigate version`)
   - Error message
   - Relevant logs
   - Steps to reproduce

---

## See Also

- [[Configuration]] - Configuration reference
- [[Error-Codes]] - Error code reference
- [[FAQ]] - Frequently asked questions
