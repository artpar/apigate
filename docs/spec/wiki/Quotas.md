# Quotas

**Quotas** limit total API usage over a billing period (monthly).

---

## Overview

While rate limits control request **frequency**, quotas control total **volume**:

```
┌────────────────────────────────────────────────────────────────┐
│                  Rate Limits vs Quotas                          │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Rate Limit (short-term)           Quota (long-term)           │
│  ┌─────────────────────┐           ┌─────────────────────┐     │
│  │ 60 requests/minute  │           │ 10,000 requests/mo  │     │
│  │                     │           │                     │     │
│  │ Prevents burst      │           │ Caps monthly usage  │     │
│  │ abuse               │           │ for billing         │     │
│  └─────────────────────┘           └─────────────────────┘     │
│                                                                 │
│  User at 55 req/min: ✓ OK          User at 9,500/mo: ⚠ Warning │
│  User at 65 req/min: ✗ 429         User at 10,500/mo: Depends  │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Quota Properties

| Property | Type | Description |
|----------|------|-------------|
| `requests_per_month` | int | Monthly request limit (0 = unlimited) |
| `quota_grace_percent` | int | Allowance before hard block (default: 0) |
| `quota_enforcement` | enum | How to handle exceeded quota |

---

## Enforcement Modes

### Hard (Default)

Blocks requests when quota exceeded:

```bash
apigate plans create \
  --monthly-quota 10000 \
  --quota-enforcement hard
```

At 10,001 requests:
```http
HTTP/1.1 429 Too Many Requests

{
  "errors": [{
    "status": "429",
    "code": "quota_exceeded",
    "title": "Quota Exceeded",
    "detail": "Monthly quota of 10000 requests exceeded"
  }]
}
```

### Soft

Allows overage but flags the request:

```bash
apigate plans create \
  --monthly-quota 10000 \
  --quota-enforcement soft
```

At 10,001 requests:
- Request proceeds
- Response includes warning header
- Usage recorded for potential overage billing

```http
HTTP/1.1 200 OK
X-Quota-Warning: Monthly quota exceeded
X-Quota-Used: 10001
X-Quota-Limit: 10000
```

### Warn

Sends notifications but doesn't block:

```bash
apigate plans create \
  --monthly-quota 10000 \
  --quota-enforcement warn
```

At 8,000 requests (80%): Warning email sent
At 9,500 requests (95%): Critical warning sent
At 10,000+ requests: Over-quota notification, requests continue

---

## Grace Percentage

Buffer before hard enforcement kicks in:

```bash
apigate plans create \
  --monthly-quota 10000 \
  --quota-grace-percent 10 \
  --quota-enforcement hard
```

| Usage | Status |
|-------|--------|
| 0 - 9,999 | Normal |
| 10,000 | Quota reached, warning |
| 10,001 - 11,000 | Grace period (10%) |
| 11,001+ | Hard blocked |

---

## Quota Response Headers

Every response includes quota information:

```http
X-Quota-Limit: 10000
X-Quota-Used: 5234
X-Quota-Remaining: 4766
X-Quota-Reset: 2025-02-01T00:00:00Z
```

| Header | Description |
|--------|-------------|
| `X-Quota-Limit` | Monthly quota |
| `X-Quota-Used` | Requests used this period |
| `X-Quota-Remaining` | Requests remaining |
| `X-Quota-Reset` | When quota resets (month start) |

---

## Configuring Quotas

### Per Plan

```bash
# CLI
apigate plans create \
  --name "Starter" \
  --monthly-quota 10000

apigate plans create \
  --name "Pro" \
  --monthly-quota 100000

apigate plans create \
  --name "Enterprise" \
  --monthly-quota 0  # Unlimited
```

### Via API

```bash
curl -X POST http://localhost:8080/admin/plans \
  -d '{
    "name": "Pro",
    "requests_per_month": 100000,
    "quota_grace_percent": 10,
    "quota_enforcement": "hard"
  }'
```

---

## Quota Reset

### Automatic Monthly Reset

Quotas reset on the 1st of each month at 00:00 UTC:

```
Jan 1  → Counter: 0
Jan 15 → Counter: 5,234
Jan 31 → Counter: 9,876
Feb 1  → Counter: 0 (reset!)
```

### Custom Reset Date

Set per-user billing cycle:

```bash
# User signed up on 15th - reset on 15th each month
apigate users update <id> --billing-cycle-day 15
```

---

## Quota Notifications

### Configure Thresholds

```bash
apigate settings set quota_warning_thresholds "50,80,95,100"
```

Notifications sent at each threshold:
- 50%: Informational
- 80%: Warning
- 95%: Critical
- 100%: Quota reached

### Notification Channels

```bash
# Email notifications
apigate settings set quota_notification_email true

# Webhook notifications
apigate webhooks create \
  --name "quota-alerts" \
  --url "https://slack.com/webhook/xxx" \
  --events "quota.warning,quota.exceeded"
```

---

## Quota Management

### Check User Quota

```bash
# CLI
apigate users quota <user-id>

# Output:
# Plan: Pro (100,000 requests/month)
# Used: 45,234 (45.2%)
# Remaining: 54,766
# Resets: Feb 1, 2025
```

### Via API

```bash
curl http://localhost:8080/admin/users/<id>/quota
```

```json
{
  "data": {
    "type": "quota",
    "attributes": {
      "limit": 100000,
      "used": 45234,
      "remaining": 54766,
      "percent_used": 45.2,
      "resets_at": "2025-02-01T00:00:00Z"
    }
  }
}
```

### Reset Quota Manually

Emergency quota reset:

```bash
apigate users reset-quota <user-id>
```

### Add Quota Adjustment

Grant additional requests:

```bash
# Add 5000 bonus requests
apigate users adjust-quota <user-id> --add 5000 --reason "Compensation for outage"
```

---

## Overage Billing

### Track Overage

With `soft` enforcement, track overage for billing:

```bash
apigate analytics overage --user <id> --period 2025-01
```

### Calculate Overage Cost

```bash
# Plan: 10,000 included, $0.001 per extra request
# Used: 15,000
# Overage: 5,000 × $0.001 = $5.00
```

---

## Multiple Quota Types

### Request Quotas

```bash
apigate plans create --monthly-quota 10000  # 10K requests
```

### Byte Quotas

```bash
apigate plans create \
  --monthly-quota 10000 \
  --monthly-bytes-quota 1073741824  # 1GB
```

### Custom Unit Quotas

For metered APIs (AI tokens, compute units):

```yaml
# Route with custom metering
metering_mode: custom
metering_expr: "response.body.tokens_used"
```

---

## Quota Strategies

### 1. Simple Tiers

| Plan | Quota | Price |
|------|-------|-------|
| Free | 1,000 | $0 |
| Pro | 100,000 | $49 |
| Business | 1,000,000 | $299 |

### 2. Pay-As-You-Go

- Base quota included
- Overage billed per request

```bash
apigate plans create \
  --name "PayGo" \
  --monthly-quota 10000 \
  --overage-rate-cents 0.1 \
  --quota-enforcement soft
```

### 3. Unlimited with Fair Use

```bash
apigate plans create \
  --name "Unlimited" \
  --monthly-quota 0 \
  --rate-limit 100  # Fair use via rate limit
```

---

## Client Best Practices

### 1. Monitor Quota Headers

```python
def check_quota(response):
    remaining = int(response.headers.get('X-Quota-Remaining', 0))
    limit = int(response.headers.get('X-Quota-Limit', 0))

    percent_used = (limit - remaining) / limit * 100

    if percent_used > 80:
        log.warning(f"Quota warning: {percent_used}% used")

    if percent_used > 95:
        log.critical(f"Quota critical: {percent_used}% used")
```

### 2. Implement Graceful Degradation

```python
def api_call():
    response = requests.get(url, headers={'X-API-Key': key})

    if response.status_code == 429:
        error = response.json()['errors'][0]
        if error['code'] == 'quota_exceeded':
            # Quota exceeded - fail gracefully
            return cached_response()
        else:
            # Rate limited - retry later
            time.sleep(int(response.headers.get('Retry-After', 60)))
            return api_call()

    return response
```

---

## See Also

- [[Plans]] - Configure quotas per plan
- [[Rate-Limiting]] - Per-minute limits
- [[Usage-Tracking]] - How usage is recorded
- [[Billing]] - Overage billing setup
