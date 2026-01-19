# Frequently Asked Questions

Common questions about APIGate.

---

## General

### What is APIGate?

APIGate is a self-hosted API gateway that adds authentication, rate limiting, usage tracking, and billing to any API.

### What are the system requirements?

- **Minimum**: 512MB RAM, 1 CPU, 1GB disk
- **Recommended**: 2GB RAM, 2 CPU, 10GB disk
- **OS**: Linux, macOS, Windows (Docker recommended)

### Is APIGate free?

Yes, APIGate is open source and free to self-host.

---

## Installation

### How do I install APIGate?

Several options:

```bash
# Docker (recommended)
docker run -p 8080:8080 -p 9090:9090 artpar/apigate

# Binary
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-linux-amd64 -o apigate
chmod +x apigate
./apigate serve

# From source
git clone https://github.com/artpar/apigate
cd apigate
make build
./apigate serve
```

See [[Installation]] for details.

### What database does APIGate use?

SQLite. This is the only supported database. Data is stored in a single file (default: `apigate.db`).

### Can I run multiple instances?

APIGate uses SQLite which supports single-instance deployments. For horizontal scaling, you would need to deploy multiple independent instances or use a load balancer with sticky sessions.

---

## API Keys

### How are API keys secured?

Keys are hashed with bcrypt before storage. Only the hash is stored—the original key cannot be recovered.

### Can I see an API key after creation?

No. Keys are shown only once at creation. If lost, create a new key.

### How do I rotate API keys?

1. Create new key
2. Update your application
3. Revoke old key

```bash
apigate keys create --user <id> --name "New Key"
apigate keys revoke <old-key-id>
```

### Can API keys have different permissions?

API keys inherit their permissions from the user's plan. All keys for a user have the same rate limit and quota based on their plan.

---

## Rate Limiting

### How does rate limiting work?

Token bucket algorithm. Each API key gets tokens based on their plan's `rate_limit_per_minute`. Tokens refill continuously.

### Can I disable rate limiting?

Set a very high limit when creating a plan:

```bash
apigate plans create \
  --id "unlimited" \
  --name "Unlimited" \
  --rate-limit 1000000 \
  --requests -1
```

### Is rate limiting per key or per user?

Per API key. Each key has its own independent token bucket.

---

## Plans & Billing

### How do I create a free plan?

```bash
apigate plans create \
  --id "free" \
  --name "Free" \
  --rate-limit 60 \
  --requests 1000 \
  --price 0 \
  --default
```

### How do I integrate with Stripe?

1. Get Stripe API keys
2. Configure in APIGate:
```bash
apigate settings set payment.provider stripe
apigate settings set payment.stripe.api_key "sk_xxx" --encrypted
```
3. Link plans to Stripe prices

See [[Payment-Stripe]] for details.

### Can I have usage-based billing?

Yes. Use `metering_expr` on routes to calculate costs:

```yaml
metering_expr: "respBody.tokens_used"
```

---

## Routes & Upstreams

### Can I route to multiple backends?

Yes. Create multiple upstreams and routes:

```bash
# Create upstreams
apigate upstreams create --name "Users API" --url "http://users:8080"
apigate upstreams create --name "Orders API" --url "http://orders:8080"

# Create routes
apigate routes create --name "users-route" --path "/users/*" --upstream <users-upstream-id>
apigate routes create --name "orders-route" --path "/orders/*" --upstream <orders-upstream-id>
```

### How do I rewrite paths?

Path rewriting is configured via the Admin UI or API. The `path_rewrite` field takes an expression.

### Can I transform requests/responses?

Yes. See [[Transformations]].

---

## Security

### How do I enable HTTPS?

Option 1: ACME (automatic):
```bash
apigate settings set tls.enabled true
apigate settings set tls.mode acme
apigate settings set tls.domain "api.example.com"
apigate settings set tls.acme_email "admin@example.com"
```

Option 2: Manual certificates:
```bash
apigate settings set tls.enabled true
apigate settings set tls.mode manual
apigate settings set tls.cert_path "/path/to/cert.pem"
apigate settings set tls.key_path "/path/to/key.pem"
```

### How do I secure the admin UI?

1. Restrict admin port to internal network
2. Use strong admin passwords
3. Enable HTTPS
4. Set up firewall rules

### Are passwords encrypted?

Yes. Passwords are bcrypt hashed. Sensitive data (API keys, certificates, OAuth tokens) is encrypted at rest.

---

## OAuth

### Which OAuth providers are supported?

- Google
- GitHub
- Any OpenID Connect provider

### How do I enable "Sign in with Google"?

1. Create OAuth credentials in Google Cloud Console
2. Configure in APIGate:
```bash
apigate settings set oauth.google.enabled true
apigate settings set oauth.google.client_id "xxx"
apigate settings set oauth.google.client_secret "xxx" --encrypted
```

See [[OAuth]] for details.

---

## Groups/Teams

### Can users share API keys?

Yes, via Groups. Create a group and add members:

```bash
# Create a group
apigate groups create --name "Acme Corp" --owner <user-id>

# Add members
apigate group-members create --group <group-id> --user <user-id> --role member
```

Group API keys are shared by all members.

### What roles are available?

- **Owner** - Full control, billing
- **Admin** - Manage members and keys
- **Member** - Use API keys, view usage

---

## Troubleshooting

### Why am I getting 429 errors?

Rate limit or quota exceeded. Check headers:
- `X-RateLimit-Remaining` - Rate limit tokens left
- `X-Quota-Used` / `X-Quota-Limit` - Monthly quota

### Why am I getting 502 errors?

Upstream service unavailable. Check:
```bash
# Check route configuration
apigate routes get <route-id>

# Test upstream directly
curl -v <upstream-url>/health
```

### How do I view logs?

Logs are output to stdout/stderr by default. Use your system's logging infrastructure or redirect to a file:

```bash
./apigate serve 2>&1 | tee apigate.log
```

For debug logging:
```bash
APIGATE_LOG_LEVEL=debug ./apigate serve
```

See [[Troubleshooting]] for more.

---

## Performance

### How many requests can APIGate handle?

Depends on hardware, but typically:
- Single instance: 10,000+ req/sec on modern hardware

### How do I scale APIGate?

For a single instance:
1. Use adequate hardware (SSD for database)
2. Tune SQLite settings if needed
3. Use a reverse proxy/CDN for static content

For high availability, deploy multiple independent instances behind a load balancer.

---

## Customization

### Can I customize the customer portal?

Yes. See [[Branding]] for branding options.

### Can I add custom fields to users?

Not directly in schema, but you can:
1. Use the `metadata` pattern via settings
2. Store in separate table via custom module

### Can I write custom plugins?

Currently no plugin system. Contribute via pull requests.

---

## See Also

- [[Quick-Start]] - Getting started
- [[Troubleshooting]] - Problem solving
- [[Configuration]] - All settings
