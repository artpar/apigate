# Tutorial: Basic API Setup

Set up APIGate to protect a simple REST API.

---

## Prerequisites

- APIGate installed
- A backend API to protect

---

## Step 1: Start APIGate

```bash
# Start with default settings
apigate serve

# Or with Docker
docker run -p 8080:8080 -p 9090:9090 artpar/apigate
```

Access admin UI at: `http://localhost:9090`

---

## Step 2: Configure Upstream

Point to your backend API:

```bash
# CLI
apigate upstreams create \
  --name "My Backend" \
  --url "http://localhost:3000"

# Or set default upstream via environment variable
export APIGATE_UPSTREAM_URL=http://localhost:3000
```

---

## Step 3: Create a Plan

```bash
apigate plans create \
  --id "free" \
  --name "Free" \
  --rate-limit 60 \
  --requests 1000 \
  --default
```

---

## Step 4: Create a User

```bash
apigate users create \
  --email test@example.com \
  --name "Test User" \
  --plan free
```

---

## Step 5: Create an API Key

```bash
apigate keys create \
  --user <user-id> \
  --name "Test Key"

# Output:
# API Key created: ak_abc123... (save this!)
```

---

## Step 6: Test the API

```bash
# Without key - rejected
curl http://localhost:8080/api/users
# {"error": {"code": "missing_api_key"}}

# With key - success
curl -H "X-API-Key: ak_abc123..." http://localhost:8080/api/users
# [{"id": 1, "name": "John"}]
```

---

## Step 7: Check Usage

```bash
# View user's usage summary
apigate usage summary --user <user-id>

# View in admin UI
open http://localhost:8080/ui/users/<user-id>
```

---

## What's Next?

- [[Routes]] - Add specific routes
- [[Rate-Limiting]] - Configure rate limits
- [[Plans]] - Create more plans
- [[Tutorial-Monetization]] - Add billing

---

## See Also

- [[Quick-Start]] - Quick start guide
- [[Upstreams]] - Upstream configuration
- [[API-Keys]] - Key management
