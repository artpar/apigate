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

Payment provider price IDs must be linked via the **Admin UI**, not CLI.

### Steps

1. Create your plan in APIGate via CLI:

   ```bash
   apigate plans create \
     --name "Pro" \
     --rate-limit-per-minute 1000 \
     --requests-per-month 100000 \
     --price-monthly 2900
   ```

2. Create the corresponding price in your payment provider's dashboard

3. In APIGate Admin UI:
   - Go to **Plans**
   - Click on the plan to edit
   - Enter the price ID in the appropriate field:
     - **Stripe Price ID** (e.g., `price_xxx`)
     - **Paddle Price ID** (e.g., `pri_xxx`)
     - **LemonSqueezy Variant ID** (e.g., `xxx`)
   - Save the plan

> **Note**: There are no CLI flags like `--stripe-price-id` - price linking must be done via Admin UI.

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
