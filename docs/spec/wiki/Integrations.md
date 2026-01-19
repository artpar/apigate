# Integrations

APIGate integrates with external services through a capability/provider pattern.

---

## Overview

Integrations are organized by capability:

| Capability | Purpose | Providers |
|------------|---------|-----------|
| **Payment** | Billing | Stripe, Paddle, LemonSqueezy |
| **Email** | Sending emails | SMTP, SendGrid |
| **Cache** | Caching | Redis, Memory |
| **Storage** | File storage | S3, Disk |
| **Queue** | Background jobs | Redis, Memory |
| **Notification** | Alerts | Slack, Webhook |
| **OAuth** | Social login | Google, GitHub, OIDC |
| **TLS** | Certificates | ACME |

---

## Configuration

Each integration is configured via environment variables:

```bash
# Payment
PAYMENT_PROVIDER=stripe
STRIPE_API_KEY=sk_xxx

# Email
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.example.com

# Cache
CACHE_PROVIDER=redis
REDIS_URL=redis://localhost:6379
```

---

## Provider Documentation

### Payment

- [[Payment-Stripe]] - Stripe integration
- [[Payment-Paddle]] - Paddle integration
- [[Payment-LemonSqueezy]] - LemonSqueezy integration

### Authentication

- [[OAuth]] - OAuth providers (Google, GitHub, OIDC)

### Infrastructure

- [[Email-Configuration]] - Email providers
- [[Database-Setup]] - Database configuration
- [[Certificates]] - TLS/ACME setup

---

## Full Reference

See [[Providers]] for complete documentation of all capabilities and providers.

---

## See Also

- [[Providers]] - Complete provider reference
- [[Configuration]] - All configuration options
