# Provider Integrations

APIGate uses external providers for payment processing and email delivery.

---

## Overview

APIGate supports pluggable providers for:
- **Payment** - Subscription billing (Stripe, Paddle, LemonSqueezy)
- **Email** - Transactional emails (SMTP)

Providers are configured through database settings, not environment variables directly.

---

## Payment Providers

Payment providers handle subscription billing, invoicing, and customer management.

### Stripe

Full-featured payment integration.

```bash
# Via settings CLI
apigate settings set payment.provider stripe
apigate settings set payment.stripe.secret_key "sk_live_xxx" --encrypted
apigate settings set payment.stripe.public_key "pk_live_xxx"
apigate settings set payment.stripe.webhook_secret "whsec_xxx" --encrypted
```

**Features**:
- Subscription management
- Usage-based billing
- Customer portal
- Webhook events

**Webhook URL**: `https://your-domain.com/webhooks/stripe`

### Paddle

Alternative payment provider with built-in tax handling.

```bash
apigate settings set payment.provider paddle
apigate settings set payment.paddle.api_key "xxx" --encrypted
apigate settings set payment.paddle.public_key "xxx"
apigate settings set payment.paddle.webhook_secret "xxx" --encrypted
```

**Webhook URL**: `https://your-domain.com/webhooks/paddle`

### LemonSqueezy

Simple payment provider for indie developers.

```bash
apigate settings set payment.provider lemonsqueezy
apigate settings set payment.lemonsqueezy.api_key "xxx" --encrypted
apigate settings set payment.lemonsqueezy.store_id "xxx"
apigate settings set payment.lemonsqueezy.webhook_secret "xxx" --encrypted
```

**Webhook URL**: `https://your-domain.com/webhooks/lemonsqueezy`

### Dummy (Testing)

Simulates successful payments for development/testing.

```bash
apigate settings set payment.provider dummy
```

All payment operations succeed without external calls.

### None (Default)

Disables payment processing. Subscriptions and billing will not work.

```bash
apigate settings set payment.provider none
```

---

## Email Providers

Email providers send transactional emails (password reset, verification, welcome).

### SMTP

Standard email via any SMTP server.

```bash
apigate settings set email.provider smtp
apigate settings set email.smtp.host smtp.example.com
apigate settings set email.smtp.port 587
apigate settings set email.smtp.username user
apigate settings set email.smtp.password secret --encrypted
apigate settings set email.from_address noreply@example.com
apigate settings set email.from_name "APIGate"
apigate settings set email.smtp.use_tls true
```

See [[Email-Configuration]] for common SMTP configurations.

### Mock (Development)

Stores emails in memory for testing. Does not send actual emails.

```bash
apigate settings set email.provider mock
```

### None (Default)

Disables email sending. Password reset and email verification will not work.

```bash
apigate settings set email.provider none
```

---

## OAuth Providers

OAuth providers enable social login. See [[OAuth]] for detailed configuration.

### Google

```bash
apigate settings set oauth.google.enabled true
apigate settings set oauth.google.client_id "xxx.googleusercontent.com"
apigate settings set oauth.google.client_secret "xxx" --encrypted
```

### GitHub

```bash
apigate settings set oauth.github.enabled true
apigate settings set oauth.github.client_id "xxx"
apigate settings set oauth.github.client_secret "xxx" --encrypted
```

### Generic OIDC

```bash
apigate settings set oauth.oidc.enabled true
apigate settings set oauth.oidc.name "My IdP"
apigate settings set oauth.oidc.issuer_url "https://your-idp.com"
apigate settings set oauth.oidc.client_id "xxx"
apigate settings set oauth.oidc.client_secret "xxx" --encrypted
```

---

## TLS Providers

See [[Certificates]] for TLS/ACME configuration.

```bash
apigate settings set tls.enabled true
apigate settings set tls.mode acme
apigate settings set tls.domain "api.example.com"
apigate settings set tls.acme_email "admin@example.com"
```

---

## Development Configuration

Use test providers during development:

```bash
apigate settings set payment.provider dummy
apigate settings set email.provider mock
```

---

## See Also

- [[Configuration]] - Full configuration reference
- [[Email-Configuration]] - Email provider details
- [[OAuth]] - OAuth provider setup
- [[Certificates]] - TLS configuration
- [[Payment-Stripe]] - Stripe integration guide
- [[Payment-Paddle]] - Paddle integration guide
- [[Payment-LemonSqueezy]] - LemonSqueezy integration guide
