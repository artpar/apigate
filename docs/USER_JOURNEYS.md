# APIGate User Journeys

This document defines all user journeys through the APIGate platform, serving as both documentation and the source of truth for automated screenshot/GIF capture.

---

## Overview

APIGate has two primary user personas:

| Persona | Description | Entry Point |
|---------|-------------|-------------|
| **API Seller (Admin)** | Sets up and manages the API monetization platform | `/ui/` |
| **API Buyer (Customer)** | Signs up, gets API keys, and uses the API | `/portal/` |

---

## Journey Map

```
┌─────────────────────────────────────────────────────────────────────┐
│                        API SELLER JOURNEYS                          │
├─────────────────────────────────────────────────────────────────────┤
│  J1: First-Time Setup                                               │
│      └── Setup Wizard (3 steps) → Admin Dashboard                   │
│                                                                     │
│  J2: Plan Management                                                │
│      └── Create Plans → Set Default → Configure Pricing             │
│                                                                     │
│  J3: Monitor & Manage                                               │
│      └── Dashboard → Users → API Keys → Usage                       │
│                                                                     │
│  J4: Configure Platform                                             │
│      └── Settings → Payment Provider → Email → Upstream             │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                       API BUYER JOURNEYS                            │
├─────────────────────────────────────────────────────────────────────┤
│  J5: Customer Onboarding                                            │
│      └── Sign Up → Login → Dashboard                                │
│                                                                     │
│  J6: Get API Access                                                 │
│      └── Create API Key → Copy Key → Test API                       │
│                                                                     │
│  J7: Monitor Usage                                                  │
│      └── View Usage → Check Limits → Quota Warnings                 │
│                                                                     │
│  J8: Upgrade Plan                                                   │
│      └── View Plans → Select Plan → Payment → Confirmation          │
│                                                                     │
│  J9: Self-Service Documentation                                     │
│      └── Docs Portal → Try It → Code Examples                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## J1: First-Time Setup (Admin)

**Goal:** Complete initial platform setup from scratch.

**Preconditions:** Fresh APIGate installation, no admin account exists.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 1.1 | Navigate to `/` | `j1-01-welcome` | Redirected to setup wizard |
| 1.2 | Enter upstream URL | `j1-02-upstream` | URL validated successfully |
| 1.3 | Create admin account | `j1-03-admin` | Email and password entered |
| 1.4 | Create first plan | `j1-04-plan` | Plan details configured |
| 1.5 | Complete setup | `j1-05-complete` | Redirected to admin dashboard |
| 1.6 | View dashboard | `j1-06-dashboard` | Dashboard with getting started checklist |

### Success Criteria
- Admin can access `/ui/` with credentials
- First plan is created and set as default
- Upstream URL is configured
- Getting started checklist visible

### GIF Capture Points
- `j1-setup-wizard.gif`: Steps 1.1 → 1.5 (entire wizard flow)

---

## J2: Plan Management (Admin)

**Goal:** Create and manage pricing plans.

**Preconditions:** Admin logged in, at least one plan exists.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 2.1 | Navigate to Plans | `j2-01-plans-list` | Plans list with table |
| 2.2 | Click Create New | `j2-02-create-form` | Empty plan creation form |
| 2.3 | Fill Free tier | `j2-03-free-plan` | Plan with 0 price, 1K requests |
| 2.4 | Submit and create | `j2-04-created` | Plan appears in list |
| 2.5 | Create Pro plan | `j2-05-pro-plan` | $29/mo, 100K requests |
| 2.6 | Set as default | `j2-06-default` | Default badge visible |
| 2.7 | Edit plan | `j2-07-edit` | Edit form with existing values |
| 2.8 | Disable plan | `j2-08-disabled` | Plan marked as disabled |

### Success Criteria
- Multiple plans visible in list
- Default plan indicator works
- Plans can be enabled/disabled
- Price displayed correctly (cents to dollars)

### GIF Capture Points
- `j2-create-plan.gif`: Steps 2.2 → 2.4

---

## J3: Monitor & Manage (Admin)

**Goal:** Monitor platform usage and manage users.

**Preconditions:** Admin logged in, users and API keys exist.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 3.1 | View dashboard | `j3-01-dashboard` | Stats cards visible |
| 3.2 | Navigate to Users | `j3-02-users-list` | User table with status |
| 3.3 | View user details | `j3-03-user-detail` | User info and plan |
| 3.4 | Navigate to API Keys | `j3-04-keys-list` | All API keys listed |
| 3.5 | View key usage | `j3-05-key-detail` | Usage stats for key |
| 3.6 | Suspend user | `j3-06-suspend` | Confirmation dialog |
| 3.7 | View suspended | `j3-07-suspended` | User status changed |
| 3.8 | Reactivate user | `j3-08-reactivated` | User active again |

### Success Criteria
- Dashboard shows real-time stats
- User actions (suspend/activate) work
- API key list shows usage
- Status changes reflected immediately

### GIF Capture Points
- `j3-suspend-user.gif`: Steps 3.6 → 3.8

---

## J4: Configure Platform (Admin)

**Goal:** Configure payment, email, and other platform settings.

**Preconditions:** Admin logged in.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 4.1 | Navigate to Settings | `j4-01-settings` | Settings page |
| 4.2 | Payment section | `j4-02-payment` | Provider selection |
| 4.3 | Enter Stripe keys | `j4-03-stripe` | API keys entered |
| 4.4 | Email section | `j4-04-email` | SMTP configuration |
| 4.5 | Test email | `j4-05-email-test` | Test email sent |
| 4.6 | Upstream section | `j4-06-upstream` | Backend URL config |
| 4.7 | Save settings | `j4-07-saved` | Success message |

### Success Criteria
- Settings persist after refresh
- Stripe keys saved securely
- Email test works
- Upstream URL can be changed

---

## J5: Customer Onboarding

**Goal:** New customer signs up and accesses dashboard.

**Preconditions:** At least one plan exists, signup enabled.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 5.1 | Navigate to portal | `j5-01-portal-home` | Portal landing page |
| 5.2 | Click Sign Up | `j5-02-signup-form` | Registration form |
| 5.3 | Fill details | `j5-03-filled-form` | Name, email, password |
| 5.4 | Submit form | `j5-04-success` | Account created message |
| 5.5 | Login | `j5-05-login` | Login form |
| 5.6 | Enter credentials | `j5-06-credentials` | Email and password |
| 5.7 | View dashboard | `j5-07-dashboard` | Customer dashboard |

### Success Criteria
- User created with default plan
- Can login immediately after signup
- Dashboard shows plan info
- Getting started steps visible

### GIF Capture Points
- `j5-signup.gif`: Steps 5.2 → 5.7

---

## J6: Get API Access (Customer)

**Goal:** Create API key and make first API call.

**Preconditions:** Customer logged in.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 6.1 | Navigate to API Keys | `j6-01-keys-empty` | Empty or existing keys |
| 6.2 | Click Create Key | `j6-02-create-dialog` | Key creation dialog |
| 6.3 | Enter name | `j6-03-name-entered` | "Production" or similar |
| 6.4 | Create key | `j6-04-key-shown` | Full key displayed once |
| 6.5 | Copy key | `j6-05-copied` | Copy confirmation |
| 6.6 | View in list | `j6-06-in-list` | Key with masked value |
| 6.7 | Navigate to Try It | `j6-07-try-it` | API tester page |
| 6.8 | Enter key | `j6-08-key-entered` | Key pasted in field |
| 6.9 | Make request | `j6-09-response` | API response displayed |

### Success Criteria
- Key only shown once (security)
- Copy to clipboard works
- Key works in Try It section
- Response shows rate limit headers

### GIF Capture Points
- `j6-create-key.gif`: Steps 6.2 → 6.5
- `j6-test-api.gif`: Steps 6.7 → 6.9

---

## J7: Monitor Usage (Customer)

**Goal:** Track API usage and understand limits.

**Preconditions:** Customer logged in, has made API calls.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 7.1 | Navigate to Usage | `j7-01-usage-page` | Usage overview |
| 7.2 | View current period | `j7-02-current` | Requests this month |
| 7.3 | View quota bar | `j7-03-quota-bar` | Visual progress indicator |
| 7.4 | Check rate limit | `j7-04-rate-limit` | Requests per minute |
| 7.5 | View by day | `j7-05-daily` | Daily breakdown chart |
| 7.6 | At 80% usage | `j7-06-warning` | Yellow warning shown |
| 7.7 | At 95% usage | `j7-07-critical` | Red critical warning |

### Success Criteria
- Usage updates in real-time
- Quota warnings at correct thresholds
- Clear visual indicators
- Data matches actual usage

---

## J8: Upgrade Plan (Customer)

**Goal:** Upgrade from free to paid plan.

**Preconditions:** Customer on free plan, payment provider configured.

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 8.1 | Navigate to Plans | `j8-01-plans-page` | Available plans |
| 8.2 | Compare plans | `j8-02-comparison` | Feature comparison |
| 8.3 | Select Pro plan | `j8-03-select-pro` | Plan highlighted |
| 8.4 | Click Upgrade | `j8-04-upgrade-btn` | Payment redirect |
| 8.5 | Stripe checkout | `j8-05-stripe` | Stripe payment page |
| 8.6 | Complete payment | `j8-06-complete` | Success callback |
| 8.7 | View new plan | `j8-07-upgraded` | Pro plan active |
| 8.8 | Verify limits | `j8-08-new-limits` | Higher limits shown |

### Success Criteria
- Plan comparison is clear
- Stripe integration works
- Plan changes immediately
- New limits take effect

### GIF Capture Points
- `j8-upgrade.gif`: Steps 8.3 → 8.7

---

## J9: Self-Service Documentation (Customer)

**Goal:** Learn API and test endpoints.

**Preconditions:** Customer logged in (or anonymous for public docs).

### Steps

| Step | Action | Screenshot Key | Expected Outcome |
|------|--------|----------------|------------------|
| 9.1 | Navigate to Docs | `j9-01-docs-home` | Documentation portal |
| 9.2 | View Quickstart | `j9-02-quickstart` | Getting started guide |
| 9.3 | View Authentication | `j9-03-auth` | How to authenticate |
| 9.4 | View API Reference | `j9-04-reference` | Endpoint documentation |
| 9.5 | View Examples | `j9-05-examples` | Code samples |
| 9.6 | Select language | `j9-06-language` | JavaScript/Python/cURL |
| 9.7 | Open Try It | `j9-07-try-it` | Interactive tester |
| 9.8 | Test endpoint | `j9-08-tested` | Response displayed |

### Success Criteria
- All doc sections accessible
- Code examples are copyable
- Try It works with valid key
- Multiple languages available

### GIF Capture Points
- `j9-docs-tour.gif`: Steps 9.1 → 9.5

---

## Error Journeys

### E1: Invalid API Key

| Step | Action | Screenshot Key |
|------|--------|----------------|
| E1.1 | Make request with bad key | `e1-01-invalid-key` |
| E1.2 | View error response | `e1-02-error-401` |

### E2: Rate Limited

| Step | Action | Screenshot Key |
|------|--------|----------------|
| E2.1 | Exceed rate limit | `e2-01-rate-exceeded` |
| E2.2 | View 429 response | `e2-02-error-429` |
| E2.3 | Check Retry-After | `e2-03-retry-after` |

### E3: Quota Exceeded

| Step | Action | Screenshot Key |
|------|--------|----------------|
| E3.1 | Exceed monthly quota | `e3-01-quota-exceeded` |
| E3.2 | View quota error | `e3-02-error-503` |
| E3.3 | Upgrade prompt | `e3-03-upgrade-prompt` |

---

## Screenshot Automation

### Directory Structure

```
docs/
├── screenshots/
│   ├── j1-setup/           # First-time setup screenshots
│   ├── j2-plans/           # Plan management
│   ├── j3-monitor/         # Monitoring & management
│   ├── j4-config/          # Platform configuration
│   ├── j5-onboarding/      # Customer onboarding
│   ├── j6-api-access/      # API key creation & testing
│   ├── j7-usage/           # Usage monitoring
│   ├── j8-upgrade/         # Plan upgrade
│   ├── j9-docs/            # Documentation portal
│   └── errors/             # Error state screenshots
├── gifs/
│   ├── j1-setup-wizard.gif
│   ├── j2-create-plan.gif
│   ├── j5-signup.gif
│   ├── j6-create-key.gif
│   ├── j6-test-api.gif
│   ├── j8-upgrade.gif
│   └── j9-docs-tour.gif
└── USER_JOURNEYS.md         # This file
```

### Automation Script Usage

```bash
# Capture all journeys
./scripts/capture-journeys.sh all

# Capture specific journey
./scripts/capture-journeys.sh j1

# Capture only GIFs
./scripts/capture-journeys.sh --gifs-only

# Capture with fresh database (clean slate)
./scripts/capture-journeys.sh --fresh all
```

### Screenshot Naming Convention

```
{journey}-{step}-{description}.png

Examples:
j1-01-welcome.png
j5-04-success.png
e1-02-error-401.png
```

### GIF Parameters

| Parameter | Value |
|-----------|-------|
| Frame rate | 2 FPS |
| Delay between steps | 1500ms |
| Resolution | 1280x720 |
| Quality | 80 |
| Max duration | 15 seconds |

---

## Validation Checklist

For each journey, verify:

- [ ] All screenshots captured at correct resolution
- [ ] No sensitive data visible (passwords, real API keys)
- [ ] UI elements are in expected state
- [ ] Loading states are resolved
- [ ] Error messages are clear
- [ ] Success messages are visible
- [ ] Navigation is logical
- [ ] Mobile breakpoints work (if applicable)

---

## Maintenance

### When to Update Screenshots

1. UI component changes
2. New features added to journey
3. Error message wording changes
4. Branding/styling updates

### When to Update GIFs

1. Multi-step workflow changes
2. Significant UX improvements
3. New user-facing features

### Automation Triggers

Screenshots should be automatically captured:
- On pull requests with UI changes
- Before releases
- After accessibility improvements
