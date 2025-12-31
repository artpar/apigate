# E2: Rate Limiting Errors

> **Protecting the API while guiding users to success.**

---

## Overview

Rate limiting protects the API from abuse and ensures fair usage. When limits are exceeded, clear guidance helps users adapt their usage patterns.

---

## Error Response

### Rate Limit Exceeded (429)

**Trigger:** Too many requests in the time window.

**Response:**
```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Please slow down.",
    "limit": 10,
    "window": "1m",
    "retry_after": 45
  }
}
```

**Headers:**
```
HTTP/1.1 429 Too Many Requests
Retry-After: 45
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705315860
```

---

## Rate Limit Headers

All API responses include rate limit headers:

| Header | Description | Example |
|--------|-------------|---------|
| `X-RateLimit-Limit` | Requests allowed per window | `10` |
| `X-RateLimit-Remaining` | Requests left in window | `7` |
| `X-RateLimit-Reset` | Unix timestamp when limit resets | `1705315860` |
| `Retry-After` | Seconds to wait (on 429 only) | `45` |

---

## User Recovery

### Immediate Actions

1. **Wait** - Respect `Retry-After` header
2. **Check remaining** - Use `X-RateLimit-Remaining`
3. **Implement backoff** - Exponential backoff strategy

### Code Example

```javascript
async function apiCall(url) {
  const response = await fetch(url, {
    headers: { 'X-API-Key': API_KEY }
  });

  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    console.log(`Rate limited. Waiting ${retryAfter} seconds...`);
    await sleep(retryAfter * 1000);
    return apiCall(url); // Retry
  }

  return response.json();
}
```

### Long-term Solutions

1. **Batch requests** - Combine multiple operations
2. **Cache responses** - Don't re-fetch unchanged data
3. **Queue requests** - Spread over time
4. **Upgrade plan** - Higher limits available

---

## Plan Limits

| Plan | Requests/Minute | Best For |
|------|-----------------|----------|
| Free | 10 | Testing, development |
| Pro | 600 | Production applications |
| Enterprise | 6,000 | High-volume usage |

---

## UX Guidelines

### Proactive Communication

- Show rate limit in dashboard
- Include headers in every response
- Warn when approaching limit

### Error Message Clarity

```
❌ Bad: "Too many requests"
✅ Good: "Rate limit exceeded. You can make 10 requests per minute. Retry in 45 seconds."
```

---

## Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| 429 Response | Hit rate limit | `errors/e2-01-rate-limited.png` |
| Retry-After | Response headers | `errors/e2-02-headers.png` |
| Dashboard view | Rate limit info | `errors/e2-03-dashboard.png` |

---

## Related

- [J7: Usage Monitoring](../customer/j7-usage-monitoring.md)
- [J8: Plan Upgrade](../customer/j8-plan-upgrade.md)
