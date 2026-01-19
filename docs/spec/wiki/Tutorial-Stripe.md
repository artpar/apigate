# Tutorial: Stripe Integration

Accept payments and manage subscriptions with Stripe.

---

## Overview

Integrate Stripe to:
- Accept credit card payments
- Manage recurring subscriptions
- Auto-assign plans on payment
- Handle upgrades and downgrades
- Process refunds and cancellations

---

## Prerequisites

- APIGate running with plans configured
- Stripe account (test mode is fine)
- Stripe CLI (optional, for webhook testing)

---

## Step 1: Create Stripe Account

1. Sign up at https://stripe.com
2. Complete account verification
3. Get your API keys from Dashboard → Developers → API Keys

You'll need:
- **Publishable key**: `pk_test_xxx` (for frontend)
- **Secret key**: `sk_test_xxx` (for backend)

---

## Step 2: Configure APIGate for Stripe

```bash
# Set payment provider
apigate settings set payment_provider stripe

# Configure API key
apigate settings set stripe_secret_key "sk_test_xxx"

# (We'll set webhook secret in Step 5)
```

Or via environment:
```bash
export APIGATE_PAYMENT_PROVIDER=stripe
export APIGATE_STRIPE_SECRET_KEY=sk_test_xxx
```

---

## Step 3: Create Products in Stripe

Create products matching your APIGate plans:

### Via Stripe Dashboard

1. Go to Products → Add Product
2. Create each plan:

**Starter Plan:**
- Name: `Starter`
- Description: `25,000 API requests per month`
- Pricing: `$29.00/month` recurring
- Note the Price ID: `price_xxx`

**Pro Plan:**
- Name: `Pro`
- Description: `100,000 API requests per month`
- Pricing: `$99.00/month` recurring
- Note the Price ID: `price_yyy`

### Via Stripe CLI

```bash
# Create Starter product and price
stripe products create \
  --name="Starter" \
  --description="25,000 API requests per month"

stripe prices create \
  --product="prod_xxx" \
  --unit-amount=2900 \
  --currency=usd \
  --recurring[interval]=month

# Create Pro product and price
stripe products create \
  --name="Pro" \
  --description="100,000 API requests per month"

stripe prices create \
  --product="prod_yyy" \
  --unit-amount=9900 \
  --currency=usd \
  --recurring[interval]=month
```

---

## Step 4: Link Stripe Prices to APIGate Plans

```bash
# Link Starter plan
apigate plans update Starter --stripe-price-id "price_xxx"

# Link Pro plan
apigate plans update Pro --stripe-price-id "price_yyy"
```

Verify the link:
```bash
apigate plans get Starter
# Shows: stripe_price_id: price_xxx
```

---

## Step 5: Set Up Webhooks

Stripe webhooks notify APIGate of payment events.

### Create Webhook Endpoint

In Stripe Dashboard → Developers → Webhooks:

1. Click **Add endpoint**
2. Endpoint URL: `https://your-domain.com/webhooks/stripe`
3. Select events:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.paid`
   - `invoice.payment_failed`
4. Click **Add endpoint**
5. Copy the **Signing secret**: `whsec_xxx`

### Configure Webhook Secret

```bash
apigate settings set stripe_webhook_secret "whsec_xxx"
```

### Test Webhooks Locally

Using Stripe CLI:

```bash
# Forward webhooks to local APIGate
stripe listen --forward-to localhost:8080/webhooks/stripe

# In another terminal, trigger test events
stripe trigger checkout.session.completed
```

---

## Step 6: Enable Checkout Flow

### Portal Checkout

When Stripe is configured, the portal automatically shows:
- **Upgrade** buttons on free accounts
- **Change Plan** for existing subscribers
- **Payment method** management

### Custom Checkout

Create checkout sessions via API:

```bash
curl -X POST http://localhost:8080/api/portal/checkout \
  -H "Authorization: Bearer USER_SESSION" \
  -d '{
    "plan_id": "pro-plan-id",
    "success_url": "https://yoursite.com/success",
    "cancel_url": "https://yoursite.com/pricing"
  }'
```

Response:
```json
{
  "checkout_url": "https://checkout.stripe.com/pay/cs_xxx"
}
```

Redirect user to `checkout_url` to complete payment.

---

## Step 7: Test the Flow

### Test Mode Cards

Use these test cards:
- **Success**: `4242 4242 4242 4242`
- **Decline**: `4000 0000 0000 0002`
- **Requires auth**: `4000 0025 0000 3155`

### Complete Test Purchase

1. Register a new user in portal
2. Click **Upgrade to Starter**
3. Enter test card: `4242 4242 4242 4242`
4. Any future date, any CVC
5. Complete purchase
6. User automatically upgraded to Starter plan

### Verify in APIGate

```bash
apigate users get test@example.com
# Shows: plan: Starter
```

### Verify in Stripe

Dashboard → Customers → Find customer → Shows active subscription

---

## Step 8: Handle Subscription Events

APIGate automatically handles these events:

| Stripe Event | APIGate Action |
|--------------|----------------|
| `checkout.session.completed` | Create/update user, assign plan |
| `customer.subscription.updated` | Change plan if price changed |
| `customer.subscription.deleted` | Downgrade to free plan |
| `invoice.payment_failed` | Send warning email |

### Custom Event Handling

Forward events to your system:

```bash
apigate webhooks create \
  --name "Billing Events" \
  --url "https://your-backend.com/billing" \
  --events "subscription.created,subscription.upgraded,subscription.cancelled"
```

---

## Step 9: Manage Subscriptions

### View Subscription

```bash
apigate users subscription test@example.com
```

Output:
```
Subscription: sub_xxx
Plan: Pro
Status: active
Current period: 2025-01-19 to 2025-02-19
Payment method: •••• 4242
Next invoice: $99.00 on 2025-02-19
```

### Cancel Subscription

```bash
# Via portal: User clicks "Cancel Subscription"

# Via API
curl -X POST http://localhost:8080/admin/users/<id>/cancel-subscription

# Via CLI
apigate users cancel-subscription test@example.com
```

Subscription cancels at end of billing period.

### Upgrade/Downgrade

```bash
# Change plan
apigate users change-plan test@example.com --plan Enterprise
```

Stripe prorates automatically.

---

## Step 10: Go Live

### Switch to Live Mode

1. Complete Stripe account verification
2. Get live API keys
3. Update APIGate:

```bash
apigate settings set stripe_secret_key "sk_live_xxx"
apigate settings set stripe_webhook_secret "whsec_live_xxx"
```

4. Create live webhook endpoint in Stripe

### Checklist

- [ ] Stripe account verified
- [ ] Live API keys configured
- [ ] Webhook endpoint created
- [ ] Products/prices created in live mode
- [ ] Plans linked to live price IDs
- [ ] Test purchase in live mode (use real card)

---

## Advanced: Usage-Based Billing

For metered APIs, report usage to Stripe:

### Configure Usage Metering

```bash
# Create metered price in Stripe
stripe prices create \
  --product="prod_xxx" \
  --currency=usd \
  --recurring[interval]=month \
  --recurring[usage_type]=metered \
  --unit-amount-decimal=0.001
```

### Report Usage

APIGate reports usage automatically at end of billing period, or you can trigger manually:

```bash
apigate billing report-usage --period 2025-01
```

---

## Troubleshooting

### Webhook Not Receiving Events

1. Check webhook URL is accessible
2. Verify signing secret is correct
3. Check Stripe Dashboard → Webhooks → Recent events
4. Look for failed deliveries

### User Not Upgraded After Payment

1. Check webhook logs: `apigate logs --filter stripe`
2. Verify `customer.email` matches user email
3. Check Stripe customer metadata

### Subscription Shows Wrong Plan

1. Verify price ID matches plan:
   ```bash
   apigate plans get <plan>
   stripe prices retrieve <price_id>
   ```
2. Check for multiple subscriptions

### Payment Failed

1. Check Stripe Dashboard for decline reason
2. User receives email to update payment method
3. Subscription status: `past_due`

---

## Complete Integration Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Stripe Integration Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Customer clicks "Upgrade"                                    │
│       │                                                          │
│       ▼                                                          │
│  2. APIGate creates Stripe Checkout Session                      │
│       │                                                          │
│       ▼                                                          │
│  3. Customer enters payment on Stripe                            │
│       │                                                          │
│       ▼                                                          │
│  4. Stripe sends webhook: checkout.session.completed             │
│       │                                                          │
│       ▼                                                          │
│  5. APIGate receives webhook                                     │
│       │                                                          │
│       ▼                                                          │
│  6. APIGate assigns user to paid plan                            │
│       │                                                          │
│       ▼                                                          │
│  7. User immediately gets higher limits                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Next Steps

1. **[[Billing]]** - Advanced billing configurations
2. **[[Webhooks]]** - Forward billing events
3. **[[Analytics]]** - Track revenue metrics
4. **[[Tutorial-Production]]** - Production deployment

---

## Summary

You've learned how to:

1. ✅ Create Stripe account and get API keys
2. ✅ Configure APIGate for Stripe
3. ✅ Create products and prices in Stripe
4. ✅ Link Stripe prices to APIGate plans
5. ✅ Set up webhooks for payment events
6. ✅ Enable checkout in customer portal
7. ✅ Test the complete payment flow
8. ✅ Handle subscription lifecycle
9. ✅ Go live with real payments

Your API now accepts credit card payments!
