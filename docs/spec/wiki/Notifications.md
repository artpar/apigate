# Notifications

APIGate supports webhook-based notifications for important events.

---

## Overview

Notifications are delivered via the webhook system. To receive notifications, configure a webhook endpoint and subscribe to the events you want to be notified about.

See [[Webhooks]] for webhook configuration.
See [[Events]] for the list of supported events.

---

## Notification-Relevant Events

| Event | Description |
|-------|-------------|
| `usage.threshold` | User reached usage threshold (e.g., 80% quota) |
| `usage.limit` | User reached usage limit |
| `payment.failed` | Payment failed |
| `payment.success` | Payment succeeded |
| `key.created` | API key was created |
| `key.revoked` | API key was revoked |

---

## Setting Up Notifications

### Via Web UI

1. Navigate to **Webhooks** in the admin dashboard
2. Click **Create Webhook**
3. Enter your notification endpoint URL
4. Select the events you want to be notified about
5. Save and note the signing secret

### Example: Slack Integration

To send notifications to Slack, you can use Slack's incoming webhooks:

1. Create a Slack incoming webhook at https://api.slack.com/apps
2. Create an APIGate webhook pointing to your Slack webhook URL
3. Subscribe to desired events

Note: APIGate sends its standard webhook payload format. For custom Slack formatting, you'll need an intermediate service to transform the payload.

---

## See Also

- [[Webhooks]] - Webhook configuration
- [[Events]] - All available events
