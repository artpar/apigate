# Metering API

The Metering API allows external services to submit usage events for billing purposes.

---

## Overview

APIGate tracks usage for requests that pass through its proxy automatically. However, external services need to report their own usage events for:

- Deployment lifecycle events (start, stop, scale)
- Compute time billing
- Storage usage
- Custom resource consumption

---

## Authentication

The metering API requires a **service API key** with the `meter:write` scope.

```bash
curl -X POST http://localhost:8080/api/v1/meter \
  -H "Authorization: Bearer <service-api-key>" \
  -H "Content-Type: application/vnd.api+json"
```

See [[API-Keys#service-api-keys]] for creating service keys.

---

## Submit Usage Events

**POST** `/api/v1/meter`

Submit one or more usage events for billing.

### Request Body

```json
{
  "data": [
    {
      "type": "usage_events",
      "attributes": {
        "id": "evt_abc123",
        "user_id": "usr_xyz789",
        "event_type": "deployment.started",
        "resource_id": "depl_456",
        "resource_type": "deployment",
        "quantity": 1,
        "metadata": {
          "template_id": "tmpl_789",
          "region": "us-east-1"
        },
        "timestamp": "2026-01-19T12:00:00Z"
      }
    }
  ]
}
```

### Event Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Idempotency key (prevents duplicate billing) |
| `user_id` | string | Yes | User to attribute usage to |
| `event_type` | string | Yes | Category of event |
| `resource_id` | string | No | Identifier of the resource used |
| `resource_type` | string | No | Type of resource |
| `quantity` | float64 | No | Units consumed (default: 1.0) |
| `metadata` | object | No | Arbitrary key-value context |
| `timestamp` | timestamp | No | When event occurred (default: now) |

### Response: Success (202 Accepted)

```json
{
  "meta": {
    "accepted": 1,
    "rejected": 0,
    "errors": []
  }
}
```

### Response: Partial Success (202 Accepted)

```json
{
  "meta": {
    "accepted": 2,
    "rejected": 1,
    "errors": [
      {
        "index": 1,
        "id": "evt_dup123",
        "code": "duplicate_event",
        "detail": "Event with this ID already processed"
      }
    ]
  }
}
```

---

## Query Usage Events

**GET** `/api/v1/meter`

Query submitted usage events (admin only).

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_id` | string | Filter by user ID |
| `event_type` | string | Filter by event type |
| `resource_type` | string | Filter by resource type |
| `start_date` | timestamp | Events after this time |
| `end_date` | timestamp | Events before this time |
| `page[number]` | int | Page number (default: 1) |
| `page[size]` | int | Page size (default: 50, max: 100) |

### Response (200 OK)

```json
{
  "data": [
    {
      "type": "usage_events",
      "id": "evt_abc123",
      "attributes": {
        "user_id": "usr_xyz789",
        "event_type": "deployment.started",
        "resource_id": "depl_456",
        "resource_type": "deployment",
        "quantity": 1,
        "metadata": {
          "template_id": "tmpl_789",
          "region": "us-east-1"
        },
        "timestamp": "2026-01-19T12:00:00Z",
        "source": "hoster-service",
        "created_at": "2026-01-19T12:00:01Z"
      }
    }
  ],
  "links": {
    "self": "/api/v1/meter?page[number]=1&page[size]=50",
    "next": "/api/v1/meter?page[number]=2&page[size]=50"
  },
  "meta": {
    "total_count": 150,
    "page_count": 3
  }
}
```

---

## Event Types

Event types follow a `{resource}.{action}` naming convention:

| Event Type | Description | Default Quantity |
|------------|-------------|------------------|
| `api.request` | API request | 1 |
| `deployment.created` | Deployment created | 1 |
| `deployment.started` | Deployment started running | 1 |
| `deployment.stopped` | Deployment stopped | 1 |
| `deployment.deleted` | Deployment removed | 1 |
| `compute.minutes` | Compute time in minutes | minutes |
| `storage.gb_hours` | Storage in GB-hours | gb_hours |
| `bandwidth.gb` | Data transfer in GB | gb |
| `custom.*` | Custom event types | varies |

Custom event types can be defined by prefixing with `custom.`.

---

## Idempotency

### Event ID Deduplication

Each event must have a unique `id` (idempotency key):
- Same ID submitted twice = second submission ignored
- IDs are scoped per service key
- IDs are retained for 30 days, then purged

### Batch Idempotency

Use `Idempotency-Key` header for batch-level deduplication:
- Entire batch is atomic (all succeed or none)
- Retrying same batch with same header = same result

---

## Rate Limits

| Limit | Value | Scope |
|-------|-------|-------|
| Requests per minute | 100 | Per service key |
| Events per request | 1000 | Per batch |
| Event age | 7 days | Maximum timestamp age |

---

## Billing Integration

### Quota Counting

External events count toward the user's monthly quota:

1. Events are aggregated per billing period (monthly)
2. `quantity` field is multiplied by event type's cost multiplier
3. Total is added to user's usage count
4. Quota warnings/enforcement apply as configured in plan

### Cost Multipliers

| Event Type | Default Multiplier | Notes |
|------------|-------------------|-------|
| `api.request` | 1.0 | Standard request |
| `compute.minutes` | 0.1 | 10 minutes = 1 request equivalent |
| `storage.gb_hours` | 0.01 | 100 GB-hours = 1 request equivalent |
| Custom | 1.0 | Configurable per event type |

---

## Error Codes

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `invalid_event_type` | 422 | Invalid Event Type | Unknown event type |
| `duplicate_event` | 409 | Duplicate Event | Event ID already processed |
| `user_not_found` | 422 | User Not Found | user_id doesn't exist |
| `invalid_quantity` | 422 | Invalid Quantity | Quantity <= 0 |
| `invalid_timestamp` | 422 | Invalid Timestamp | Timestamp in future or too old |
| `insufficient_scope` | 403 | Insufficient Scope | API key lacks `meter:write` scope |

---

## Example: Deployment Service Integration

A deployment service (like Hoster) can report lifecycle events:

```bash
# When deployment starts
curl -X POST http://localhost:8080/api/v1/meter \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{
    "data": [{
      "type": "usage_events",
      "attributes": {
        "id": "evt_depl_123_start",
        "user_id": "usr_abc",
        "event_type": "deployment.started",
        "resource_id": "depl_123",
        "resource_type": "deployment",
        "metadata": {"template": "nodejs", "plan": "pro"}
      }
    }]
  }'

# Report compute time (every hour)
curl -X POST http://localhost:8080/api/v1/meter \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{
    "data": [{
      "type": "usage_events",
      "attributes": {
        "id": "evt_compute_123_2026011912",
        "user_id": "usr_abc",
        "event_type": "compute.minutes",
        "resource_id": "depl_123",
        "quantity": 60,
        "timestamp": "2026-01-19T12:00:00Z"
      }
    }]
  }'
```

---

## See Also

- [[API-Keys#service-api-keys]] - Creating service keys
- [[Usage-Metering]] - Metering configuration
- [[Usage-Tracking]] - How usage is recorded
- [[Billing]] - Billing integration
- [[Quotas]] - Quota enforcement
