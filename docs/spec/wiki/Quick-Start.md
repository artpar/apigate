# Quick Start

Get APIGate running and proxy your first API in 5 minutes.

---

## Step 1: Start APIGate

```bash
./apigate serve
```

You'll see:
```
APIGate starting...
Admin UI:    http://localhost:8080/ui
Portal:      http://localhost:8080/portal
Docs:        http://localhost:8080/docs
API:         http://localhost:8080/api
```

---

## Step 2: Complete Setup

1. Open http://localhost:8080 in your browser
2. You'll be redirected to the setup wizard
3. Create your admin account:
   - Email: `admin@example.com`
   - Password: `your-secure-password`

---

## Step 3: Create an Upstream

An **upstream** is your backend API that APIGate will proxy requests to.

### Via Admin UI

1. Go to **Upstreams** in the sidebar
2. Click **Add Upstream**
3. Fill in:
   - **Name**: `my-api`
   - **Base URL**: `https://api.example.com`
4. Click **Save**

### Via CLI

```bash
./apigate upstreams create \
  --name "my-api" \
  --url "https://api.example.com"
```

### Via API

```bash
curl -X POST http://localhost:8080/api/upstreams \
  -H "Content-Type: application/vnd.api+json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "data": {
      "type": "upstreams",
      "attributes": {
        "name": "my-api",
        "base_url": "https://api.example.com"
      }
    }
  }'
```

---

## Step 4: Create a Route

A **route** maps incoming requests to your upstream.

### Via Admin UI

1. Go to **Routes** in the sidebar
2. Click **Add Route**
3. Fill in:
   - **Name**: `api-v1`
   - **Path Pattern**: `/v1/*`
   - **Match Type**: `prefix`
   - **Upstream**: Select `my-api`
4. Click **Save**

### Via CLI

```bash
./apigate routes create \
  --name "api-v1" \
  --path "/v1/*" \
  --upstream "my-api"
```

---

## Step 5: Create a Plan

A **plan** defines rate limits and quotas for your customers.

### Via Admin UI

1. Go to **Plans** in the sidebar
2. Click **Add Plan**
3. Fill in:
   - **Name**: `Free`
   - **Rate Limit**: `60` requests/minute
   - **Monthly Quota**: `1000` requests
   - **Price**: `0` (free plan)
4. Click **Save**

---

## Step 6: Create a Test User

### Via Admin UI

1. Go to **Users** in the sidebar
2. Click **Add User**
3. Fill in:
   - **Email**: `test@example.com`
   - **Plan**: Select `Free`
4. Click **Save**

---

## Step 7: Create an API Key

### Via Admin UI

1. Go to **API Keys** in the sidebar
2. Click **Add Key**
3. Fill in:
   - **User**: Select `test@example.com`
   - **Name**: `Test Key`
4. Click **Save**
5. **Copy the API key** - it's only shown once!

The key looks like: `ak_abc123def456...`

---

## Step 8: Test Your API

```bash
# Make a request through APIGate
curl -H "X-API-Key: ak_YOUR_KEY_HERE" \
  http://localhost:8080/v1/endpoint

# Check rate limit headers in response:
# X-RateLimit-Limit: 60
# X-RateLimit-Remaining: 59
# X-RateLimit-Reset: 1704067200
```

---

## What Just Happened?

```
┌──────────┐     ┌──────────┐     ┌──────────────┐
│  Client  │────▶│ APIGate  │────▶│ Your Backend │
│          │     │          │     │              │
│ API Key  │     │ • Auth   │     │ api.example  │
│ X-API-Key│     │ • Rate   │     │ .com/v1/...  │
└──────────┘     │ • Usage  │     └──────────────┘
                 └──────────┘
```

1. Client sends request with API key
2. APIGate validates the key
3. APIGate checks rate limit
4. APIGate forwards to upstream
5. APIGate records usage
6. Response returned to client

---

## Next Steps

- [[Tutorial-Monetization]] - Add paid plans
- [[Rate-Limiting]] - Configure rate limits
- [[Customer-Portal]] - Let customers self-serve
- [[API-Documentation]] - Auto-generate docs
