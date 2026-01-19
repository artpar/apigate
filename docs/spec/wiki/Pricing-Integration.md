# Pricing Integration

Connect APIGate plans to your payment provider's pricing.

---

## Overview

APIGate plans map to payment provider prices:

```
┌─────────────────────────────────────────────────────────────┐
│                    Price Mapping                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   APIGate Plan          Payment Provider                   │
│   ─────────────         ─────────────────                  │
│   Pro Plan        ────▶  Stripe: price_xxx                 │
│   ($29/month)            Paddle: pri_xxx                   │
│                          Lemon: variant_xxx                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Linking Plans to Prices

### Stripe

```bash
apigate plans update <id> --stripe-price-id "price_xxx"
```

### Paddle

```bash
apigate plans update <id> --paddle-price-id "pri_xxx"
```

### LemonSqueezy

```bash
apigate plans update <id> --lemon-variant-id "xxx"
```

---

## Creating Linked Plans

```bash
# Create plan with Stripe price
apigate plans create \
  --name "Pro" \
  --rate-limit 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900 \
  --stripe-price-id "price_xxx"
```

---

## Synchronization

Plans sync automatically via webhooks:

1. User subscribes → Webhook received
2. APIGate matches price ID to plan
3. User's plan updated

---

## See Also

- [[Plans]] - Plan configuration
- [[Payment-Stripe]] - Stripe setup
- [[Payment-Paddle]] - Paddle setup
- [[Payment-LemonSqueezy]] - LemonSqueezy setup
- [[Billing]] - Billing overview
