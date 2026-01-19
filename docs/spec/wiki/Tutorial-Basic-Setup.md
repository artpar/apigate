# Tutorial: Basic Setup

Set up APIGate from scratch and proxy your first API in 15 minutes.

---

## Prerequisites

- APIGate binary downloaded
- Terminal access
- A backend API to proxy (or use our test API)

---

## Step 1: Start APIGate

Create a directory and start APIGate:

```bash
mkdir apigate-demo
cd apigate-demo

./apigate serve
```

You'll see:
```
APIGate v1.0.0 starting...
Database: ./data/apigate.db (created)
Admin UI:    http://localhost:8080/ui
Portal:      http://localhost:8080/portal
API:         http://localhost:8080/api
```

---

## Step 2: Complete Setup Wizard

1. Open http://localhost:8080 in your browser
2. You'll see the setup wizard
3. Create your admin account:
   - **Email**: `admin@example.com`
   - **Password**: Choose a strong password
4. Click **Complete Setup**

You're now logged into the Admin UI.

---

## Step 3: Create an Upstream

An upstream is your backend API. Let's create one.

### Using a Test API

We'll proxy JSONPlaceholder (a free test API):

1. Click **Upstreams** in the sidebar
2. Click **Add Upstream**
3. Fill in:
   - **Name**: `jsonplaceholder`
   - **Base URL**: `https://jsonplaceholder.typicode.com`
4. Click **Save**

### Using Your Own API

If you have your own API:

1. Click **Upstreams** → **Add Upstream**
2. Fill in:
   - **Name**: `my-api`
   - **Base URL**: `https://api.yourservice.com`
   - **Auth Type**: If your API requires auth
   - **Auth Value**: Your API key/token
3. Click **Save**

---

## Step 4: Create a Route

Routes map incoming requests to upstreams.

1. Click **Routes** in the sidebar
2. Click **Add Route**
3. Fill in:
   - **Name**: `public-api`
   - **Path Pattern**: `/api/*`
   - **Match Type**: `prefix`
   - **Upstream**: Select `jsonplaceholder`
4. Click **Save**

Now requests to `http://localhost:8080/api/*` will proxy to JSONPlaceholder.

---

## Step 5: Create a Plan

Plans define rate limits and quotas for customers.

1. Click **Plans** in the sidebar
2. Click **Add Plan**
3. Fill in:
   - **Name**: `Free`
   - **Price**: `0` (free tier)
   - **Rate Limit**: `60` requests per minute
   - **Monthly Quota**: `1000` requests
4. Check **Default Plan** (new users get this plan)
5. Click **Save**

---

## Step 6: Create a Test User

1. Click **Users** in the sidebar
2. Click **Add User**
3. Fill in:
   - **Email**: `test@example.com`
   - **Name**: `Test User`
   - **Plan**: Select `Free`
4. Click **Save**

---

## Step 7: Create an API Key

1. Click **API Keys** in the sidebar
2. Click **Add Key**
3. Fill in:
   - **User**: Select `test@example.com`
   - **Name**: `Test Key`
4. Click **Save**
5. **IMPORTANT**: Copy the API key shown - it's only displayed once!

The key looks like: `ak_abc123def456...`

---

## Step 8: Test Your Setup

Open a terminal and test with curl:

```bash
# Replace YOUR_API_KEY with the key you copied
API_KEY="ak_your_key_here"

# Make a request through APIGate
curl -H "X-API-Key: $API_KEY" \
  http://localhost:8080/api/users/1
```

You should get a JSON response:
```json
{
  "id": 1,
  "name": "Leanne Graham",
  "username": "Bret",
  "email": "Sincere@april.biz"
  ...
}
```

Check the response headers:
```bash
curl -i -H "X-API-Key: $API_KEY" \
  http://localhost:8080/api/users/1
```

You'll see rate limit headers:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
X-RateLimit-Reset: 1704067260
```

---

## Step 9: View Usage

Back in the Admin UI:

1. Click **Analytics** in the sidebar
2. You'll see your test request recorded
3. Click on the user to see their usage details

---

## Step 10: Test Rate Limiting

Let's verify rate limiting works:

```bash
# Make 65 requests quickly
for i in {1..65}; do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -H "X-API-Key: $API_KEY" \
    http://localhost:8080/api/users/1
done
```

You'll see:
- First 60 requests: `200`
- Remaining requests: `429` (rate limited)

---

## What You've Built

```
┌─────────────────────────────────────────────────────────────┐
│                        Your API Gateway                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Client Request                                              │
│  curl -H "X-API-Key: ak_xxx" localhost:8080/api/users/1     │
│       │                                                      │
│       ▼                                                      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   APIGate                            │    │
│  │  ✓ Authenticate API key                             │    │
│  │  ✓ Check rate limit (60/min)                        │    │
│  │  ✓ Check quota (1000/month)                         │    │
│  │  ✓ Match route (/api/*)                             │    │
│  └─────────────────────────────────────────────────────┘    │
│       │                                                      │
│       ▼                                                      │
│  Upstream: jsonplaceholder.typicode.com/users/1             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Next Steps

Now that you have a basic setup:

1. **[[Tutorial-Monetization]]** - Add paid plans and billing
2. **[[Tutorial-Stripe]]** - Integrate Stripe payments
3. **[[Customer-Portal]]** - Let customers self-serve
4. **[[Transformations]]** - Modify requests/responses
5. **[[Tutorial-Production]]** - Deploy to production

---

## Troubleshooting

### "Unauthorized" Error

- Check the API key is correct
- Verify key hasn't expired
- Ensure user isn't suspended

### "Not Found" Error

- Check route path pattern matches your request
- Verify route is enabled
- Check upstream URL is correct

### "Rate Limited" but Count Seems Wrong

- Rate limits are per minute
- Wait 60 seconds and try again
- Check if route has its own rate limit

### Upstream Timeout

- Check upstream URL is accessible
- Verify network connectivity
- Try increasing timeout in upstream settings

---

## Complete CLI Version

If you prefer CLI over UI:

```bash
# Start APIGate
./apigate serve &

# Create admin user (you'll be prompted for password)
./apigate admin create --email admin@example.com

# Create upstream
./apigate upstreams create \
  --name "jsonplaceholder" \
  --url "https://jsonplaceholder.typicode.com"

# Create route
./apigate routes create \
  --name "public-api" \
  --path "/api/*" \
  --upstream "jsonplaceholder"

# Create plan
./apigate plans create \
  --id "free" \
  --name "Free" \
  --rate-limit 60 \
  --requests 1000 \
  --price 0 \
  --default

# Create user
./apigate users create \
  --email "test@example.com" \
  --name "Test User" \
  --plan "free"

# Create API key
./apigate keys create \
  --user "test@example.com" \
  --name "Test Key"
# Output: ak_abc123def456...

# Test it
curl -H "X-API-Key: ak_abc123def456..." \
  http://localhost:8080/api/users/1
```

---

## Summary

In this tutorial, you:

1. ✅ Started APIGate
2. ✅ Completed setup wizard
3. ✅ Created an upstream (backend API)
4. ✅ Created a route (URL mapping)
5. ✅ Created a plan (rate limits & quota)
6. ✅ Created a user (API customer)
7. ✅ Created an API key (authentication)
8. ✅ Tested the complete flow
9. ✅ Verified rate limiting works

You now have a fully functional API gateway!
