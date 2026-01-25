# APIGate - Claude Code Instructions

> **This file governs how Claude Code operates on this codebase. All changes must align with documented standards.**

---

## High-Level Architectural Goals

### 1. Values as Boundaries

Test configuration inputs, not implementation details. If a function accepts config values, test ALL input combinations produce correct outputs.

```go
// Config has boolean field → test both values
func NewProvider(cfg Config) // cfg.Staging = true/false

// Tests MUST cover both:
{staging: false, wantURL: production}
{staging: true, wantURL: staging}
```

### 2. Pluggable Modules (Capability/Provider Pattern)

**Capabilities** define interfaces. **Providers** implement them.

```
core/modules/capabilities/   ← Interface definitions
  tls.yaml                   ← "capability: tls"
  payment.yaml               ← "capability: payment"
  email.yaml                 ← "capability: email"

core/modules/providers/      ← Implementations
  tls_acme.yaml              ← "meta.implements: [tls]"
  payment_stripe.yaml        ← "meta.implements: [payment]"
  email_smtp.yaml            ← "meta.implements: [email]"
```

**Core Data Modules** (NOT pluggable):
- `setting.yaml` - key-value config store
- `certificate.yaml` - certificate data storage
- `user.yaml`, `plan.yaml`, `route.yaml` - core entities

These are data stores, not capability providers.

### 3. Module YAML is Source of Truth

Module YAML defines everything:
- Schema (fields, types, constraints)
- Actions (CRUD + custom)
- Channels (HTTP endpoints, CLI commands)
- Hooks (events)

**The runtime generates from YAML.** Never write hand-coded handlers that duplicate module definitions.

```yaml
# If YAML says this:
channels:
  http:
    serve:
      endpoints:
        - { action: get_by_domain, path: "/domain/{domain}" }

# Runtime MUST generate that endpoint. No exceptions.
```

### 4. No Backward Compatibility / Legacy Maintenance

- Don't maintain parallel paths (old + new)
- Delete legacy code, don't wrap it
- No feature flags for "old way" vs "new way"
- Single implementation path only

### 5. No Unnecessary Transforms in Glue Layers

Glue code connects components. It should NOT:
- Transform data shapes
- Add business logic
- Validate beyond type checking

Pass data through. Let boundaries handle their own concerns.

---

## Governing Documents

Before making any changes, understand and follow these documents:

| Document | Purpose | When to Reference |
|----------|---------|-------------------|
| **[PROJECT_STANDARDS.md](PROJECT_STANDARDS.md)** | Core principles, release blockers | Every change |
| **[docs/spec/](docs/spec/)** | API specification (JSON:API, errors, resources) | **Any API change** |
| **[docs/user_journeys/](docs/user_journeys/)** | User flows, UX requirements | UI/UX changes |
| **[docs/SYSTEM_ARCHITECTURE.md](docs/SYSTEM_ARCHITECTURE.md)** | Module system, data models | Architectural changes |
| **[docs/TECHNICAL_FEATURES.md](docs/TECHNICAL_FEATURES.md)** | Feature inventory | Adding features |
| **[docs/USER_GUIDE.md](docs/USER_GUIDE.md)** | End-user documentation | User-facing changes |

### API Specification (docs/spec/) - Source of Truth

The `docs/spec/` directory contains the **authoritative specification** for all API behavior:

| Spec Document | Governs | Implementation |
|---------------|---------|----------------|
| [json-api.md](docs/spec/json-api.md) | Response format, document structure | `pkg/jsonapi/` |
| [error-codes.md](docs/spec/error-codes.md) | All error codes and HTTP statuses | `pkg/jsonapi/errors.go` |
| [pagination.md](docs/spec/pagination.md) | Pagination parameters and behavior | `pkg/jsonapi/pagination.go` |
| [resource-types.md](docs/spec/resource-types.md) | All API resource types and attributes | `adapters/http/admin/` |

**Spec-Code Alignment Rules:**
- If behavior is in the spec, it MUST be implemented in code
- If behavior is in code, it MUST be documented in the spec
- Tests MUST verify behavior matches the specification
- Wiki is synced from docs/spec/ (not the other way)

---

## The Five Principles (Release Blockers)

Every change must satisfy ALL five principles from PROJECT_STANDARDS.md:

```
1. SELF-ONBOARDING    - Can users start without human help?
2. SELF-SERVICE       - Can users do this entirely via UI?
3. SELF-DOCUMENTING   - Is there a single source of truth?
4. TYPE SAFETY        - Is Go code explicitly typed?
5. TEST COVERAGE      - Is coverage >80%?
```

**If a change violates any principle, stop and discuss before proceeding.**

---

## Test Requirements (CI Enforced)

**These are enforced by CI - violations block merge:**

1. **Coverage threshold**: Total coverage must be >=80%
2. **Coverage delta**: PRs cannot decrease coverage
3. **Boundary testing**: Every exported function with config inputs must test all input values

Example - function with boolean config:
```go
// Function
func NewProvider(cfg Config) (*Provider, error)

// REQUIRED test - both values of cfg.Staging
func TestNewProvider(t *testing.T) {
    tests := []struct{
        staging bool
        wantURL string
    }{
        {false, productionURL},  // MUST test
        {true, stagingURL},      // MUST test
    }
    // ...
}
```

**Pre-commit hook enforces locally. CI is the final gate.**

---

## Before Making Changes

### 1. Check Documentation First

Before writing code, read relevant documentation:

```
User-facing change?     → Read docs/user_journeys/{relevant_journey}.md
API change?             → Read docs/TECHNICAL_FEATURES.md
Architecture change?    → Read docs/SYSTEM_ARCHITECTURE.md
Module change?          → Read the module's YAML definition
```

### 2. Identify Documentation Impact

Ask yourself:
- [ ] Will this change affect any user journey (J1-J9)?
- [ ] Does this add/modify/remove an API endpoint?
- [ ] Does this change the module schema?
- [ ] Does this affect error messages or codes?
- [ ] Does this change configuration options?

If YES to any, documentation updates are **required** with the code change.

### 3. Check Single Source of Truth

Identify where the truth lives:

| Concept | Source of Truth | Derived |
|---------|-----------------|---------|
| **API response format** | `docs/spec/json-api.md` | `pkg/jsonapi/` implementation |
| **Error codes** | `docs/spec/error-codes.md` | `pkg/jsonapi/errors.go` |
| **Pagination behavior** | `docs/spec/pagination.md` | `pkg/jsonapi/pagination.go` |
| **Resource types** | `docs/spec/resource-types.md` | Handler implementations |
| API endpoints | Go handlers | OpenAPI, docs |
| Module schema | YAML in core/modules/ | UI forms |
| CLI commands | Cobra definitions | CLI help |
| Config options | Env var constants | README |

**Never update derived documentation directly - update the source.**

### 4. API Change Workflow

For any API behavior change:

```
1. UPDATE SPEC FIRST
   → Edit docs/spec/{relevant-file}.md
   → Define exact expected behavior

2. IMPLEMENT CODE
   → Write code that matches spec exactly
   → Use pkg/jsonapi/ builders

3. WRITE/UPDATE TESTS
   → Tests verify spec compliance
   → Include response format assertions

4. SYNC TO WIKI (optional)
   → gh wiki sync (when ready)
```

---

## After Making Changes

### 1. Reflect on Documentation Impact

After completing a code change, verify:

```
□ Did I change user-facing behavior?
  → Update relevant journey in docs/user_journeys/

□ Did I add/change an API endpoint?
  → Ensure handler has OpenAPI annotations
  → Update docs/TECHNICAL_FEATURES.md if significant

□ Did I add/change a module?
  → Update module YAML schema
  → UI forms auto-update from schema

□ Did I add/change error codes?
  → Update error constants (source of truth)
  → Error docs derive from constants

□ Did I add/change configuration?
  → Update env var constants
  → README derives from constants
```

### 2. Update Affected Documentation

If documentation needs updating, do it in the **same commit** as the code change:

```
Good:  "Add password reset flow" (includes code + journey update)
Bad:   "Add password reset flow" then later "Update docs for password reset"
```

### 3. Verify No Documentation Drift

Run mental check:
- Does docs/TECHNICAL_FEATURES.md still accurately describe the system?
- Do user journeys still match the actual UI flow?
- Are all error codes documented?

---

## Documentation Update Guidelines

### User Journeys (docs/user_journeys/)

When updating a journey document:

1. **Update the step-by-step flow** if UI/UX changes
2. **Update screenshot references** (capture will regenerate)
3. **Update metrics/KPIs** if success criteria change
4. **Update error handling** if new errors added
5. **Keep business context accurate**

### Technical Features (docs/TECHNICAL_FEATURES.md)

When updating:

1. **Add new features** to appropriate section
2. **Update API endpoints** if changed
3. **Update CLI commands** if changed
4. **Mark deprecated features** (don't remove immediately)

### System Architecture (docs/SYSTEM_ARCHITECTURE.md)

When updating:

1. **Update module definitions** if schema changes
2. **Update data models** if entities change
3. **Update flow diagrams** if request flow changes
4. **Keep capability descriptions current**

---

## Code Style Enforcement

### Go Server (Type Safety Required)

```go
// REQUIRED: Explicit types
func GetUser(id string) (*User, error)

// FORBIDDEN: interface{} without assertion
func GetData() interface{}  // NO

// REQUIRED: Type assertion with check
user, ok := ctx.Value("user").(User)
if !ok {
    return ErrInvalidContext
}

// FORBIDDEN: Untyped JSON
var data map[string]interface{}  // NO

// REQUIRED: Typed structs
var req CreateUserRequest
```

### Error Handling

```go
// REQUIRED: All errors handled
result, err := doSomething()
if err != nil {
    return fmt.Errorf("context: %w", err)
}

// FORBIDDEN: Ignored errors
doSomething()  // NO - error ignored
```

---

## Test Requirements

### For Every Code Change

1. **Unit tests** for new functions (aim for >80% coverage)
2. **Integration tests** if components interact
3. **E2E test updates** if user journey affected

### Before Completing

```bash
# Must pass
go test ./...
go vet ./...

# Coverage check
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
# Must be >80%
```

---

## Commit Message Format

Include documentation updates in commit message:

```
feat: Add password reset flow

- Add POST /auth/reset-password endpoint
- Add email template for reset link
- Update J5-onboarding journey with reset flow
- Add password reset to TECHNICAL_FEATURES.md

Docs updated:
- docs/user_journeys/customer/j5-onboarding.md
- docs/TECHNICAL_FEATURES.md
```

---

## Red Flags - Stop and Discuss

Stop and ask before proceeding if:

1. **Documentation conflict** - Code doesn't match what docs describe
2. **Missing source of truth** - Can't identify single source
3. **Breaking user journey** - Change would break J1-J9 flow
4. **Type safety violation** - Need to use `interface{}` without assertion
5. **Coverage drop** - Change would reduce coverage below 80%
6. **Self-service violation** - Feature requires admin/CLI/DB access

---

## Quick Reference

### Change Checklist

```
BEFORE:
[ ] Read relevant documentation
[ ] Identify documentation impact
[ ] Understand single source of truth

DURING:
[ ] Follow type safety requirements
[ ] Write tests alongside code
[ ] Keep changes aligned with principles

AFTER:
[ ] Update affected documentation (same commit)
[ ] Verify no documentation drift
[ ] Run tests and coverage check
```

### Documentation Locations

```
API Specification  → docs/spec/               # Source of truth for API behavior
  - JSON:API format  → docs/spec/json-api.md
  - Error codes      → docs/spec/error-codes.md
  - Pagination       → docs/spec/pagination.md
  - Resource types   → docs/spec/resource-types.md

User guides        → docs/USER_GUIDE.md
User journeys      → docs/user_journeys/
Technical features → docs/TECHNICAL_FEATURES.md
Architecture       → docs/SYSTEM_ARCHITECTURE.md
Module schemas     → core/modules/**/*.yaml
JSON:API impl      → pkg/jsonapi/             # Implements docs/spec/
API handlers       → adapters/http/admin/     # Uses pkg/jsonapi/
```

---

## CI/CD Automation

### Automated Workflows

| Workflow | Trigger | What It Does |
|----------|---------|--------------|
| `ci.yml` | Push/PR to main | Tests, coverage check, builds, vet, docs validation |
| `release.yml` | Tag `v*` | Builds all platforms, GitHub release, Docker, Homebrew |
| `wiki-sync.yml` | Push to main (docs/spec) | Syncs docs/spec to GitHub wiki |

### Creating a Release

Use the release script to automate version bumping and tagging:

```bash
# Patch release (bug fixes): v0.1.10 -> v0.1.11
./scripts/prepare-release.sh patch

# Minor release (new features): v0.1.10 -> v0.2.0
./scripts/prepare-release.sh minor

# Major release (breaking changes): v0.1.10 -> v1.0.0
./scripts/prepare-release.sh major
```

The script will:
1. Fetch latest tags
2. Calculate next version
3. Show commits since last release
4. Prompt for confirmation
5. Create and push tag

GitHub Actions then automatically:
1. Runs tests
2. Builds binaries (Linux, macOS, Windows)
3. Creates GitHub release with changelog
4. Builds/pushes Docker image
5. Updates Homebrew formula
6. Syncs wiki from docs/spec

### Manual Wiki Sync

```bash
# Preview changes
./scripts/sync-wiki.sh --dry-run

# Sync to wiki
./scripts/sync-wiki.sh
```

---

## GitHub Wiki Synchronization

The GitHub wiki mirrors `docs/spec/` for external visibility.

**Automated**: Wiki syncs automatically on push to main when `docs/spec/` changes.

### First-Time Setup

The wiki must be initialized via GitHub UI before git access works:
1. Go to https://github.com/artpar/apigate/wiki
2. Click "Create the first page"
3. Save any content (will be replaced by sync)

### Wiki Structure

| Wiki Page | Source |
|-----------|--------|
| Home | `docs/spec/README.md` |
| JSON:API-Format | `docs/spec/json-api.md` |
| Error-Codes | `docs/spec/error-codes.md` |
| Pagination | `docs/spec/pagination.md` |
| Resource-Types | `docs/spec/resource-types.md` |
| TLS-Certificates | `docs/spec/tls-certificates.md` |
| Metering-API | `docs/spec/metering-api.md` |

**Important**: Always edit `docs/spec/` first. Wiki syncs automatically on push to main.

---

## Implementation Notes

### HTTP Channel Module Endpoints

The HTTP channel (`core/channel/http/http.go`) respects module YAML endpoint definitions:

1. **Explicit Endpoints** (from `channels.http.serve.endpoints`):
   - Registered via `registerExplicitEndpoints()`
   - Use custom base_path if defined
   - Support all HTTP methods (GET, POST, PUT, PATCH, DELETE)
   - Path parameters extracted via `chi.URLParam(r, "param")`

2. **Implicit CRUD** (fallback when no endpoints defined):
   - Generated from `mod.Actions`
   - Uses `mod.Plural` as base path

### Module Types

| Type | Location | Pattern |
|------|----------|---------|
| Core Data Modules | `core/modules/*.yaml` | No `meta.implements` |
| Capability Definitions | `core/modules/capabilities/*.yaml` | `capability: name` |
| Provider Implementations | `core/modules/providers/*.yaml` | `meta.implements: [cap]` |

**Example**: `setting.yaml` is a core data module (stores config), while `tls_acme.yaml` is a provider (implements TLS capability).

### Custom Action Handlers

Custom actions in module YAML (e.g., `get_by_domain`, `list_expiring`) are executed via:

```go
result, err := c.runtime.Execute(ctx, mod.Source.Name, action.Name, input)
```

Path parameters are extracted and passed in `input.Data`:
- `{id}` → `data["id"]`
- `{domain}` → `data["domain"]` or `input.Lookup`
- `{prefix}` → `data["prefix"]`

### When Removing Legacy Handlers

When module YAML replaces hand-coded handlers:
1. Delete the handler file (e.g., `settings.go`)
2. Remove route registration from router
3. Remove tests for deleted handlers
4. Update documentation (spec + wiki)

---

## Remember

> **Documentation is not separate from code - it IS the code.**
>
> Every code change that affects behavior should update documentation in the same commit.
>
> If documentation can become stale, the architecture is wrong.
>
> **The spec defines behavior. Code implements the spec. Tests verify the spec.**
