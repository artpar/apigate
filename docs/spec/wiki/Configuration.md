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

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_HOST` | `0.0.0.0` | Listen address |
| `APIGATE_PORT` | `8080` | Listen port |
| `APIGATE_BASE_URL` | `http://localhost:8080` | Public URL |
| `APIGATE_TLS_CERT` | - | TLS certificate path |
| `APIGATE_TLS_KEY` | - | TLS private key path |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_DATABASE_PATH` | `./data/apigate.db` | SQLite database path |

### Security

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_SECRET_KEY` | (generated) | Encryption key for secrets |
| `APIGATE_SESSION_SECRET` | (generated) | Session signing key |
| `APIGATE_SESSION_DURATION` | `24h` | Session expiration |
| `APIGATE_CORS_ORIGINS` | `*` | Allowed CORS origins |

### Admin

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_ADMIN_EMAIL` | - | Initial admin email |
| `APIGATE_ADMIN_PASSWORD` | - | Initial admin password |
| `APIGATE_REQUIRE_SETUP` | `true` | Require setup wizard |

### Proxy

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_DEFAULT_TIMEOUT` | `30000` | Default upstream timeout (ms) |
| `APIGATE_MAX_IDLE_CONNS` | `100` | Max idle connections per upstream |
| `APIGATE_IDLE_CONN_TIMEOUT` | `90000` | Idle connection timeout (ms) |

### Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `APIGATE_RATE_LIMIT_PER_USER` | `false` | Share bucket across user's keys |

### Quotas

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_QUOTA_ENABLED` | `true` | Enable quota tracking |
| `APIGATE_QUOTA_WARNING_THRESHOLDS` | `80,95,100` | Warning percentages |

### Email

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_EMAIL_PROVIDER` | `log` | Email provider (smtp, sendgrid, log) |
| `APIGATE_SMTP_HOST` | - | SMTP server host |
| `APIGATE_SMTP_PORT` | `587` | SMTP server port |
| `APIGATE_SMTP_USER` | - | SMTP username |
| `APIGATE_SMTP_PASSWORD` | - | SMTP password |
| `APIGATE_SMTP_FROM` | - | From email address |
| `APIGATE_SENDGRID_API_KEY` | - | SendGrid API key |

### Payment

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_PAYMENT_PROVIDER` | `dummy` | Payment provider |
| `APIGATE_STRIPE_SECRET_KEY` | - | Stripe secret key |
| `APIGATE_STRIPE_WEBHOOK_SECRET` | - | Stripe webhook secret |
| `APIGATE_PADDLE_VENDOR_ID` | - | Paddle vendor ID |
| `APIGATE_PADDLE_API_KEY` | - | Paddle API key |
| `APIGATE_LEMONSQUEEZY_API_KEY` | - | LemonSqueezy API key |

### Portal

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_PORTAL_ENABLED` | `true` | Enable customer portal |
| `APIGATE_PORTAL_REGISTRATION` | `true` | Allow self-registration |
| `APIGATE_PORTAL_COMPANY_NAME` | `APIGate` | Company name in portal |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `APIGATE_LOG_FORMAT` | `text` | Log format (text, json) |
| `APIGATE_LOG_FILE` | - | Log file path (stdout if empty) |

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
| `--host` | `APIGATE_HOST` | Listen address |
| `--port` | `APIGATE_PORT` | Listen port |
| `--database` | `APIGATE_DATABASE_PATH` | Database path |
| `--log-level` | `APIGATE_LOG_LEVEL` | Logging level |
| `--config` | - | Config file path |

---

## Configuration File

Create `apigate.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  base_url: https://api.example.com

database:
  path: ./data/apigate.db

security:
  secret_key: your-secret-key-here
  session_duration: 24h
  cors_origins:
    - https://app.example.com
    - https://admin.example.com

proxy:
  default_timeout_ms: 30000
  max_idle_conns: 100

rate_limit:
  enabled: true
  per_user: false

quota:
  enabled: true
  warning_thresholds: [80, 95, 100]

email:
  provider: smtp
  smtp:
    host: smtp.example.com
    port: 587
    user: apigate@example.com
    password: ${SMTP_PASSWORD}
    from: noreply@example.com

payment:
  provider: stripe
  stripe:
    secret_key: ${STRIPE_SECRET_KEY}
    webhook_secret: ${STRIPE_WEBHOOK_SECRET}

portal:
  enabled: true
  registration: true
  company_name: "Acme API"

logging:
  level: info
  format: json
  file: /var/log/apigate/apigate.log
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
APIGATE_PORT=8080
APIGATE_DATABASE_PATH=/data/apigate.db
APIGATE_SECRET_KEY=your-secret-key
APIGATE_STRIPE_SECRET_KEY=sk_live_xxx
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
      - APIGATE_DATABASE_PATH=/data/apigate.db
      - APIGATE_SECRET_KEY=${APIGATE_SECRET_KEY}
      - APIGATE_STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
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
  APIGATE_PORT: "8080"
  APIGATE_LOG_LEVEL: "info"
  APIGATE_LOG_FORMAT: "json"
  APIGATE_PORTAL_ENABLED: "true"
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: apigate-secrets
type: Opaque
stringData:
  APIGATE_SECRET_KEY: "your-secret-key"
  APIGATE_STRIPE_SECRET_KEY: "sk_live_xxx"
  APIGATE_DATABASE_PATH: "/data/apigate.db"
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
APIGATE_STRIPE_SECRET_KEY=sk_live_xxx

# Good: Use secret manager
APIGATE_STRIPE_SECRET_KEY=${vault:secret/apigate/stripe_key}
```

### 2. Rotate Secrets Regularly

```bash
# Generate new secret key
apigate secrets rotate --type session

# Generate new encryption key
apigate secrets rotate --type encryption
```

### 3. Limit CORS Origins

```bash
# Bad: Allow all origins
APIGATE_CORS_ORIGINS=*

# Good: Specific origins
APIGATE_CORS_ORIGINS=https://app.example.com,https://admin.example.com
```

### 4. Use TLS in Production

```bash
APIGATE_TLS_CERT=/etc/ssl/certs/apigate.crt
APIGATE_TLS_KEY=/etc/ssl/private/apigate.key
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
ls -la $(dirname $APIGATE_DATABASE_PATH)

# Check permissions
touch $APIGATE_DATABASE_PATH
```

---

## See Also

- [[Installation]] - Initial setup
- [[Production]] - Production deployment
- [[Security]] - Security configuration
- [[Troubleshooting]] - Common issues
