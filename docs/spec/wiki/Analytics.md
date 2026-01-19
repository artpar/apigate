# Analytics

APIGate provides usage tracking and Prometheus metrics for monitoring.

---

## Overview

Analytics features:
- Request usage tracking per user/plan
- Prometheus metrics endpoint
- Admin API for usage statistics

---

## Viewing Usage Statistics

### Admin API

Get usage statistics via the admin API:

```bash
# Get usage summary (default: last month)
curl http://localhost:8080/admin/usage \
  -H "Authorization: Bearer <session_id>"

# Filter by period
curl http://localhost:8080/admin/usage?period=week

# Filter by user
curl http://localhost:8080/admin/usage?user_id=user_xxx

# Custom date range
curl "http://localhost:8080/admin/usage?start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z"
```

### Response Format

```json
{
  "meta": {
    "period": "month",
    "start_date": "2024-01-01T00:00:00Z",
    "end_date": "2024-02-01T00:00:00Z",
    "summary": {
      "total_requests": 15420,
      "total_users": 45,
      "total_keys": 67
    },
    "by_user": [
      {
        "user_id": "user_abc123",
        "email": "customer@example.com",
        "plan_id": "pro",
        "requests": 5200,
        "bytes_in": 1048576,
        "bytes_out": 2097152
      }
    ],
    "by_plan": [
      {
        "plan_id": "pro",
        "plan_name": "Pro",
        "user_count": 12,
        "requests": 8500
      }
    ]
  }
}
```

---

## Prometheus Metrics

Enable the Prometheus metrics endpoint for monitoring.

### Configuration

```bash
# Enable metrics endpoint
APIGATE_METRICS_ENABLED=true

# Custom path (default: /metrics)
APIGATE_METRICS_PATH=/metrics
```

### Metrics Endpoint

Access at `http://localhost:8080/metrics`

### Available Metrics

```prometheus
# Request metrics
apigate_requests_total{method="GET", path="/api/users", status="200", plan_id="pro"}
apigate_request_duration_seconds{method="GET", path="/api/users", status="200"}
apigate_requests_in_flight

# Authentication metrics
apigate_auth_failures_total{reason="invalid_key"}

# Rate limiting metrics
apigate_rate_limit_hits_total{plan_id="free", user_id="user_xxx"}
apigate_rate_limit_tokens{plan_id="free", user_id="user_xxx"}

# Usage metrics
apigate_usage_requests_total{user_id="user_xxx", plan_id="pro"}
apigate_usage_bytes_total{user_id="user_xxx", plan_id="pro", direction="in"}

# Upstream metrics
apigate_upstream_duration_seconds{method="GET", status="200"}
apigate_upstream_errors_total{type="timeout"}
apigate_upstream_requests_in_flight

# Configuration metrics
apigate_config_reloads_total
apigate_config_reload_errors_total
apigate_config_last_reload_timestamp
```

### Grafana Integration

Import a Grafana dashboard to visualize metrics:

1. Add Prometheus as a data source in Grafana
2. Create dashboards using the `apigate_*` metrics
3. Set up alerts for error rates or latency thresholds

---

## Web Dashboard

The admin web UI at `/dashboard` shows:
- Request volume graphs
- Usage by user
- Usage by plan

---

## See Also

- [[Usage-Metering]] - How usage is recorded
- [[Plans]] - Plan-based quota management
- [[Configuration]] - Environment variables
