# Billing

APIGate integrates with payment providers to handle subscription billing.

---

## Overview

Billing in APIGate connects:
- **Plans** - Define pricing and limits
- **Payment Providers** - Handle actual payments
- **Usage Tracking** - Metered billing

```
+-------------------------------------------------------------+
|                    Billing Flow                             |
+-------------------------------------------------------------+
|                                                             |
|   Plan (APIGate)  <->  Price (Provider)                     |
|        |                    |                               |
|        v                    v                               |
|   User subscribes    Payment processed                      |
|        |                    |                               |
|        v                    v                               |
|   Usage tracked      Invoice generated                      |
|        |                    |                               |
|        v                    v                               |
|   Quota enforced     Billing portal                         |
|                                                             |
+-------------------------------------------------------------+
```

---

## Payment Providers

APIGate supports multiple payment providers:

| Provider | Best For | Docs |
|----------|----------|------|
| **Stripe** | Full-featured billing | [[Payment-Stripe]] |
| **Paddle** | Tax compliance | [[Payment-Paddle]] |
| **LemonSqueezy** | Indie developers | [[Payment-LemonSqueezy]] |
| **None** | Development/testing | Default |

---

## Setting Up Billing

### 1. Configure Provider

Set the billing mode in your configuration:

```yaml
billing:
  mode: stripe  # or "paddle", "lemonsqueezy", "none"
  stripe_key: "sk_xxx"
```

Or via environment variables:

```bash
APIGATE_BILLING_MODE=stripe
APIGATE_BILLING_STRIPE_KEY=sk_xxx
```

### 2. Create Plans

Plans define the rate limits, quotas, and pricing:

```bash
apigate plans create \
  --id pro \
  --name "Pro" \
  --rate-limit 1000 \
  --requests 100000 \
  --price 2900
```

**Note**: Payment provider price IDs (like Stripe's `price_xxx`) are linked through the web UI or database, not via CLI flags.

### 3. Link to Provider

After creating a plan, link it to your payment provider's price in:
- **Stripe**: Create a price in Stripe Dashboard, then update plan's `stripe_price_id`
- **Paddle**: Create a product in Paddle Dashboard, then update plan's `paddle_price_id`
- **LemonSqueezy**: Create a variant, then update plan's `lemon_variant_id`

---

## Pricing Models

### Fixed Monthly

```bash
apigate plans create \
  --id pro \
  --name "Pro" \
  --price 2900  # $29.00
```

### Usage-Based

```bash
apigate plans create \
  --id payg \
  --name "Pay As You Go" \
  --price 0 \
  --overage 1  # $0.01 per request over quota
```

> **Note**: Usage includes both proxy requests AND external events submitted via the [[Metering-API]]. External services can report deployments, compute time, storage, and other billable resources.

### Tiered

Create multiple plans with different limits:

```bash
apigate plans create --id free --name "Free" --requests 1000 --price 0
apigate plans create --id pro --name "Pro" --requests 100000 --price 2900
apigate plans create --id enterprise --name "Enterprise" --requests -1 --price 29900  # -1 = unlimited
```

---

## Customer Portal

Users manage their billing through the customer portal:
- View current plan
- Upgrade/downgrade
- Update payment method
- View invoices
- Cancel subscription

Access at: `/portal/billing`

---

## See Also

- [[Plans]] - Plan configuration
- [[Payment-Stripe]] - Stripe integration
- [[Payment-Paddle]] - Paddle integration
- [[Payment-LemonSqueezy]] - LemonSqueezy integration
- [[Usage-Metering]] - Usage tracking
