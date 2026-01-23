# Request Lifecycle

This page describes the complete journey of an API request through APIGate.

---

## Overview

APIGate processes requests differently based on whether the matched route requires authentication.

### Route Matching First

**Important**: Route matching happens FIRST to determine if authentication is required.

```
┌─────────────────────────────────────────────────────────────────┐
│                      Request Lifecycle                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Client Request                                                │
│        │                                                        │
│        ▼                                                        │
│   ┌─────────────┐                                               │
│   │ 1. Match    │  Find matching route by host/path/method     │
│   │    Route    │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐     No     ┌─────────────────────────┐       │
│   │auth_required├───────────►│ PUBLIC ROUTE PATH       │       │
│   │   = true?   │            │ Skip to step 10         │       │
│   └──────┬──────┘            │ (Transform & Forward)   │       │
│          │ Yes               └─────────────────────────┘       │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 2. Extract  │  X-API-Key header or Authorization Bearer    │
│   │    API Key  │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 3. Validate │  Check prefix format                         │
│   │    Format   │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 4. Lookup   │  Find key by prefix in store                 │
│   │    Key      │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 5. Verify   │  bcrypt hash comparison                      │
│   │    Hash     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 6. Validate │  Check expiry, revocation                    │
│   │    Key      │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 7. Check    │  Verify user status is active                │
│   │    User     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 8. Check    │  Monthly request/compute limits              │
│   │    Quota    │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │ 9. Check    │  Token bucket rate limiting                  │
│   │    Rate     │                                               │
│   └──────┬──────┘                                               │
│          │                                                      │
│          ▼                                                      │
│   ┌─────────────┐                                               │
│   │10. Resolve  │  Add X-Entitlement-* headers                 │
│   │ Entitlements│                                               │
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

### 1. Match Route

Find matching route by host, path, method, and headers:

```go
match := s.routeService.Match(req.Method, req.Path, req.Headers)
```

Routes are matched by:
- `host_pattern` with `host_match_type` (exact, wildcard, regex)
- `path_pattern` with `match_type` (exact, prefix, regex)
- `methods` array
- `headers` conditions
- `priority` (higher matches first)

**If the matched route has `auth_required: false`, skip to step 11 (Transform Request).**

### 2. Extract API Key

API key is extracted from (in order):
1. `X-API-Key` header
2. `Authorization: Bearer {key}` header
3. `api_key` query parameter (if enabled)

```go
// From adapters/http/handler.go
apiKey := extractAPIKey(r)
```

### 3. Validate Format

Check key matches expected prefix format:

```go
// From domain/key/validate.go
prefix, valid := key.ValidateFormat(req.APIKey, s.keyPrefix)
```

Keys must start with configured prefix (default: `ak_`).

### 4. Lookup Key

Find key record by prefix in store:

```go
keys, err := s.keys.Get(ctx, prefix)
```

Uses indexed prefix lookup for efficiency.

### 5. Verify Hash

Compare provided key against stored bcrypt hash:

```go
bcrypt.CompareHashAndPassword(k.Hash, []byte(req.APIKey))
```

### 6. Validate Key

Check key is not expired or revoked:

```go
validation := key.Validate(matchedKey, now)
// Checks: expires_at, revoked_at
```

### 7. Check User

Load user and verify status is `active`:

```go
user, err := s.users.Get(ctx, matchedKey.UserID)
if user.Status != "active" {
    // Return 403 user_suspended
}
```

### 8. Check Quota

Monthly quota check with grace period:

```go
quotaResult = quota.Check(quotaState, quotaCfg, increment)
```

Supports meter types:
- `requests` - Count each request as 1
- `compute_units` - Use estimated cost per request

Returns headers: `X-Quota-Used`, `X-Quota-Limit`, `X-Quota-Reset`

### 9. Check Rate Limit

Token bucket rate limiting:

```go
rlResult, newRLState := ratelimit.Check(rlState, rlConfig, now)
```

Based on plan's `rate_limit_per_minute` setting.

Returns headers: `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`

### 10. Resolve Entitlements

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

## Public Routes (auth_required: false)

When a route has `auth_required: false`, the request follows a shortened path:

```
1. Match Route
   └── auth_required = false
       ▼
11. Transform Request
12. Rewrite Path
13. Forward to Upstream
14. Transform Response
15. Calculate Cost
16. Record Usage (with anonymous user/key)
```

**What's skipped for public routes:**
- API key extraction and validation (steps 2-6)
- User lookup and status check (step 7)
- Quota enforcement (step 8)
- Rate limiting (step 9)
- Entitlement headers (step 10)

**What still applies:**
- Request/response transformations
- Path rewriting
- Upstream authentication (backend credentials injected by transform)
- Usage logging (with `anonymous` user/key IDs)

**Use cases:**
- Reverse proxy for deployed applications
- Health check endpoints
- Webhook receivers
- Static content serving

See [[Routes#Public Routes (No Authentication)]] for configuration details.

---

## Error Responses

Errors can occur at various stages:

| Stage | Error Code | HTTP Status |
|-------|------------|-------------|
| Match Route | `route_not_found` | 404 |
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
