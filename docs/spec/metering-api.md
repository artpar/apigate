# Metering API Specification

> Implementation: `adapters/http/admin/meter.go`

The Metering API allows external services to submit usage events for billing purposes. This enables downstream services to report their own usage (deployments, compute time, storage, etc.) that doesn't pass through APIGate's proxy.

## Overview

### Purpose

APIGate tracks usage for requests that pass through its proxy automatically. However, external services need to report their own usage events for:
- Deployment lifecycle events (start, stop, scale)
- Compute time billing
- Storage usage
- Custom resource consumption

### Authentication

The metering API requires a **service API key** with the `meter:write` scope. This is a special key type that:
- Is not tied to a specific user
- Can submit events on behalf of any user
- Should only be issued to trusted services

## Resource Type

**Type**: `usage_events`

### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Idempotency key (prevents duplicate billing) |
| `user_id` | string | Yes | User to attribute usage to |
| `event_type` | string | Yes | Category of event (e.g., `deployment.started`) |
| `resource_id` | string | No | Identifier of the resource used |
| `resource_type` | string | No | Type of resource (e.g., `deployment`, `storage`) |
| `quantity` | float64 | No | Units consumed (default: 1.0) |
| `metadata` | object | No | Arbitrary key-value context |
| `timestamp` | timestamp | No | When event occurred (default: now) |

### Event Types

Event types follow a `{resource}.{action}` naming convention:

| Event Type | Description | Default Quantity |
|------------|-------------|------------------|
| `api.request` | API request (use for external API calls) | 1 |
| `deployment.created` | Deployment created | 1 |
| `deployment.started` | Deployment started running | 1 |
| `deployment.stopped` | Deployment stopped | 1 |
| `deployment.deleted` | Deployment removed | 1 |
| `compute.minutes` | Compute time in minutes | minutes |
| `storage.gb_hours` | Storage in GB-hours | gb_hours |
| `bandwidth.gb` | Data transfer in GB | gb |
| `custom.*` | Custom event types | varies |

Custom event types can be defined by prefixing with `custom.`.

## Endpoints

### Submit Usage Events

**POST** `/api/v1/meter`

Submit one or more usage events for billing.

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | `Bearer <service-api-key>` |
| `Content-Type` | Yes | `application/vnd.api+json` |
| `Idempotency-Key` | No | Alternative idempotency key for the batch |

#### Request Body

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

#### Response: Success (202 Accepted)

```json
{
  "meta": {
    "accepted": 1,
    "rejected": 0,
    "errors": []
  }
}
```

#### Response: Partial Success (202 Accepted)

When some events are accepted and others rejected:

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

#### Response: All Rejected (422 Unprocessable Entity)

When all events fail validation:

```json
{
  "errors": [
    {
      "status": "422",
      "code": "validation_error",
      "title": "Validation Failed",
      "detail": "user_id is required",
      "source": {
        "pointer": "/data/0/attributes/user_id"
      }
    }
  ]
}
```

### Query Usage Events

**GET** `/api/v1/meter`

Query submitted usage events (admin only).

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `user_id` | string | Filter by user ID |
| `event_type` | string | Filter by event type |
| `resource_type` | string | Filter by resource type |
| `start_date` | timestamp | Events after this time |
| `end_date` | timestamp | Events before this time |
| `page[number]` | int | Page number (default: 1) |
| `page[size]` | int | Page size (default: 50, max: 100) |

#### Response (200 OK)

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

## Error Codes

### Metering-Specific Errors

| Code | Status | Title | When Used |
|------|--------|-------|-----------|
| `invalid_event_type` | 422 | Invalid Event Type | Unknown event type |
| `duplicate_event` | 409 | Duplicate Event | Event ID already processed |
| `user_not_found` | 422 | User Not Found | user_id doesn't exist |
| `invalid_quantity` | 422 | Invalid Quantity | Quantity <= 0 |
| `invalid_timestamp` | 422 | Invalid Timestamp | Timestamp in future or too old |
| `insufficient_scope` | 403 | Insufficient Scope | API key lacks `meter:write` scope |

### Error Examples

#### Invalid Event Type

```json
{
  "errors": [{
    "status": "422",
    "code": "invalid_event_type",
    "title": "Invalid Event Type",
    "detail": "Event type 'unknown.event' is not recognized",
    "source": {
      "pointer": "/data/0/attributes/event_type"
    }
  }]
}
```

#### Duplicate Event

```json
{
  "errors": [{
    "status": "409",
    "code": "duplicate_event",
    "title": "Duplicate Event",
    "detail": "Event with ID 'evt_abc123' has already been processed"
  }]
}
```

#### Insufficient Scope

```json
{
  "errors": [{
    "status": "403",
    "code": "insufficient_scope",
    "title": "Insufficient Scope",
    "detail": "API key requires 'meter:write' scope to submit usage events"
  }]
}
```

## Billing Integration

### Quota Counting

External events count toward the user's monthly quota:

1. Events are aggregated per billing period (monthly)
2. `quantity` field is multiplied by event type's cost multiplier
3. Total is added to user's usage count
4. Quota warnings/enforcement apply as configured in plan

### Cost Multipliers

Event types can have different cost multipliers:

| Event Type | Default Multiplier | Notes |
|------------|-------------------|-------|
| `api.request` | 1.0 | Standard request |
| `compute.minutes` | 0.1 | 10 minutes = 1 request equivalent |
| `storage.gb_hours` | 0.01 | 100 GB-hours = 1 request equivalent |
| Custom | 1.0 | Configurable per event type |

### Overage Billing

When user exceeds quota:
1. Overage is calculated as `(total_usage - quota_limit) * overage_price`
2. External events included in overage calculation
3. Invoice generated at billing period end

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

## Rate Limits

| Limit | Value | Scope |
|-------|-------|-------|
| Requests per minute | 100 | Per service key |
| Events per request | 1000 | Per batch |
| Event age | 7 days | Maximum timestamp age |

## Service API Keys

### Creating a Service Key

Service keys are created via admin API with `meter:write` scope:

```json
POST /admin/keys
{
  "data": {
    "type": "api_keys",
    "attributes": {
      "name": "Hoster Metering Service",
      "scopes": ["meter:write"],
      "service": true
    }
  }
}
```

### Key Attributes

| Attribute | Description |
|-----------|-------------|
| `service` | `true` for service keys |
| `scopes` | Must include `meter:write` |
| `source_name` | Identifies the service in event logs |

## Implementation Notes

### Domain Model

**File**: `domain/usage/event.go`

Extended Event struct:
```go
type Event struct {
    // Existing fields
    ID             string
    KeyID          string
    UserID         string
    // ...

    // New fields for external events
    EventType      string            // "deployment.started", etc.
    ResourceID     string            // What was used
    ResourceType   string            // Resource category
    Quantity       float64           // Units (default 1.0)
    Source         string            // "proxy" or service name
    Metadata       map[string]string // Arbitrary context
    IdempotencyKey string            // For deduplication
}
```

### Storage

**Table**: `usage_events`

New columns:
- `event_type` VARCHAR(255)
- `resource_id` VARCHAR(255)
- `resource_type` VARCHAR(255)
- `quantity` REAL DEFAULT 1.0
- `source` VARCHAR(255)
- `metadata` TEXT (JSON)
- `idempotency_key` VARCHAR(255) UNIQUE

### Aggregation

Events are aggregated in `domain/usage/aggregate.go`:
- Group by `user_id` and `event_type`
- Sum `quantity` values
- Apply cost multipliers
- Return totals for billing

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-19 | Initial metering API specification |
