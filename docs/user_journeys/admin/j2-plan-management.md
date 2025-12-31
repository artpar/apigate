# J2: Plan Management

> **Defining the revenue model - where business strategy meets product.**

---

## Business Context

### Why This Journey Matters

Plans are the **core monetization mechanism** of APIGate. How API Sellers structure their plans directly determines their revenue potential and customer acquisition strategy.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PLAN STRATEGY IMPACT                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Plan Design  ───▶  Customer Acquisition  ───▶  Revenue           │
│                                                                     │
│   Free Tier         Lowers barrier            Conversion funnel     │
│   Pro Tier          Captures serious users    Core revenue          │
│   Enterprise        High-value accounts       Expansion revenue     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact

| Plan Strategy | Customer Impact | Revenue Impact |
|---------------|-----------------|----------------|
| **Free tier only** | High signups, low revenue | Limited |
| **No free tier** | Fewer signups, qualified leads | Faster revenue |
| **Freemium** | High signups, conversion funnel | Balanced growth |
| **Usage-based** | Pay-as-you-go flexibility | Scales with usage |

### Business Success Criteria

- [ ] API Seller can create a free tier in < 1 minute
- [ ] API Seller can create a paid tier with Stripe in < 2 minutes
- [ ] Plan changes take effect immediately for new users
- [ ] Existing users can be migrated between plans
- [ ] Revenue projections visible in dashboard

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Seller (completed setup, exploring monetization) |
| **Prior Action** | Completed J1 Setup, exploring admin dashboard |
| **Mental State** | Strategic thinking, planning business model |
| **Expectation** | "I need to figure out my pricing strategy" |

### What Triggered This Journey?

- Completed initial setup, now planning monetization
- Preparing to launch to first customers
- Adjusting pricing based on market feedback
- Adding new tier for enterprise customers

### User Goals

1. **Primary:** Create plans that match their business model
2. **Secondary:** Set appropriate limits that protect their backend
3. **Tertiary:** Configure payment integration for paid plans

### User Questions at This Stage

- "What pricing model should I use?"
- "How do I set up a free trial?"
- "What happens when users exceed limits?"
- "Can I change plans after users sign up?"

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Plans  │───▶│ Create  │───▶│Configure│───▶│ Manage  │
│  List   │    │  Plan   │    │ Details │    │ Active  │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     │              │              │              │
     │              │              │              │
     ▼              ▼              ▼              ▼
  View all     Define tier    Set limits    Edit/disable
```

### Step 1: View Plans List

**URL:** `/ui/plan`

**Purpose:** See all existing plans and their status.

#### UI Elements

| Element | Type | Description |
|---------|------|-------------|
| Plans table | Data table | All plans with key metrics |
| Create button | Primary button | Opens plan creation form |
| Search/filter | Input | Filter plans by name/status |

#### Table Columns

| Column | Description | Sortable |
|--------|-------------|----------|
| Name | Plan identifier | Yes |
| Price | Monthly price in dollars | Yes |
| Requests/Month | Quota limit | Yes |
| Rate Limit | Requests per minute | Yes |
| Users | Number of users on plan | Yes |
| Status | Enabled/Disabled | Yes |
| Default | Default plan badge | No |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Plans list | Page load | `j2-plans/01-plans-list.png` |
| Empty state | No plans | `j2-plans/01-plans-empty.png` |
| With users | Plans have subscribers | `j2-plans/01-plans-with-users.png` |

---

### Step 2: Create New Plan

**URL:** `/ui/plan/new`

**Purpose:** Define a new pricing tier.

#### UI Elements

| Element | Type | Validation | Notes |
|---------|------|------------|-------|
| Name | Text input | Required, unique | e.g., "Free", "Pro", "Enterprise" |
| Description | Textarea | Optional | Shown in customer portal |
| Price (Monthly) | Number input | >= 0 | In dollars |
| Requests/Month | Number input | > 0 | 0 = unlimited |
| Rate Limit/Minute | Number input | > 0 | Requests per minute |
| Trial Days | Number input | >= 0 | Free trial period |
| Stripe Price ID | Text input | Optional | Required for payment |
| Enabled | Toggle | - | Active or draft |
| Is Default | Toggle | - | Auto-assign to new users |

#### Form Sections

```
┌─────────────────────────────────────────────────────────────┐
│ Basic Information                                           │
├─────────────────────────────────────────────────────────────┤
│ Name: [________________]                                    │
│ Description: [__________________________________________]   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Pricing                                                     │
├─────────────────────────────────────────────────────────────┤
│ Monthly Price: $[____]     Trial Days: [____]              │
│ Stripe Price ID: [________________] (for paid plans)        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Limits                                                      │
├─────────────────────────────────────────────────────────────┤
│ Requests per Month: [________]                              │
│ Rate Limit (per minute): [____]                             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Settings                                                    │
├─────────────────────────────────────────────────────────────┤
│ [✓] Enabled     [ ] Set as Default Plan                    │
└─────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Empty form | Page load | `j2-plans/02-create-empty.png` |
| Free tier | Sample free plan | `j2-plans/02-create-free.png` |
| Pro tier | Sample paid plan | `j2-plans/02-create-pro.png` |
| Enterprise | High-value plan | `j2-plans/02-create-enterprise.png` |
| Validation errors | Submit invalid | `j2-plans/02-create-errors.png` |

---

### Step 3: Configure Plan Details

**Purpose:** Fine-tune plan configuration.

#### Plan Templates

**Free Tier (Recommended First Plan)**
```
Name: Free
Price: $0
Requests/Month: 1,000
Rate Limit: 10/min
Trial Days: 0
Default: Yes
```

**Starter Tier**
```
Name: Starter
Price: $9
Requests/Month: 10,000
Rate Limit: 60/min
Trial Days: 7
```

**Pro Tier**
```
Name: Pro
Price: $29
Requests/Month: 100,000
Rate Limit: 600/min
Trial Days: 14
```

**Enterprise Tier**
```
Name: Enterprise
Price: $99
Requests/Month: 1,000,000
Rate Limit: 6000/min
Trial Days: 30
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Template selection | Choosing template | `j2-plans/03-templates.png` |
| Custom configuration | Manual entry | `j2-plans/03-custom.png` |

---

### Step 4: Manage Active Plans

**Purpose:** Edit, enable/disable, and manage existing plans.

#### Actions Available

| Action | Description | Impact |
|--------|-------------|--------|
| **Edit** | Modify plan details | Changes apply to new users |
| **Disable** | Hide from signup | Existing users keep access |
| **Enable** | Make available | Shows in customer portal |
| **Set Default** | Auto-assign to signups | One default at a time |
| **Delete** | Remove plan | Only if no users assigned |

#### Editing Existing Plans

| Field | Can Change | Impact on Existing Users |
|-------|------------|--------------------------|
| Name | Yes | Display only |
| Description | Yes | Display only |
| Price | Yes | Only new subscriptions |
| Limits | Yes | Immediate for all users |
| Enabled | Yes | Hides from portal |
| Default | Yes | Only affects new signups |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Edit form | Click edit | `j2-plans/04-edit.png` |
| Disable confirmation | Click disable | `j2-plans/04-disable-confirm.png` |
| Set default | Click set default | `j2-plans/04-set-default.png` |
| Delete blocked | Has users | `j2-plans/04-delete-blocked.png` |

---

## UX Analysis

### Cognitive Load Assessment

| Task | Decisions Required | Cognitive Load |
|------|-------------------|----------------|
| View plans | 0 | Very Low |
| Create free plan | 4 (name, quota, rate, default) | Low-Medium |
| Create paid plan | 6 (+ price, stripe ID, trial) | Medium |
| Edit existing | Variable | Low |

### Friction Analysis

```
Friction Score: 1 (Low) to 5 (High)

View Plans        █░░░░░░░░░ 1/5
Create Free Plan  ███░░░░░░░ 3/5  (what values to choose?)
Create Paid Plan  ████░░░░░░ 4/5  (Stripe integration adds complexity)
Edit Plan         ██░░░░░░░░ 2/5

Main Friction Point: Stripe Price ID requirement for paid plans
Mitigation: Link to Stripe docs, inline help
```

### Guidance Needed

1. **What limits should I set?**
   - Provide industry benchmarks
   - Show calculator: "At $29/mo with 100K requests, you earn $0.00029/request"

2. **How does Stripe integration work?**
   - Step-by-step inline guide
   - Link to create Stripe product/price

3. **What happens at limit?**
   - Clear explanation: "Requests return 429/503 error"
   - Option for overage billing (future feature)

---

## Emotional Map

```
                     Emotional State During Plan Management

Delight  ─┐                              ┌─────────────● Success!
          │                            ╱
          │                          ╱
Neutral  ─┼────●─────────────●─────●
          │     ╲           ╱
          │       ╲       ╱
Anxiety  ─┴─────────●───●
          │
          └────┬─────────┬─────────┬─────────┬─────────
            View List  Create   Configure  Live!
```

### Emotional Journey

| Stage | Emotion | Trigger | Design Response |
|-------|---------|---------|-----------------|
| **View List** | Neutral | Orienting | Clear, organized table |
| **Start Create** | Slight anxiety | "What values?" | Templates and defaults |
| **Pricing Decision** | Uncertainty | Business decision | Guidance, examples |
| **Stripe Config** | Frustration possible | External system | Clear instructions |
| **Plan Live** | Accomplishment | Successfully created | Success message |

---

## Metrics & KPIs

### Primary Metrics

| Metric | Definition | Target |
|--------|------------|--------|
| **Plans per Seller** | Average plans created | 3+ |
| **Paid Plan Rate** | Sellers with paid plans | > 60% |
| **Time to First Paid Plan** | From setup to paid plan | < 1 day |

### Analytics Events

```javascript
// Plan creation
analytics.track('plan_created', {
  plan_type: 'paid', // 'free' or 'paid'
  price_cents: 2900,
  has_trial: true,
  trial_days: 14
});

// Plan modified
analytics.track('plan_modified', {
  plan_id: 'xxx',
  fields_changed: ['price', 'rate_limit']
});
```

---

## Edge Cases & Errors

### Validation Errors

| Field | Error | Message |
|-------|-------|---------|
| Name | Duplicate | "A plan with this name already exists" |
| Price | Negative | "Price cannot be negative" |
| Stripe ID | Invalid | "Invalid Stripe Price ID format" |
| Delete | Has users | "Cannot delete plan with active users" |

### Business Logic Errors

| Scenario | Prevention |
|----------|------------|
| Disable only plan | Require at least one enabled |
| Delete default plan | Must set new default first |
| Price change for subscribed | Only affects new subscribers |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j2-plan-management
requires_auth: admin
viewport: 1280x720

steps:
  - name: plans-list
    url: /ui/plan
    wait: networkidle

  - name: create-empty
    url: /ui/plan/new
    wait: networkidle

  - name: create-free
    url: /ui/plan/new
    actions:
      - fill: input[name="name"]
        value: "Free"
      - fill: input[name="requests_per_month"]
        value: "1000"
      - fill: input[name="rate_limit_per_minute"]
        value: "10"

  - name: create-pro
    url: /ui/plan/new
    actions:
      - fill: input[name="name"]
        value: "Pro"
      - fill: input[name="price_monthly"]
        value: "2900"
      - fill: input[name="requests_per_month"]
        value: "100000"
```

### GIF Sequence

**j2-create-plan.gif**
- Frame 1: Empty form (1s)
- Frame 2: Name entered (1s)
- Frame 3: Limits configured (2s)
- Frame 4: Submit (1s)
- Frame 5: Success in list (2s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J1: Setup](j1-first-time-setup.md) | Creates first plan |
| [J4: Configuration](j4-platform-config.md) | Stripe setup required |
| [J8: Upgrade](../customer/j8-plan-upgrade.md) | Customer sees these plans |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
