# Entitlements

**Entitlements** are feature flags that control what capabilities are available to users based on their plan.

---

## Overview

Entitlements enable granular feature access control:

```
┌────────────────────────────────────────────────────────────────┐
│                    Entitlement System                           │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐     ┌──────────────────┐     ┌─────────────┐  │
│  │ Entitlement │────▶│ Plan-Entitlement │◀────│    Plan     │  │
│  │             │     │                  │     │             │  │
│  │ • name      │     │ • value          │     │ • Pro       │  │
│  │ • type      │     │ • enabled        │     │ • Enterprise│  │
│  │ • default   │     │                  │     │             │  │
│  └─────────────┘     └──────────────────┘     └─────────────┘  │
│         │                     │                      │          │
│         │                     ▼                      │          │
│         │            ┌──────────────┐               │          │
│         └───────────▶│   Request    │◀──────────────┘          │
│                      │              │                           │
│                      │ X-Entitlement│                           │
│                      │   Headers    │                           │
│                      └──────────────┘                           │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Entitlement Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Entitlement name (required, unique) |
| `display_name` | string | Human-readable name |
| `description` | string | Entitlement description |
| `category` | string | Grouping category (default: "feature") |
| `value_type` | enum | boolean, number, string |
| `default_value` | string | Default value if not set (default: "true") |
| `header_name` | string | Custom header name for upstream |
| `enabled` | bool | Entitlement active |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

---

## Value Types

### Boolean (Default)

Simple on/off feature flags:

```bash
apigate entitlements create \
  --name "webhooks" \
  --display-name "Webhooks" \
  --value-type boolean \
  --default-value "false"
```

Header: `X-Entitlement-Webhooks: true` or `X-Entitlement-Webhooks: false`

### Number

Numeric limits or quantities:

```bash
apigate entitlements create \
  --name "max_api_keys" \
  --display-name "Max API Keys" \
  --value-type number \
  --default-value "3"
```

Header: `X-Entitlement-Max-Api-Keys: 10`

### String

Text values for configuration:

```bash
apigate entitlements create \
  --name "support_tier" \
  --display-name "Support Tier" \
  --value-type string \
  --default-value "community"
```

Header: `X-Entitlement-Support-Tier: priority`

---

## Creating Entitlements

### Admin UI

1. Go to **Entitlements** in sidebar
2. Click **New Entitlement**
3. Fill in:
   - **Name**: Internal identifier (lowercase, underscores)
   - **Display Name**: Human-readable name
   - **Description**: What this entitlement controls
   - **Value Type**: boolean, number, or string
   - **Default Value**: Value when not assigned
4. Click **Create**

### CLI

```bash
# Boolean feature flag
apigate entitlements create \
  --name "advanced_analytics" \
  --display-name "Advanced Analytics" \
  --value-type boolean \
  --default-value "false"

# Numeric limit
apigate entitlements create \
  --name "rate_limit_multiplier" \
  --display-name "Rate Limit Multiplier" \
  --value-type number \
  --default-value "1"

# String configuration
apigate entitlements create \
  --name "data_retention" \
  --display-name "Data Retention Period" \
  --value-type string \
  --default-value "30d"
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/entitlements \
  -H "Content-Type: application/json" \
  -d '{
    "name": "webhooks",
    "display_name": "Webhooks",
    "description": "Enable webhook notifications",
    "value_type": "boolean",
    "default_value": "false",
    "enabled": true
  }'
```

---

## Plan-Entitlements

Link entitlements to plans with specific values:

### Assign to Plan

```bash
# CLI
apigate plan-entitlements create \
  --plan "pro" \
  --entitlement "webhooks" \
  --value "true"

# Different value for enterprise
apigate plan-entitlements create \
  --plan "enterprise" \
  --entitlement "max_api_keys" \
  --value "100"
```

### Plan-Entitlement Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `plan_id` | string | Plan this applies to |
| `entitlement_id` | string | Entitlement being assigned |
| `value` | string | Value for this plan |
| `enabled` | bool | Whether assignment is active |

### Example: Tiered Features

```bash
# Create entitlements
apigate entitlements create --name "webhooks" --value-type boolean --default "false"
apigate entitlements create --name "analytics" --value-type boolean --default "false"
apigate entitlements create --name "max_api_keys" --value-type number --default "3"
apigate entitlements create --name "support_tier" --value-type string --default "community"

# Free plan - minimal features
# (uses defaults: no webhooks, no analytics, 3 keys, community support)

# Pro plan - more features
apigate plan-entitlements create --plan "pro" --entitlement "webhooks" --value "true"
apigate plan-entitlements create --plan "pro" --entitlement "analytics" --value "true"
apigate plan-entitlements create --plan "pro" --entitlement "max_api_keys" --value "10"
apigate plan-entitlements create --plan "pro" --entitlement "support_tier" --value "email"

# Enterprise plan - all features
apigate plan-entitlements create --plan "enterprise" --entitlement "webhooks" --value "true"
apigate plan-entitlements create --plan "enterprise" --entitlement "analytics" --value "true"
apigate plan-entitlements create --plan "enterprise" --entitlement "max_api_keys" --value "unlimited"
apigate plan-entitlements create --plan "enterprise" --entitlement "support_tier" --value "priority"
```

---

## Headers Sent to Upstream

When a request is proxied, APIGate injects entitlement headers:

### Default Header Format

```
X-Entitlement-{Name}: {value}
```

Name is converted to Title-Case with dashes.

### Example Headers

```http
GET /api/users HTTP/1.1
Host: api.example.com
X-API-Key: ak_xxx

# Injected by APIGate:
X-User-ID: usr_abc123
X-User-Plan: pro
X-Entitlement-Webhooks: true
X-Entitlement-Analytics: true
X-Entitlement-Max-Api-Keys: 10
X-Entitlement-Support-Tier: email
```

### Custom Header Names

Override the default header name:

```bash
apigate entitlements create \
  --name "max_api_keys" \
  --header-name "X-Max-Keys"
```

Now sends: `X-Max-Keys: 10` instead of `X-Entitlement-Max-Api-Keys: 10`

---

## Using Entitlements in Your API

### Check Boolean Entitlement

```go
func handler(w http.ResponseWriter, r *http.Request) {
    webhooksEnabled := r.Header.Get("X-Entitlement-Webhooks") == "true"

    if !webhooksEnabled {
        http.Error(w, "Webhooks not available on your plan", 403)
        return
    }

    // Process webhook request...
}
```

### Check Numeric Entitlement

```go
func handler(w http.ResponseWriter, r *http.Request) {
    maxKeys := r.Header.Get("X-Entitlement-Max-Api-Keys")
    limit, _ := strconv.Atoi(maxKeys)

    currentKeys := countUserKeys(userID)
    if currentKeys >= limit {
        http.Error(w, "API key limit reached", 403)
        return
    }

    // Create key...
}
```

### Check String Entitlement

```go
func handler(w http.ResponseWriter, r *http.Request) {
    supportTier := r.Header.Get("X-Entitlement-Support-Tier")

    switch supportTier {
    case "priority":
        routeToPriorityQueue(r)
    case "email":
        routeToEmailQueue(r)
    default:
        routeToCommunityForum(r)
    }
}
```

---

## Admin UI Pages

### Entitlements List (`/entitlements`)

View all entitlements:
- Name and display name
- Value type
- Default value
- Status (enabled/disabled)

### Entitlement Edit (`/entitlements/:id`)

Edit entitlement:
- Display name and description
- Category
- Header name customization
- Enable/disable

### Plan-Entitlements (`/plans/:id`)

On the plan edit page, manage entitlement assignments:
- See all available entitlements
- Set values for this plan
- Enable/disable per-plan

---

## Categories

Organize entitlements with categories:

```bash
# Feature flags
apigate entitlements create --name "webhooks" --category "features"
apigate entitlements create --name "analytics" --category "features"

# Limits
apigate entitlements create --name "max_api_keys" --category "limits"
apigate entitlements create --name "rate_limit_multiplier" --category "limits"

# Support
apigate entitlements create --name "support_tier" --category "support"
apigate entitlements create --name "sla_level" --category "support"
```

Categories help organize the admin UI.

---

## Best Practices

### 1. Use Clear Naming

```bash
# Good - descriptive
apigate entitlements create --name "advanced_analytics"
apigate entitlements create --name "max_api_keys_per_user"

# Bad - ambiguous
apigate entitlements create --name "feature1"
apigate entitlements create --name "limit"
```

### 2. Set Meaningful Defaults

```bash
# Good - restrictive default
apigate entitlements create --name "webhooks" --default "false"

# Then enable for paid plans
apigate plan-entitlements create --plan "pro" --entitlement "webhooks" --value "true"
```

### 3. Use Number Type for Limits

```bash
# Good - numeric
apigate entitlements create --name "max_team_members" --value-type number --default "5"

# Bad - boolean for limits
apigate entitlements create --name "has_team_members" --value-type boolean
```

### 4. Document Entitlements

Use descriptions to explain what each entitlement controls:

```bash
apigate entitlements create \
  --name "advanced_analytics" \
  --description "Enables detailed analytics dashboard with custom reports and export functionality"
```

---

## See Also

- [[Plans]] - Plan management
- [[Routes]] - Request routing
- [[Transformations]] - Header transformation
