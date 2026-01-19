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

## See Also

- [[Routes]] - Route configuration
- [[Quotas]] - Quota enforcement
- [[Usage-Tracking]] - Usage recording
- [[Plans]] - Plan metering settings
