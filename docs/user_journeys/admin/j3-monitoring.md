# J3: Monitoring & Management

> **Running your API business - visibility into users, usage, and revenue.**

---

## Business Context

### Why This Journey Matters

Monitoring is how API Sellers **understand their business health**. Without visibility into users, usage patterns, and revenue, they can't make informed decisions or identify problems.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MONITORING VALUE                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Usage Data  ───▶  Business Insights  ───▶  Actions               │
│                                                                     │
│   • Request volume    • Growth trends       • Adjust limits        │
│   • Error rates       • Churn signals       • Contact at-risk      │
│   • User activity     • Revenue forecast    • Plan adjustments     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact

| Insight | Business Action | Revenue Impact |
|---------|-----------------|----------------|
| High-usage free user | Prompt to upgrade | Conversion |
| Inactive paid user | Reach out, reduce churn | Retention |
| Error spike | Fix issues fast | Customer satisfaction |
| API key abuse | Revoke and protect | Cost control |

### Business Success Criteria

- [ ] Admin can see key metrics within 5 seconds of dashboard load
- [ ] Usage data updates in real-time (< 1 minute delay)
- [ ] User actions (suspend, activate) take effect immediately
- [ ] Export data for external analysis
- [ ] Alerts for anomalies (future)

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Seller (operational mode, managing business) |
| **Prior Action** | Setup complete, has customers using the API |
| **Mental State** | Operational, checking on business health |
| **Expectation** | "How is my API business doing?" |

### What Triggered This Journey?

- Daily check on business metrics
- Customer support inquiry
- Investigating an issue or anomaly
- Preparing business reports

### User Goals

1. **Primary:** Understand current business health at a glance
2. **Secondary:** Investigate specific user or usage issues
3. **Tertiary:** Take action on problematic accounts

---

## The Journey

### Overview

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│Dashboard │───▶│  Users   │───▶│ API Keys │───▶│ Actions  │
│ Overview │    │  List    │    │   List   │    │(suspend) │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
      │               │               │
      └───────────────┴───────────────┘
                      │
              All accessible from
                 navigation
```

---

### Step 1: Dashboard Overview

**URL:** `/ui/`

**Purpose:** Quick health check of the entire platform.

#### Dashboard Cards

| Card | Metric | Trend |
|------|--------|-------|
| **Total Users** | Count of registered users | vs last period |
| **Active API Keys** | Keys that made requests recently | vs last period |
| **Requests Today** | API calls in last 24h | Sparkline |
| **Monthly Revenue** | MRR from paid subscriptions | vs last month |

#### Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  APIGate Admin Dashboard                           [User ▼] [Logout]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐│
│  │ Total Users │  │ Active Keys │  │Requests 24h │  │  MRR        ││
│  │     127     │  │     89      │  │   45,230    │  │   $1,247    ││
│  │   +12% ▲    │  │   +5% ▲     │  │   +23% ▲    │  │   +8% ▲     ││
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    Request Volume (7 days)                      ││
│  │  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂               ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Getting Started                      Recent Activity               │
│  ─────────────────                    ───────────────               │
│  [✓] Connect API                      • John signed up (2m ago)     │
│  [✓] Create admin                     • API key created (15m ago)   │
│  [✓] Create plan                      • Pro upgrade (1h ago)        │
│  [ ] First customer                   • Rate limit hit (2h ago)     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Full dashboard | Page load | `j3-monitor/01-dashboard.png` |
| With activity | Has users | `j3-monitor/01-dashboard-active.png` |
| Empty state | No users yet | `j3-monitor/01-dashboard-empty.png` |

---

### Step 2: User Management

**URL:** `/ui/user`

**Purpose:** View and manage customer accounts.

#### User List Table

| Column | Description | Actions |
|--------|-------------|---------|
| Name | User's display name | Click for detail |
| Email | User's email address | - |
| Plan | Current subscription | - |
| Status | Active/Suspended/Cancelled | Status pill |
| Requests | This month's usage | Progress bar |
| Created | Account creation date | - |
| Actions | Menu | View, Suspend, etc. |

#### User Detail View

```
┌─────────────────────────────────────────────────────────────────────┐
│ User: John Developer                                    [← Back]    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Profile                           Plan & Usage                     │
│  ───────                           ────────────                     │
│  Email: john@example.com           Plan: Pro ($29/mo)               │
│  Status: Active                    Requests: 45,230 / 100,000       │
│  Created: Jan 15, 2024             Rate Limit: 600/min              │
│                                    Renewal: Feb 15, 2024            │
│                                                                     │
│  API Keys (2)                                                       │
│  ────────────                                                       │
│  • Production (ak_abc1...7890)     Last used: 5 min ago            │
│  • Development (ak_def1...2345)    Last used: 3 days ago           │
│                                                                     │
│  Usage History                                                      │
│  ─────────────                                                      │
│  [Chart showing 30-day request history]                             │
│                                                                     │
│  Actions                                                            │
│  ───────                                                            │
│  [Change Plan]  [Suspend User]  [Reset Usage]                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### User Actions

| Action | Effect | Confirmation |
|--------|--------|--------------|
| **Suspend** | Blocks all API access | Yes, with reason |
| **Activate** | Restores API access | No |
| **Change Plan** | Moves to different tier | Yes |
| **Delete** | Removes account entirely | Yes, type email |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| User list | Page load | `j3-monitor/02-users-list.png` |
| User detail | Click user | `j3-monitor/02-user-detail.png` |
| Suspend dialog | Click suspend | `j3-monitor/02-suspend-dialog.png` |
| User suspended | After suspend | `j3-monitor/02-user-suspended.png` |

---

### Step 3: API Key Management

**URL:** `/ui/api_key`

**Purpose:** Monitor all API keys across all users.

#### API Key List Table

| Column | Description | Notes |
|--------|-------------|-------|
| Key (masked) | ak_xxxx...yyyy | Only prefix/suffix shown |
| Name | User-given name | e.g., "Production" |
| User | Owner email | Link to user |
| Requests | Total requests made | All time |
| Last Used | Most recent request | Relative time |
| Created | Key creation date | - |
| Status | Active/Revoked | Status pill |

#### Key Detail View

| Section | Information |
|---------|-------------|
| Key Info | Masked key, name, created date |
| Usage Stats | Total requests, errors, data transfer |
| Recent Requests | Last 10 requests with details |
| Owner | Link to user profile |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Key list | Page load | `j3-monitor/03-keys-list.png` |
| Key detail | Click key | `j3-monitor/03-key-detail.png` |
| Revoke dialog | Click revoke | `j3-monitor/03-revoke-dialog.png` |

---

### Step 4: Taking Action

**Purpose:** Respond to issues identified during monitoring.

#### Common Actions

| Scenario | Detection | Action |
|----------|-----------|--------|
| Abusive user | High error rate, complaints | Suspend account |
| Trial expired | Trial days elapsed | Contact or auto-downgrade |
| High usage free | Approaching limits | Prompt upgrade email |
| Inactive paid | No requests in 30 days | Retention outreach |
| Key compromised | Suspicious patterns | Revoke key |

#### Suspend User Flow

1. Navigate to user detail
2. Click "Suspend User"
3. Enter reason (required)
4. Confirm action
5. User immediately loses API access
6. User sees "Account suspended" on login

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Suspend confirmation | Click suspend | `j3-monitor/04-suspend-confirm.png` |
| Reason entry | In dialog | `j3-monitor/04-suspend-reason.png` |
| Success message | After suspend | `j3-monitor/04-suspend-success.png` |

---

## UX Analysis

### Information Hierarchy

```
Dashboard (30 seconds)
├── Health metrics at a glance
├── Trends and anomalies
└── Quick links to details

User List (2 minutes)
├── Overview of all users
├── Sort and filter
└── Quick actions

User Detail (1-5 minutes)
├── Full user context
├── Usage deep dive
└── Account management
```

### Cognitive Load Assessment

| View | Information Density | Cognitive Load |
|------|---------------------|----------------|
| Dashboard | Medium | Low (familiar patterns) |
| User List | High | Medium (scanning) |
| User Detail | High | Medium-High (decision making) |

### Accessibility

| Requirement | Implementation |
|-------------|----------------|
| Data tables | Proper ARIA roles |
| Charts | Alt text descriptions |
| Color coding | Also uses icons/text |
| Actions | Clear button labels |

---

## Emotional Map

```
                     Emotional State During Monitoring

Delight  ─┐
          │           ●─────●  Business growing!
          │         ╱       ╲
Neutral  ─┼────●───●─────────●───────●
          │    │               ╲   ╱
          │    │                 ●    Issue found
Anxiety  ─┴────┴───────────────────
          │
          └────┬─────────┬─────────┬─────────
            Dashboard  Drill down  Action
```

### Emotional Triggers

| Trigger | Emotion | Design Response |
|---------|---------|-----------------|
| Metrics up | Delight | Green indicators, upward arrows |
| Metrics down | Concern | Yellow/red indicators |
| Issue found | Determination | Clear action buttons |
| Action taken | Resolution | Success confirmation |

---

## Metrics & KPIs

### Dashboard Engagement

| Metric | Definition | Target |
|--------|------------|--------|
| **Dashboard visits** | Daily active admins | > 70% daily |
| **Time on dashboard** | Average session | 2-5 minutes |
| **Drill-down rate** | View user/key details | > 30% of visits |

### Operational Metrics

| Metric | Definition | Alert |
|--------|------------|-------|
| Data freshness | Time since last update | > 5 min |
| Action success rate | Actions completed | < 95% |
| Page load time | Dashboard render | > 3 sec |

---

## Edge Cases & Errors

### Data Freshness

| Scenario | User Message | Recovery |
|----------|--------------|----------|
| Stale data | "Last updated X min ago" | Auto-refresh |
| Backend down | "Unable to fetch metrics" | Retry button |

### Action Errors

| Action | Error | Recovery |
|--------|-------|----------|
| Suspend | User already suspended | Show current status |
| Delete | Has active subscription | Cancel subscription first |
| Revoke key | Already revoked | Show revoked status |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j3-monitoring
requires_auth: admin
requires_data: true  # Need users and usage data
viewport: 1280x720

steps:
  - name: dashboard
    url: /ui/
    wait: networkidle

  - name: users-list
    url: /ui/user
    wait: networkidle

  - name: user-detail
    url: /ui/user/{first_user_id}
    wait: networkidle

  - name: keys-list
    url: /ui/api_key
    wait: networkidle

  - name: suspend-dialog
    url: /ui/user/{first_user_id}
    actions:
      - click: button:has-text("Suspend")
      - wait: dialog
```

### GIF Sequence

**j3-user-management.gif**
- Frame 1: User list view (2s)
- Frame 2: Click into user (1s)
- Frame 3: User detail view (3s)
- Frame 4: Usage chart (2s)
- Frame 5: Back to list (1s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J2: Plans](j2-plan-management.md) | Plan info shown in user details |
| [J7: Usage](../customer/j7-usage-monitoring.md) | Customer's view of same data |
| [E2: Rate Limiting](../errors/rate-limiting.md) | Error handling |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
