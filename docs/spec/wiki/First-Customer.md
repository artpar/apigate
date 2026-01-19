# First Customer

Guide to onboarding your first API customer.

---

## Prerequisites

1. APIGate is installed and running
2. At least one plan exists
3. Upstream configured

---

## Step 1: Create User Account

### Via Admin UI

1. Go to **Users** > **New User**
2. Enter email and name
3. Select plan
4. Click **Create**

### Via CLI

```bash
apigate users create \
  --email customer@example.com \
  --name "First Customer" \
  --plan pro
```

---

## Step 2: Create API Key

### Via Admin UI

1. Go to user's detail page
2. Click **Create API Key**
3. Enter name (e.g., "Production Key")
4. Copy the key (shown only once!)

### Via CLI

```bash
apigate keys create \
  --user <user-id> \
  --name "Production Key"
```

---

## Step 3: Share with Customer

Send customer:
1. API key
2. API documentation URL (`/docs`)
3. Base URL for API calls

Example email:

```
Welcome to our API!

Your API Key: ak_xxx...

Base URL: https://api.example.com

Documentation: https://api.example.com/docs

Quick start:
curl -H "X-API-Key: ak_xxx" https://api.example.com/v1/resource
```

---

## Step 4: Verify Usage

After customer makes requests:

```bash
# Check usage summary
apigate usage summary --user <user-id>

# View recent requests
apigate usage recent --user <user-id> --limit 20
```

---

## Customer Self-Service

For customer self-registration:

1. Enable customer portal
2. Share signup URL: `/portal/signup`
3. Customers create own account and keys

See [[Customer-Portal]] for setup.

---

## See Also

- [[Quick-Start]] - Initial setup
- [[Users]] - User management
- [[API-Keys]] - Key management
- [[Customer-Portal]] - Self-service portal
