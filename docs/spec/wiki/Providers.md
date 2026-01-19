# Provider Integrations

APIGate uses a **capability/provider** pattern for external integrations, allowing you to swap implementations without code changes.

---

## Overview

```
┌────────────────────────────────────────────────────────────────┐
│                   Capability/Provider Pattern                   │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐                                               │
│  │ Capability  │◀───── Defines the interface                   │
│  │  (Payment)  │                                               │
│  └──────┬──────┘                                               │
│         │                                                       │
│         │ implements                                            │
│         │                                                       │
│  ┌──────┴──────┬──────────────┬──────────────┐                 │
│  │             │              │              │                 │
│  ▼             ▼              ▼              ▼                 │
│ ┌───────┐  ┌───────┐    ┌───────┐    ┌───────┐               │
│ │Stripe │  │Paddle │    │Lemon  │    │Dummy  │               │
│ │       │  │       │    │Squeezy│    │(Test) │               │
│ └───────┘  └───────┘    └───────┘    └───────┘               │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Capabilities

APIGate defines these capability interfaces:

| Capability | Purpose | Providers |
|------------|---------|-----------|
| **Payment** | Subscription billing | Stripe, Paddle, LemonSqueezy, Dummy |
| **Email** | Sending emails | SMTP, SendGrid, Log (testing) |
| **Cache** | Caching data | Redis, Memory |
| **Storage** | File storage | S3, Disk, Memory |
| **Queue** | Background jobs | Redis, Memory |
| **Notification** | Alerts | Slack, Webhook, Log |
| **OAuth** | Social login | Google, GitHub, OIDC |
| **TLS** | Certificate management | ACME |

---

## Payment Providers

### Stripe

Full-featured payment integration.

```bash
# Environment
PAYMENT_PROVIDER=stripe
STRIPE_API_KEY=sk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# CLI
apigate settings set payment.provider stripe
apigate settings set payment.stripe.api_key "sk_live_xxx"
apigate settings set payment.stripe.webhook_secret "whsec_xxx"
```

**Features**:
- Subscription management
- Usage-based billing
- Customer portal
- Webhooks for payment events
- Automatic plan sync

**Webhook Events**:
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.paid`
- `invoice.payment_failed`

### Paddle

Alternative payment provider with built-in tax handling.

```bash
PAYMENT_PROVIDER=paddle
PADDLE_VENDOR_ID=123456
PADDLE_API_KEY=xxx
PADDLE_WEBHOOK_SECRET=xxx
```

### LemonSqueezy

Simple payment provider for indie developers.

```bash
PAYMENT_PROVIDER=lemonsqueezy
LEMON_API_KEY=xxx
LEMON_STORE_ID=xxx
LEMON_WEBHOOK_SECRET=xxx
```

### Dummy (Testing)

No-op provider for development.

```bash
PAYMENT_PROVIDER=dummy
```

All payment operations succeed without external calls.

---

## Email Providers

### SMTP

Standard email via SMTP server.

```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=apigate@example.com
SMTP_PASSWORD=xxx
SMTP_FROM=noreply@example.com
SMTP_TLS=true
```

### SendGrid

Email via SendGrid API.

```bash
EMAIL_PROVIDER=sendgrid
SENDGRID_API_KEY=SG.xxx
SENDGRID_FROM=noreply@example.com
```

### Log (Testing)

Logs emails to console instead of sending.

```bash
EMAIL_PROVIDER=log
```

---

## Cache Providers

### Redis

Production-grade caching with Redis.

```bash
CACHE_PROVIDER=redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=xxx
REDIS_DB=0
```

### Memory

In-process cache (single instance only).

```bash
CACHE_PROVIDER=memory
```

**Note**: Memory cache is lost on restart and doesn't work across multiple instances.

---

## Storage Providers

### S3

AWS S3 or S3-compatible storage.

```bash
STORAGE_PROVIDER=s3
S3_BUCKET=apigate-storage
S3_REGION=us-east-1
S3_ACCESS_KEY=AKIA...
S3_SECRET_KEY=xxx
S3_ENDPOINT=  # Optional, for S3-compatible services
```

### Disk

Local filesystem storage.

```bash
STORAGE_PROVIDER=disk
STORAGE_PATH=/var/lib/apigate/storage
```

### Memory

In-memory storage (testing only).

```bash
STORAGE_PROVIDER=memory
```

---

## Queue Providers

### Redis

Redis-based job queue.

```bash
QUEUE_PROVIDER=redis
REDIS_URL=redis://localhost:6379
```

**Used for**:
- Webhook delivery
- Email sending
- Background analytics

### Memory

In-process queue (single instance only).

```bash
QUEUE_PROVIDER=memory
```

---

## Notification Providers

### Slack

Notifications to Slack channels.

```bash
NOTIFICATION_PROVIDER=slack
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx
SLACK_CHANNEL=#api-alerts
```

### Webhook

Generic HTTP webhook notifications.

```bash
NOTIFICATION_PROVIDER=webhook
NOTIFICATION_WEBHOOK_URL=https://example.com/webhook
NOTIFICATION_WEBHOOK_SECRET=xxx
```

### Log (Testing)

Logs notifications to console.

```bash
NOTIFICATION_PROVIDER=log
```

---

## OAuth Providers

See [[OAuth]] for detailed configuration.

### Google

```bash
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=xxx.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=xxx
```

### GitHub

```bash
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=xxx
OAUTH_GITHUB_CLIENT_SECRET=xxx
```

### Generic OIDC

```bash
OAUTH_OIDC_ENABLED=true
OAUTH_OIDC_ISSUER=https://your-idp.com
OAUTH_OIDC_CLIENT_ID=xxx
OAUTH_OIDC_CLIENT_SECRET=xxx
```

---

## TLS Providers

See [[Certificates]] for detailed configuration.

### ACME (Let's Encrypt)

```bash
TLS_ACME_ENABLED=true
TLS_ACME_EMAIL=admin@example.com
TLS_ACME_DIRECTORY=https://acme-v02.api.letsencrypt.org/directory
```

---

## Provider Mapping

APIGate maintains mappings between local entities and external provider IDs:

| Entity Type | Provider | External ID |
|-------------|----------|-------------|
| user | stripe | cus_xxx |
| plan | stripe | price_xxx |
| subscription | stripe | sub_xxx |
| user | paddle | 123456 |
| plan | paddle | pri_xxx |

### How Mappings Work

```
┌──────────────┐     ┌────────────────────┐     ┌───────────┐
│  Local User  │────▶│  Provider Mapping  │────▶│  Stripe   │
│  usr_abc123  │     │                    │     │  cus_xxx  │
└──────────────┘     │  provider: stripe  │     └───────────┘
                     │  entity: user      │
                     │  local: usr_abc123 │
                     │  external: cus_xxx │
                     └────────────────────┘
```

### View Mappings

```bash
# All mappings for a provider
apigate provider-mappings list --provider stripe

# Mappings for an entity
apigate provider-mappings lookup stripe user usr_abc123
```

---

## Testing Providers

Use test/dummy providers in development:

```bash
# Development configuration
PAYMENT_PROVIDER=dummy
EMAIL_PROVIDER=log
CACHE_PROVIDER=memory
STORAGE_PROVIDER=memory
QUEUE_PROVIDER=memory
NOTIFICATION_PROVIDER=log
```

---

## Health Checks

Each provider supports connection testing:

```bash
# Test all providers
apigate providers test

# Test specific provider
apigate providers test --payment
apigate providers test --email
apigate providers test --cache
```

### API Health Endpoint

```bash
curl http://localhost:8080/admin/health

# Response includes provider status
{
  "status": "healthy",
  "providers": {
    "payment": { "provider": "stripe", "status": "connected" },
    "email": { "provider": "smtp", "status": "connected" },
    "cache": { "provider": "redis", "status": "connected" }
  }
}
```

---

## Adding Custom Providers

APIGate's provider system is extensible. To add a custom provider:

1. Implement the capability interface
2. Register the provider
3. Configure via environment variables

See the source code in `core/modules/capabilities/` and `core/modules/providers/` for examples.

---

## Best Practices

### 1. Use Production Providers

```bash
# Production
PAYMENT_PROVIDER=stripe
CACHE_PROVIDER=redis
QUEUE_PROVIDER=redis

# Don't use memory providers in production
```

### 2. Configure Webhooks

Most payment providers require webhooks for event processing:

```bash
# Stripe webhook URL
https://your-domain.com/webhooks/stripe

# Paddle webhook URL
https://your-domain.com/webhooks/paddle
```

### 3. Monitor Provider Health

Set up alerts for provider failures:

```bash
# Slack notification on provider errors
apigate settings set notification.slack.enabled true
apigate settings set notification.events "provider.error"
```

### 4. Use Secrets Management

Never commit secrets to version control:

```bash
# Good - environment variables
STRIPE_API_KEY=${STRIPE_API_KEY}

# Bad - hardcoded
STRIPE_API_KEY=sk_live_xxx
```

---

## Environment Variable Reference

### Payment
| Variable | Description |
|----------|-------------|
| `PAYMENT_PROVIDER` | Provider name (stripe, paddle, lemonsqueezy, dummy) |
| `STRIPE_API_KEY` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |

### Email
| Variable | Description |
|----------|-------------|
| `EMAIL_PROVIDER` | Provider name (smtp, sendgrid, log) |
| `SMTP_HOST` | SMTP server hostname |
| `SMTP_PORT` | SMTP server port |
| `SMTP_USERNAME` | SMTP username |
| `SMTP_PASSWORD` | SMTP password |
| `SMTP_FROM` | From email address |
| `SENDGRID_API_KEY` | SendGrid API key |

### Cache
| Variable | Description |
|----------|-------------|
| `CACHE_PROVIDER` | Provider name (redis, memory) |
| `REDIS_URL` | Redis connection URL |

### Storage
| Variable | Description |
|----------|-------------|
| `STORAGE_PROVIDER` | Provider name (s3, disk, memory) |
| `S3_BUCKET` | S3 bucket name |
| `S3_REGION` | AWS region |
| `STORAGE_PATH` | Disk storage path |

### Queue
| Variable | Description |
|----------|-------------|
| `QUEUE_PROVIDER` | Provider name (redis, memory) |

### Notification
| Variable | Description |
|----------|-------------|
| `NOTIFICATION_PROVIDER` | Provider name (slack, webhook, log) |
| `SLACK_WEBHOOK_URL` | Slack webhook URL |

---

## See Also

- [[Configuration]] - Full configuration reference
- [[OAuth]] - OAuth provider setup
- [[Certificates]] - TLS provider setup
- [[Webhooks]] - Payment webhooks
