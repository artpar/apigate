# Events

APIGate emits webhook events for usage, billing, and API key lifecycle actions.

---

## Supported Event Types

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

## Event Payload

All webhook events follow this structure:

```json
{
  "id": "evt_a1b2c3d4e5f6g7h8",
  "type": "key.created",
  "timestamp": "2024-01-15T10:30:00Z",
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
| `type` | Event type from the supported list |
| `timestamp` | ISO 8601 timestamp when the event occurred |
| `data` | Event-specific data (varies by event type) |

---

## Webhook Signature

All webhook deliveries are signed with HMAC-SHA256. The signature is included in the `X-Webhook-Signature` header.

To verify:

```go
expectedSig := hmac.New(sha256.New, []byte(secret))
expectedSig.Write(payload)
valid := hmac.Equal(signature, expectedSig.Sum(nil))
```

---

## Subscribing to Events

Webhooks are configured through the web UI:

1. Navigate to **Webhooks** in the admin dashboard
2. Click **Create Webhook**
3. Enter the endpoint URL
4. Select events to subscribe to
5. Save and note the signing secret

---

## Retry Behavior

Failed webhook deliveries are retried with exponential backoff:

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1 minute |
| 3 | 5 minutes |
| 4 | 30 minutes |

Retries occur on:
- HTTP 5xx errors (server errors)
- HTTP 408 (request timeout)
- HTTP 429 (rate limited)

---

## See Also

- [[Webhooks]] - Webhook configuration details
