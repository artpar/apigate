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
│  User at 55 req/min: OK            User at 9,500/mo: Warning   │
│  User at 65 req/min: 429           User at 10,000+/mo: Blocked │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Quota Properties

Plans define quota limits via `requests_per_month`:

| Property | Type | Description |
|----------|------|-------------|
| `requests_per_month` | int | Monthly request limit (-1 = unlimited) |

---

## Configuring Quotas

### Via Admin UI

1. Go to **Plans** in the sidebar
2. Click **Add Plan** or edit existing
3. Set **Requests per Month**
4. Click **Save**

### Via CLI

```bash
# Create plan with quota (requests_per_month)
apigate plans create \
  --id "starter" \
  --name "Starter" \
  --rate-limit 60 \
  --requests 10000

# Unlimited quota (-1)
apigate plans create \
  --id "enterprise" \
  --name "Enterprise" \
  --rate-limit 1000 \
  --requests -1

# List plans with quotas
apigate plans list
```

### Via API

```bash
curl -X POST http://localhost:8080/admin/plans \
  -H "Content-Type: application/vnd.api+json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "data": {
      "type": "plans",
      "attributes": {
        "name": "Pro",
        "rate_limit_per_minute": 300,
        "requests_per_month": 100000
      }
    }
  }'
```

---

## Quota Enforcement

When a user exceeds their monthly quota:

```http
HTTP/1.1 402 Payment Required
Content-Type: application/vnd.api+json

{
  "errors": [{
    "status": "402",
    "code": "quota_exceeded",
    "title": "Quota Exceeded",
    "detail": "Monthly request quota exceeded"
  }]
}
```

---

## Quota Response Headers

Responses include quota information:

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
| `X-Quota-Reset` | When quota resets |

---

## Quota Reset

### Monthly Reset

Quotas reset at the start of each billing period. For most users, this is the 1st of each month at 00:00 UTC.

---

## Overage Billing

Plans can include overage pricing for requests beyond the included quota:

```bash
# Create plan with overage rate
apigate plans create \
  --id "paygo" \
  --name "Pay As You Go" \
  --rate-limit 100 \
  --requests 10000 \
  --overage 1  # $0.01 per request over limit
```

When quota is exceeded:
- Additional requests may still be allowed
- Overage is recorded for billing
- User is charged based on `overage_price`

---

## Quota Strategies

### 1. Simple Tiers

| Plan | Quota | Price |
|------|-------|-------|
| Free | 1,000 | $0 |
| Pro | 100,000 | $49 |
| Business | 1,000,000 | $299 |

### 2. Pay-As-You-Go

- Base quota included in plan
- Overage billed per request

```bash
apigate plans create \
  --id "paygo" \
  --name "PayGo" \
  --requests 10000 \
  --overage 1
```

### 3. Unlimited with Rate Limits

```bash
apigate plans create \
  --id "unlimited" \
  --name "Unlimited" \
  --requests -1 \
  --rate-limit 100
```

Fair use enforced via rate limiting, not quotas.

---

## Monitoring Quota Usage

### Via Admin UI

1. Go to **Users** in the sidebar
2. Click on a user
3. View **Usage** section showing current quota consumption

### Via API

```bash
curl http://localhost:8080/admin/users/<id> \
  -H "Cookie: session=YOUR_SESSION"
```

The response includes usage data in the `meta` section.

---

## Client Best Practices

### 1. Monitor Quota Headers

```python
def check_quota(response):
    remaining = int(response.headers.get('X-Quota-Remaining', 0))
    limit = int(response.headers.get('X-Quota-Limit', 0))

    if limit > 0:
        percent_used = (limit - remaining) / limit * 100

        if percent_used > 80:
            log.warning(f"Quota warning: {percent_used}% used")

        if percent_used > 95:
            log.critical(f"Quota critical: {percent_used}% used")
```

### 2. Handle Quota Errors

```python
def api_call():
    response = requests.get(url, headers={'X-API-Key': key})

    if response.status_code == 402:
        # Quota exceeded - need to upgrade plan or wait for reset
        error = response.json()['errors'][0]
        raise QuotaExceededError(error['detail'])

    if response.status_code == 429:
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
- [[Billing]] - Payment integration
