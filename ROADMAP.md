# APIGate Roadmap: Correctness, Consistency, Completeness

## High-Level Goals
- **100% Self-Onboardability**: New users can start without external help
- **100% Self-Documentation**: Every feature is discoverable and documented
- **100% Self-Serve**: All operations available through WebUI (CLI parity)
- **Values as Boundaries**: Explicit data flow, no hidden state

---

## Current State Assessment

| Category | Status | Score |
|----------|--------|-------|
| Hook Infrastructure | Working but 94% unused | 6% |
| YAML Hook Processing | Parsed but not executed | 0% |
| WebUI CRUD Operations | Complete | 100% |
| WebUI Custom Actions | Partial (not wired) | 30% |
| API Completeness | Full | 100% |
| Self-Documentation | Good | 80% |
| Data Integrity | Multiple issues | 60% |

---

## Phase 0: Critical Fixes (Foundation)

### 0.1 Fix Type Mismatch Bug
**Problem**: `avg_duration_ns` scanned as int64 but AVG() returns float64
**Files**:
- `core/analytics/sqlite.go:447,490`
- `core/exporter/log.go:69`

**Fix**: Change SQL to `CAST(AVG(duration_ns) AS INTEGER)` or change Go type to float64

### 0.2 Fix SQL Injection Risk
**Problem**: OrderBy parameter used without validation
**File**: `core/storage/sqlite.go:228-236`

**Fix**: Whitelist allowed column names before interpolation

### 0.3 Add Missing Foreign Key Constraints
**Problem**: auth_tokens, user_sessions, rate_limit_state lack FK constraints
**File**: New migration `009_foreign_keys.sql`

**Fix**: Add proper REFERENCES with ON DELETE CASCADE/RESTRICT

---

## Phase 1: Hook System Completion

### 1.1 YAML Hook Auto-Registration
**Gap**: Hooks declared in YAML are parsed but never registered with runtime

**Implementation**:
```
bootstrap/hooks.go
├── RegisterHooks()           # Existing - hardcoded hooks
├── RegisterYAMLHooks()       # NEW - auto-register from YAML
│   ├── Parse mod.Hooks map
│   ├── Create handler for each hook type
│   └── Call rt.OnHook() for each
└── Hook type handlers:
    ├── emitHandler()         # Publish to event bus
    ├── callHandler()         # Invoke registered function
    ├── emailHandler()        # Send via email adapter
    └── webhookHandler()      # HTTP POST to URL
```

**Files to modify**:
- `bootstrap/hooks.go` - Add RegisterYAMLHooks()
- `bootstrap/modules.go` - Call RegisterYAMLHooks() after module load

### 1.2 Function Registry for `call:` Hooks
**Gap**: `call: reload_router` declared but no function lookup

**Implementation**:
```go
// core/runtime/functions.go (NEW)
type FunctionRegistry struct {
    funcs map[string]func(ctx context.Context, event HookEvent) error
}

func (r *FunctionRegistry) Register(name string, fn func(...) error)
func (r *FunctionRegistry) Call(name string, event HookEvent) error
```

**Built-in functions to implement**:
| Function | Module | Purpose |
|----------|--------|---------|
| `reload_router` | route, upstream | Refresh routing table |
| `clear_other_defaults` | plan | Unset is_default on other plans |
| `sync_to_stripe` | plan | Optional payment provider sync |
| `send_verification_email` | user | Email via adapter |

### 1.3 Event Bus for `emit:` Hooks
**Gap**: No event publication/subscription system

**Implementation**:
```go
// core/events/bus.go (NEW)
type EventBus struct {
    subscribers map[string][]EventHandler
}

func (b *EventBus) Publish(event string, data map[string]any)
func (b *EventBus) Subscribe(event string, handler EventHandler)
```

**Events to support**:
- `key.created`, `key.revoked`
- `user.created`, `user.updated`
- `route.created`, `route.updated`, `route.deleted`
- `upstream.created`, `upstream.updated`, `upstream.deleted`
- `setting.changed`

---

## Phase 2: WebUI Feature Parity

### 2.1 Wire Custom Actions in List View
**Gap**: DynamicTable has customActions but ModuleList doesn't pass onAction

**File**: `webui/src/pages/ModuleList.tsx`

**Fix**: Add onAction handler that calls API endpoint

### 2.2 Add Usage Analytics Dashboard
**Gap**: CLI has `apigate usage summary|history|recent`, WebUI has nothing

**Implementation**:
- New page: `webui/src/pages/UsageDashboard.tsx`
- New API: Already exists at analytics endpoints
- Components: Charts, tables, date range picker

### 2.3 Add Admin User Management
**Gap**: CLI has `apigate admin`, WebUI has no equivalent

**Implementation**:
- New page: `webui/src/pages/AdminSettings.tsx`
- Features: Create admin, reset password, delete admin

### 2.4 Add Password Management
**Gap**: No way to change password in WebUI

**Implementation**:
- Add to user edit form or separate modal
- Call `set_password` action endpoint

---

## Phase 3: Self-Documentation Completion

### 3.1 Add Example Values to OpenAPI
**Gap**: No example field values in spec

**File**: `core/openapi/generator.go`

**Fix**: Generate examples from field type and constraints

### 3.2 Document Filterable Fields
**Gap**: Schema doesn't indicate which fields support filtering

**File**: `core/schema/introspect.go`

**Fix**: Add `filterable: true` to FieldSchema for indexed fields

### 3.3 Add Constraint Descriptions
**Gap**: Validation rules not exposed in schema

**File**: `core/channel/http/schema.go`

**Fix**: Include constraint details (min, max, pattern) in field schema

---

## Phase 4: Data Integrity Hardening

### 4.1 Strict Unknown Field Rejection
**Gap**: Unknown fields silently ignored

**File**: `core/validation/validator.go:97`

**Fix**: Return error or warning for unknown fields

### 4.2 Database-Level Constraints
**Gap**: Constraints only checked in Go, not in SQL

**Implementation**:
- Generate CHECK constraints in CREATE TABLE
- Enforce at both application and database layer

### 4.3 Reference Validation
**Gap**: ref_exists constraint not actually checked

**File**: `core/validation/validator.go:174`

**Fix**: Query database to verify referenced record exists

---

## Implementation Order

```
Week 1: Foundation (Phase 0)
├── Day 1: Fix avg_duration_ns type mismatch
├── Day 2: Fix SQL injection in OrderBy
└── Day 3: Add FK constraints migration

Week 2: Hook System (Phase 1.1-1.2)
├── Day 1-2: YAML hook auto-registration
├── Day 3-4: Function registry + reload_router
└── Day 5: Integration testing

Week 3: Hook System (Phase 1.3) + WebUI (Phase 2.1)
├── Day 1-2: Event bus implementation
├── Day 3: Wire custom actions in WebUI
└── Day 4-5: Testing and fixes

Week 4: WebUI Parity (Phase 2.2-2.4)
├── Day 1-2: Usage analytics dashboard
├── Day 3: Admin user management
└── Day 4-5: Password management

Week 5: Documentation + Hardening (Phase 3-4)
├── Day 1-2: OpenAPI improvements
├── Day 3: Schema documentation
└── Day 4-5: Data integrity hardening
```

---

## Success Criteria

### Correctness
- [ ] No type mismatch errors in logs
- [ ] No SQL injection vectors
- [ ] All FK constraints enforced
- [ ] All YAML-declared hooks execute

### Consistency
- [ ] Dashboard and sidebar use same URL pattern
- [ ] All modules follow same hook registration pattern
- [ ] CLI and WebUI have identical capabilities
- [ ] YAML declarations match runtime behavior

### Completeness
- [ ] 100% of declared hooks implemented
- [ ] 100% of custom actions accessible in WebUI
- [ ] 100% of fields have descriptions
- [ ] 100% of API endpoints documented in OpenAPI

---

## Architecture Principles

### Values as Boundaries
```
YAML Definition → Parse → Derive → Register → Execute → Return
     ↓              ↓        ↓         ↓          ↓        ↓
  Explicit      No magic  Computed  Explicit   Traced   Complete
  schema        inference  values   handlers   flow     response
```

### No Hidden State
- Hooks receive all context via HookEvent
- Results returned via ActionResult.Meta
- No global variables for inter-hook communication

### Fail Loud
- Unknown fields: Error (not silent ignore)
- Missing references: Error (not silent skip)
- Type mismatches: Error (not coerce)

---

## Files Reference

### Core Infrastructure
| File | Purpose |
|------|---------|
| `core/runtime/runtime.go` | Hook dispatch, action execution |
| `core/runtime/functions.go` | NEW: Function registry |
| `core/events/bus.go` | NEW: Event publication |
| `bootstrap/hooks.go` | Hook registration |
| `bootstrap/modules.go` | Module loading |

### Data Layer
| File | Purpose |
|------|---------|
| `core/storage/sqlite.go` | Database operations |
| `core/validation/validator.go` | Input validation |
| `core/analytics/sqlite.go` | Usage metrics |
| `migrations/*.sql` | Schema definitions |

### API Layer
| File | Purpose |
|------|---------|
| `core/channel/http/http.go` | HTTP handlers |
| `core/channel/http/schema.go` | Schema introspection |
| `core/openapi/generator.go` | OpenAPI spec |

### WebUI
| File | Purpose |
|------|---------|
| `webui/src/pages/ModuleList.tsx` | List view |
| `webui/src/pages/ModuleView.tsx` | Create/edit |
| `webui/src/pages/Dashboard.tsx` | Module overview |
| `webui/src/pages/UsageDashboard.tsx` | NEW: Analytics |
| `webui/src/pages/AdminSettings.tsx` | NEW: Admin mgmt |
