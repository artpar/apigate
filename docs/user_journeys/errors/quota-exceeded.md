# E3: Quota Exceeded Errors

> **When monthly limits are reached - upgrade path is critical.**

---

## Overview

Quota exceeded errors occur when a user has consumed their entire monthly request allocation. This is a key monetization moment - guide users to upgrade.

---

## Error Response

### Quota Exceeded (503)

**Trigger:** Monthly request quota fully consumed.

**Response:**
```json
{
  "error": {
    "code": "quota_exceeded",
    "message": "Monthly quota exceeded. Upgrade your plan for more requests.",
    "quota": 1000,
    "used": 1000,
    "resets_at": "2024-02-01T00:00:00Z",
    "upgrade_url": "/portal/plans"
  }
}
```

**Headers:**
```
HTTP/1.1 503 Service Unavailable
X-Quota-Limit: 1000
X-Quota-Used: 1000
X-Quota-Resets: 2024-02-01T00:00:00Z
```

---

## User Recovery Options

### Option 1: Wait for Reset

If the user can wait:
- Quota resets on the 1st of each month
- `resets_at` field shows exact time
- No action required

### Option 2: Upgrade Plan

For immediate access:
1. Go to `/portal/plans`
2. Select higher tier
3. Complete payment
4. New quota immediately available

### Option 3: Contact Support

For special circumstances:
- Temporary quota increase
- Enterprise custom limits
- Billing questions

---

## UX Guidelines

### Progressive Warnings

| Usage | Action |
|-------|--------|
| 50% | Info banner (optional) |
| 80% | Yellow warning |
| 95% | Red critical warning |
| 100% | Blocked + upgrade prompt |

### Upgrade-Focused Messaging

```
┌─────────────────────────────────────────────────────────────────────┐
│ ❌ Monthly quota exceeded                                           │
│                                                                     │
│ You've used all 1,000 requests for January.                        │
│                                                                     │
│ Options:                                                            │
│ • Upgrade to Pro for 100,000 requests/month ($29)                  │
│ • Wait until Feb 1 for your quota to reset                         │
│                                                                     │
│ [Upgrade Now]                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### Don't Punish Users

- Don't delete their data
- Don't revoke their keys
- Keep portal accessible
- Allow viewing usage history

---

## Quota Headers

All responses include quota headers:

| Header | Description |
|--------|-------------|
| `X-Quota-Limit` | Monthly request limit |
| `X-Quota-Used` | Requests used this month |
| `X-Quota-Remaining` | Requests remaining |
| `X-Quota-Resets` | ISO timestamp of reset |

---

## Plan Quotas

| Plan | Monthly Requests | Price |
|------|------------------|-------|
| Free | 1,000 | $0 |
| Pro | 100,000 | $29 |
| Enterprise | 1,000,000 | $99 |

---

## Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| 503 Response | Quota exceeded | `errors/e3-01-quota-exceeded.png` |
| Portal blocked | Try to use portal | `errors/e3-02-portal-blocked.png` |
| Upgrade prompt | On exceeded page | `errors/e3-03-upgrade-prompt.png` |

---

## Related

- [J7: Usage Monitoring](../customer/j7-usage-monitoring.md)
- [J8: Plan Upgrade](../customer/j8-plan-upgrade.md)
- [E2: Rate Limiting](rate-limiting.md)
