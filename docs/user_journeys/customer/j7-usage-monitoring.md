# J7: Usage Monitoring

> **Awareness drives action - showing usage before it becomes a problem.**

---

## Business Context

### Why This Journey Matters

Usage monitoring creates **natural upgrade triggers**. When customers see they're approaching limits, they're primed to consider paid plans. It's also critical for trust - customers hate surprise limit hits.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    USAGE-DRIVEN UPGRADE FUNNEL                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Active User ──▶ 50% Usage ──▶ 80% Warning ──▶ Consider Upgrade   │
│                                                                     │
│   "I'm using this"  "Getting value"  "Need more"   "Worth paying"  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact (for API Seller)

| Usage Stage | Action | Revenue Potential |
|-------------|--------|-------------------|
| 0-50% | Happy usage | Engagement |
| 50-80% | Soft prompt | Pre-qualified lead |
| 80-95% | Strong prompt | High intent |
| 100% | Block/warn | Convert or churn |

### Business Success Criteria

- [ ] Usage updates within 1 minute of API calls
- [ ] Clear visual progress indicators
- [ ] Proactive warnings at 80% and 95%
- [ ] Easy path to upgrade from usage page
- [ ] Historical usage data available

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Buyer (active user checking on usage) |
| **Prior Action** | Has been using the API |
| **Mental State** | Monitoring, planning, possibly concerned |
| **Expectation** | "How much have I used? Am I close to limits?" |

### What Triggered This Journey?

- Periodic check on usage
- Received usage warning email
- Planning capacity needs
- Hit rate limit, checking status
- Preparing to upgrade

### User Goals

1. **Primary:** Understand current usage vs limits
2. **Secondary:** Anticipate when they'll hit limits
3. **Tertiary:** Make informed upgrade decisions

---

## The Journey

### Overview

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│Dashboard │───▶│  Usage   │───▶│ Warning  │───▶│ Decision │
│  Glance  │    │  Detail  │    │  State   │    │ Upgrade? │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
```

---

### Step 1: Usage at a Glance (Dashboard)

**URL:** `/portal/`

**Purpose:** Quick visibility without deep dive.

#### Dashboard Usage Card

```
┌─────────────────────────────────────────────────────────────────────┐
│  Your Plan: Free                                                    │
│                                                                     │
│  Monthly Requests                                                   │
│  ████████████████████░░░░░░░░░░░░░░░░░░░░  450 / 1,000 (45%)       │
│                                                                     │
│  Rate Limit: 10 requests/minute                                     │
│  Resets: Feb 1, 2024                                               │
│                                                                     │
│  [View Usage Details]                          [Upgrade Plan]       │
└─────────────────────────────────────────────────────────────────────┘
```

#### Progress Bar Colors

| Usage | Color | Message |
|-------|-------|---------|
| 0-70% | Green | On track |
| 70-90% | Yellow | Approaching limit |
| 90-100% | Red | Near/at limit |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Low usage | < 50% | `j7-usage/01-dashboard-low.png` |
| Medium usage | 50-80% | `j7-usage/01-dashboard-medium.png` |
| High usage | > 80% | `j7-usage/01-dashboard-high.png` |

---

### Step 2: Usage Details Page

**URL:** `/portal/usage`

**Purpose:** Deep dive into usage patterns.

#### Usage Page Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  Usage & Billing                                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Current Billing Period: Jan 1 - Jan 31, 2024                       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Total Requests                                                 ││
│  │                                                                 ││
│  │       7,450 / 10,000                                           ││
│  │  ████████████████████████████░░░░░░░░░░  74.5%                 ││
│  │                                                                 ││
│  │  At current rate, you'll reach your limit on Jan 28            ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Usage Stats                                                        │
│  ───────────                                                        │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │ Today         │  │ This Week     │  │ Avg/Day       │           │
│  │    312        │  │   2,184       │  │    240        │           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
│                                                                     │
│  Daily Usage (Last 30 days)                                         │
│  ──────────────────────────                                         │
│  ▂▃▅▆▇█▇▆▅▄▃▂▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅                                    │
│  1  5    10    15    20    25    30                                │
│                                                                     │
│  By Endpoint                                                        │
│  ───────────                                                        │
│  /api/data        ████████████████████  4,200 (56%)                │
│  /api/search      ████████████          2,400 (32%)                │
│  /api/users       ████                    850 (12%)                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Usage Breakdown

| Section | Information |
|---------|-------------|
| **Period summary** | Current period, days remaining |
| **Progress** | Visual bar, percentage, prediction |
| **Quick stats** | Today, week, average |
| **Daily chart** | 30-day trend |
| **By endpoint** | Breakdown by API path |
| **By key** | Usage per API key |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Usage page | Page load | `j7-usage/02-usage-page.png` |
| With chart | Data loaded | `j7-usage/02-usage-chart.png` |
| By endpoint | Scroll down | `j7-usage/02-by-endpoint.png` |

---

### Step 3: Warning States

**Purpose:** Alert users approaching limits.

#### Warning Levels

| Level | Threshold | Display | Action |
|-------|-----------|---------|--------|
| **Info** | 50% | Blue banner | None |
| **Warning** | 80% | Yellow banner | Soft upgrade prompt |
| **Critical** | 95% | Red banner | Strong upgrade prompt |
| **Exceeded** | 100% | Red block | Blocked, must upgrade |

#### Warning Banner (80%)

```
┌─────────────────────────────────────────────────────────────────────┐
│ ⚠️ You've used 80% of your monthly requests                        │
│                                                                     │
│ At your current usage rate, you'll reach your limit in 5 days.     │
│ Consider upgrading to avoid interruption.                          │
│                                                      [View Plans]   │
└─────────────────────────────────────────────────────────────────────┘
```

#### Critical Banner (95%)

```
┌─────────────────────────────────────────────────────────────────────┐
│ 🚨 You've used 95% of your monthly requests                        │
│                                                                     │
│ Only 500 requests remaining. Upgrade now to continue using         │
│ the API without interruption.                                      │
│                                                                     │
│ [Upgrade to Pro - 100K requests/month]                             │
└─────────────────────────────────────────────────────────────────────┘
```

#### Exceeded State (100%)

```
┌─────────────────────────────────────────────────────────────────────┐
│ ❌ Monthly quota exceeded                                           │
│                                                                     │
│ You've used all 10,000 requests for this month.                    │
│ Your quota resets on Feb 1, 2024.                                  │
│                                                                     │
│ Upgrade now for immediate access:                                  │
│                                                                     │
│ [Upgrade to Pro - $29/mo]  [Upgrade to Enterprise - $99/mo]        │
│                                                                     │
│ Or wait 5 days for your quota to reset.                            │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| 80% warning | Usage at 80% | `j7-usage/03-warning-80.png` |
| 95% critical | Usage at 95% | `j7-usage/03-warning-95.png` |
| 100% exceeded | Usage at 100% | `j7-usage/03-exceeded.png` |

---

### Step 4: Rate Limit Information

**Purpose:** Understand per-minute limits.

#### Rate Limit Display

```
┌─────────────────────────────────────────────────────────────────────┐
│  Rate Limiting                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Your Limit: 10 requests per minute                                 │
│                                                                     │
│  When you exceed this limit:                                        │
│  • You'll receive a 429 (Too Many Requests) response               │
│  • The Retry-After header tells you when to retry                  │
│  • Your quota is not affected by rate limiting                     │
│                                                                     │
│  Tips for staying under the limit:                                  │
│  • Add delays between requests                                      │
│  • Implement exponential backoff                                    │
│  • Consider upgrading for higher limits                             │
│                                                                     │
│  Your Plan: Free (10/min)  |  Pro: 600/min  |  Enterprise: 6000/min│
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Rate limit info | Expand section | `j7-usage/04-rate-limit.png` |

---

## UX Analysis

### Information Hierarchy

```
Dashboard (5 seconds)
├── Progress bar - quick visual
├── Numbers - current/total
└── CTA - View details or Upgrade

Usage Page (30-60 seconds)
├── Period context
├── Detailed breakdown
├── Historical chart
└── By endpoint/key analysis
```

### Cognitive Load

| View | Information Density | User Effort |
|------|---------------------|-------------|
| Dashboard card | Low | Glance |
| Usage page | Medium-High | Study |
| Warning banner | Low | Notice |

### Warning Psychology

The warning system is designed to:
1. **Not annoy** - Only show when relevant
2. **Build urgency** - Progressively stronger
3. **Offer solution** - Always include upgrade path
4. **Be honest** - Accurate predictions

### Accessibility

| Requirement | Implementation |
|-------------|----------------|
| Color meaning | Also uses icons/text |
| Progress bars | ARIA values |
| Charts | Alt text summaries |
| Warning banners | ARIA roles |

---

## Emotional Map

```
                     Emotional State During Usage Monitoring

Delight  ─┐                          ●────────● Upgrade completed
          │                        ╱
Neutral  ─┼────●───────●─────────●
          │             ╲       ╱
          │               ╲   ╱
Anxiety  ─┴─────────────────●
          │                  │
          │                  At limit
          └────┬─────────┬───┴───┬─────────┬─────────
            Check   Growing   Warning   Decision
```

### Emotional Triggers

| Stage | Emotion | Design Response |
|-------|---------|-----------------|
| Low usage | Comfortable | Green indicators |
| Growing | Engaged | Positive framing |
| Warning | Concerned | Clear action path |
| At limit | Anxious/Frustrated | Immediate solution |

---

## Metrics & KPIs

### Usage Page Engagement

| Metric | Definition | Target |
|--------|------------|--------|
| **Page views** | Visits to usage page | 2x/week |
| **Time on page** | Average duration | 30-60 sec |
| **Upgrade clicks** | From usage page | > 10% of views |

### Warning Effectiveness

| Metric | Definition | Target |
|--------|------------|--------|
| **Warning seen rate** | Users who see 80% | 60% of actives |
| **Warning → Upgrade** | Conversion from warning | > 15% |
| **Exceeded → Upgrade** | Conversion from 100% | > 30% |

### Analytics Events

```javascript
// Usage page viewed
analytics.track('usage_page_viewed', {
  usage_percent: 74,
  days_remaining: 15
});

// Warning shown
analytics.track('usage_warning_shown', {
  level: '80_percent',
  requests_remaining: 2000
});

// Upgrade clicked from usage
analytics.track('upgrade_clicked', {
  source: 'usage_page',
  current_usage_percent: 85
});
```

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j7-usage-monitoring
requires_auth: customer
requires_usage_data: true
viewport: 1280x720

steps:
  - name: dashboard-low
    url: /portal/
    setup:
      set_usage_percent: 30
    wait: networkidle

  - name: dashboard-high
    url: /portal/
    setup:
      set_usage_percent: 85
    wait: networkidle

  - name: usage-page
    url: /portal/usage
    wait: networkidle

  - name: warning-80
    url: /portal/usage
    setup:
      set_usage_percent: 80
    wait: text=80%

  - name: warning-95
    setup:
      set_usage_percent: 95

  - name: exceeded
    setup:
      set_usage_percent: 100
```

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J6: API Access](j6-api-access.md) | Using the API |
| [J8: Upgrade](j8-plan-upgrade.md) | When ready to upgrade |
| [E2: Rate Limiting](../errors/rate-limiting.md) | Per-minute limits |
| [E3: Quota Exceeded](../errors/quota-exceeded.md) | Monthly limits |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
