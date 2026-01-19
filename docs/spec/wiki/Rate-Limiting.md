# Rate Limiting

**Rate limiting** protects your API from abuse by limiting request frequency per API key.

---

## Overview

APIGate uses a **token bucket algorithm** for rate limiting:

```
┌────────────────────────────────────────────────────────────────┐
│                     Token Bucket Algorithm                      │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐                                                │
│  │   Bucket    │ ← Tokens refill at steady rate                 │
│  │ ┌─────────┐ │                                                │
│  │ │ ● ● ● ● │ │   Capacity: 60 tokens (rate_limit)            │
│  │ │ ● ● ● ● │ │   Refill: 1 token per second                  │
│  │ │ ● ●     │ │                                                │
│  │ └─────────┘ │                                                │
│  └──────┬──────┘                                                │
│         │                                                       │
│         ▼                                                       │
│  Request arrives → Take 1 token                                 │
│  ├── Token available → Request proceeds                         │
│  └── Bucket empty → 429 Too Many Requests                       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## How It Works

1. Each API key has a token bucket
2. Bucket fills at `rate_limit / 60` tokens per second
3. Each request consumes 1 token
4. Bucket has maximum capacity (`rate_limit_per_minute`)
5. Empty bucket = rate limited

### Example

Plan with `rate_limit_per_minute: 60`:
- Bucket holds up to 60 tokens
- Refills 1 token per second
- Steady 1 req/sec: Never limited
- Burst 10 req/sec: Works for 6 seconds, then limited

---

## Configuration

### Per-Plan Rate Limits

Rate limits are configured on plans. All users on a plan share the same rate limit settings.

#### Via Admin UI

1. Go to **Plans** in the sidebar
2. Click **Add Plan** or edit existing
3. Set **Rate Limit** (requests per minute)
4. Click **Save**

#### Via CLI

```bash
# Create plan with rate limit
apigate plans create \
  --id "pro" \
  --name "Pro" \
  --rate-limit 600 \
  --requests 100000

# List plans with their rate limits
apigate plans list
```

#### Via API

```bash
curl -X POST http://localhost:8080/admin/plans \
  -H "Content-Type: application/vnd.api+json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "data": {
      "type": "plans",
      "attributes": {
        "name": "Pro",
        "rate_limit_per_minute": 600,
        "requests_per_month": 100000
      }
    }
  }'
```

### Global Burst Setting

Burst capacity is configured globally via settings:

```bash
apigate settings set ratelimit.burst_tokens 10
```

The default is 5 tokens. This allows brief bursts above the steady rate.

---

## Response Headers

Every response includes rate limit headers:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 599
X-RateLimit-Reset: 1704067260
```

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Max requests per minute |
| `X-RateLimit-Remaining` | Tokens left in bucket |
| `X-RateLimit-Reset` | Unix timestamp when bucket refills |

---

## Rate Limited Response

When rate limited, API returns:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/vnd.api+json
Retry-After: 5
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704067265

{
  "errors": [{
    "status": "429",
    "code": "rate_limited",
    "title": "Too Many Requests",
    "detail": "Rate limit exceeded. Retry after 5 seconds."
  }]
}
```

### Retry-After Header

Indicates seconds until requests are allowed again.

---

## Rate Limit Strategies

### 1. Tiered Limits

Different limits per plan tier:

| Plan | Rate Limit | Use Case |
|------|------------|----------|
| Free | 60/min | Evaluation |
| Starter | 300/min | Small apps |
| Pro | 1000/min | Production |
| Enterprise | 10000/min | High scale |

### 2. Plan-Based Limits

Each plan has its own rate limit. Users automatically get the rate limit from their assigned plan.

---

## Per-Key Buckets

Each API key has its own independent token bucket:

```
User with 3 API keys:
├── Key A: 60 tokens
├── Key B: 60 tokens  (independent)
└── Key C: 60 tokens
```

This allows users to distribute requests across multiple keys if needed.

---

## Client Best Practices

### 1. Respect Headers

```python
import time
import requests

def api_call():
    response = requests.get(url, headers={'X-API-Key': key})

    remaining = int(response.headers.get('X-RateLimit-Remaining', 0))

    if remaining < 10:
        # Slow down proactively
        time.sleep(1)

    if response.status_code == 429:
        retry_after = int(response.headers.get('Retry-After', 60))
        time.sleep(retry_after)
        return api_call()  # Retry

    return response
```

### 2. Implement Exponential Backoff

```python
def api_call_with_backoff(max_retries=5):
    for attempt in range(max_retries):
        response = requests.get(url, headers={'X-API-Key': key})

        if response.status_code != 429:
            return response

        wait_time = (2 ** attempt) + random.uniform(0, 1)
        time.sleep(wait_time)

    raise Exception("Max retries exceeded")
```

### 3. Queue Requests

```python
import time

class RateLimitedClient:
    def __init__(self, rate_limit=60):
        self.interval = 60 / rate_limit
        self.last_request = 0

    def request(self, url):
        now = time.time()
        wait = self.last_request + self.interval - now
        if wait > 0:
            time.sleep(wait)

        self.last_request = time.time()
        return requests.get(url, headers={'X-API-Key': key})
```

---

## Troubleshooting

### Unexpected Rate Limiting

**Symptom**: Getting 429s before expected

**Causes**:
1. Multiple keys consuming from same user's allocation
2. Burst consumed, waiting for refill
3. Clock skew between client and server

**Debug**:
```bash
# Check plan rate limit
apigate plans get <plan-id>
```

### Rate Limits Not Applying

**Symptom**: No rate limiting despite configuration

**Causes**:
1. Plan has `rate_limit_per_minute: 0` (unlimited)
2. Request bypassing authentication
3. API key not associated with a plan

**Debug**:
```bash
apigate plans list
apigate keys list
```

---

## See Also

- [[Plans]] - Configure rate limits per plan
- [[Quotas]] - Monthly usage limits
- [[API-Keys]] - Per-key authentication
- [[Troubleshooting]] - Common issues
