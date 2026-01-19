# Webhooks

**Webhooks** notify external systems when events occur in APIGate.

---

## Overview

Webhooks push real-time notifications to your systems when events like API key creation, usage threshold, or payment events occur.

---

## Supported Events

| Event | Description |
|-------|-------------|
| `usage.threshold` | User reached usage threshold (e.g., 80% quota) |
| `usage.limit` | User reached usage limit |
| `key.created` | API key was created |
| `key.revoked` | API key was revoked |
| `subscription.start` | Subscription started |
| `subscription.end` | Subscription ended |
| `subscription.renew` | Subscription renewed |
| `plan.changed` | User changed plans |
| `payment.success` | Payment succeeded |
| `payment.failed` | Payment failed |
| `invoice.created` | Invoice was created |
| `test` | Test event for webhook validation |

---

## Creating Webhooks

Webhooks are created and managed through the admin web UI:

1. Go to **Webhooks** in the sidebar
2. Click **Add Webhook**
3. Configure:
   - **Name**: Descriptive name
   - **URL**: Endpoint to receive webhooks
   - **Events**: Select events to subscribe
4. Click **Save**

A signing secret is automatically generated for signature verification.

---

## Webhook Payload

```json
{
  "id": "evt_a1b2c3d4e5f6g7h8",
  "type": "key.created",
  "timestamp": "2025-01-19T10:30:00Z",
  "data": {
    "key_id": "key_abc123",
    "user_id": "usr_xyz789",
    "name": "Production Key"
  }
}
```

### Payload Fields

| Field | Description |
|-------|-------------|
| `id` | Unique event identifier (prefixed with `evt_`) |
| `type` | Event type |
| `timestamp` | ISO 8601 timestamp |
| `data` | Event-specific data |

---

## Webhook Security

### Signature Verification

Every webhook includes a signature header:

```
X-Webhook-Signature: <signature>
```

The signature is an HMAC-SHA256 hex digest of the request body using your webhook secret.

### Verifying Signatures

**Python:**

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

**JavaScript:**

```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}
```

**Go:**

```go
func verifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

---

## Retry Policy

Failed webhooks are retried with exponential backoff:

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1 minute |
| 3 | 5 minutes |
| 4 | 30 minutes |

After 3 failed attempts, the delivery is marked as failed.

### Retries Triggered On

- HTTP 5xx errors (server errors)
- HTTP 408 (request timeout)
- HTTP 429 (rate limited)

### Response Requirements

Your endpoint should:
- Return `2xx` status code for success
- Respond within 30 seconds
- Return error status for failures (triggers retry)

---

## Best Practices

### 1. Verify Signatures

Always verify webhook signatures to prevent spoofing.

### 2. Respond Quickly

Return 200 immediately, process asynchronously:

```python
@app.post("/webhook")
def handle_webhook(request):
    verify_signature(request)
    queue.enqueue(process_webhook, request.json)
    return {"status": "queued"}, 200
```

### 3. Handle Duplicates

Events may be delivered multiple times. Use `id` for deduplication:

```python
def process_webhook(event):
    if already_processed(event['id']):
        return
    mark_processed(event['id'])
    handle_event(event)
```

### 4. Use HTTPS

Always use HTTPS endpoints in production for security.

---

## Testing Webhooks

### Local Development

Use tools like ngrok for local testing:

```bash
ngrok http 3000
```

Then configure your webhook URL with the ngrok URL.

---

## See Also

- [[Events]] - All available events
- [[Notifications]] - Notification overview
