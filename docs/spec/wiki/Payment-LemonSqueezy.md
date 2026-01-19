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
# Environment variables
PAYMENT_PROVIDER=lemonsqueezy
LEMON_API_KEY=xxx
LEMON_STORE_ID=xxx
LEMON_WEBHOOK_SECRET=xxx

# Or via CLI
apigate settings set payment.provider lemonsqueezy
apigate settings set payment.lemon.api_key "xxx"
apigate settings set payment.lemon.store_id "xxx"
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

## Plan Synchronization

### Create Products in LemonSqueezy

1. Create products with variants in LemonSqueezy
2. Link variants to APIGate plans:

```bash
apigate plans create \
  --name "Pro" \
  --rate-limit 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900 \
  --lemon-variant-id "xxx"
```

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

Use LemonSqueezy test mode:

```bash
LEMON_TEST_MODE=true
```

---

## See Also

- [[Plans]] - Plan configuration
- [[Providers]] - Provider overview
- [[Webhooks]] - Webhook handling
