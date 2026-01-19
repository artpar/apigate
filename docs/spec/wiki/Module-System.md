# Module System

APIGate uses a YAML-based module system to define entities, their schemas, APIs, and CLI commands.

---

## Overview

Modules are defined in `core/modules/*.yaml` and describe:

- **Schema** - Field definitions with types and constraints
- **Actions** - Custom operations beyond CRUD
- **Channels** - HTTP API and CLI exposure
- **Hooks** - Lifecycle event handlers
- **Meta** - Display metadata

```
core/modules/
├── user.yaml
├── plan.yaml
├── api_key.yaml
├── route.yaml
├── upstream.yaml
├── group.yaml
├── entitlement.yaml
├── webhook.yaml
├── certificate.yaml
└── ...
```

---

## Module Structure

### Basic Example

```yaml
# user.yaml
module: user

schema:
  email:         { type: email, unique: true, lookup: true, required: true }
  password_hash: { type: secret, internal: true }
  name:          { type: string, default: "" }
  plan_id:       { type: ref, to: plan, default: "free" }
  status:        { type: enum, values: [pending, active, suspended, cancelled], default: active }

actions:
  suspend:
    set: { status: suspended }
    description: Suspend a user account

  activate:
    set: { status: active }
    description: Activate a user account

channels:
  http:
    serve:
      enabled: true
      base_path: /api/users
      endpoints:
        - { action: list, method: GET, path: "/" }
        - { action: get, method: GET, path: "/{id}" }
        - { action: create, method: POST, path: "/", auth: admin }
        - { action: update, method: PATCH, path: "/{id}", auth: admin }
        - { action: delete, method: DELETE, path: "/{id}", auth: admin }
        - { action: suspend, method: POST, path: "/{id}/suspend", auth: admin }
        - { action: activate, method: POST, path: "/{id}/activate", auth: admin }

  cli:
    serve:
      enabled: true
      command: users
      commands:
        - action: list
          columns: [id, email, name, plan_id, status]
        - action: get
          args:
            - { name: id, required: true }
        - action: create
          flags:
            - { param: email, required: true }
            - { param: name }
            - { param: plan_id, name: plan, default: free }

hooks:
  after_create:
    - emit: user.created

meta:
  description: User accounts
  icon: users
  display_name: Users
  plural: Users
```

---

## Schema Definition

### Field Properties

| Property | Type | Description |
|----------|------|-------------|
| `type` | string | Field type (see types below) |
| `required` | bool | Must be provided on create |
| `unique` | bool | Value must be unique |
| `lookup` | bool | Indexed for fast lookup |
| `default` | any | Default value if not provided |
| `internal` | bool | Hidden from API responses |
| `immutable` | bool | Cannot be changed after create |
| `description` | string | Field documentation |

### Field Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `name: { type: string }` |
| `int` | Integer | `count: { type: int, default: 0 }` |
| `bool` | Boolean | `enabled: { type: bool, default: true }` |
| `email` | Email address (validated) | `email: { type: email }` |
| `timestamp` | Date/time | `created_at: { type: timestamp }` |
| `secret` | Encrypted at rest | `password: { type: secret }` |
| `json` | JSON object/array | `config: { type: json }` |
| `enum` | One of defined values | `status: { type: enum, values: [a, b] }` |
| `ref` | Reference to another module | `user_id: { type: ref, to: user }` |

### Reference Fields

```yaml
# Reference to another module
user_id: { type: ref, to: user, required: true }
plan_id: { type: ref, to: plan }
```

References create foreign key relationships and enable:
- Cascade deletes (configurable)
- API joins/includes
- Validation

---

## Actions

Actions define operations beyond basic CRUD:

### Simple Set Action

```yaml
actions:
  enable:
    set: { enabled: true }
    description: Enable this item

  disable:
    set: { enabled: false }
    description: Disable this item
```

### Action with Input

```yaml
actions:
  set_priority:
    input:
      - { name: priority, type: int, required: true }
    description: Set route priority
```

### Custom Action

```yaml
actions:
  revoke:
    custom: true
    description: Revoke with reason
    # Implemented in Go handler
```

---

## Channels

### HTTP Channel

```yaml
channels:
  http:
    serve:
      enabled: true
      base_path: /api/users
      endpoints:
        - { action: list, method: GET, path: "/" }
        - { action: get, method: GET, path: "/{id}" }
        - { action: create, method: POST, path: "/", auth: admin }
        - { action: update, method: PATCH, path: "/{id}", auth: admin }
        - { action: delete, method: DELETE, path: "/{id}", auth: admin }
```

**Endpoint Properties:**
- `action` - CRUD action or custom action name
- `method` - HTTP method
- `path` - URL path (relative to base_path)
- `auth` - Required auth level (admin, user, none)

### CLI Channel

```yaml
channels:
  cli:
    serve:
      enabled: true
      command: users
      commands:
        - action: list
          columns: [id, email, name, status]
        - action: get
          args:
            - { name: id, required: true }
        - action: create
          flags:
            - { param: email, required: true }
            - { param: name }
            - { param: plan_id, name: plan, short: p }
```

**Command Properties:**
- `action` - Action to execute
- `args` - Positional arguments
- `flags` - Named flags
- `columns` - Columns to display for list
- `confirm` - Confirmation prompt for destructive actions

---

## Hooks

Lifecycle hooks for events:

```yaml
hooks:
  # Emit events
  after_create:
    - emit: user.created

  after_update:
    - emit: user.updated

  after_delete:
    - emit: user.deleted

  # Call internal functions
  after_create:
    - call: reload_router
```

**Hook Points:**
- `before_create` / `after_create`
- `before_update` / `after_update`
- `before_delete` / `after_delete`

**Hook Actions:**
- `emit: event.name` - Emit event for webhooks
- `call: function_name` - Call internal function

---

## Meta

Display metadata for UI:

```yaml
meta:
  description: User accounts for API access
  icon: users
  display_name: Users
  plural: Users
```

---

## Capabilities and Providers

Special module types for integrations:

### Capability Definition

```yaml
# core/modules/capabilities/payment.yaml
module: payment
type: capability

interface:
  create_customer:
    input: [user_id, email, name]
    output: [customer_id]

  create_subscription:
    input: [customer_id, plan_id]
    output: [subscription_id]
```

### Provider Implementation

```yaml
# core/modules/providers/stripe.yaml
module: stripe
type: provider
implements: payment

config:
  api_key: { type: secret, required: true }
  webhook_secret: { type: secret }

meta:
  display_name: Stripe
  description: Stripe payment processing
```

---

## Module Files

| Module | Description |
|--------|-------------|
| `user.yaml` | User accounts |
| `plan.yaml` | Subscription plans |
| `api_key.yaml` | API keys |
| `route.yaml` | Routing rules |
| `upstream.yaml` | Backend services |
| `group.yaml` | Teams/organizations |
| `group_member.yaml` | Team membership |
| `group_invite.yaml` | Team invitations |
| `entitlement.yaml` | Feature flags |
| `plan_entitlement.yaml` | Plan-feature links |
| `webhook.yaml` | Webhook configs |
| `webhook_delivery.yaml` | Delivery attempts |
| `setting.yaml` | System settings |
| `certificate.yaml` | TLS certificates |
| `oauth_identity.yaml` | OAuth identities |
| `oauth_state.yaml` | OAuth CSRF tokens |

---

## See Also

- [[Architecture]] - System architecture
- [[Resource-Types]] - API resource documentation
- [[Configuration]] - Configuration reference
