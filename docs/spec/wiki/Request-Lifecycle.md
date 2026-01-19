# Request Lifecycle

This page describes the complete journey of an API request through APIGate.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Request Lifecycle                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Client Request                                                │
│        │                                                        │
│        ▼                                                        │
│   ┌─────────────┐                                               │
│   │ 1. Extract  │  X-API-Key header or Authorization Bearer    │
│   │    API Key  │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 2. Validate │  Check prefix format                         │
│   │    Format   │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 3. Lookup   │  Find key by prefix in store                 │
│   │    Key      │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 4. Verify   │  bcrypt hash comparison                      │
│   │    Hash     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 5. Validate │  Check expiry, revocation                    │
│   │    Key      │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 6. Check    │  Verify user status is active                │
│   │    User     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 7. Check    │  Monthly request/compute limits              │
│   │    Quota    │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 8. Check    │  Token bucket rate limiting                  │
│   │    Rate     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 9. Resolve  │  Add X-Entitlement-* headers                 │
│   │ Entitlements│                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │10. Match    │  Find matching route by path/method          │
│   │    Route    │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │11. Transform│  Apply request_transform rules               │
│   │    Request  │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │12. Rewrite  │  Apply path_rewrite expression               │
│   │    Path     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │13. Forward  │  Send to upstream (route or default)         │
│   │    Upstream │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │14. Transform│  Apply response_transform rules              │
│   │    Response │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │15. Calculate│  Evaluate metering_expr                      │
│   │    Cost     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │16. Record   │  Store usage event                           │
│   │    Usage    │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   Client Response                                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Step Details

### 1. Extract API Key

API key is extracted from (in order):
1. `X-API-Key` header
2. `Authorization: Bearer {key}` header
3. `api_key` query parameter (if enabled)

```go
// From adapters/http/handler.go
apiKey := extractAPIKey(r)
```

### 2. Validate Format

Check key matches expected prefix format:

```go
// From domain/key/validate.go
prefix, valid := key.ValidateFormat(req.APIKey, s.keyPrefix)
```

Keys must start with configured prefix (default: `ak_`).

### 3. Lookup Key

Find key record by prefix in store:

```go
keys, err := s.keys.Get(ctx, prefix)
```

Uses indexed prefix lookup for efficiency.

### 4. Verify Hash

Compare provided key against stored bcrypt hash:

```go
bcrypt.CompareHashAndPassword(k.Hash, []byte(req.APIKey))
```

### 5. Validate Key

Check key is not expired or revoked:

```go
validation := key.Validate(matchedKey, now)
// Checks: expires_at, revoked_at
```

### 6. Check User

Load user and verify status is `active`:

```go
user, err := s.users.Get(ctx, matchedKey.UserID)
if user.Status != "active" {
    // Return 403 user_suspended
}
```

### 7. Check Quota

Monthly quota check with grace period:

```go
quotaResult = quota.Check(quotaState, quotaCfg, increment)
```

Supports meter types:
- `requests` - Count each request as 1
- `compute_units` - Use estimated cost per request

Returns headers: `X-Quota-Used`, `X-Quota-Limit`, `X-Quota-Reset`

### 8. Check Rate Limit

Token bucket rate limiting:

```go
rlResult, newRLState := ratelimit.Check(rlState, rlConfig, now)
```

Based on plan's `rate_limit_per_minute` setting.

Returns headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`

### 9. Resolve Entitlements

Add entitlement headers based on user's plan:

```go
userEntitlements := entitlement.ResolveForPlan(
    user.PlanID,
    dynCfg.Entitlements,
    dynCfg.PlanEntitlements,
)
entitlementHeaders := entitlement.ToHeaders(userEntitlements)
```

Adds headers like `X-Entitlement-Webhooks: true`.

### 10. Match Route

Find matching route by path and method:

```go
match := s.routeService.Match(req.Method, req.Path, req.Headers)
```

Routes are matched by:
- `path_pattern` with `match_type` (exact, prefix, regex)
- `methods` array
- `headers` conditions
- `priority` (higher matches first)

### 11. Transform Request

Apply request transformations if route defines them:

```go
req, err = s.transformService.TransformRequest(ctx, req, matchedRoute.RequestTransform, &auth)
```

Can modify headers, body, add authentication.

### 12. Rewrite Path

Apply path rewriting expression:

```go
newPath, err := s.transformService.EvalString(ctx, matchedRoute.PathRewrite, rewriteCtx)
```

Context includes `path`, `pathParams`, `method`.

### 13. Forward to Upstream

Send request to backend service:

```go
// Route's upstream if matched, else default
if routeUpstream != nil {
    resp, err = s.upstream.ForwardTo(ctx, req, routeUpstream)
} else {
    resp, err = s.upstream.Forward(ctx, req)
}
```

### 14. Transform Response

Apply response transformations:

```go
resp, err = s.transformService.TransformResponse(ctx, resp, matchedRoute.ResponseTransform, &auth)
```

Can modify headers, body, status code.

### 15. Calculate Cost

Evaluate metering expression:

```go
val, err := s.transformService.EvalFloat(ctx, matchedRoute.MeteringExpr, meteringCtx)
```

Context includes:
- `status` - Response status code
- `responseBytes` - Response body size
- `requestBytes` - Request body size
- `respBody` - Parsed JSON response (if applicable)

### 16. Record Usage

Store usage event for billing/analytics:

```go
s.usage.Record(ctx, usage.Event{
    KeyID:     auth.KeyID,
    UserID:    auth.UserID,
    Path:      req.Path,
    Method:    req.Method,
    Status:    resp.Status,
    Cost:      costMult,
    Timestamp: now,
})
```

---

## Error Responses

Errors can occur at various stages:

| Stage | Error Code | HTTP Status |
|-------|------------|-------------|
| Extract Key | `missing_api_key` | 401 |
| Validate Format | `invalid_api_key` | 401 |
| Lookup Key | `invalid_api_key` | 401 |
| Verify Hash | `invalid_api_key` | 401 |
| Validate Key | `key_expired` / `key_revoked` | 401 |
| Check User | `user_suspended` | 403 |
| Check Quota | `quota_exceeded` | 402 |
| Check Rate | `rate_limit_exceeded` | 429 |
| Transform | `transform_error` | 500 |
| Forward | `upstream_error` | 502 |

---

## Streaming Requests

For SSE/streaming protocols, the lifecycle is similar but:

1. Response is streamed chunk-by-chunk
2. Usage is recorded after stream completes
3. Metering can access accumulated stream data

```go
if h.service.ShouldStream(req) {
    h.handleStreamingRequest(w, r, ctx, req)
    return
}
```

---

## See Also

- [[Routes]] - Route configuration
- [[Transformations]] - Transform rules
- [[Rate-Limiting]] - Rate limit details
- [[Quotas]] - Quota configuration
