# Tutorial: API Monetization

Turn your API into a revenue stream with tiered pricing plans.

---

## Overview

In this tutorial, you'll:
1. Create a pricing strategy
2. Set up multiple plans
3. Configure usage-based limits
4. Enable customer self-service
5. Track usage and revenue

---

## Prerequisites

- APIGate running with basic setup complete
- (Optional) Stripe account for payments

---

## Step 1: Define Your Pricing Strategy

Before creating plans, decide your pricing model:

### Option A: Flat Rate Plans

Fixed monthly price, fixed limits:

| Plan | Price | Rate Limit | Monthly Quota |
|------|-------|------------|---------------|
| Free | $0 | 60/min | 1,000 |
| Starter | $29 | 300/min | 25,000 |
| Pro | $99 | 1,000/min | 100,000 |
| Enterprise | Custom | Custom | Unlimited |

### Option B: Usage-Based

Pay per request after free tier:

| Tier | Included | Overage |
|------|----------|---------|
| Free | 1,000 | Blocked |
| Pay-as-you-go | 0 | $0.001/req |
| Volume | 100,000 | $0.0005/req |

### Option C: Hybrid

Base subscription + usage overage:

| Plan | Base | Included | Overage |
|------|------|----------|---------|
| Starter | $29 | 50,000 | $0.001/req |
| Pro | $99 | 250,000 | $0.0005/req |

For this tutorial, we'll use **Option A** (Flat Rate).

---

## Step 2: Create Your Plans

### Free Tier

The free tier lets users try your API:

```bash
apigate plans create \
  --id free \
  --name "Free" \
  --description "Perfect for testing and small projects" \
  --price 0 \
  --rate-limit 60 \
  --requests 1000 \
  --default
```

Or via Admin UI:
1. Go to **Plans** → **Add Plan**
2. Fill in details
3. Check **Default Plan**
4. Save

### Starter Plan

For small businesses and side projects:

```bash
apigate plans create \
  --id starter \
  --name "Starter" \
  --description "For growing projects" \
  --price 2900 \
  --rate-limit 300 \
  --requests 25000
```

### Pro Plan

For production applications:

```bash
apigate plans create \
  --id pro \
  --name "Pro" \
  --description "For production workloads" \
  --price 9900 \
  --rate-limit 1000 \
  --requests 100000
```

### Enterprise Plan

For high-volume customers:

```bash
apigate plans create \
  --id enterprise \
  --name "Enterprise" \
  --description "Custom limits and dedicated support" \
  --price 0 \
  --rate-limit 10000 \
  --requests -1
```

Note: Enterprise is $0 because pricing is custom. Use `--requests -1` for unlimited.

---

## Step 3: Quota Behavior

By default, when users exceed their monthly quota:
- Requests are rejected with HTTP 429 status
- Rate limit headers show the quota status

For paid plans with overage billing, configure the `--overage` flag when creating plans:

```bash
# Starter plan with overage at $0.001 per request
apigate plans create \
  --id starter \
  --name "Starter" \
  --price 2900 \
  --rate-limit 300 \
  --requests 25000 \
  --overage 1
```

---

## Step 4: Enable Customer Portal

Let customers sign up and manage their subscriptions:

```bash
# Enable portal
apigate settings set portal.enabled true

# Branding
apigate settings set portal.app_name "Your API Company"
apigate settings set custom.logo_url "https://yoursite.com/logo.png"
```

Now customers can:
1. Register at `/portal/register`
2. View their usage
3. Manage API keys
4. Upgrade/downgrade plans

---

## Step 5: Add Payment Integration (Optional)

### Stripe Integration

```bash
# Configure Stripe
apigate settings set payment.provider stripe
apigate settings set payment.stripe.secret_key "sk_live_xxx" --encrypted
apigate settings set payment.stripe.webhook_secret "whsec_xxx" --encrypted
```

Link Stripe prices to plans via the Admin UI:
1. Go to **Plans** → Edit plan
2. Enter the Stripe Price ID
3. Save

Now customers can subscribe with a credit card through the portal.

### Without Payment Integration

You can still monetize manually:
1. Customer contacts you
2. You create invoice externally
3. Manually assign them to paid plan via Admin UI

---

## Step 6: Create Value Tiers

Differentiate plans with rate limits:

### Rate Limit Tiers

Each plan has its own rate limit configured via `--rate-limit`:

| Plan | Requests/min | Monthly Quota |
|------|--------------|---------------|
| Free | 60 | 1,000 |
| Starter | 300 | 25,000 |
| Pro | 1000 | 100,000 |

### Support Tiers

Document in your pricing page:
- Free: Community support only
- Starter: Email support (48h response)
- Pro: Priority email (24h response)
- Enterprise: Dedicated Slack channel

---

## Step 7: Monitor Usage

Track usage via the Admin UI:

1. Go to **Analytics** in the sidebar
2. View overall API usage
3. Click on individual users to see their usage

Or use the CLI:

```bash
# View user usage summary
apigate usage summary --user user@example.com

# View usage history
apigate usage history --user user@example.com --periods 6
```

Key metrics to track:
- **MRR** (Monthly Recurring Revenue)
- **Conversion Rate** (Free → Paid)
- **Churn Rate** (Cancellations)
- **ARPU** (Average Revenue Per User)

---

## Step 8: Create Pricing Page Content

Add to your website:

```markdown
# API Pricing

## Free
$0/month
- 1,000 requests/month
- 60 requests/minute
- Community support
[Sign Up Free]

## Starter
$29/month
- 25,000 requests/month
- 300 requests/minute
- Email support
[Start Trial]

## Pro
$99/month
- 100,000 requests/month
- 1,000 requests/minute
- Priority support
[Get Started]

## Enterprise
Custom pricing
- Unlimited requests
- Custom rate limits
- Dedicated support
- SLA guarantee
[Contact Sales]
```

---

## Step 9: Monitor and Optimize

### Weekly Review

Use the Admin UI Analytics page to:
- View usage trends
- Identify high-usage users
- Track plan distribution

### CLI Usage Reports

```bash
# Check individual user usage
apigate usage summary --user user@example.com

# View recent requests
apigate usage recent --user user@example.com --limit 50
```

---

## Complete Plan Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                     Your Pricing Tiers                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐     │
│  │   Free   │   │ Starter  │   │   Pro    │   │Enterprise│     │
│  │   $0     │   │   $29    │   │   $99    │   │  Custom  │     │
│  ├──────────┤   ├──────────┤   ├──────────┤   ├──────────┤     │
│  │ 1K/mo    │   │ 25K/mo   │   │ 100K/mo  │   │Unlimited │     │
│  │ 60/min   │   │ 300/min  │   │ 1000/min │   │ Custom   │     │
│  │ Hard cap │   │ Soft cap │   │ Warn     │   │ Flexible │     │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘     │
│       │              │              │              │            │
│       └──────────────┴──────────────┴──────────────┘            │
│                    Upgrade Path                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Next Steps

1. **[[Tutorial-Stripe]]** - Full Stripe integration guide
2. **[[Customer-Portal]]** - Customize the portal
3. **[[Webhooks]]** - Notify on subscription changes
4. **[[Analytics]]** - Deep dive into metrics

---

## Summary

You've learned how to:

1. ✅ Design a pricing strategy
2. ✅ Create multiple pricing plans
3. ✅ Understand quota behavior
4. ✅ Enable customer self-service
5. ✅ (Optional) Add payment integration
6. ✅ Differentiate plan tiers
7. ✅ Monitor usage

Your API is now ready to generate revenue!
