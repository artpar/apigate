# APIGate Project Standards

> **These are non-negotiable release blockers. Failing any of these criteria blocks release.**

---

## Core Principles

APIGate is built on five foundational principles. Every feature, every change, every release must satisfy all five.

| # | Principle | Summary |
|---|-----------|---------|
| 1 | **Self-Onboarding** | Zero human intervention required for any user to start |
| 2 | **Self-Service** | Users can accomplish everything without contacting support |
| 3 | **Self-Documenting** | Single source of truth, code and docs always in sync |
| 4 | **Type Safety** | Explicit types everywhere, no `any`, no runtime type errors |
| 5 | **Test Coverage** | 80%+ coverage, all critical paths tested |

---

## Principle 1: Self-Onboarding

### Definition

Every user type (API Seller, API Buyer) must be able to complete their onboarding journey without any human assistance, documentation reading beyond tooltips, or external support.

### Requirements

| Requirement | Validation |
|-------------|------------|
| Setup wizard completes end-to-end without docs | E2E test |
| Customer signup to first API call < 5 minutes | UX testing |
| No required configuration files | Architecture review |
| No required CLI commands post-install | User journey test |
| All form fields have inline help/validation | UI audit |
| Error messages include recovery actions | Error catalog review |

### Measurable Criteria

```
┌─────────────────────────────────────────────────────────────────────┐
│ SELF-ONBOARDING METRICS                          Target    Blocker │
├─────────────────────────────────────────────────────────────────────┤
│ Admin setup completion rate (no docs)            > 95%      < 80%  │
│ Customer signup completion rate                  > 90%      < 75%  │
│ Time to first successful API call                < 5 min    > 15m  │
│ Setup wizard abandonment rate                    < 10%      > 25%  │
│ Support tickets during onboarding                < 5%       > 15%  │
└─────────────────────────────────────────────────────────────────────┘
```

### Validation

- [ ] Fresh install E2E test passes (admin journey)
- [ ] Fresh signup E2E test passes (customer journey)
- [ ] UX audit by non-technical user completes successfully
- [ ] No "how do I...?" questions in first 10 minutes

### Implementation Checklist

Every new feature must answer:
- [ ] Can a new user discover this without documentation?
- [ ] Are all inputs validated with helpful error messages?
- [ ] Is there a sensible default that works for 80% of users?
- [ ] Does the UI guide the user to the next step?

---

## Principle 2: Self-Service

### Definition

Every operation a user might need to perform must be available through the UI without requiring admin intervention, support tickets, or direct database access.

### Requirements

| Capability | Admin (API Seller) | Customer (API Buyer) |
|------------|-------------------|----------------------|
| Account creation | Setup wizard | Signup form |
| Password reset | Self-service | Self-service |
| Plan management | Full CRUD | View, upgrade |
| API key management | View all, revoke | Create, revoke own |
| Usage monitoring | All users | Own usage |
| Billing management | Configure | View, update payment |
| Account deletion | N/A | Self-service |
| Data export | Available | Available |

### Measurable Criteria

```
┌─────────────────────────────────────────────────────────────────────┐
│ SELF-SERVICE METRICS                             Target    Blocker │
├─────────────────────────────────────────────────────────────────────┤
│ Operations requiring support                     0%        > 5%    │
│ Features accessible via UI                       100%      < 95%   │
│ Admin-only operations (non-security)             0         > 3     │
│ Database-only operations                         0         > 0     │
│ Support ticket rate for "how do I..."            < 2%      > 10%   │
└─────────────────────────────────────────────────────────────────────┘
```

### Validation

- [ ] Every user journey (J1-J9) completable via UI
- [ ] No feature requires CLI/database access for normal operation
- [ ] Password reset flow works end-to-end
- [ ] Account deletion available (with appropriate warnings)
- [ ] Data export available in standard format

### Implementation Checklist

Every new feature must answer:
- [ ] Can the user do this entirely through the UI?
- [ ] Is there an undo/recovery path?
- [ ] Are confirmations provided for destructive actions?
- [ ] Does the user know the operation succeeded?

---

## Principle 3: Self-Documenting

### Definition

Documentation and implementation must be derived from single sources of truth. No separate docs that can drift from code. Types, schemas, and behavior must be introspectable.

### Single Sources of Truth

| Concept | Source of Truth | Derived From |
|---------|-----------------|--------------|
| API endpoints | Go handlers + OpenAPI | Auto-generated docs |
| Module schema | YAML definitions | UI forms, validation |
| Error codes | errors.go constants | Error documentation |
| CLI commands | Cobra definitions | CLI help, docs |
| Configuration | Env var definitions | README, docs |
| Database schema | Migrations | Entity documentation |

### Requirements

| Requirement | Implementation |
|-------------|----------------|
| API docs generated from code | OpenAPI spec from handlers |
| UI forms generated from schema | Module YAML → React forms |
| Error messages from code constants | Single error catalog |
| CLI help from command definitions | Cobra auto-help |
| No hand-written API docs that can drift | Generated only |

### Measurable Criteria

```
┌─────────────────────────────────────────────────────────────────────┐
│ SELF-DOCUMENTING METRICS                         Target    Blocker │
├─────────────────────────────────────────────────────────────────────┤
│ Hand-written API endpoint docs                   0         > 0     │
│ Schema-driven UI forms                           100%      < 100%  │
│ Documented vs actual endpoints match             100%      < 100%  │
│ Error codes in catalog vs code match             100%      < 100%  │
│ Stale documentation issues                       0         > 3     │
└─────────────────────────────────────────────────────────────────────┘
```

### Validation

- [ ] OpenAPI spec matches actual endpoints (automated test)
- [ ] All module UIs generated from YAML schemas
- [ ] Error catalog matches error constants in code
- [ ] No TODO comments about updating docs
- [ ] CI check for schema/docs drift

### Implementation Checklist

Every new feature must answer:
- [ ] What is the single source of truth for this?
- [ ] How are docs/UI derived from that source?
- [ ] Is there a CI check to catch drift?
- [ ] Can this documentation become stale? If so, how to prevent?

---

## Principle 4: Type Safety

### Definition

All code must be explicitly typed. No implicit any, no runtime type coercion surprises, no "trust me" casting. Types are documentation that the compiler enforces.

### Requirements

#### Go (Backend)

| Requirement | Enforcement |
|-------------|-------------|
| No `interface{}` without type assertion | Linter rule |
| All public functions documented | golint |
| Error handling explicit | errcheck linter |
| No panics in production code | Panic-free guarantee |
| Struct fields typed and tagged | Required |

#### TypeScript (Frontend)

| Requirement | Enforcement |
|-------------|-------------|
| `strict: true` in tsconfig | CI check |
| No `any` type | ESLint rule |
| No `@ts-ignore` without justification | PR review |
| Props interfaces for all components | Required |
| API responses typed | Generated from OpenAPI |

### Measurable Criteria

```
┌─────────────────────────────────────────────────────────────────────┐
│ TYPE SAFETY METRICS                              Target    Blocker │
├─────────────────────────────────────────────────────────────────────┤
│ TypeScript strict mode                           enabled   disabled│
│ `any` types in codebase                          0         > 10    │
│ `interface{}` without assertion                  0         > 5     │
│ Runtime type errors in logs                      0         > 0     │
│ @ts-ignore comments                              0         > 5     │
│ Untyped API responses                            0         > 0     │
└─────────────────────────────────────────────────────────────────────┘
```

### Validation

- [ ] `tsc --noEmit` passes with strict mode
- [ ] ESLint no-any rule passes
- [ ] Go vet passes
- [ ] No type assertion errors in production logs
- [ ] API types generated from OpenAPI spec

### Implementation Checklist

Every new code must answer:
- [ ] Are all function parameters and returns typed?
- [ ] Are error cases handled explicitly?
- [ ] Would a type change here break compilation (good) or runtime (bad)?
- [ ] Is the type narrow enough to prevent misuse?

---

## Principle 5: Test Coverage

### Definition

Minimum 80% code coverage, with 100% coverage of critical paths (auth, payment, data integrity). Tests are not optional - they are part of the feature.

### Coverage Requirements

| Category | Minimum Coverage | Critical Paths |
|----------|------------------|----------------|
| **Overall** | 80% | N/A |
| **Authentication** | 95% | Login, API key validation |
| **Payment** | 95% | Checkout, webhooks |
| **Data integrity** | 95% | User CRUD, plan assignment |
| **API proxy** | 90% | Request forwarding, rate limiting |
| **UI components** | 70% | Forms, navigation |

### Test Types Required

| Type | Purpose | Coverage |
|------|---------|----------|
| **Unit tests** | Individual functions | 80% lines |
| **Integration tests** | Component interaction | Key flows |
| **E2E tests** | Full user journeys | J1-J9 |
| **Contract tests** | API compatibility | All endpoints |
| **Property tests** | Edge cases | Rate limiting, validation |

### Measurable Criteria

```
┌─────────────────────────────────────────────────────────────────────┐
│ TEST COVERAGE METRICS                            Target    Blocker │
├─────────────────────────────────────────────────────────────────────┤
│ Overall code coverage                            > 80%     < 70%   │
│ Auth module coverage                             > 95%     < 90%   │
│ Payment module coverage                          > 95%     < 90%   │
│ E2E tests passing                                100%      < 100%  │
│ Flaky test rate                                  < 1%      > 5%    │
│ Test run time                                    < 5 min   > 15m   │
└─────────────────────────────────────────────────────────────────────┘
```

### Validation

- [ ] `go test -cover ./...` shows > 80%
- [ ] Critical path coverage verified
- [ ] All E2E journeys pass
- [ ] No skipped tests without justification
- [ ] Coverage report in CI

### Implementation Checklist

Every new feature must include:
- [ ] Unit tests for new functions
- [ ] Integration tests for component interaction
- [ ] E2E test updates if user journey affected
- [ ] Edge case tests for error paths
- [ ] Coverage delta in PR (must not decrease)

---

## Release Criteria

### Pre-Release Checklist

A release is blocked if ANY of these fail:

```
┌─────────────────────────────────────────────────────────────────────┐
│ RELEASE BLOCKER CHECKLIST                                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ SELF-ONBOARDING                                                     │
│ [ ] Fresh install E2E passes                                        │
│ [ ] Customer signup E2E passes                                      │
│ [ ] No required manual steps documented                             │
│                                                                     │
│ SELF-SERVICE                                                        │
│ [ ] All user journeys (J1-J9) work via UI                          │
│ [ ] No new admin-only operations for user tasks                     │
│ [ ] Password reset works                                            │
│                                                                     │
│ SELF-DOCUMENTING                                                    │
│ [ ] OpenAPI spec matches handlers                                   │
│ [ ] No documentation drift detected                                 │
│ [ ] Error catalog up to date                                        │
│                                                                     │
│ TYPE SAFETY                                                         │
│ [ ] TypeScript strict mode passes                                   │
│ [ ] No new `any` types                                              │
│ [ ] Go vet passes                                                   │
│                                                                     │
│ TEST COVERAGE                                                       │
│ [ ] Overall coverage > 80%                                          │
│ [ ] Critical path coverage > 95%                                    │
│ [ ] All E2E tests pass                                              │
│ [ ] No new flaky tests                                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Automated Checks (CI)

These must pass in CI before merge:

```yaml
# .github/workflows/standards.yml
checks:
  - name: type-safety
    commands:
      - tsc --noEmit --strict
      - golangci-lint run
    blocker: true

  - name: test-coverage
    commands:
      - go test -coverprofile=coverage.out ./...
      - go tool cover -func=coverage.out | grep total | awk '{print $3}' | check-threshold 80
    blocker: true

  - name: e2e-tests
    commands:
      - npx playwright test
    blocker: true

  - name: docs-drift
    commands:
      - ./scripts/check-openapi-drift.sh
      - ./scripts/check-error-catalog.sh
    blocker: true
```

### Manual Checks (Pre-Release)

Before each release:

1. **Fresh Install Test**
   - Deploy to clean environment
   - Complete setup wizard as new user
   - Time the process (must be < 5 minutes)

2. **User Journey Audit**
   - Walk through J1-J9 manually
   - Note any friction points
   - Verify all screenshots match current UI

3. **Documentation Review**
   - Verify README accuracy
   - Check all links work
   - Confirm version numbers updated

---

## Tracking & Reporting

### Weekly Standards Report

Generate weekly with:

```bash
./scripts/standards-report.sh
```

Output:

```
APIGate Standards Report - Week of Jan 15, 2024
================================================

SELF-ONBOARDING
  Setup completion rate: 94% (Target: >95%) ⚠️
  Signup completion rate: 92% (Target: >90%) ✓
  Time to first API call: 3.2 min (Target: <5 min) ✓

SELF-SERVICE
  UI-accessible features: 100% ✓
  Support tickets (how-to): 1.8% (Target: <2%) ✓

SELF-DOCUMENTING
  OpenAPI drift: 0 endpoints ✓
  Error catalog match: 100% ✓

TYPE SAFETY
  TypeScript any count: 0 ✓
  Go interface{} unasserted: 0 ✓

TEST COVERAGE
  Overall: 82.3% (Target: >80%) ✓
  Auth module: 96.1% (Target: >95%) ✓
  Payment module: 94.8% (Target: >95%) ⚠️
  E2E pass rate: 100% ✓

BLOCKERS: 2 warnings, 0 critical
```

### Dashboard (Future)

Display standards compliance in admin dashboard:
- Current coverage percentage
- E2E test status
- Documentation freshness
- Type safety score

---

## Exceptions Process

### When Standards Can Be Waived

Never for:
- Security-related code (auth, payment)
- Core user journeys
- Type safety violations

Rarely, with justification, for:
- Experimental features (flagged)
- Third-party integration edge cases
- Performance-critical code (documented)

### Exception Request

```markdown
## Standards Exception Request

**Principle:** [Which principle]
**Scope:** [What code/feature]
**Justification:** [Why exception needed]
**Mitigation:** [How risk is managed]
**Timeline:** [When will this be resolved]
**Approver:** [Who approved]
```

---

## Appendix: Quick Reference

### For New Features

```
Before starting:
[ ] Identify which user journey this affects
[ ] Determine single source of truth
[ ] Plan test coverage approach

Before PR:
[ ] Self-onboarding: Can a new user discover this?
[ ] Self-service: Is this fully UI-accessible?
[ ] Self-documenting: Is the source of truth clear?
[ ] Type safety: All types explicit?
[ ] Test coverage: >80% for new code?

Before merge:
[ ] CI passes
[ ] Coverage not decreased
[ ] E2E still passes
```

### For Code Review

```
Reviewer checklist:
[ ] Types are explicit and narrow
[ ] Tests cover happy path and errors
[ ] UI includes validation and help text
[ ] No hand-written docs that could drift
[ ] Feature is discoverable without docs
```

---

## Document History

| Date | Change | Author |
|------|--------|--------|
| 2024-XX-XX | Initial standards document | Team |

---

*These standards exist to ensure APIGate delivers on its promise: Turn any API into a revenue stream. In 5 minutes. Every deviation from these standards is a step away from that promise.*
