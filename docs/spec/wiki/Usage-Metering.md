# Usage Metering

Configure how API usage is measured for billing and quotas.

---

## Overview

Usage metering determines how requests count against quotas:

| Mode | Description | Use Case |
|------|-------------|----------|
| `request` | 1 unit per request | Simple APIs |
| `bytes` | By response size | Data APIs |
| `response_field` | From response JSON | AI/ML APIs |
| `custom` | Custom expression | Complex pricing |

---

## Request-Based (Default)

Every request counts as 1 unit:

```bash
apigate routes update <id> --metering-mode request
```

---

## Byte-Based

Meter by response size:

```bash
apigate routes update <id> \
  --metering-mode bytes \
  --metering-expr "responseBytes / 1024"  # KB
```

---

## Response Field

Extract usage from response JSON:

```bash
# AI API that returns token count
apigate routes update <id> \
  --metering-mode response_field \
  --metering-expr "respBody.usage.total_tokens"
```

Response example:
```json
{
  "result": "...",
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150
  }
}
```

---

## Custom Expressions

Complex metering logic:

```bash
# Charge based on model tier
apigate routes update <id> \
  --metering-mode custom \
  --metering-expr "respBody.model == 'gpt-4' ? respBody.tokens * 2 : respBody.tokens"
```

### Expression Context

Available variables:

| Variable | Type | Description |
|----------|------|-------------|
| `status` | int | Response status code |
| `requestBytes` | int | Request body size |
| `responseBytes` | int | Response body size |
| `respBody` | any | Parsed JSON response |
| `path` | string | Request path |
| `method` | string | HTTP method |

### SSE/Streaming Context

For streaming responses:

| Variable | Type | Description |
|----------|------|-------------|
| `sseEvents` | int | Number of SSE events |
| `allData` | []byte | Accumulated data |

---

## Plan Configuration

Configure metering at plan level:

```bash
apigate plans update <id> \
  --meter-type compute_units \
  --estimated-cost-per-req 10
```

---

---

## External Event Ingestion

External services (like downstream applications) can submit usage events directly to APIGate for billing purposes. This enables tracking usage that doesn't pass through the proxy.

### Use Cases

- **Deployment lifecycle** - Track when deployments start/stop
- **Compute time** - Report compute minutes used
- **Storage usage** - Report storage GB-hours
- **Custom resources** - Any billable resource

### API Endpoint

```
POST /api/v1/meter
Authorization: Bearer <service-api-key>
```

External events require a **service API key** with `meter:write` scope. See [[API-Keys#service-api-keys]] for details.

### Event Format

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

### Event Types

| Event Type | Description |
|------------|-------------|
| `api.request` | API request (for external API calls) |
| `deployment.created` | Deployment created |
| `deployment.started` | Deployment started running |
| `deployment.stopped` | Deployment stopped |
| `compute.minutes` | Compute time in minutes |
| `storage.gb_hours` | Storage in GB-hours |
| `bandwidth.gb` | Data transfer in GB |
| `custom.*` | Custom event types |

### Billing Integration

External events count toward the user's monthly quota:
- Events are aggregated per billing period
- `quantity` field multiplied by cost multiplier
- Total added to user's usage count
- Quota warnings/enforcement apply as configured

See [[Metering-API]] for full specification.

---

## See Also

- [[Routes]] - Route configuration
- [[Quotas]] - Quota enforcement
- [[Usage-Tracking]] - Usage recording
- [[Plans]] - Plan metering settings
- [[Metering-API]] - External event ingestion API
