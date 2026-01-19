# Integrations

APIGate integrates with external services through a capability/provider pattern.

---

## Overview

Integrations are organized by capability:

| Capability | Purpose | Providers |
|------------|---------|-----------|
| **Payment** | Billing | Stripe, Paddle, LemonSqueezy, Dummy, None |
| **Email** | Sending emails | SMTP, Mock, None |
| **OAuth** | Social login | Google, GitHub, OIDC |
| **TLS** | Certificates | ACME, Manual |

---

## Configuration

Integrations are configured via the settings system:

```bash
# Payment
apigate settings set payment.provider stripe
apigate settings set payment.stripe.secret_key "sk_xxx" --encrypted
apigate settings set payment.stripe.public_key "pk_xxx"

# Email
apigate settings set email.provider smtp
apigate settings set email.smtp.host "smtp.example.com"
apigate settings set email.smtp.port "587"
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
