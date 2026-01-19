# LemonSqueezy Integration

LemonSqueezy is a simple payment solution ideal for indie developers and small teams.

---

## Setup

### 1. Create LemonSqueezy Account

1. Sign up at [lemonsqueezy.com](https://lemonsqueezy.com)
2. Create a store
3. Get API key from Settings > API

### 2. Configure APIGate

```bash
# Via CLI settings
apigate settings set payment.provider lemonsqueezy
apigate settings set payment.lemonsqueezy.api_key "xxx" --encrypted
apigate settings set payment.lemonsqueezy.store_id "xxx"
apigate settings set payment.lemonsqueezy.webhook_secret "xxx" --encrypted
```

### 3. Set Up Webhooks

In LemonSqueezy Dashboard > Settings > Webhooks:

1. Add webhook URL: `https://your-domain.com/webhooks/lemonsqueezy`
2. Add signing secret
3. Select events:
   - `subscription_created`
   - `subscription_updated`
   - `subscription_cancelled`
   - `subscription_payment_success`
   - `subscription_payment_failed`

---

## Plan Configuration

### 1. Create Plans in APIGate

```bash
apigate plans create \
  --name "Pro" \
  --rate-limit-per-minute 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900
```

### 2. Link to LemonSqueezy Variant

After creating the plan:

1. Create a product with variants in LemonSqueezy Dashboard
2. In APIGate Admin UI, go to **Plans** and edit the plan
3. Enter the LemonSqueezy variant ID in the LemonSqueezy Variant ID field
4. Save the plan

> **Note**: LemonSqueezy variant IDs must be linked via the Admin UI, not CLI.

---

## Benefits

- **Simple setup**: Quick to get started
- **Merchant of record**: Handles taxes and compliance
- **Built-in affiliate system**: Easy referral programs
- **License keys**: Built-in software licensing

---

## Checkout Flow

LemonSqueezy uses hosted checkout:

1. User clicks "Subscribe"
2. Redirected to LemonSqueezy checkout
3. User completes payment
4. Webhook notifies APIGate
5. User redirected back, plan activated

---

## Webhook Events

| Event | APIGate Action |
|-------|----------------|
| `subscription_created` | Activate user plan |
| `subscription_updated` | Update user plan |
| `subscription_cancelled` | Revert to free plan |
| `subscription_payment_success` | Record payment |
| `subscription_payment_failed` | Send notification |

---

## Testing

Use LemonSqueezy test mode in the dashboard to test checkout flows with test payments.

---

## See Also

- [[Plans]] - Plan configuration
- [[Providers]] - Provider overview
- [[Webhooks]] - Webhook handling
- [[Metering-API]] - External usage events
