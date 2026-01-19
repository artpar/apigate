# Documentation Verification Checklist

> **Purpose**: Systematically verify LLM-generated documentation against actual codebase to detect hallucinations and ensure accuracy.

---

## Verification Methodology

### Principles

1. **Code is truth** - Documentation must match what code actually does
2. **Existence checks first** - Verify things exist before checking details
3. **Test coverage validates** - If tests pass, behavior is likely correct
4. **Cross-reference multiple sources** - Never trust a single source

### Hallucination Detection Patterns

Common LLM hallucination patterns to watch for:

| Pattern | Example | Detection Method |
|---------|---------|------------------|
| **Invented endpoints** | `POST /api/v1/sync` that doesn't exist | Grep for route registration |
| **Wrong parameter names** | `user_id` vs actual `userId` | Check struct field tags |
| **Fabricated features** | "Supports WebSocket streaming" | Search for feature code |
| **Incorrect defaults** | "Default rate limit is 1000/min" | Check config/constants |
| **Non-existent error codes** | `ERR_QUOTA_SOFT_LIMIT` | Check error definitions |
| **Wrong HTTP methods** | "DELETE /users" vs actual "POST /users/delete" | Check route handlers |
| **Imaginary config options** | `ENABLE_DARK_MODE=true` | Check env var parsing |

---

## Verification Progress Tracker

### Legend
- [ ] Not started
- [~] In progress
- [x] Verified correct
- [!] Issues found (see notes)

---

## Section 1: API Specification (docs/spec/)

### 1.1 JSON:API Format (`docs/spec/json-api.md`)

| Check | Status | Verified By | Date | Notes |
|-------|--------|-------------|------|-------|
| Document structure matches `pkg/jsonapi/types.go` | [ ] | | | |
| Resource object fields match actual structs | [ ] | | | |
| Relationship format matches implementation | [ ] | | | |
| Links format is correct | [ ] | | | |
| Meta format is correct | [ ] | | | |
| Example responses match actual API output | [ ] | | | |

**Verification Commands:**
```bash
# Check types.go for actual structure
grep -A 20 "type Document struct" pkg/jsonapi/types.go

# Compare example output with actual
curl -s http://localhost:8080/admin/users | jq '.data[0]'
```

### 1.2 Error Codes (`docs/spec/error-codes.md`)

| Check | Status | Verified By | Date | Notes |
|-------|--------|-------------|------|-------|
| All documented error codes exist in `pkg/jsonapi/errors.go` | [ ] | | | |
| HTTP status codes match implementation | [ ] | | | |
| Error code strings match exactly | [ ] | | | |
| Error messages/titles are accurate | [ ] | | | |
| No undocumented error codes in code | [ ] | | | |

**Verification Commands:**
```bash
# Extract all error codes from implementation
grep -E "NewError\([0-9]+," pkg/jsonapi/errors.go

# Extract all error codes from docs
grep -E "^\| `[a-z_]+`" docs/spec/error-codes.md
```

### 1.3 Pagination (`docs/spec/pagination.md`)

| Check | Status | Verified By | Date | Notes |
|-------|--------|-------------|------|-------|
| Query parameter names match code | [ ] | | | |
| Default page size is accurate | [ ] | | | |
| Max page size is accurate | [ ] | | | |
| Link generation format matches | [ ] | | | |
| Cursor pagination (if documented) exists | [ ] | | | |

**Verification Commands:**
```bash
# Check pagination implementation
grep -r "page\[" adapters/http/
grep -r "PageSize\|PageNumber" pkg/jsonapi/
```

### 1.4 Resource Types (`docs/spec/resource-types.md`)

| Check | Status | Verified By | Date | Notes |
|-------|--------|-------------|------|-------|
| All documented resource types have handlers | [ ] | | | |
| Attribute names match JSON tags | [ ] | | | |
| Relationship names are accurate | [ ] | | | |
| Required vs optional fields match validation | [ ] | | | |
| No undocumented resource types in code | [ ] | | | |

**Verification Commands:**
```bash
# Find all resource types in handlers
grep -r '"type":' adapters/http/admin/*.go

# Check YAML module definitions
ls core/modules/*.yaml
```

---

## Section 2: Wiki Documentation (docs/spec/wiki/)

### 2.1 Core Concepts

| Document | Status | Checks |
|----------|--------|--------|
| `Home.md` | [ ] | Feature list exists, links work |
| `Architecture.md` | [ ] | Component names match code structure |
| `Installation.md` | [ ] | Commands actually work |
| `Quick-Start.md` | [ ] | All steps reproducible |
| `Configuration.md` | [ ] | All env vars exist in code |

### 2.2 Features

| Document | Status | Code Location | Verification Method |
|----------|--------|---------------|---------------------|
| `Upstreams.md` | [ ] | `adapters/http/upstream.go` | Compare fields |
| `Routes.md` | [ ] | `adapters/http/admin/routes.go` | Compare fields |
| `Rate-Limiting.md` | [ ] | `core/ratelimit/` | Check algorithms |
| `Quotas.md` | [ ] | `core/quota/` | Check implementation |
| `Transformations.md` | [ ] | Search for transform code | Verify exists |
| `Webhooks.md` | [ ] | `core/modules/webhook.yaml` | Compare schema |
| `Usage-Tracking.md` | [ ] | Search for tracking code | Verify exists |
| `Groups.md` | [ ] | `core/modules/group.yaml` | Compare schema |
| `OAuth.md` | [ ] | `core/modules/capabilities/oauth.yaml` | Compare providers |
| `Certificates.md` | [ ] | `core/modules/certificate.yaml` | Compare schema |
| `Customer-Portal.md` | [ ] | `web/` directory | Check routes |

### 2.3 Tutorials

| Document | Status | Verification Method |
|----------|--------|---------------------|
| `Tutorial-Basic-Setup.md` | [ ] | Execute all commands |
| `Tutorial-Monetization.md` | [ ] | Check referenced features exist |
| `Tutorial-Stripe.md` | [ ] | Check Stripe integration code |
| `Tutorial-Production.md` | [ ] | Verify deployment configs |
| `Tutorial-Basic-API.md` | [ ] | Execute all API calls |
| `Tutorial-Custom-Portal.md` | [ ] | Check customization code |

### 2.4 Payment Providers

| Document | Status | YAML Definition | Provider Code |
|----------|--------|-----------------|---------------|
| `Payment-Stripe.md` | [ ] | `payment_stripe.yaml` | Check impl |
| `Payment-Paddle.md` | [ ] | Check if exists | Check impl |
| `Payment-LemonSqueezy.md` | [ ] | Check if exists | Check impl |

### 2.5 Reference

| Document | Status | Verification Method |
|----------|--------|---------------------|
| `CLI-Reference.md` | [ ] | Run `apigate --help`, compare all commands |
| `API-Reference.md` | [ ] | Compare with route registrations |
| `Error-Codes.md` (wiki) | [ ] | Should match `docs/spec/error-codes.md` |

---

## Section 3: User Journeys (docs/user_journeys/)

### 3.1 Admin Journeys

| Journey | Status | UI Routes Exist | API Endpoints Exist | Flow Testable |
|---------|--------|-----------------|---------------------|---------------|
| `j1-first-time-setup.md` | [ ] | [ ] | [ ] | [ ] |
| `j2-plan-management.md` | [ ] | [ ] | [ ] | [ ] |
| `j3-monitoring.md` | [ ] | [ ] | [ ] | [ ] |
| `j4-platform-config.md` | [ ] | [ ] | [ ] | [ ] |

### 3.2 Customer Journeys

| Journey | Status | UI Routes Exist | API Endpoints Exist | Flow Testable |
|---------|--------|-----------------|---------------------|---------------|
| `j5-onboarding.md` | [ ] | [ ] | [ ] | [ ] |
| `j6-api-access.md` | [ ] | [ ] | [ ] | [ ] |
| `j7-usage-monitoring.md` | [ ] | [ ] | [ ] | [ ] |
| `j8-plan-upgrade.md` | [ ] | [ ] | [ ] | [ ] |
| `j9-documentation.md` | [ ] | [ ] | [ ] | [ ] |

### 3.3 Error Journeys

| Journey | Status | Error Codes Match | UX Flow Accurate |
|---------|--------|-------------------|------------------|
| `authentication-errors.md` | [ ] | [ ] | [ ] |
| `rate-limiting.md` | [ ] | [ ] | [ ] |
| `quota-exceeded.md` | [ ] | [ ] | [ ] |

---

## Section 4: Technical Documentation

### 4.1 SYSTEM_ARCHITECTURE.md

| Check | Status | Notes |
|-------|--------|-------|
| Module list matches `core/modules/*.yaml` | [ ] | |
| Capability list matches `core/modules/capabilities/*.yaml` | [ ] | |
| Provider list matches `core/modules/providers/*.yaml` | [ ] | |
| Data models match actual structs | [ ] | |
| Flow diagrams match code paths | [ ] | |

### 4.2 TECHNICAL_FEATURES.md

| Check | Status | Notes |
|-------|--------|-------|
| Listed features have implementations | [ ] | |
| API endpoints exist and work | [ ] | |
| CLI commands exist and work | [ ] | |
| Configuration options exist | [ ] | |

### 4.3 USER_GUIDE.md

| Check | Status | Notes |
|-------|--------|-------|
| All referenced UI pages exist | [ ] | |
| All referenced API endpoints work | [ ] | |
| Screenshots match current UI | [ ] | |
| Instructions are accurate | [ ] | |

---

## Section 5: Module YAML Definitions

### 5.1 Core Modules

| Module YAML | Status | Fields Match DB Schema | Documented Accurately |
|-------------|--------|------------------------|----------------------|
| `user.yaml` | [ ] | [ ] | [ ] |
| `route.yaml` | [ ] | [ ] | [ ] |
| `upstream.yaml` | [ ] | [ ] | [ ] |
| `plan.yaml` | [ ] | [ ] | [ ] |
| `setting.yaml` | [ ] | [ ] | [ ] |
| `api_key.yaml` | [ ] | [ ] | [ ] |
| `group.yaml` | [ ] | [ ] | [ ] |
| `group_member.yaml` | [ ] | [ ] | [ ] |
| `group_invite.yaml` | [ ] | [ ] | [ ] |
| `entitlement.yaml` | [ ] | [ ] | [ ] |
| `plan_entitlement.yaml` | [ ] | [ ] | [ ] |
| `webhook.yaml` | [ ] | [ ] | [ ] |
| `webhook_delivery.yaml` | [ ] | [ ] | [ ] |
| `certificate.yaml` | [ ] | [ ] | [ ] |
| `oauth_identity.yaml` | [ ] | [ ] | [ ] |
| `oauth_state.yaml` | [ ] | [ ] | [ ] |

### 5.2 Capabilities

| Capability YAML | Status | Has Providers | Documented |
|-----------------|--------|---------------|------------|
| `auth.yaml` | [ ] | [ ] | [ ] |
| `cache.yaml` | [ ] | [ ] | [ ] |
| `email.yaml` | [ ] | [ ] | [ ] |
| `notification.yaml` | [ ] | [ ] | [ ] |
| `oauth.yaml` | [ ] | [ ] | [ ] |
| `payment.yaml` | [ ] | [ ] | [ ] |
| `queue.yaml` | [ ] | [ ] | [ ] |
| `reconciliation.yaml` | [ ] | [ ] | [ ] |
| `storage.yaml` | [ ] | [ ] | [ ] |
| `sync.yaml` | [ ] | [ ] | [ ] |
| `tls.yaml` | [ ] | [ ] | [ ] |
| `data_source.yaml` | [ ] | [ ] | [ ] |

### 5.3 Providers

| Provider YAML | Status | Implementation Exists | Documented |
|---------------|--------|----------------------|------------|
| `auth_builtin.yaml` | [ ] | [ ] | [ ] |
| `cache_memory.yaml` | [ ] | [ ] | [ ] |
| `cache_redis.yaml` | [ ] | [ ] | [ ] |
| `email_smtp.yaml` | [ ] | [ ] | [ ] |
| `email_sendgrid.yaml` | [ ] | [ ] | [ ] |
| `email_log.yaml` | [ ] | [ ] | [ ] |
| `notification_slack.yaml` | [ ] | [ ] | [ ] |
| `notification_webhook.yaml` | [ ] | [ ] | [ ] |
| `notification_log.yaml` | [ ] | [ ] | [ ] |
| `oauth_google.yaml` | [ ] | [ ] | [ ] |
| `oauth_github.yaml` | [ ] | [ ] | [ ] |
| `oauth_oidc.yaml` | [ ] | [ ] | [ ] |
| `payment_stripe.yaml` | [ ] | [ ] | [ ] |
| `payment_dummy.yaml` | [ ] | [ ] | [ ] |
| `queue_memory.yaml` | [ ] | [ ] | [ ] |
| `queue_redis.yaml` | [ ] | [ ] | [ ] |
| `reconciliation_default.yaml` | [ ] | [ ] | [ ] |
| `storage_disk.yaml` | [ ] | [ ] | [ ] |
| `storage_s3.yaml` | [ ] | [ ] | [ ] |
| `storage_memory.yaml` | [ ] | [ ] | [ ] |
| `sync_table.yaml` | [ ] | [ ] | [ ] |
| `tls_acme.yaml` | [ ] | [ ] | [ ] |

---

## Verification Scripts

### Script 1: Extract All API Endpoints from Code

```bash
#!/bin/bash
# Save as: scripts/verify-endpoints.sh

echo "=== Registered Routes ==="
grep -rn "router\.\(GET\|POST\|PUT\|PATCH\|DELETE\|Handle\)" adapters/http/ | \
  sed 's/.*"\([^"]*\)".*/\1/' | sort | uniq

echo ""
echo "=== Handler Functions ==="
grep -rn "func.*Handler\|func Handle" adapters/http/admin/*.go | \
  grep -v "_test.go"
```

### Script 2: Extract All Error Codes

```bash
#!/bin/bash
# Save as: scripts/verify-errors.sh

echo "=== Error Codes in Code ==="
grep -E "NewError\(" pkg/jsonapi/errors.go | \
  sed 's/.*NewError(\([0-9]*\), "\([^"]*\)".*/\1 \2/'

echo ""
echo "=== Error Codes in Docs ==="
grep -E "^\| `[a-z_]+`" docs/spec/error-codes.md
```

### Script 3: Extract All Env Vars

```bash
#!/bin/bash
# Save as: scripts/verify-config.sh

echo "=== Env Vars in Code ==="
grep -rh "os\.Getenv\|viper\.Get" --include="*.go" . | \
  grep -oE '"[A-Z_]+"' | sort | uniq

echo ""
echo "=== Env Vars in Docs ==="
grep -E "^[A-Z_]+=" docs/spec/wiki/Configuration.md 2>/dev/null || \
  echo "No Configuration.md found"
```

### Script 4: Compare Module YAML vs Documentation

```bash
#!/bin/bash
# Save as: scripts/verify-modules.sh

echo "=== Module YAMLs ==="
ls core/modules/*.yaml | xargs -I{} basename {} .yaml

echo ""
echo "=== Documented Resource Types ==="
grep -E "^### " docs/spec/resource-types.md | sed 's/### //'
```

---

## Verification Session Template

When starting a verification session, copy this template:

```markdown
## Verification Session: [DATE]

**Verifier**: [NAME/AI]
**Focus Area**: [Section being verified]
**Commit**: [git commit hash]

### Files Verified

| File | Result | Issues |
|------|--------|--------|
| | | |

### Issues Found

1. **[FILE:LINE]** - Description of issue
   - Expected: X
   - Actual: Y
   - Severity: High/Medium/Low

### Corrections Made

| File | Change | Commit |
|------|--------|--------|
| | | |

### Session Notes

[Any observations, patterns noticed, etc.]
```

---

## Issue Severity Classification

| Severity | Definition | Examples |
|----------|------------|----------|
| **Critical** | Completely wrong, would cause errors | Non-existent endpoint documented |
| **High** | Misleading, could cause confusion | Wrong parameter name |
| **Medium** | Inaccurate but not harmful | Wrong default value |
| **Low** | Minor discrepancy | Typo, formatting issue |

---

## Automated Verification Ideas

Future automation opportunities:

1. **OpenAPI diff** - Generate OpenAPI from code, compare to docs
2. **Type extraction** - Parse Go structs, compare to documented types
3. **Test coverage check** - Ensure documented features have tests
4. **Link checker** - Verify all doc links resolve
5. **Example validator** - Run documented curl examples, check responses

---

## Quick Reference: Key Code Locations

| Concept | Primary Code Location |
|---------|----------------------|
| API handlers | `adapters/http/admin/*.go` |
| Route registration | `adapters/http/admin/routes.go` |
| JSON:API types | `pkg/jsonapi/types.go` |
| Error definitions | `pkg/jsonapi/errors.go` |
| Module schemas | `core/modules/*.yaml` |
| Capabilities | `core/modules/capabilities/*.yaml` |
| Providers | `core/modules/providers/*.yaml` |
| Web handlers | `web/handlers.go` |
| Main entry | `main.go` |

---

## Appendix: Document Inventory

Total documents to verify: ~80

| Category | Count |
|----------|-------|
| docs/spec/ (core) | 5 |
| docs/spec/wiki/ | ~50 |
| docs/user_journeys/ | 12 |
| docs/ (top-level) | 6 |
| Module YAMLs | 50 |

Estimated effort: 2-4 hours per section for thorough verification.
