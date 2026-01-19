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
APIGATE_BILLING_MODE=stripe
APIGATE_BILLING_STRIPE_KEY=sk_live_xxx

# Or via CLI settings
apigate settings set payment.provider stripe
apigate settings set payment.stripe.secret_key "sk_live_xxx" --encrypted
apigate settings set payment.stripe.public_key "pk_live_xxx"
apigate settings set payment.stripe.webhook_secret "whsec_xxx" --encrypted
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
4. Copy the signing secret to webhook_secret setting

---

## Plan Configuration

### 1. Create Plans in APIGate

```bash
apigate plans create \
  --name "Pro" \
  --rate-limit-per-minute 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900 \
  --trial-days 14
```

### 2. Link to Stripe Price

After creating the plan in APIGate:

1. Create a corresponding price in Stripe Dashboard
2. In APIGate Admin UI, go to **Plans** and edit the plan
3. Enter the Stripe price ID (e.g., `price_xxx`) in the Stripe Price ID field
4. Save the plan

> **Note**: Stripe price IDs must be linked via the Admin UI, not CLI.

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

Users access the portal via: `/portal/billing`

---

## Usage-Based Billing

For metered billing:

```bash
# Create metered plan
apigate plans create \
  --name "Pay As You Go" \
  --price-monthly 0 \
  --overage-price 1
```

Link to a Stripe metered price via Admin UI. APIGate reports usage to Stripe automatically at the end of each billing period.

External services can also report usage via the [[Metering-API]].

---

## Testing

Use Stripe test mode:

```bash
apigate settings set payment.stripe.secret_key "sk_test_xxx" --encrypted
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

**Solution**: Ensure webhook secret matches the endpoint's signing secret:
```bash
apigate settings set payment.stripe.webhook_secret "whsec_xxx" --encrypted
```

### Customer Not Found

**Error**: `No such customer: cus_xxx`

**Solution**: The user's Stripe customer mapping may be stale. Check the user record in Admin UI and verify the Stripe customer ID.

### Subscription Sync Issues

1. Check webhook logs in Stripe Dashboard
2. Verify webhook endpoint is receiving events
3. Check APIGate logs for processing errors

---

## See Also

- [[Plans]] - Plan configuration
- [[Providers]] - Provider overview
- [[Webhooks]] - Webhook handling
- [[Metering-API]] - External usage events
- [[Tutorial-Stripe]] - Step-by-step tutorial
