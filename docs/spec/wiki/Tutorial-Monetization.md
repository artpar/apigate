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
  --name "Free" \
  --description "Perfect for testing and small projects" \
  --price 0 \
  --rate-limit 60 \
  --monthly-quota 1000 \
  --default true
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
  --name "Starter" \
  --description "For growing projects" \
  --price 2900 \
  --rate-limit 300 \
  --monthly-quota 25000
```

### Pro Plan

For production applications:

```bash
apigate plans create \
  --name "Pro" \
  --description "For production workloads" \
  --price 9900 \
  --rate-limit 1000 \
  --monthly-quota 100000 \
  --features "priority_support,webhooks,analytics"
```

### Enterprise Plan

For high-volume customers:

```bash
apigate plans create \
  --name "Enterprise" \
  --description "Custom limits and dedicated support" \
  --price 0 \
  --rate-limit 10000 \
  --monthly-quota 0 \
  --features "priority_support,webhooks,analytics,dedicated_account,sla"
```

Note: Enterprise is $0 because pricing is custom.

---

## Step 3: Configure Quota Enforcement

Decide what happens when users exceed quota:

### Hard Limit (Recommended for Free)

Block requests when quota exceeded:

```bash
apigate plans update Free \
  --quota-enforcement hard \
  --quota-grace-percent 0
```

### Soft Limit (Good for Paid Plans)

Allow overage, potentially charge later:

```bash
apigate plans update Starter \
  --quota-enforcement soft \
  --quota-grace-percent 20
```

### Warnings Only (Premium Plans)

```bash
apigate plans update Pro \
  --quota-enforcement warn \
  --quota-grace-percent 50
```

---

## Step 4: Set Up Quota Warnings

Alert users before they hit limits:

```bash
apigate settings set quota_warning_thresholds "50,80,95,100"
apigate settings set quota_notification_email true
```

Users will receive emails at:
- 50% - Informational
- 80% - Warning
- 95% - Critical warning
- 100% - Quota reached

---

## Step 5: Enable Customer Portal

Let customers sign up and manage their subscriptions:

```bash
# Enable portal
apigate settings set portal_enabled true
apigate settings set portal_registration_enabled true

# Branding
apigate settings set portal_company_name "Your API Company"
apigate settings set portal_logo_url "https://yoursite.com/logo.png"
```

Now customers can:
1. Register at `/portal/register`
2. View their usage
3. Manage API keys
4. Upgrade/downgrade plans

---

## Step 6: Add Payment Integration (Optional)

### Stripe Integration

```bash
# Configure Stripe
apigate settings set payment_provider stripe
apigate settings set stripe_secret_key "sk_live_xxx"
apigate settings set stripe_webhook_secret "whsec_xxx"
```

Create Stripe products and link to plans:

```bash
# Link Stripe price to plan
apigate plans update Starter --stripe-price-id "price_xxx"
apigate plans update Pro --stripe-price-id "price_yyy"
```

Now customers can subscribe with a credit card through the portal.

### Without Payment Integration

You can still monetize manually:
1. Customer contacts you
2. You create invoice externally
3. Manually assign them to paid plan

```bash
apigate users update customer@example.com --plan Pro
```

---

## Step 7: Create Value Tiers

Differentiate plans with features:

### API Access Levels

```bash
# Free: Read-only access
apigate routes update users-api --required-features ""
apigate routes update users-write --required-features "write_access"

# Create route that requires paid plan
apigate routes create \
  --name "analytics-api" \
  --path "/api/analytics/*" \
  --required-features "analytics"
```

### Rate Limit Tiers

Higher plans get more capacity:

| Plan | Requests/min | Burst |
|------|--------------|-------|
| Free | 60 | 60 |
| Starter | 300 | 500 |
| Pro | 1000 | 2000 |

### Support Tiers

Document in your pricing page:
- Free: Community support only
- Starter: Email support (48h response)
- Pro: Priority email (24h response)
- Enterprise: Dedicated Slack channel

---

## Step 8: Set Up Analytics

Track your business metrics:

```bash
# View revenue by plan
apigate analytics revenue

# View plan distribution
apigate analytics plans

# View upgrade funnel
apigate analytics upgrades
```

Key metrics to track:
- **MRR** (Monthly Recurring Revenue)
- **Conversion Rate** (Free → Paid)
- **Churn Rate** (Cancellations)
- **ARPU** (Average Revenue Per User)

---

## Step 9: Create Pricing Page Content

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
- Advanced analytics
- Webhooks
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

## Step 10: Monitor and Optimize

### Weekly Review

```bash
# Check plan performance
apigate analytics weekly

# Users approaching quota
apigate users list --quota-percent-gt 80

# Recent upgrades
apigate analytics upgrades --days 7
```

### Identify Optimization Opportunities

```bash
# High-usage free users (potential upgrades)
apigate users list --plan Free --quota-percent-gt 50

# Low-usage paid users (churn risk)
apigate users list --plan Starter --quota-percent-lt 10
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
3. ✅ Configure quota enforcement
4. ✅ Set up usage warnings
5. ✅ Enable customer self-service
6. ✅ (Optional) Add payment integration
7. ✅ Differentiate plan features
8. ✅ Set up business analytics

Your API is now ready to generate revenue!
