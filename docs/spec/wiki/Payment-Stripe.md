# Stripe Integration

Stripe is the recommended payment provider for production deployments.

---

## Setup

### 1. Create Stripe Account

1. Sign up at [stripe.com](https://stripe.com)
2. Complete business verification
3. Get API keys from Dashboard > Developers > API keys

### 2. Configure APIGate

```bash
# Environment variables
PAYMENT_PROVIDER=stripe
STRIPE_API_KEY=sk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Or via CLI
apigate settings set payment.provider stripe
apigate settings set payment.stripe.api_key "sk_live_xxx"
apigate settings set payment.stripe.webhook_secret "whsec_xxx"
```

### 3. Set Up Webhooks

In Stripe Dashboard > Developers > Webhooks:

1. Click "Add endpoint"
2. URL: `https://your-domain.com/webhooks/stripe`
3. Select events:
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.paid`
   - `invoice.payment_failed`
4. Copy the signing secret to `STRIPE_WEBHOOK_SECRET`

---

## Plan Synchronization

### Create Plans in Stripe

```bash
# Create a Stripe price, then link to APIGate plan
apigate plans create \
  --name "Pro" \
  --rate-limit 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900 \
  --stripe-price-id "price_xxx"
```

### Automatic Sync

When configured, APIGate automatically:
- Creates Stripe customers for new users
- Creates subscriptions when users select a plan
- Updates subscriptions on plan changes
- Cancels subscriptions on user cancellation

---

## Customer Portal

Stripe's Customer Portal allows users to:
- Update payment methods
- View invoices
- Cancel subscriptions

```bash
# Enable portal redirect
apigate settings set payment.stripe.portal_enabled true
```

Users access via: `/portal/billing`

---

## Usage-Based Billing

For metered billing:

```bash
# Create metered plan
apigate plans create \
  --name "Pay As You Go" \
  --price-monthly 0 \
  --overage-price 1 \
  --stripe-price-id "price_metered_xxx"
```

APIGate reports usage to Stripe automatically at the end of each billing period.

---

## Testing

Use Stripe test mode:

```bash
STRIPE_API_KEY=sk_test_xxx
```

Test card numbers:
- Success: `4242424242424242`
- Decline: `4000000000000002`
- Requires auth: `4000002500003155`

---

## Webhook Events

| Event | APIGate Action |
|-------|----------------|
| `customer.subscription.created` | Activate user plan |
| `customer.subscription.updated` | Update user plan |
| `customer.subscription.deleted` | Revert to free plan |
| `invoice.paid` | Record payment |
| `invoice.payment_failed` | Send notification, retry |

---

## Troubleshooting

### Webhook Signature Failed

**Error**: `Webhook signature verification failed`

**Solution**: Ensure `STRIPE_WEBHOOK_SECRET` matches the endpoint's signing secret.

### Customer Not Found

**Error**: `No such customer: cus_xxx`

**Solution**: The user's Stripe customer was deleted. Clear the mapping:
```bash
apigate provider-mappings delete stripe user <user-id>
```

### Subscription Sync Issues

```bash
# Force sync from Stripe
apigate stripe sync --user <user-id>
```

---

## See Also

- [[Plans]] - Plan configuration
- [[Providers]] - Provider overview
- [[Webhooks]] - Webhook handling
- [[Tutorial-Stripe]] - Step-by-step tutorial
