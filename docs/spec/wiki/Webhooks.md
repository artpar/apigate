# Webhooks

**Webhooks** notify external systems when events occur in APIGate.

---

## Overview

Webhooks push real-time notifications to your systems:

```
┌────────────────────────────────────────────────────────────────┐
│                      Webhook Flow                               │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Event Occurs in APIGate                                        │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │            Webhook Delivery System                   │       │
│  │  • Queue event                                       │       │
│  │  • Sign payload                                      │       │
│  │  • Retry on failure                                  │       │
│  └─────────────────────────────────────────────────────┘       │
│       │                                                         │
│       ▼                                                         │
│  POST https://your-server.com/webhook                           │
│  {                                                              │
│    "event": "user.created",                                     │
│    "data": { ... },                                             │
│    "timestamp": "2025-01-19T10:30:00Z"                          │
│  }                                                              │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Webhook Events

### User Events

| Event | Triggered When |
|-------|----------------|
| `user.created` | New user registered |
| `user.updated` | User details changed |
| `user.deleted` | User deleted |
| `user.suspended` | User suspended |
| `user.activated` | User reactivated |

### API Key Events

| Event | Triggered When |
|-------|----------------|
| `key.created` | New API key created |
| `key.revoked` | API key revoked |

### Usage Events

| Event | Triggered When |
|-------|----------------|
| `quota.warning` | Quota threshold reached (80%, 95%) |
| `quota.exceeded` | Monthly quota exceeded |
| `rate_limit.exceeded` | Rate limit hit |

### Subscription Events

| Event | Triggered When |
|-------|----------------|
| `subscription.created` | New subscription started |
| `subscription.upgraded` | Plan upgraded |
| `subscription.downgraded` | Plan downgraded |
| `subscription.cancelled` | Subscription cancelled |
| `subscription.expired` | Subscription expired |

### System Events

| Event | Triggered When |
|-------|----------------|
| `upstream.healthy` | Upstream recovered |
| `upstream.unhealthy` | Upstream health check failed |

---

## Creating Webhooks

### Admin UI

1. Go to **Webhooks** in sidebar
2. Click **Add Webhook**
3. Configure:
   - **Name**: Descriptive name
   - **URL**: Endpoint to receive webhooks
   - **Events**: Select events to subscribe
   - **Secret**: For signature verification
4. Click **Save**

### CLI

```bash
# Create webhook
apigate webhooks create \
  --name "Slack Notifications" \
  --url "https://hooks.slack.com/services/xxx" \
  --events "user.created,quota.exceeded"

# With custom secret
apigate webhooks create \
  --name "Backend Integration" \
  --url "https://api.internal/webhooks" \
  --events "*" \
  --secret "whsec_your_secret_here"
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Webhook",
    "url": "https://api.example.com/webhook",
    "events": ["user.created", "quota.exceeded"],
    "secret": "whsec_xxx",
    "enabled": true
  }'
```

---

## Webhook Payload

### Standard Format

```json
{
  "id": "evt_abc123",
  "event": "user.created",
  "timestamp": "2025-01-19T10:30:00Z",
  "data": {
    "id": "usr_xyz789",
    "email": "user@example.com",
    "name": "John Doe",
    "plan_id": "plan_free",
    "created_at": "2025-01-19T10:30:00Z"
  },
  "metadata": {
    "source": "portal",
    "ip": "192.168.1.1"
  }
}
```

### Event-Specific Payloads

#### user.created

```json
{
  "event": "user.created",
  "data": {
    "id": "usr_xyz789",
    "email": "user@example.com",
    "name": "John Doe",
    "plan_id": "plan_free"
  }
}
```

#### quota.exceeded

```json
{
  "event": "quota.exceeded",
  "data": {
    "user_id": "usr_xyz789",
    "user_email": "user@example.com",
    "plan_name": "Free",
    "quota_limit": 1000,
    "quota_used": 1001
  }
}
```

#### subscription.upgraded

```json
{
  "event": "subscription.upgraded",
  "data": {
    "user_id": "usr_xyz789",
    "previous_plan": "Free",
    "new_plan": "Pro",
    "effective_at": "2025-01-19T10:30:00Z"
  }
}
```

---

## Webhook Security

### Signature Verification

Every webhook includes a signature header:

```
X-Webhook-Signature: sha256=abc123def456...
```

### Verifying Signatures

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(f"sha256={expected}", signature)

# In your webhook handler
@app.post("/webhook")
def handle_webhook(request):
    payload = request.body
    signature = request.headers.get("X-Webhook-Signature")

    if not verify_signature(payload, signature, WEBHOOK_SECRET):
        return {"error": "Invalid signature"}, 401

    event = json.loads(payload)
    process_event(event)
    return {"status": "ok"}, 200
```

```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
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
| 5 | 2 hours |
| 6 | 24 hours |

After 6 failed attempts, webhook is marked as failed.

### Response Requirements

Your endpoint should:
- Return `2xx` status code for success
- Respond within 30 seconds
- Return error status for failures (triggers retry)

---

## Managing Webhooks

### List Webhooks

```bash
# CLI
apigate webhooks list

# API
curl http://localhost:8080/admin/webhooks
```

### View Webhook

```bash
# CLI
apigate webhooks get <id>

# API
curl http://localhost:8080/admin/webhooks/<id>
```

### Update Webhook

```bash
# CLI
apigate webhooks update <id> --url "https://new-url.com/webhook"

# API
curl -X PUT http://localhost:8080/admin/webhooks/<id> \
  -d '{"url": "https://new-url.com/webhook"}'
```

### Disable Webhook

```bash
# CLI
apigate webhooks disable <id>

# API
curl -X POST http://localhost:8080/admin/webhooks/<id>/disable
```

### Delete Webhook

```bash
# CLI
apigate webhooks delete <id>

# API
curl -X DELETE http://localhost:8080/admin/webhooks/<id>
```

---

## Webhook History

### View Delivery History

```bash
# Recent deliveries
apigate webhooks history <id>

# Filter by status
apigate webhooks history <id> --status failed

# API
curl http://localhost:8080/admin/webhooks/<id>/deliveries
```

### Delivery Details

```json
{
  "id": "del_abc123",
  "webhook_id": "wh_xyz789",
  "event": "user.created",
  "status": "delivered",
  "attempts": 1,
  "response_code": 200,
  "response_time_ms": 234,
  "created_at": "2025-01-19T10:30:00Z",
  "delivered_at": "2025-01-19T10:30:00Z"
}
```

### Retry Failed Delivery

```bash
# CLI
apigate webhooks retry <delivery-id>

# API
curl -X POST http://localhost:8080/admin/webhooks/deliveries/<id>/retry
```

---

## Testing Webhooks

### Send Test Event

```bash
# CLI
apigate webhooks test <id> --event user.created

# API
curl -X POST http://localhost:8080/admin/webhooks/<id>/test \
  -d '{"event": "user.created"}'
```

### Local Testing

Use tools like ngrok for local development:

```bash
# Start ngrok
ngrok http 3000

# Create webhook with ngrok URL
apigate webhooks create \
  --url "https://abc123.ngrok.io/webhook" \
  --events "*"
```

---

## Common Integrations

### Slack

```bash
apigate webhooks create \
  --name "Slack Alerts" \
  --url "https://hooks.slack.com/services/T00/B00/xxx" \
  --events "quota.exceeded,user.created"
```

Transform payload for Slack:
```json
{
  "text": "New user registered: ${data.email}"
}
```

### Discord

```bash
apigate webhooks create \
  --name "Discord Alerts" \
  --url "https://discord.com/api/webhooks/xxx/yyy" \
  --events "quota.exceeded"
```

### Zapier

```bash
apigate webhooks create \
  --name "Zapier Integration" \
  --url "https://hooks.zapier.com/hooks/catch/xxx/yyy" \
  --events "*"
```

---

## Best Practices

### 1. Verify Signatures

Always verify webhook signatures to prevent spoofing.

### 2. Respond Quickly

Return 200 immediately, process asynchronously:

```python
@app.post("/webhook")
def handle_webhook(request):
    # Verify signature
    verify_signature(request)

    # Queue for async processing
    queue.enqueue(process_webhook, request.json)

    # Return immediately
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

### 4. Monitor Failures

Set up alerts for webhook failures:

```bash
apigate webhooks create \
  --name "Webhook Monitor" \
  --url "https://alerts.example.com" \
  --events "webhook.failed"
```

---

## See Also

- [[Events]] - All available events
- [[Integrations]] - Third-party integrations
- [[Notifications]] - Notification system
