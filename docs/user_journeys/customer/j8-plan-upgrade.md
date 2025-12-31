# J8: Plan Upgrade

> **The monetization moment - converting free users to paying customers.**

---

## Business Context

### Why This Journey Matters

Plan upgrades are **the primary revenue event** for API Sellers. This journey directly determines their ability to monetize their API. Every friction point here costs real money.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    UPGRADE CONVERSION FUNNEL                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   View Plans ──▶ Select Plan ──▶ Checkout ──▶ Payment ──▶ Upgraded │
│      100%          60%           45%          35%         32%       │
│                                                                     │
│   Target: 50%+ of users who view plans complete an upgrade         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact (for API Seller)

| Stage | Improvement | Revenue Impact |
|-------|-------------|----------------|
| View → Select | +10% | More qualified leads |
| Select → Checkout | +10% | Higher intent |
| Checkout → Complete | +10% | Direct revenue |

### Business Success Criteria

- [ ] Plans comparison is clear and compelling
- [ ] Checkout completes in < 2 minutes
- [ ] Payment succeeds on first attempt > 95%
- [ ] New limits apply immediately
- [ ] Receipt email sent automatically

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Buyer (active user ready to pay) |
| **Prior Action** | Using API, approaching limits, or needs features |
| **Mental State** | Evaluating value, making purchase decision |
| **Expectation** | "I need more capacity, is it worth the price?" |

### What Triggered This Journey?

- Approaching or exceeded quota
- Need higher rate limits
- Requires features only in paid plans
- Planning production deployment
- Business growth requires more capacity

### User Goals

1. **Primary:** Get more API capacity
2. **Secondary:** Understand what they're paying for
3. **Tertiary:** Complete payment quickly and securely

### User Questions at This Stage

- "Is this worth the price?"
- "What exactly do I get?"
- "Can I downgrade later?"
- "Is my payment secure?"

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Trigger│───▶│  Plans  │───▶│ Select  │───▶│Checkout │───▶│ Success │
│         │    │ Compare │    │  Plan   │    │(Stripe) │    │         │
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

---

### Step 1: Upgrade Trigger

**Sources:**
- Usage warning banner
- Dashboard "Upgrade" button
- Navigation menu "Plans"
- Usage exceeded page
- Email notification

#### Entry Points

| Source | Context | Intent Level |
|--------|---------|--------------|
| Dashboard button | Browsing | Low-Medium |
| 80% warning | Approaching limit | Medium |
| 95% warning | Critical | High |
| Exceeded page | Blocked | Very High |
| Email | External | Medium |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Dashboard CTA | Dashboard view | `j8-upgrade/01-dashboard-cta.png` |
| Warning CTA | Usage warning | `j8-upgrade/01-warning-cta.png` |
| Exceeded CTA | Quota exceeded | `j8-upgrade/01-exceeded-cta.png` |

---

### Step 2: Plans Comparison

**URL:** `/portal/plans`

**Purpose:** Help user choose the right plan.

#### Plans Page Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  Choose Your Plan                                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │     Free      │  │   ⭐ Pro      │  │  Enterprise   │           │
│  │   Current     │  │  Popular      │  │               │           │
│  │               │  │               │  │               │           │
│  │    $0/mo      │  │   $29/mo      │  │   $99/mo      │           │
│  │               │  │               │  │               │           │
│  │ ────────────  │  │ ────────────  │  │ ────────────  │           │
│  │               │  │               │  │               │           │
│  │ 1,000 req/mo  │  │ 100,000 req   │  │ 1,000,000 req │           │
│  │ 10 req/min    │  │ 600 req/min   │  │ 6,000 req/min │           │
│  │ Basic support │  │ Email support │  │ Priority supp │           │
│  │               │  │ 14-day trial  │  │ 30-day trial  │           │
│  │               │  │               │  │               │           │
│  │ [Current]     │  │ [Upgrade]     │  │ [Upgrade]     │           │
│  │               │  │               │  │               │           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
│                                                                     │
│  All plans include:                                                 │
│  ✓ Unlimited API keys  ✓ Real-time usage  ✓ SSL encryption         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Plan Comparison Elements

| Element | Purpose |
|---------|---------|
| Price | Monthly cost |
| Request quota | Monthly limit |
| Rate limit | Per-minute limit |
| Features | What's included |
| Trial | Free trial period |
| CTA | Upgrade button |

#### Pricing Psychology

- **Anchor high** - Show enterprise first (or last for contrast)
- **Highlight recommended** - Star/badge on best value
- **Show current** - Mark user's current plan
- **Contrast savings** - Per-request cost reduction

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Plans comparison | Page load | `j8-upgrade/02-plans-comparison.png` |
| Plan highlighted | Hover on plan | `j8-upgrade/02-plan-hover.png` |
| Mobile view | Responsive | `j8-upgrade/02-plans-mobile.png` |

---

### Step 3: Select Plan

**Action:** Click "Upgrade" on chosen plan

**Purpose:** Confirm selection and proceed to payment.

#### Confirmation Modal (Optional)

```
┌─────────────────────────────────────────────────────────────────────┐
│  Upgrade to Pro                                           [×]       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  You're upgrading from Free to Pro                                  │
│                                                                     │
│  What you'll get:                                                   │
│  • 100,000 requests/month (was 1,000)                              │
│  • 600 requests/minute (was 10)                                    │
│  • Email support                                                    │
│  • 14-day free trial                                               │
│                                                                     │
│  Price: $29/month                                                   │
│  First charge: After 14-day trial                                   │
│                                                                     │
│  [Cancel]                              [Proceed to Payment]         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Confirmation | Click upgrade | `j8-upgrade/03-confirmation.png` |

---

### Step 4: Payment (Stripe Checkout)

**URL:** Redirect to Stripe Checkout or embedded form

**Purpose:** Collect payment information securely.

#### Stripe Checkout Experience

Stripe Checkout provides:
- Secure payment form (PCI compliant)
- Multiple payment methods (cards, wallets)
- Address collection if needed
- Receipt handling

#### Checkout Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│  stripe                                          Powered by Stripe  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [API Name] - Pro Plan                                              │
│  $29.00 per month after 14-day trial                               │
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  Email                                                              │
│  [john@example.com________________________________]                 │
│                                                                     │
│  Card information                                                   │
│  [4242 4242 4242 4242]  [12/28]  [123]                             │
│                                                                     │
│  Name on card                                                       │
│  [John Developer_______________________________]                    │
│                                                                     │
│  Country                                                            │
│  [United States ▼]                                                  │
│                                                                     │
│  [Start trial - first charge Feb 1]                                │
│                                                                     │
│  By confirming, you agree to our Terms of Service.                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Stripe checkout | Redirect | `j8-upgrade/04-stripe-checkout.png` |
| Payment form filled | Enter details | `j8-upgrade/04-payment-filled.png` |

---

### Step 5: Success

**URL:** `/portal/upgrade/success` or `/portal/dashboard`

**Purpose:** Confirm upgrade and show new benefits.

#### Success Page

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                    🎉 Welcome to Pro!                               │
│                                                                     │
│  Your upgrade is complete. Your new limits are active now.         │
│                                                                     │
│  What's changed:                                                    │
│  ─────────────────                                                  │
│  • Monthly requests: 1,000 → 100,000                               │
│  • Rate limit: 10/min → 600/min                                    │
│  • Support: Basic → Email                                          │
│                                                                     │
│  Your 14-day trial has started. First charge: Feb 1, 2024          │
│                                                                     │
│  A receipt has been sent to john@example.com                        │
│                                                                     │
│                    [Go to Dashboard]                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Post-Upgrade Actions

1. Update user's plan in database
2. Apply new rate limits immediately
3. Reset quota counter (or prorate)
4. Send receipt email
5. Trigger webhook (for API Seller)

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Success page | Payment complete | `j8-upgrade/05-success.png` |
| Updated dashboard | After redirect | `j8-upgrade/05-dashboard-upgraded.png` |

---

## UX Analysis

### Decision Support

The plans page must answer:
1. **What do I need?** - Clear feature comparison
2. **What's the value?** - Price per request calculation
3. **What if it's wrong?** - Downgrade/cancel policy
4. **Is it safe?** - Security badges, trusted payment

### Friction Points

| Step | Friction | Mitigation |
|------|----------|------------|
| Plan comparison | Too many options | Recommend best value |
| Checkout redirect | Context switch | Embedded checkout option |
| Payment form | Many fields | Wallet payment (Apple/Google) |
| Trial confusion | When charged? | Clear first charge date |

### Trust Signals

- Stripe branding (known secure)
- SSL indicator
- Money-back guarantee (if offered)
- Customer testimonials (optional)

### Mobile Experience

| Aspect | Consideration |
|--------|---------------|
| Plan cards | Stack vertically |
| Comparison | Swipe between plans |
| Checkout | Mobile-optimized |
| Wallet payments | Apple Pay, Google Pay |

---

## Emotional Map

```
                     Emotional State During Upgrade

Delight  ─┐                                              ┌─ ● Success!
          │                                            ╱
          │                                          ╱
Neutral  ─┼────●───────────────────────────────────●
          │      ╲                               ╱
          │        ╲         ●─────●───────────●
Anxiety  ─┴──────────●─────╱       Payment
          │              View     commit
          │             plans
          └────┬─────────┬─────────┬─────────┬─────────
            Trigger   Compare   Checkout   Done!
```

### Emotional Triggers

| Stage | Emotion | Design Response |
|-------|---------|-----------------|
| Trigger | Urgency/Need | Clear path to solution |
| Compare | Evaluation | Easy comparison |
| Checkout | Commitment anxiety | Trust signals |
| Payment | Risk feeling | Secure indicators |
| Success | Relief, excitement | Celebration, confirmation |

---

## Metrics & KPIs

### Conversion Metrics

| Metric | Definition | Target |
|--------|------------|--------|
| **View → Select** | Click on upgrade | > 40% |
| **Select → Checkout** | Start checkout | > 70% |
| **Checkout → Complete** | Finish payment | > 80% |
| **End-to-end** | View to complete | > 25% |

### Revenue Metrics

| Metric | Definition |
|--------|------------|
| **Upgrade rate** | % of free users who upgrade |
| **Time to upgrade** | Days from signup to paid |
| **Average plan value** | Revenue per upgrade |
| **Trial conversion** | Trial → Paid |

### Analytics Events

```javascript
// Plans viewed
analytics.track('plans_page_viewed', {
  current_plan: 'free',
  trigger: 'usage_warning' // or 'menu', 'dashboard'
});

// Plan selected
analytics.track('plan_selected', {
  plan: 'pro',
  price: 2900
});

// Checkout started
analytics.track('checkout_started', {
  plan: 'pro',
  has_trial: true
});

// Upgrade completed
analytics.track('upgrade_completed', {
  plan: 'pro',
  price: 2900,
  trial_days: 14,
  time_to_upgrade_seconds: 180
});
```

---

## Edge Cases & Errors

### Payment Errors

| Error | Cause | User Message |
|-------|-------|--------------|
| Card declined | Insufficient funds | "Card was declined. Try another card." |
| Invalid card | Wrong number | "Card number is invalid" |
| Expired card | Past expiry | "Card has expired" |
| 3D Secure failed | Auth failed | "Authentication failed" |

### System Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| Stripe timeout | Network issue | Retry button |
| Webhook failed | System error | Manual verification |
| Plan not found | Config error | Contact support |

### Edge Cases

| Scenario | Handling |
|----------|----------|
| Already on plan | Show "Current" badge |
| Downgrade | Different flow (not covered here) |
| Multiple upgrades quickly | Prevent duplicate charges |
| Tab closed during checkout | Session recovery |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j8-plan-upgrade
requires_auth: customer
viewport: 1280x720

steps:
  - name: dashboard-cta
    url: /portal/
    wait: networkidle

  - name: plans-comparison
    url: /portal/plans
    wait: networkidle

  - name: confirmation
    url: /portal/plans
    actions:
      - click: button:has-text("Upgrade"):nth(0)
      - wait: dialog

  - name: success
    url: /portal/upgrade/success
    notes: Mock successful payment callback
```

### GIF Sequence

**j8-upgrade.gif**
- Frame 1: Dashboard with upgrade CTA (1s)
- Frame 2: Plans comparison page (2s)
- Frame 3: Click Upgrade (1s)
- Frame 4: Confirmation modal (1s)
- Frame 5: Stripe checkout (2s)
- Frame 6: Success page (2s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J7: Usage](j7-usage-monitoring.md) | Trigger for upgrade |
| [J2: Plans](../admin/j2-plan-management.md) | Admin creates plans |
| [J4: Config](../admin/j4-platform-config.md) | Payment setup |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
