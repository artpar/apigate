# Paddle Integration

Paddle handles payments, tax compliance, and merchant of record responsibilities.

---

## Setup

### 1. Create Paddle Account

1. Sign up at [paddle.com](https://paddle.com)
2. Complete verification
3. Get credentials from Paddle Dashboard

### 2. Configure APIGate

```bash
# Environment variables
PAYMENT_PROVIDER=paddle
PADDLE_VENDOR_ID=123456
PADDLE_API_KEY=xxx
PADDLE_WEBHOOK_SECRET=xxx

# Or via CLI
apigate settings set payment.provider paddle
apigate settings set payment.paddle.vendor_id "123456"
apigate settings set payment.paddle.api_key "xxx"
```

### 3. Set Up Webhooks

In Paddle Dashboard > Developer Tools > Notifications:

1. Add webhook URL: `https://your-domain.com/webhooks/paddle`
2. Select events:
   - `subscription_created`
   - `subscription_updated`
   - `subscription_cancelled`
   - `subscription_payment_succeeded`
   - `subscription_payment_failed`

---

## Plan Synchronization

### Create Plans in Paddle

1. Create products in Paddle Dashboard
2. Link to APIGate plans:

```bash
apigate plans create \
  --name "Pro" \
  --rate-limit 1000 \
  --requests-per-month 100000 \
  --price-monthly 2900 \
  --paddle-price-id "pri_xxx"
```

---

## Benefits

- **Tax handling**: Paddle calculates and remits VAT/sales tax
- **Merchant of record**: Paddle handles refunds, chargebacks
- **Global payments**: Supports many currencies and payment methods

---

## Checkout Flow

Paddle uses overlay checkout:

1. User clicks "Subscribe"
2. Paddle checkout opens in overlay
3. User completes payment
4. Webhook notifies APIGate
5. User plan activated

---

## Webhook Events

| Event | APIGate Action |
|-------|----------------|
| `subscription_created` | Activate user plan |
| `subscription_updated` | Update user plan |
| `subscription_cancelled` | Revert to free plan |
| `subscription_payment_succeeded` | Record payment |
| `subscription_payment_failed` | Send notification |

---

## Testing

Use Paddle sandbox:

```bash
PADDLE_ENVIRONMENT=sandbox
PADDLE_VENDOR_ID=sandbox_vendor_id
```

---

## See Also

- [[Plans]] - Plan configuration
- [[Providers]] - Provider overview
- [[Webhooks]] - Webhook handling
