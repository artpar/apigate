# Usage Tracking

**Usage tracking** records every API request for analytics, billing, and monitoring.

---

## Overview

Every request through APIGate is tracked:

```
┌────────────────────────────────────────────────────────────────┐
│                    Usage Tracking Flow                          │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Request arrives                                                │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │              Request Processing                      │       │
│  │  • Route matching                                    │       │
│  │  • Authentication                                    │       │
│  │  • Proxy to upstream                                 │       │
│  └─────────────────────────────────────────────────────┘       │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │              Usage Recording (async)                 │       │
│  │                                                      │       │
│  │  • User ID          • Response status               │       │
│  │  • API Key ID       • Response time (ms)            │       │
│  │  • Route ID         • Request bytes                 │       │
│  │  • Method           • Response bytes                │       │
│  │  • Path             • Metered units                 │       │
│  │  • Timestamp        • IP address                    │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## What's Tracked

### Request Metadata

| Field | Description |
|-------|-------------|
| `user_id` | Authenticated user |
| `api_key_id` | API key used |
| `route_id` | Matched route |
| `method` | HTTP method (GET, POST, etc.) |
| `path` | Request path |
| `query_params` | Query string (optional) |
| `timestamp` | Request time |

### Response Metadata

| Field | Description |
|-------|-------------|
| `status_code` | HTTP status (200, 404, etc.) |
| `latency_ms` | Response time |
| `request_bytes` | Request body size |
| `response_bytes` | Response body size |
| `error_code` | Error code if failed |

### Metering Data

| Field | Description |
|-------|-------------|
| `metered_units` | Usage units (mode-dependent) |
| `metering_mode` | How units calculated |

---

## Metering Modes

### Request Count (Default)

Each request = 1 unit.

```yaml
route:
  metering_mode: request
```

### Byte Count

Units based on data transfer.

```yaml
route:
  metering_mode: bytes
```

Formula: `(request_bytes + response_bytes) / 1024` (KB)

### Response Field

Extract units from response JSON.

```yaml
route:
  metering_mode: response_field
  metering_expr: "response.body.tokens_used"
```

Useful for AI APIs that report token usage.

### Custom Expression

Calculate custom units.

```yaml
route:
  metering_mode: custom
  metering_expr: "request.body.items.length * 0.5"
```

---

## Viewing Usage

### Admin UI

1. Go to **Analytics** in sidebar
2. View:
   - Total requests (by period)
   - Requests by user
   - Requests by route
   - Error rates
   - Latency percentiles

### CLI

```bash
# Summary
apigate analytics summary

# By user
apigate analytics users --sort requests --limit 10

# By route
apigate analytics routes --sort requests --limit 10

# By time period
apigate analytics requests --from 2025-01-01 --to 2025-01-31
```

### API

```bash
# Summary
curl http://localhost:8080/admin/analytics/summary

# User usage
curl http://localhost:8080/admin/analytics/users

# Time series
curl "http://localhost:8080/admin/analytics/requests?from=2025-01-01&to=2025-01-31"
```

---

## User Usage

### Get User's Usage

```bash
# CLI
apigate users usage <user-id>

# API
curl http://localhost:8080/admin/users/<id>/usage
```

Response:
```json
{
  "data": {
    "type": "usage",
    "attributes": {
      "period": "2025-01",
      "total_requests": 8234,
      "total_bytes": 45234567,
      "total_metered_units": 8234,
      "quota_limit": 10000,
      "quota_used_percent": 82.34,
      "by_route": [
        {"route_id": "...", "route_name": "users-api", "requests": 5000},
        {"route_id": "...", "route_name": "orders-api", "requests": 3234}
      ],
      "by_status": {
        "2xx": 8100,
        "4xx": 120,
        "5xx": 14
      }
    }
  }
}
```

### Export Usage

```bash
# CSV export
apigate analytics export \
  --user <user-id> \
  --from 2025-01-01 \
  --to 2025-01-31 \
  --format csv > usage.csv

# JSON export
apigate analytics export \
  --format json > usage.json
```

---

## Aggregations

### Daily Aggregates

Automatic daily rollups:

```bash
apigate analytics daily --user <id>
```

### Monthly Aggregates

For billing:

```bash
apigate analytics monthly --user <id>
```

### Custom Aggregations

```bash
# Requests by hour
apigate analytics aggregate \
  --group-by hour \
  --from 2025-01-19

# Requests by endpoint
apigate analytics aggregate \
  --group-by path \
  --user <id>
```

---

## Real-Time Monitoring

### Live Dashboard

```bash
# Watch requests in real-time
apigate analytics live

# Filter by user
apigate analytics live --user <id>

# Filter by status
apigate analytics live --status 5xx
```

### Metrics Endpoint

```bash
curl http://localhost:8080/metrics

# Prometheus format
# apigate_requests_total{method="GET",status="200"} 12345
# apigate_request_duration_seconds{quantile="0.99"} 0.234
# apigate_active_connections 42
```

---

## Retention

### Configure Retention

```bash
# Keep detailed logs for 30 days
apigate settings set usage_retention_days 30

# Keep aggregates for 365 days
apigate settings set usage_aggregate_retention_days 365
```

### Cleanup

```bash
# Manual cleanup
apigate analytics cleanup --older-than 30d

# Automatic (runs daily)
APIGATE_USAGE_CLEANUP_ENABLED=true
```

---

## Privacy

### Anonymization

```bash
# Don't store IP addresses
apigate settings set usage_store_ip false

# Don't store paths
apigate settings set usage_store_path false
```

### Data Export (GDPR)

```bash
# Export user's data
apigate users export-data <user-id> > user-data.json
```

### Data Deletion

```bash
# Delete user's usage data
apigate users delete-usage <user-id>
```

---

## Billing Integration

### Usage-Based Billing

Track usage for billing:

```bash
# Get billable usage
apigate billing usage \
  --user <id> \
  --period 2025-01
```

Response:
```json
{
  "user_id": "usr_xxx",
  "period": "2025-01",
  "plan": "Pro",
  "included_units": 100000,
  "used_units": 115000,
  "overage_units": 15000,
  "overage_rate_cents": 0.1,
  "overage_amount_cents": 1500
}
```

### Send to Payment Provider

```bash
# Report usage to Stripe
apigate billing report-usage --period 2025-01
```

---

## Performance

### Async Recording

Usage is recorded asynchronously to minimize latency:
- Request completes immediately
- Usage written to buffer
- Buffer flushed periodically (default: 1 second)

### Batch Writes

Multiple usage records written in batches:

```bash
# Configure batch size
apigate settings set usage_batch_size 100

# Configure flush interval
apigate settings set usage_flush_interval_ms 1000
```

### Storage Optimization

```bash
# Enable compression
apigate settings set usage_compression true

# Use separate database
APIGATE_USAGE_DATABASE_PATH=/data/usage.db
```

---

## Troubleshooting

### Usage Not Recording

1. Check tracking enabled:
   ```bash
   apigate settings get usage_tracking_enabled
   ```

2. Check disk space for database

3. Check logs for errors:
   ```bash
   apigate logs --filter usage
   ```

### Incorrect Counts

1. Check metering mode on route
2. Verify time zone settings
3. Check for duplicate records

### Performance Impact

1. Increase batch size
2. Increase flush interval
3. Use separate database for usage

---

## See Also

- [[Quotas]] - Monthly limits
- [[Rate-Limiting]] - Request frequency
- [[Analytics]] - Dashboards and reports
- [[Billing]] - Usage-based billing
