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
4. Bucket has maximum capacity (`rate_limit_burst` or `rate_limit`)
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

```bash
# Via CLI
apigate plans create \
  --name "Pro" \
  --rate-limit 600 \
  --rate-limit-burst 1000

# Via API
curl -X POST http://localhost:8080/admin/plans \
  -d '{
    "name": "Pro",
    "rate_limit_per_minute": 600,
    "rate_limit_burst": 1000
  }'
```

### Rate Limit Properties

| Property | Default | Description |
|----------|---------|-------------|
| `rate_limit_per_minute` | 60 | Sustained request rate |
| `rate_limit_burst` | same as rate_limit | Maximum burst capacity |

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

## Per-Route Rate Limits

Override plan limits for specific routes:

```bash
apigate routes create \
  --name "expensive-operation" \
  --path "/api/export/*" \
  --rate-limit 10  # Override to 10/min regardless of plan
```

Route rate limits take precedence over plan limits.

---

## Shared vs Separate Buckets

### Default: Per-Key Buckets

Each API key has its own bucket:

```
User with 3 API keys:
├── Key A: 60 tokens
├── Key B: 60 tokens  (independent)
└── Key C: 60 tokens
```

### Per-User Buckets

Share bucket across all user's keys:

```bash
apigate settings set rate_limit_per_user true
```

```
User with 3 API keys:
└── Shared bucket: 60 tokens
    ├── Key A uses from shared
    ├── Key B uses from shared
    └── Key C uses from shared
```

---

## Burst Handling

### Allow Bursts

```bash
apigate plans create \
  --rate-limit 60 \
  --rate-limit-burst 120
```

- Sustained: 60 req/min
- Burst: Up to 120 requests if bucket is full
- After burst: Must wait for refill

### No Burst

```bash
apigate plans create \
  --rate-limit 60 \
  --rate-limit-burst 60
```

Strict 1 req/sec maximum.

---

## Rate Limit Strategies

### 1. Tiered Limits

Different limits per plan tier:

| Plan | Rate Limit | Burst | Use Case |
|------|------------|-------|----------|
| Free | 60/min | 60 | Evaluation |
| Starter | 300/min | 500 | Small apps |
| Pro | 1000/min | 2000 | Production |
| Enterprise | 10000/min | 20000 | High scale |

### 2. Endpoint-Specific Limits

Protect expensive operations:

```bash
# Normal endpoints: use plan limit
apigate routes create --name "users-api" --path "/api/users/*"

# Expensive endpoint: lower limit
apigate routes create --name "export" --path "/api/export/*" --rate-limit 5

# Critical endpoint: higher limit
apigate routes create --name "health" --path "/health" --rate-limit 1000
```

### 3. Time-Based Limits

Different limits by time (requires custom module):

```yaml
# peak_hours: 9am-6pm → stricter limits
# off_peak: other times → relaxed limits
```

---

## Monitoring Rate Limits

### View Current State

```bash
# Check specific key's bucket
apigate keys rate-limit <key-id>

# Output:
# Bucket: 45/60 tokens
# Refill rate: 1/sec
# Next reset: 15s
```

### Analytics

```bash
# Keys hitting rate limits
apigate analytics rate-limits --period 24h

# Top rate-limited keys
apigate analytics rate-limits --sort hits --limit 10
```

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
from queue import Queue
import threading
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
1. Multiple keys sharing user bucket
2. Route-specific limit lower than plan
3. Burst consumed, waiting for refill

**Debug**:
```bash
apigate keys rate-limit <key-id>
apigate routes get <route-id>  # Check for route-level limit
```

### Rate Limits Not Applying

**Symptom**: No rate limiting despite configuration

**Causes**:
1. Plan has `rate_limit_per_minute: 0` (unlimited)
2. Request bypassing authentication
3. Admin role (may bypass limits)

**Debug**:
```bash
apigate plans get <plan-id>
```

---

## See Also

- [[Plans]] - Configure rate limits per plan
- [[Quotas]] - Monthly usage limits
- [[API-Keys]] - Per-key authentication
- [[Troubleshooting]] - Common issues
