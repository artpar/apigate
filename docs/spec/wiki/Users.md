# Users

**Users** are your API customers - the people or organizations consuming your API.

---

## Overview

Users are the core entity for API access management:

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Relationships                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                          ┌──────────┐                            │
│                          │   User   │                            │
│                          │          │                            │
│                          │ • email  │                            │
│                          │ • status │                            │
│                          └────┬─────┘                            │
│                               │                                  │
│           ┌───────────────────┼───────────────────┐              │
│           │                   │                   │              │
│           ▼                   ▼                   ▼              │
│    ┌──────────┐       ┌──────────┐       ┌──────────┐           │
│    │   Plan   │       │ API Keys │       │  Usage   │           │
│    │          │       │          │       │  Events  │           │
│    │ • limits │       │ • prefix │       │ • method │           │
│    │ • price  │       │ • scopes │       │ • path   │           │
│    └──────────┘       └──────────┘       └──────────┘           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## User Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `email` | string | Email address (required, unique) |
| `name` | string | Display name |
| `password_hash` | string | Hashed password (internal) |
| `plan_id` | string | Assigned plan |
| `status` | enum | active, suspended, pending |
| `role` | enum | admin, customer |
| `metadata` | object | Custom key-value data |
| `stripe_customer_id` | string | Stripe customer reference |
| `email_verified_at` | timestamp | Email verification time |
| `created_at` | timestamp | Registration time |
| `last_login_at` | timestamp | Last login time |

---

## User Statuses

| Status | Description | API Access |
|--------|-------------|------------|
| `active` | Normal state | Full access |
| `pending` | Awaiting verification | Limited |
| `suspended` | Account suspended | Blocked |

---

## Creating Users

### Admin UI

1. Go to **Users** in sidebar
2. Click **Add User**
3. Fill in:
   - **Email**: User's email
   - **Name**: Display name (optional)
   - **Plan**: Select a plan
4. Click **Save**

### Customer Self-Registration

Customers register via the portal:

1. Visit `/portal/register`
2. Enter email and password
3. Verify email (if configured)
4. Automatically assigned default plan

### CLI

```bash
# Create customer
apigate users create \
  --email "customer@example.com" \
  --name "Acme Corp" \
  --plan "free"

# Create admin
apigate users create \
  --email "admin@company.com" \
  --role admin
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@example.com",
    "name": "Acme Corp",
    "plan_id": "plan-id-here",
    "status": "active"
  }'
```

---

## User Roles

### Customer (Default)

- Can access customer portal
- Can manage own API keys
- Can view own usage
- Cannot access admin UI

### Admin

- Full access to admin UI
- Can manage all users
- Can configure system
- Can view all analytics

```bash
# Promote to admin
apigate users update <id> --role admin

# Demote to customer
apigate users update <id> --role customer
```

---

## Managing Users

### List Users

```bash
# CLI
apigate users list
apigate users list --plan "pro"
apigate users list --status "active"

# API
curl http://localhost:8080/admin/users
curl "http://localhost:8080/admin/users?plan_id=xxx"
```

### Get User

```bash
# CLI
apigate users get <id>

# API
curl http://localhost:8080/admin/users/<id>
```

### Update User

```bash
# CLI
apigate users update <id> --plan "pro"
apigate users update <id> --name "New Name"

# API
curl -X PUT http://localhost:8080/admin/users/<id> \
  -H "Content-Type: application/json" \
  -d '{"plan_id": "new-plan-id"}'
```

### Suspend User

```bash
# CLI
apigate users suspend <id>

# API
curl -X POST http://localhost:8080/admin/users/<id>/suspend
```

All API keys immediately stop working.

### Reactivate User

```bash
# CLI
apigate users activate <id>

# API
curl -X POST http://localhost:8080/admin/users/<id>/activate
```

### Delete User

```bash
# CLI
apigate users delete <id>

# API
curl -X DELETE http://localhost:8080/admin/users/<id>
```

**Warning**: Deleting a user:
- Revokes all their API keys
- Removes their usage history
- Cancels any subscriptions

---

## Password Management

### Set Password (Admin)

```bash
apigate users set-password <id> --password "new-password"
```

### Password Reset Flow

1. User requests reset: `POST /auth/forgot-password`
2. Email sent with reset token
3. User clicks link: `/auth/reset-password?token=xxx`
4. User sets new password

### Password Requirements

- Minimum 8 characters
- Configurable via settings

---

## Email Verification

### Enable Verification

```bash
apigate settings set require_email_verification true
```

### Verification Flow

1. User registers
2. Verification email sent
3. User clicks link
4. `email_verified_at` set
5. Full access granted

### Manual Verification

```bash
apigate users verify <id>
```

---

## Custom Metadata

Store additional user data:

```bash
# CLI
apigate users update <id> --metadata '{"company_size": "10-50", "industry": "fintech"}'

# API
curl -X PUT http://localhost:8080/admin/users/<id> \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "company_size": "10-50",
      "industry": "fintech",
      "signup_source": "google_ads"
    }
  }'
```

Metadata is passed to upstream in headers:
```
X-User-Meta-Company-Size: 10-50
X-User-Meta-Industry: fintech
```

---

## User Import/Export

### Export Users

```bash
# CSV export
apigate users export --format csv > users.csv

# JSON export
apigate users export --format json > users.json
```

### Import Users

```bash
# From CSV
apigate users import users.csv

# CSV format:
# email,name,plan_name
# john@example.com,John Doe,pro
# jane@example.com,Jane Smith,free
```

---

## User Metrics

### Per-User Stats

```bash
apigate users stats <id>
```

Returns:
- Total requests (this month)
- Quota usage percentage
- API keys count
- Last active timestamp

### Usage by User

```bash
# Top users by requests
apigate analytics users --sort requests --limit 10

# Users approaching quota
apigate analytics users --filter "quota_percent > 80"
```

---

## Integration with Payment Providers

### Stripe

When Stripe webhook received:
1. Customer created → User created
2. Subscription active → Plan assigned
3. Subscription canceled → Plan downgraded

```bash
# Link existing user to Stripe
apigate users update <id> --stripe-customer-id "cus_xxx"
```

### Paddle / LemonSqueezy

Similar webhook-driven flows.

---

## Best Practices

### 1. Use Email as Identifier

```bash
# Good - unique, verifiable
apigate users create --email "user@company.com"

# Avoid - manual ID management
apigate users create --id "user-123" --email "user@company.com"
```

### 2. Set Default Plan

Always have a default plan for self-registered users:

```bash
apigate plans create --name "Free" --default true
```

### 3. Monitor Inactive Users

```bash
# Find users inactive > 30 days
apigate users list --inactive-days 30
```

### 4. Use Metadata Wisely

Track business-relevant data:
- Signup source
- Company size
- Industry
- Account manager

---

## See Also

- [[API-Keys]] - User's API keys
- [[Plans]] - Assign plans to users
- [[Customer-Portal]] - Self-service interface
- [[Authentication]] - Login flows
