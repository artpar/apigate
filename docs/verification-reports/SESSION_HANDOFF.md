# Documentation Verification Session Handoff

**Last Updated**: 2026-01-19T21:00:00+05:30
**Session**: Wiki Verification Session 3

---

## Task Overview

Systematic verification of wiki documentation against actual codebase to identify and fix "hallucinated" content - incorrect CLI commands, non-existent API endpoints, wrong environment variables, fabricated features, and inaccurate claims.

---

## Progress Summary

| Metric | Count |
|--------|-------|
| **Total wiki files** | 60 |
| **Files verified** | 30 |
| **Files remaining** | 30 |
| **Issues found** | 27 |
| **Issues fixed** | 26 |

---

## Files VERIFIED (30 files - DO NOT RE-VERIFY)

### Core Spec Files (4)
- [x] docs/spec/error-codes.md
- [x] docs/spec/json-api.md
- [x] docs/spec/pagination.md
- [x] docs/spec/resource-types.md

### Wiki Files - Verified & Fixed (26)
- [x] Configuration.md
- [x] Routes.md
- [x] Upstreams.md
- [x] API-Keys.md
- [x] Authentication.md
- [x] Users.md
- [x] Plans.md
- [x] Admin-Invites.md
- [x] Analytics.md
- [x] API-Documentation.md
- [x] Branding.md
- [x] Database-Setup.md
- [x] Email-Configuration.md
- [x] Events.md
- [x] Providers.md
- [x] Notifications.md
- [x] Webhooks.md
- [x] Home.md
- [x] Rate-Limiting.md
- [x] Quotas.md
- [x] OAuth.md
- [x] Security.md
- [x] Certificates.md
- [x] Troubleshooting.md
- [x] FAQ.md
- [x] docs/spec/README.md (has 1 unfixed issue - test references)

---

## Files NOT YET VERIFIED (30 files - VERIFY NEXT)

### High Priority (likely have issues based on patterns)
- [ ] **CLI-Reference.md** - LIKELY HAS MANY HALLUCINATIONS
- [ ] **Installation.md** - Check commands
- [ ] **Quick-Start.md** - Check commands
- [ ] **Production.md** - Check deployment advice
- [ ] **Tutorial-Basic-API.md** - Check CLI examples
- [ ] **Tutorial-Basic-Setup.md** - Check CLI examples
- [ ] **Tutorial-Monetization.md** - Check CLI examples
- [ ] **Tutorial-Stripe.md** - Check settings/commands
- [ ] **Tutorial-Production.md** - Check deployment commands
- [ ] **Tutorial-Custom-Portal.md** - Check customization settings

### Medium Priority
- [ ] Architecture.md
- [ ] Request-Lifecycle.md
- [ ] Module-System.md
- [ ] Transformations.md
- [ ] Proxying.md
- [ ] Protocols.md
- [ ] Customer-Portal.md
- [ ] Usage-Tracking.md
- [ ] Usage-Metering.md
- [ ] Groups.md
- [ ] Entitlements.md
- [ ] Billing.md
- [ ] Pricing-Integration.md
- [ ] First-Customer.md

### Payment Integration (check settings/commands)
- [ ] Payment-Stripe.md
- [ ] Payment-Paddle.md
- [ ] Payment-LemonSqueezy.md

### Other
- [ ] API-Reference.md
- [ ] JSON-API-Format.md
- [ ] Error-Codes.md (wiki version)
- [ ] Pagination.md (wiki version)
- [ ] Resource-Types.md (wiki version)
- [ ] SSO.md
- [ ] Integrations.md
- [ ] _Sidebar.md

---

## Critical Knowledge for Verification

### CLI Commands That EXIST

```bash
# Core commands
apigate init
apigate serve
apigate version
apigate validate
apigate shell

# Settings
apigate settings set/get/list

# Plans
apigate plans list/get/create/delete/enable/disable
# Flags: --id, --name, --description, --rate-limit, --requests, --price, --overage, --default

# Routes
apigate routes list/get/create/update/delete/enable/disable
# Flags: --name, --path, --upstream, --match, --methods, --protocol, --priority

# Users
apigate users list/get/create/delete/activate/deactivate/set-password
# NO: users update, users suspend

# Keys
apigate keys list/create/revoke
# Flags: --user, --name, --expires
# NO: keys get, keys update, api-keys (wrong name)

# Admin
apigate admin create
# Flags: --email

# Upstreams
apigate upstreams list/get/create/update/delete/enable/disable
# Flags: --name, --url, --timeout, --auth-type, --auth-header, --auth-value

# Groups
apigate groups list/list-user/get/create/update/delete/suspend/activate
# Flags: --name, --slug, --description, --owner, --plan, --billing-email

# Group Members
apigate group-members list/list-user/create
# Flags: --group, --user, --role

# Certificates
apigate certificates list/get/get-domain/create/delete/revoke/expiring/expired
# Flags: --domain, --cert-pem, --key-pem, --chain-pem, --expires-at, --days, --reason

# Usage
apigate usage
# Flags: --user

# Generic CRUD
apigate mod <module> <action>
```

### CLI Commands That DO NOT EXIST

```bash
# THESE ARE HALLUCINATIONS - DO NOT DOCUMENT
apigate migrate
apigate doctor
apigate logs
apigate analytics
apigate webhooks
apigate webhook-deliveries
apigate oauth-identities
apigate oauth-states
apigate providers test
apigate email test/preview
apigate certificates obtain/renew/export
apigate admin invite
apigate admin invites list/revoke/resend
apigate upstreams health
apigate api-keys  # Wrong name - use 'keys'
```

### Settings Namespaces (use with `apigate settings set`)

```
custom.*           - Branding (logo_url, primary_color, support_email, docs_css, portal_css)
email.*            - Email config (provider, from_address, from_name, smtp.*)
payment.*          - Payment (provider, stripe.api_key, paddle.*, lemonsqueezy.*)
oauth.google.*     - Google OAuth
oauth.github.*     - GitHub OAuth
oauth.oidc.*       - Generic OIDC
ratelimit.*        - Rate limiting (burst_tokens)
tls.*              - TLS/HTTPS (enabled, mode, domain, acme_email, cert_path, key_path, etc.)
```

### Environment Variables

**All must have `APIGATE_` prefix:**
```
APIGATE_SERVER_HOST
APIGATE_SERVER_PORT
APIGATE_ADMIN_PORT
APIGATE_DATABASE_PATH
APIGATE_LOG_LEVEL
APIGATE_SMTP_HOST
APIGATE_SMTP_PORT
APIGATE_SMTP_USERNAME
APIGATE_SMTP_PASSWORD
APIGATE_SMTP_FROM_ADDRESS
APIGATE_SMTP_USE_TLS
```

**DO NOT document these (hallucinated):**
- TLS_ACME_* (use settings instead)
- DATABASE_* without APIGATE_ prefix
- Any env var without APIGATE_ prefix

### Key Facts

| Fact | Truth |
|------|-------|
| Database | **SQLite ONLY** - no PostgreSQL |
| Redis | **NOT SUPPORTED** |
| API Key Scopes | **DO NOT EXIST** |
| Migrations | **AUTOMATIC** on startup |
| Email Providers | smtp, mock, none only |
| Payment Providers | stripe, paddle, lemonsqueezy, dummy, none |
| Webhook Retry Count | **3** (not 6) |

---

## Verification Process

For each file:

1. **Read the wiki file**
2. **For each CLI command mentioned:**
   - Check if it exists in `cmd/apigate/*.go` or module YAML files
   - Verify flags match actual implementation
3. **For each environment variable:**
   - Must have `APIGATE_` prefix
   - Check `config/config.go` for actual vars
4. **For each setting key:**
   - Check `domain/settings/settings.go` for actual keys
5. **For any feature claims:**
   - Verify in codebase (PostgreSQL=NO, Redis=NO, Scopes=NO)
6. **Fix or flag issues**

---

## Common Hallucination Patterns Found

1. **Wrong command names**: `api-keys` instead of `keys`
2. **Non-existent subcommands**: `analytics`, `doctor`, `logs`, `migrate`
3. **Fake flags**: `--rate-limit-burst`, `--monthly-quota`, `--scopes`
4. **Wrong env var prefix**: Missing `APIGATE_` prefix
5. **Non-existent features**: PostgreSQL, Redis, API key scopes
6. **Wrong setting prefixes**: `branding.*` instead of `custom.*`
7. **Fake providers**: `sendgrid`, `log` email providers

---

## Files to Reference When Verifying

| What to Check | Source File |
|---------------|-------------|
| CLI commands | `cmd/apigate/*.go` |
| Module CLI definitions | `core/modules/*.yaml` (look for `cli:` section) |
| Settings keys | `domain/settings/settings.go` |
| Env vars | `config/config.go` |
| Webhook events | `domain/webhook/webhook.go` |
| Plan struct | `ports/ports.go` |
| Email providers | `adapters/email/factory.go` |

---

## Outstanding Issue

**docs/spec/README.md** references non-existent tests:
- `TestErrorCodesDocumented`
- `TestResourceTypesDocumented`

Decision needed: Create these tests or remove references.

---

## How to Continue

1. Start with **CLI-Reference.md** - likely has the most issues
2. Then tutorials (Tutorial-*.md) - check all CLI examples
3. Then remaining feature docs
4. Update `verification-state.json` after each file
5. Create this handoff doc at end of session

---

## Session History

| Session | Files Verified | Issues Fixed |
|---------|----------------|--------------|
| Initial Alignment | 4 spec files | 2 |
| Documentation Alignment | 7 wiki files | 6 |
| Wiki Verification | 11 wiki files | 10 |
| Wiki Verification 2 | (continuation) | - |
| Wiki Verification 3 | 8 wiki files | 8 |
| **Total** | **30 files** | **26 issues** |
