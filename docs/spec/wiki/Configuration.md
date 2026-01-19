# Configuration

Configure APIGate via environment variables, command-line flags, or runtime settings.

---

## Configuration Methods

### Priority Order

1. **Command-line flags** (highest)
2. **Environment variables**
3. **Configuration file**
4. **Runtime settings** (database)
5. **Default values** (lowest)

---

## Environment Variables

> **Implementation**: `config/config.go:241-383`

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_SERVER_HOST` | `0.0.0.0` | Listen address |
| `APIGATE_SERVER_PORT` | `8080` | Listen port |
| `APIGATE_SERVER_READ_TIMEOUT` | `30s` | HTTP read timeout (duration) |
| `APIGATE_SERVER_WRITE_TIMEOUT` | `60s` | HTTP write timeout (duration) |

### Upstream Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_UPSTREAM_URL` | **(required)** | Upstream API URL to proxy |
| `APIGATE_UPSTREAM_TIMEOUT` | - | Request timeout (duration, e.g., `30s`) |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_DATABASE_DRIVER` | `sqlite` | Database driver (sqlite, postgres) |
| `APIGATE_DATABASE_DSN` | `apigate.db` | Database connection string |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_AUTH_MODE` | `local` | Auth mode: `local` or `remote` |
| `APIGATE_AUTH_KEY_PREFIX` | `ak_` | API key prefix |
| `APIGATE_AUTH_REMOTE_URL` | - | Remote auth service URL (when mode=remote) |

### Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_RATELIMIT_ENABLED` | `true` | Enable rate limiting |
| `APIGATE_RATELIMIT_BURST` | `5` | Burst token count |
| `APIGATE_RATELIMIT_WINDOW` | `60` | Rate limit window in seconds |

### Usage Tracking

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_USAGE_MODE` | `local` | Usage mode: `local` or `remote` |
| `APIGATE_USAGE_REMOTE_URL` | - | Remote usage service URL |

### Billing

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_BILLING_MODE` | `none` | Billing mode: `none`, `stripe`, `paddle`, `lemonsqueezy`, `remote` |
| `APIGATE_BILLING_STRIPE_KEY` | - | Stripe secret key |

### Email

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_EMAIL_PROVIDER` | `none` | Email provider: `smtp`, `mock`, `none` |
| `APIGATE_SMTP_HOST` | - | SMTP server host |
| `APIGATE_SMTP_PORT` | `587` | SMTP server port |
| `APIGATE_SMTP_USERNAME` | - | SMTP username |
| `APIGATE_SMTP_PASSWORD` | - | SMTP password |
| `APIGATE_SMTP_FROM` | - | From email address |
| `APIGATE_SMTP_FROM_NAME` | (app name) | Sender display name |
| `APIGATE_SMTP_USE_TLS` | `false` | Enable TLS |

### Portal

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_PORTAL_ENABLED` | `false` | Enable customer portal |
| `APIGATE_PORTAL_BASE_URL` | - | Base URL for email links |
| `APIGATE_PORTAL_APP_NAME` | `APIGate` | Application name in portal |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `APIGATE_LOG_FORMAT` | `json` | Log format: `json` or `console` |

### Metrics & OpenAPI

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_METRICS_ENABLED` | `true` | Enable /metrics endpoint |
| `APIGATE_METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `APIGATE_OPENAPI_ENABLED` | `true` | Enable OpenAPI/Swagger |

---

## Command-Line Flags

```bash
# Start server with custom settings
apigate serve \
  --host 0.0.0.0 \
  --port 8080 \
  --database ./data/apigate.db \
  --log-level debug

# List all flags
apigate serve --help
```

### Common Flags

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--host` | `APIGATE_SERVER_HOST` | Listen address |
| `--port` | `APIGATE_SERVER_PORT` | Listen port |
| `--database` | `APIGATE_DATABASE_DSN` | Database path |
| `--log-level` | `APIGATE_LOG_LEVEL` | Logging level |
| `--config` | - | Config file path |

---

## Configuration File

> **Implementation**: `config/config.go:14-30`

Create `apigate.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 60s

upstream:
  url: https://api.backend.com
  timeout: 30s
  max_idle_conns: 100
  idle_conn_timeout: 90s

auth:
  mode: local  # "local" or "remote"
  key_prefix: "ak_"
  # For remote auth:
  # remote:
  #   url: https://auth.example.com/validate

rate_limit:
  enabled: true
  burst_tokens: 5
  window_secs: 60

usage:
  mode: local  # "local" or "remote"
  batch_size: 100
  flush_interval: 10s

billing:
  mode: stripe  # "none", "stripe", "paddle", "lemonsqueezy", "remote"
  stripe_key: ${STRIPE_SECRET_KEY}

database:
  driver: sqlite
  dsn: ./data/apigate.db

email:
  provider: smtp  # "smtp", "mock", "none"
  smtp:
    host: smtp.example.com
    port: 587
    username: apigate@example.com
    password: ${SMTP_PASSWORD}
    from: noreply@example.com
    from_name: "Acme API"
    use_tls: true

portal:
  enabled: true
  base_url: https://api.example.com
  app_name: "Acme API"

logging:
  level: info
  format: json

metrics:
  enabled: true
  path: /metrics

openapi:
  enabled: true

plans:
  - id: free
    name: Free
    rate_limit_per_minute: 60
    requests_per_month: 1000
  - id: pro
    name: Pro
    rate_limit_per_minute: 600
    requests_per_month: 100000
    price_monthly: 2900  # cents
```

### Load Config File

```bash
apigate serve --config ./apigate.yaml
```

---

## Runtime Settings

Settings stored in database, manageable via UI/API/CLI.

### View Settings

```bash
# CLI
apigate settings list
apigate settings get portal_enabled

# API
curl http://localhost:8080/admin/settings
curl http://localhost:8080/admin/settings/portal_enabled
```

### Update Settings

```bash
# CLI
apigate settings set portal_enabled true
apigate settings set portal_company_name "Acme API"

# API
curl -X PUT http://localhost:8080/admin/settings/portal_enabled \
  -d '{"value": true}'
```

### Available Runtime Settings

| Setting | Type | Description |
|---------|------|-------------|
| `portal_enabled` | bool | Enable customer portal |
| `portal_registration_enabled` | bool | Allow self-registration |
| `portal_company_name` | string | Company name |
| `portal_logo_url` | string | Logo URL |
| `portal_primary_color` | string | Primary brand color |
| `require_email_verification` | bool | Verify emails |
| `setup_completed` | bool | Setup wizard done |
| `quota_warning_thresholds` | string | Comma-separated percentages |

---

## Docker Configuration

### Environment File

Create `.env`:

```bash
APIGATE_UPSTREAM_URL=https://api.backend.com
APIGATE_SERVER_PORT=8080
APIGATE_DATABASE_DSN=/data/apigate.db
APIGATE_BILLING_STRIPE_KEY=sk_live_xxx
```

### Docker Compose

```yaml
version: '3.8'
services:
  apigate:
    image: apigate/apigate:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - APIGATE_UPSTREAM_URL=https://api.backend.com
      - APIGATE_DATABASE_DSN=/data/apigate.db
      - APIGATE_BILLING_STRIPE_KEY=${STRIPE_SECRET_KEY}
      - APIGATE_LOG_LEVEL=info
    env_file:
      - .env
```

---

## Kubernetes Configuration

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: apigate-config
data:
  APIGATE_SERVER_PORT: "8080"
  APIGATE_LOG_LEVEL: "info"
  APIGATE_LOG_FORMAT: "json"
  APIGATE_PORTAL_ENABLED: "true"
  APIGATE_METRICS_ENABLED: "true"
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: apigate-secrets
type: Opaque
stringData:
  APIGATE_UPSTREAM_URL: "https://api.backend.com"
  APIGATE_BILLING_STRIPE_KEY: "sk_live_xxx"
  APIGATE_DATABASE_DSN: "/data/apigate.db"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apigate
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: apigate
        image: apigate/apigate:latest
        envFrom:
        - configMapRef:
            name: apigate-config
        - secretRef:
            name: apigate-secrets
        volumeMounts:
        - name: data
          mountPath: /data
```

---

## Security Best Practices

### 1. Use Secret Management

```bash
# Bad: Secrets in plain text
APIGATE_BILLING_STRIPE_KEY=sk_live_xxx

# Good: Use secret manager
APIGATE_BILLING_STRIPE_KEY=${vault:secret/apigate/stripe_key}
```

### 2. Rotate Secrets Regularly

```bash
# Generate new secret key
apigate secrets rotate --type session

# Generate new encryption key
apigate secrets rotate --type encryption
```

### 3. Use a Reverse Proxy for TLS

APIGate does not handle TLS directly. Use a reverse proxy like nginx or Caddy:

```nginx
# nginx example
server {
    listen 443 ssl;
    ssl_certificate /etc/ssl/certs/apigate.crt;
    ssl_certificate_key /etc/ssl/private/apigate.key;

    location / {
        proxy_pass http://localhost:8080;
    }
}
```

---

## Validation

### Validate Configuration

```bash
# Check configuration
apigate config validate

# Output:
# ✓ Database path writable
# ✓ Secret key set
# ✓ Email provider configured
# ⚠ CORS allows all origins (consider restricting)
```

### Show Effective Configuration

```bash
# Show merged configuration
apigate config show

# Show specific section
apigate config show --section email
```

---

## Troubleshooting

### Configuration Not Applied

1. Check priority order (CLI > env > file > runtime)
2. Restart service for env changes
3. Check for typos in variable names

### Secret Not Found

```bash
# Check if secret is set
apigate config show --section security

# Regenerate if needed
apigate secrets generate
```

### Database Connection Failed

```bash
# Check path exists and is writable
ls -la $(dirname $APIGATE_DATABASE_DSN)

# Check permissions
touch $APIGATE_DATABASE_DSN
```

---

## See Also

- [[Installation]] - Initial setup
- [[Production]] - Production deployment
- [[Security]] - Security configuration
- [[Troubleshooting]] - Common issues
