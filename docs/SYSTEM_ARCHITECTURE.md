# APIGate System Architecture

## Overview

APIGate is a self-hosted API monetization platform built with a clean architecture following domain-driven design principles. This document describes the system design, module architecture, core data structures, and configuration patterns.

---

## 1. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  Web Browser          │  API Clients        │  CLI                          │
│  (Portal/Admin UI)    │  (HTTP/WS/SSE)     │  (Cobra Commands)             │
└──────────┬────────────┴─────────┬───────────┴──────────────┬────────────────┘
           │                      │                          │
           ▼                      ▼                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PRESENTATION LAYER                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  Web Portal  │  │  Admin UI    │  │  REST API    │  │  CLI Handler │     │
│  │  /portal/*   │  │  /ui/*       │  │  /api/*      │  │  apigate ... │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  Docs Portal │  │  Proxy       │  │  WebSocket   │                       │
│  │  /docs/*     │  │  Handler     │  │  Handler     │                       │
│  └──────────────┘  └──────────────┘  └──────────────┘                       │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           APPLICATION LAYER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ ProxyService │  │ RouteService │  │ AuthService  │  │ UsageService │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  Transform   │  │  Settings    │  │  Module      │                       │
│  │  Service     │  │  Service     │  │  Runtime     │                       │
│  └──────────────┘  └──────────────┘  └──────────────┘                       │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            DOMAIN LAYER                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │  User   │  │   Key   │  │  Plan   │  │  Route  │  │ Upstream│            │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘            │
│                                                                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐                         │
│  │  Usage  │  │  Quota  │  │RateLimit│  │Settings │                         │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘                         │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          INFRASTRUCTURE LAYER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────┐  ┌─────────────────────────────────────────┐   │
│  │      PORTS              │  │              ADAPTERS                    │   │
│  │  ┌─────────────────┐    │  │  ┌─────────────────┐  ┌──────────────┐  │   │
│  │  │   UserStore     │◄───┼──┼──│  SQLiteUserStore│  │   HTTPClient │  │   │
│  │  │   KeyStore      │    │  │  │  SQLiteKeyStore │  │   SMTPClient │  │   │
│  │  │   PlanStore     │    │  │  │  SQLitePlanStore│  │StripeClient  │  │   │
│  │  │   RouteStore    │    │  │  │  SQLiteRouteStore│ │ RedisClient  │  │   │
│  │  │   UsageStore    │    │  │  │  ...            │  │   ...        │  │   │
│  │  └─────────────────┘    │  │  └─────────────────┘  └──────────────┘  │   │
│  └─────────────────────────┘  └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Directory Structure

```
apigate/
├── cmd/                          # CLI entry points
│   └── apigate/
│       ├── main.go               # Application entry
│       ├── serve.go              # Server command
│       ├── init.go               # Setup wizard
│       ├── users.go              # User CLI commands
│       ├── keys.go               # Key CLI commands
│       ├── plans.go              # Plan CLI commands
│       ├── routes.go             # Route CLI commands
│       ├── upstreams.go          # Upstream CLI commands
│       ├── settings.go           # Settings CLI commands
│       └── mod.go                # Generic module CLI
│
├── core/                         # Core framework
│   ├── schema/                   # Module schema definitions
│   │   ├── module.go             # Module struct & parsing
│   │   ├── field.go              # Field type definitions
│   │   ├── action.go             # Action definitions
│   │   └── channel.go            # Channel definitions
│   │
│   ├── runtime/                  # Module runtime
│   │   ├── runtime.go            # Module loading & execution
│   │   ├── resolver.go           # Dependency resolution
│   │   └── hooks.go              # Hook execution
│   │
│   ├── channel/                  # Communication channels
│   │   ├── http/                 # HTTP REST channel
│   │   ├── cli/                  # CLI channel
│   │   └── websocket/            # WebSocket channel
│   │
│   ├── capability/               # Capability system (DI)
│   │   ├── container.go          # DI container
│   │   ├── registry.go           # Provider registry
│   │   ├── resolver.go           # Capability resolution
│   │   └── adapters/             # Capability interfaces
│   │
│   ├── storage/                  # Generic storage layer
│   │   ├── store.go              # Store interface
│   │   └── sql_builder.go        # SQL query builder
│   │
│   └── modules/                  # Module YAML definitions
│       ├── user.yaml
│       ├── plan.yaml
│       ├── api_key.yaml
│       ├── route.yaml
│       ├── upstream.yaml
│       ├── setting.yaml
│       ├── capabilities/         # Capability definitions
│       └── providers/            # Provider implementations
│
├── domain/                       # Domain models (pure business logic)
│   ├── user/                     # User aggregate
│   ├── key/                      # API key aggregate
│   ├── plan/                     # Plan aggregate
│   ├── usage/                    # Usage aggregate
│   ├── quota/                    # Quota management
│   ├── ratelimit/                # Rate limiting
│   ├── route/                    # Route aggregate
│   └── settings/                 # Settings aggregate
│
├── app/                          # Application services
│   ├── proxy.go                  # Proxy orchestration
│   ├── route.go                  # Route matching service
│   ├── transform.go              # Request/response transform
│   └── settings.go               # Settings service
│
├── ports/                        # Port interfaces (DI boundaries)
│   ├── user.go                   # UserStore interface
│   ├── key.go                    # KeyStore interface
│   └── ...
│
├── adapters/                     # Infrastructure adapters
│   ├── sqlite/                   # SQLite implementations
│   ├── http/                     # HTTP adapters
│   ├── tls/                      # TLS/ACME adapters
│   ├── email/                    # Email adapters
│   ├── payment/                  # Payment adapters
│   ├── cache/                    # Cache adapters
│   └── storage/                  # Storage adapters
│
├── bootstrap/                    # Application bootstrap
│   ├── wire.go                   # Dependency wiring
│   ├── config.go                 # Configuration loading
│   └── server.go                 # Server initialization
│
├── web/                          # Web handlers
│   ├── web.go                    # Web router
│   ├── handlers.go               # Admin UI handlers
│   ├── portal.go                 # Customer portal handlers
│   ├── portal_templates.go       # Portal HTML templates
│   ├── docs.go                   # Documentation portal
│   ├── static/                   # Static assets
│   └── templates/                # HTML templates
│
├── migrations/                   # Database migrations
│   └── *.sql                     # Migration files
│
└── docs/                         # Documentation
```

---

## 3. Module System

### 3.1 Module Definition Structure

Modules are the fundamental building blocks of APIGate. Each module is defined in YAML and represents a complete data entity with its schema, actions, channels, and hooks.

```yaml
module: module_name

meta:
  description: "Human-readable description"
  icon: "lucide-icon-name"
  display_name: "Display Name"
  plural: "Display Names"
  implements: [capability1, capability2]
  requires:
    dependency_name:
      capability: capability_type
      required: true|false
      default: "default_provider"

schema:
  field_name:
    type: string|int|bool|enum|ref|secret|json|timestamp|email|bytes
    required: true|false
    unique: true|false
    lookup: true|false
    immutable: true|false
    internal: true|false
    default: value
    description: "Field description"
    values: [value1, value2]  # For enum type
    to: target_module         # For ref type

actions:
  action_name:
    description: "What this action does"
    internal: true|false
    confirm: true|false
    auth: admin|self_or_admin
    input:
      - { name: param, type: type, required: true }
    output:
      - { name: result, type: type }
    set: { field: value }

channels:
  http:
    serve:
      enabled: true
      base_path: /api/resource
      endpoints:
        - { action: list, method: GET, path: "/" }
    consume:
      external_api:
        base: https://api.example.com
        auth: { type: bearer, bearer: ${ENV_VAR} }
        methods:
          method_name:
            method: POST
            path: /endpoint
  cli:
    serve:
      enabled: true
      command: resource
      commands:
        - action: list
          columns: [id, name, status]
  websocket:
    serve:
      enabled: true
      path: /ws/resource
      events: [created, updated, deleted]

hooks:
  before_create:
    - call: validate_data
  after_create:
    - emit: resource.created
    - call: sync_external
```

### 3.2 Field Types

| Type | Description | Attributes |
|------|-------------|------------|
| `string` | Text value | default, required, unique, lookup |
| `int` | Integer | default, required |
| `bool` | Boolean | default |
| `enum` | Predefined values | values[], default |
| `ref` | Foreign key reference | to: module_name |
| `secret` | Encrypted text | internal: true typically |
| `json` | JSON object/array | |
| `timestamp` | Date/time | |
| `email` | Email address | unique, lookup |
| `bytes` | Binary data | |

### 3.3 Built-in Modules

| Module | File | Purpose |
|--------|------|---------|
| `user` | user.yaml | User accounts |
| `plan` | plan.yaml | Pricing tiers |
| `api_key` | api_key.yaml | API authentication |
| `route` | route.yaml | Request routing |
| `upstream` | upstream.yaml | Backend services |
| `setting` | setting.yaml | Configuration |

---

## 4. Capability System

### 4.1 Architecture

The capability system implements dependency injection for pluggable providers.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        CAPABILITY CONTAINER                               │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐   │
│  │  Payment   │    │   Email    │    │   Cache    │    │  Storage   │   │
│  │ Capability │    │ Capability │    │ Capability │    │ Capability │   │
│  └─────┬──────┘    └─────┬──────┘    └─────┬──────┘    └─────┬──────┘   │
│        │                 │                 │                 │          │
│        ▼                 ▼                 ▼                 ▼          │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐   │
│  │  Providers │    │  Providers │    │  Providers │    │  Providers │   │
│  ├────────────┤    ├────────────┤    ├────────────┤    ├────────────┤   │
│  │ • stripe   │    │ • smtp     │    │ • redis    │    │ • s3       │   │
│  │ • paddle   │    │ • sendgrid │    │ • memory   │    │ • disk     │   │
│  │ • lemon    │    │ • log      │    └────────────┘    │ • memory   │   │
│  │ • dummy    │    └────────────┘                      └────────────┘   │
│  └────────────┘                                                          │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Built-in Capabilities

| Capability | Purpose | Providers |
|------------|---------|-----------|
| `payment` | Payment processing | stripe, paddle, lemon, dummy |
| `email` | Email sending | smtp, sendgrid, log |
| `cache` | Data caching | redis, memory |
| `storage` | File storage | s3, disk, memory |
| `auth` | Authentication | builtin |
| `queue` | Message queuing | redis, memory |
| `notification` | Alerts | slack, webhook, log |
| `data_source` | External data | database, API, file |
| `sync` | Data synchronization | table_sync |
| `reconciliation` | Data validation | default |

### 4.3 Capability Definition

```yaml
# capabilities/payment.yaml
capability: payment

description: |
  Payment processing capability for subscription billing,
  one-time payments, and invoice management.

actions:
  create_customer:
    input:
      - { name: email, type: email, required: true }
      - { name: name, type: string }
    output:
      - { name: customer_id, type: string }

  create_subscription:
    input:
      - { name: customer_id, type: string, required: true }
      - { name: price_id, type: string, required: true }
    output:
      - { name: subscription_id, type: string }

  handle_webhook:
    input:
      - { name: payload, type: json, required: true }
    output:
      - { name: event_type, type: string }
      - { name: processed, type: bool }

events:
  - payment.customer_created
  - payment.subscription_created
  - payment.invoice_paid

meta:
  icon: credit-card
  display_name: Payment Processing
```

### 4.4 Provider Implementation

```yaml
# providers/payment_stripe.yaml
module: payment_stripe

meta:
  description: Stripe payment processor integration
  implements: [payment]
  icon: credit-card
  display_name: Stripe

schema:
  name:           { type: string, required: true, lookup: true }
  enabled:        { type: bool, default: false }
  secret_key:     { type: secret, required: true }
  webhook_secret: { type: secret }

actions:
  test_connection:
    description: Verify Stripe API credentials
    output:
      - { name: valid, type: bool }
      - { name: error, type: string }

channels:
  http:
    consume:
      stripe:
        base: https://api.stripe.com/v1
        auth:
          type: bearer
          bearer: ${secret_key}
        methods:
          create_customer:
            method: POST
            path: /customers
            map:
              email: email
              name: name
            response:
              set:
                customer_id: id
```

### 4.5 Dependency Injection

Modules can declare dependencies on capabilities:

```yaml
module: sync_table

meta:
  implements: [sync]
  requires:
    source:
      capability: data_source
      required: true
      description: "Source data source to read from"
    target:
      capability: data_source
      required: true
      description: "Target data source to write to"
    notifier:
      capability: notification
      required: false
      default: notification_log

schema:
  source_instance: { type: string, required: true }
  target_instance: { type: string, required: true }
```

---

## 5. Core Data Models

### 5.1 User Model

```yaml
# user.yaml
schema:
  email:         { type: email, unique: true, lookup: true, required: true }
  password_hash: { type: secret, internal: true }
  name:          { type: string, default: "" }
  stripe_id:     { type: string, internal: true }
  plan_id:       { type: ref, to: plan, default: "free" }
  status:        { type: enum, values: [pending, active, suspended, cancelled], default: active }
```

**State Machine:**
```
pending → active → suspended → cancelled
                ↑            │
                └────────────┘
```

### 5.2 API Key Model

```yaml
# api_key.yaml
schema:
  user_id:    { type: ref, to: user, required: true }
  hash:       { type: secret, internal: true }
  prefix:     { type: string, lookup: true, immutable: true }
  name:       { type: string, default: "" }
  scopes:     { type: json }
  expires_at: { type: timestamp }
  revoked_at: { type: timestamp, internal: true }
  last_used:  { type: timestamp, internal: true }
```

**Key Format:** `ak_<64 hex characters>`

### 5.3 Plan Model

```yaml
# plan.yaml
schema:
  name:               { type: string, required: true, lookup: true }
  description:        { type: string, default: "" }
  rate_limit_per_minute: { type: int, default: 60 }
  requests_per_month:    { type: int, default: 1000 }
  price_monthly:      { type: int, default: 0 }  # cents
  overage_price:      { type: int, default: 0 }  # cents per request
  trial_days:         { type: int, default: 0 }
  stripe_price_id:    { type: string }
  paddle_price_id:    { type: string }
  lemon_variant_id:   { type: string }
  is_default:         { type: bool, default: false }
  enabled:            { type: bool, default: true }
```

### 5.4 Route Model

```yaml
# route.yaml
schema:
  name:           { type: string, required: true, lookup: true }
  description:    { type: string, default: "" }
  path_pattern:   { type: string, required: true }
  match_type:     { type: enum, values: [exact, prefix, regex], default: prefix }
  methods:        { type: json }
  headers:        { type: json }
  upstream_id:    { type: ref, to: upstream, required: true }
  path_rewrite:   { type: string }
  method_override: { type: string }
  request_transform:  { type: json }
  response_transform: { type: json }
  metering_expr:  { type: string, default: "1" }
  metering_mode:  { type: enum, values: [request, response_field, bytes, custom], default: request }
  protocol:       { type: enum, values: [http, http_stream, sse, websocket], default: http }
  priority:       { type: int, default: 0 }
  enabled:        { type: bool, default: true }
```

### 5.5 Upstream Model

```yaml
# upstream.yaml
schema:
  name:             { type: string, required: true, lookup: true }
  description:      { type: string, default: "" }
  base_url:         { type: string, required: true }
  timeout_ms:       { type: int, default: 30000 }
  max_idle_conns:   { type: int, default: 100 }
  idle_conn_timeout_ms: { type: int, default: 90000 }
  auth_type:        { type: enum, values: [none, header, bearer, basic], default: none }
  auth_header:      { type: string, default: "" }
  auth_value_encrypted: { type: bytes, default: null }
  enabled:          { type: bool, default: true }
```

---

## 6. Request Flow

### 6.1 Proxy Request Flow

```
┌─────────┐          ┌─────────────────────────────────────────────────────────┐
│ Client  │          │                    APIGate                               │
└────┬────┘          └────────────────────────┬────────────────────────────────┘
     │                                        │
     │  HTTP Request + X-API-Key              │
     │ ──────────────────────────────────────►│
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    1. Extract API Key     │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    2. Validate Key        │
     │                          │       (bcrypt verify)     │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    3. Load User & Plan    │
     │                          │       Check user.status   │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    4. Check Rate Limit    │
     │                          │       (Token Bucket)      │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    5. Check Quota         │
     │                          │       (Monthly Requests)  │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    6. Match Route         │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    7. Transform Request   │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    8. Forward to Upstream │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │    9. Transform Response  │
     │                          └─────────────┬─────────────┘
     │                                        │
     │                          ┌─────────────┴─────────────┐
     │                          │   10. Record Usage (async)│
     │                          └─────────────┬─────────────┘
     │                                        │
     │  HTTP Response + Rate Limit Headers    │
     │ ◄──────────────────────────────────────│
```

---

## 7. Storage Layer

### 7.1 SQLite Configuration

```go
// Connection settings
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  // 64MB
PRAGMA temp_store = MEMORY;
PRAGMA busy_timeout = 5000;
```

### 7.2 Auto-Generated Tables

Each module generates a table based on its schema:

```sql
-- Generated from user.yaml
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    name TEXT DEFAULT '',
    stripe_id TEXT,
    plan_id TEXT REFERENCES plans(id),
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

### 7.3 Store Interface

```go
type Store interface {
    List(ctx context.Context, filter Filter, opts ...ListOption) ([]Record, error)
    Get(ctx context.Context, id string) (Record, error)
    GetBy(ctx context.Context, field string, value interface{}) (Record, error)
    Create(ctx context.Context, data map[string]interface{}) (Record, error)
    Update(ctx context.Context, id string, data map[string]interface{}) (Record, error)
    Delete(ctx context.Context, id string) error
}
```

---

## 8. Configuration

### 8.1 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APIGATE_DATABASE_DSN` | SQLite database path | `apigate.db` |
| `APIGATE_SERVER_PORT` | HTTP port | `8080` |
| `APIGATE_SERVER_HOST` | Bind address | `0.0.0.0` |
| `APIGATE_LOG_LEVEL` | Log level | `info` |
| `APIGATE_LOG_FORMAT` | Log format | `json` |
| `APIGATE_JWT_SECRET` | JWT signing secret | Generated |
| `APIGATE_ENCRYPTION_KEY` | Secret encryption key | Generated |

### 8.2 Settings Storage

Settings are stored in the database:

```yaml
# Common settings keys
smtp.host: "smtp.example.com"
smtp.port: "587"
smtp.username: "user"
smtp.password: <encrypted>
smtp.from: "noreply@example.com"

payment.provider: "stripe"
payment.stripe.secret_key: <encrypted>
payment.stripe.webhook_secret: <encrypted>

auth.jwt_ttl: "24h"
auth.refresh_ttl: "168h"
auth.bcrypt_cost: "12"
```

---

## 9. Security Architecture

### 9.1 Secret Management

```
Plaintext → AES-256-GCM Encrypt → Store in SQLite
                ↑
    APIGATE_ENCRYPTION_KEY
```

### 9.2 Password Hashing

- Algorithm: bcrypt
- Cost factor: 12 (configurable)
- Used for: User passwords, API key validation

### 9.3 API Key Security

```
Generation:
  key = "ak_" + randomHex(64)  // 256-bit entropy
  hash = bcrypt(key)
  prefix = key[:15]

Storage:
  Store hash and prefix only

Validation:
  keys = findByPrefix(prefix)
  for each key:
    if bcrypt.Compare(key.hash, providedKey):
      return valid
```

---

## 10. Channels (Communication)

### 10.1 HTTP Channel

Auto-generates REST endpoints from module definitions:

```
GET     /api/{module}              → list action
POST    /api/{module}              → create action
GET     /api/{module}/{id}         → get action
PUT     /api/{module}/{id}         → update action
PATCH   /api/{module}/{id}         → update action
DELETE  /api/{module}/{id}         → delete action
POST    /api/{module}/{id}/{action} → custom action
```

### 10.2 CLI Channel

Auto-generates Cobra commands:

```bash
apigate {module} list
apigate {module} get <id>
apigate {module} create [flags]
apigate {module} update <id> [flags]
apigate {module} delete <id>
apigate {module} {action} <id>
```

### 10.3 WebSocket Channel

Real-time event streaming:

```
/ws/{module}

Events:
  - {module}.created
  - {module}.updated
  - {module}.deleted
```

---

## 11. Hooks & Events

### 11.1 Hook Types

| Hook | Trigger |
|------|---------|
| `before_create` | Before record creation |
| `after_create` | After record creation |
| `before_update` | Before record update |
| `after_update` | After record update |
| `before_delete` | Before record deletion |
| `after_delete` | After record deletion |
| `before_{action}` | Before custom action |
| `after_{action}` | After custom action |

### 11.2 Hook Actions

```yaml
hooks:
  after_create:
    - emit: resource.created    # Emit event
    - call: external_sync       # Call function
    - set: { synced: true }     # Update field
```

---

## 12. Extensibility

### 12.1 Adding Custom Modules

1. Create YAML file in `core/modules/`
2. Define schema, actions, channels
3. Restart server to load

### 12.2 Adding Custom Capabilities

1. Define capability in `core/modules/capabilities/`
2. Create provider implementations in `core/modules/providers/`
3. Register in container

### 12.3 Adding Custom Providers

1. Create provider YAML implementing capability interface
2. Configure via settings
3. Use dependency injection in modules

---

## 13. Deployment

### 13.1 Binary

```bash
# Build
go build -o apigate ./cmd/apigate

# Run
./apigate serve
```

### 13.2 Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o apigate ./cmd/apigate

FROM alpine:latest
COPY --from=builder /app/apigate /usr/local/bin/
EXPOSE 8080
CMD ["apigate", "serve"]
```

### 13.3 Production Checklist

- [ ] Set strong `APIGATE_JWT_SECRET`
- [ ] Set strong `APIGATE_ENCRYPTION_KEY`
- [ ] Configure proper database path
- [ ] Configure TLS (ACME or manual certificates)
- [ ] Configure backup for SQLite
- [ ] Set up monitoring (Prometheus)
- [ ] Configure log aggregation
- [ ] Test payment webhooks
- [ ] Verify email delivery

---

## 14. TLS Architecture

### 14.1 ACME Provider

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        TLS LAYER                                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────┐         ┌────────────────┐                          │
│  │  ACMEProvider  │────────►│ autocert.Manager│                          │
│  │                │         │                │                          │
│  │ • GetCertificate│         │ • Client (MUST be set)                    │
│  │ • RenewCertificate│       │ • DirectoryURL                            │
│  │ • RevokeCertificate│      │ • Cache                                   │
│  └────────────────┘         └────────┬───────┘                          │
│                                      │                                   │
│                                      ▼                                   │
│                            ┌────────────────┐                           │
│                            │  DBCertCache   │                           │
│                            │                │                           │
│                            │ • certificates │ (TLS certs)               │
│                            │ • acme_cache   │ (ACME account keys)       │
│                            └────────────────┘                           │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 14.2 Critical Implementation Detail

**The `autocert.Manager.Client` MUST always be explicitly configured:**

```go
provider.manager = &autocert.Manager{
    Client: &acme.Client{
        DirectoryURL: directoryURL,  // Staging or Production
    },
    // ... other fields
}
```

| Mode | DirectoryURL |
|------|--------------|
| Staging | `https://acme-staging-v02.api.letsencrypt.org/directory` |
| Production | `https://acme-v02.api.letsencrypt.org/directory` |

**Why**: Leaving `Client` as `nil` causes lazy initialization failures in production mode (Issue #48).

### 14.3 Certificate Storage

| Table | Purpose | Key |
|-------|---------|-----|
| `certificates` | TLS certificates with metadata | Domain name |
| `acme_cache` | ACME account keys | `+acme_account+<url>` |

See [TLS Certificates Spec](spec/tls-certificates.md) for full specification
