# Plans

A **plan** defines the pricing tier, rate limits, and quotas for your API customers.

---

## Overview

Plans are the foundation of API monetization. Each plan specifies:
- **Pricing**: Free, monthly, or usage-based
- **Rate limits**: Requests per minute
- **Quotas**: Monthly request allowance
- **Features**: Which capabilities are included

```
┌────────────────────────────────────────────────────────────────┐
│                        Plan Structure                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │    Free     │    │    Pro      │    │ Enterprise  │        │
│  ├─────────────┤    ├─────────────┤    ├─────────────┤        │
│  │ $0/month    │    │ $49/month   │    │ Custom      │        │
│  │ 1K req/mo   │    │ 100K req/mo │    │ Unlimited   │        │
│  │ 60 req/min  │    │ 600 req/min │    │ 6000 req/min│        │
│  │ Basic API   │    │ Full API    │    │ Full + SLA  │        │
│  └─────────────┘    └─────────────┘    └─────────────┘        │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Plan Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Display name (required) |
| `description` | string | Plan description |
| `price_cents` | int | Monthly price in cents |
| `requests_per_month` | int | Monthly quota (0 = unlimited) |
| `rate_limit_per_minute` | int | Requests per minute |
| `rate_limit_burst` | int | Burst allowance |
| `quota_grace_percent` | int | Grace percentage before hard block |
| `quota_enforcement` | enum | hard, soft, warn |
| `features` | []string | Enabled feature flags |
| `metadata` | object | Custom key-value data |
| `stripe_price_id` | string | Stripe integration |
| `paddle_price_id` | string | Paddle integration |
| `enabled` | bool | Plan availability |

---

## Creating Plans

### Admin UI

1. Go to **Plans** in sidebar
2. Click **Add Plan**
3. Configure:
   - **Name**: Plan display name
   - **Price**: Monthly cost
   - **Rate Limit**: Requests per minute
   - **Monthly Quota**: Request limit
4. Click **Save**

### CLI

```bash
# Free tier
apigate plans create \
  --name "Free" \
  --price 0 \
  --rate-limit 60 \
  --monthly-quota 1000

# Pro tier
apigate plans create \
  --name "Pro" \
  --price 4900 \
  --rate-limit 600 \
  --monthly-quota 100000

# Enterprise tier
apigate plans create \
  --name "Enterprise" \
  --price 49900 \
  --rate-limit 6000 \
  --monthly-quota 0  # unlimited
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro",
    "description": "For growing businesses",
    "price_cents": 4900,
    "requests_per_month": 100000,
    "rate_limit_per_minute": 600,
    "features": ["advanced_analytics", "webhooks"],
    "enabled": true
  }'
```

---

## Quota Settings

### Quota Enforcement Modes

| Mode | Behavior |
|------|----------|
| `hard` | Block requests when quota exceeded |
| `soft` | Allow requests but flag as over-quota |
| `warn` | Allow requests, send warning notifications |

### Grace Percentage

Allow temporary overage before hard blocking:

```bash
apigate plans create \
  --name "Pro" \
  --monthly-quota 100000 \
  --quota-grace-percent 10 \
  --quota-enforcement hard
```

- At 100,000 requests: Warning sent
- At 110,000 requests: Hard block (10% grace exceeded)

---

## Rate Limit Settings

### Per-Minute Rate Limit

```bash
# 60 requests per minute
apigate plans create --rate-limit 60

# With burst allowance
apigate plans create \
  --rate-limit 60 \
  --rate-limit-burst 100
```

### Token Bucket Algorithm

APIGate uses token bucket rate limiting:
- Bucket fills at `rate_limit / 60` tokens per second
- Maximum bucket size = `rate_limit_burst`
- Each request consumes 1 token
- Empty bucket = 429 Too Many Requests

---

## Pricing Integration

### Stripe

```bash
# Create plan with Stripe price
apigate plans create \
  --name "Pro" \
  --price 4900 \
  --stripe-price-id "price_1234567890"
```

When a user subscribes:
1. Stripe checkout completed
2. Webhook received by APIGate
3. User automatically assigned to plan

### Paddle

```bash
apigate plans create \
  --name "Pro" \
  --price 4900 \
  --paddle-price-id "pri_abc123"
```

### LemonSqueezy

```bash
apigate plans create \
  --name "Pro" \
  --price 4900 \
  --lemonsqueezy-variant-id "var_xyz789"
```

---

## Feature Flags

Control access to specific features per plan:

```bash
apigate plans create \
  --name "Enterprise" \
  --features "webhooks,analytics,priority_support,custom_domains"
```

Check features in your upstream:

```go
// Request includes X-Plan-Features header
features := r.Header.Get("X-Plan-Features")
// "webhooks,analytics,priority_support,custom_domains"
```

---

## Plan Transitions

### Upgrade

```bash
# User upgrades from Free to Pro
apigate users update <user-id> --plan "Pro"
```

Changes take effect immediately:
- New rate limits active
- New quota starts fresh or prorated

### Downgrade

```bash
# User downgrades from Pro to Free
apigate users update <user-id> --plan "Free"
```

Quota handling on downgrade:
- If current usage > new quota: Warning sent
- User can continue until period ends

---

## Plan Management

### List Plans

```bash
# CLI
apigate plans list

# API
curl http://localhost:8080/admin/plans
```

### Update Plan

```bash
# CLI
apigate plans update <id> --rate-limit 120

# API
curl -X PUT http://localhost:8080/admin/plans/<id> \
  -H "Content-Type: application/json" \
  -d '{"rate_limit_per_minute": 120}'
```

**Note**: Changes affect all users on this plan immediately.

### Disable Plan

```bash
# Prevent new signups
apigate plans update <id> --enabled false
```

Existing users remain on disabled plans.

### Delete Plan

```bash
# Only if no users assigned
apigate plans delete <id>
```

---

## Default Plans

Create a default plan for new users:

```bash
apigate plans create \
  --name "Free" \
  --price 0 \
  --rate-limit 60 \
  --monthly-quota 1000 \
  --default true
```

New users without explicit plan assignment get the default plan.

---

## Best Practices

### 1. Start Simple

```bash
# Minimum viable plans
apigate plans create --name "Free" --price 0 --rate-limit 60 --monthly-quota 1000
apigate plans create --name "Pro" --price 2900 --rate-limit 300 --monthly-quota 50000
```

### 2. Clear Differentiation

Each plan should have obvious value increase:

| Tier | Price | Requests | Rate Limit | Value Prop |
|------|-------|----------|------------|------------|
| Free | $0 | 1K/mo | 60/min | Try the API |
| Starter | $29 | 10K/mo | 120/min | Small projects |
| Pro | $99 | 100K/mo | 600/min | Production use |
| Enterprise | Custom | Unlimited | Custom | Scale + support |

### 3. Grace Periods

Always set grace percentage:

```bash
# Give 20% buffer
apigate plans create \
  --monthly-quota 10000 \
  --quota-grace-percent 20
```

### 4. Monitor Usage Patterns

Use analytics to inform plan limits:
- P95 usage should fit comfortably in plan
- P99 should fit within grace period

---

## See Also

- [[Users]] - Assign users to plans
- [[Rate-Limiting]] - How rate limits work
- [[Quotas]] - Quota management
- [[Pricing-Integration]] - Payment setup
