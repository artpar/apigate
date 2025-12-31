# J1: First-Time Setup

> **The most critical 5 minutes of our relationship with the API Seller.**

---

## Business Context

### Why This Journey Matters

First-time setup is the **activation moment** that determines whether an API Seller becomes a paying customer or abandons the product. This is our "aha moment" - when they realize APIGate actually delivers on its promise.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SETUP FUNNEL                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Download/Deploy ──▶ Start Setup ──▶ Complete Setup ──▶ First Use │
│        100%              85%              70%              60%      │
│                                                                     │
│   Target: 90%+ completion rate for users who start setup            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact

| Metric | Impact |
|--------|--------|
| **License Conversion** | Users who complete setup are 5x more likely to purchase |
| **Time to Value** | Each minute of setup time reduces conversion by ~3% |
| **Support Cost** | Failed setups generate 80% of pre-sales support tickets |

### Business Success Criteria

- [ ] Setup completes in under 5 minutes
- [ ] Zero configuration files needed
- [ ] No terminal commands required after initial start
- [ ] 90%+ setup completion rate
- [ ] Admin can see their first proxied request within 10 minutes

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Seller (Indie hacker, SaaS founder, agency dev) |
| **Prior Action** | Downloaded binary, ran Docker, or deployed to cloud |
| **Mental State** | Evaluating, slightly skeptical, time-pressured |
| **Expectation** | "This better be as easy as they claim" |

### What Triggered This Journey?

- Visited marketing site, downloaded product
- Found on GitHub/ProductHunt/HackerNews
- Recommendation from colleague
- Searching for API monetization solutions

### User Goals

1. **Primary:** Verify this product works with their API
2. **Secondary:** Understand if it's worth the effort to fully set up
3. **Tertiary:** Evaluate if it meets their security/compliance needs

### User Questions at This Stage

- "Will this work with my existing API?"
- "How long until I can test it?"
- "Do I need to change my backend code?"
- "Is my data safe?"

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Launch  │───▶│ Step 1  │───▶│ Step 2  │───▶│ Step 3  │───▶│Dashboard│
│ Binary  │    │Upstream │    │ Admin   │    │  Plan   │    │ (Done!) │
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
    │               │              │              │              │
    │               │              │              │              │
    ▼               ▼              ▼              ▼              ▼
  10 sec        60 sec          45 sec         60 sec        Success!

              Total Target Time: < 3 minutes
```

### Pre-Journey: Launch

**Action:** User runs `./apigate` or `docker run apigate`

**System Response:**
```
APIGate v1.0.0 starting...
Server listening on http://localhost:8080
Open your browser to complete setup.
```

**Screenshot:** `j1-setup/00-terminal-start.png`

---

### Step 1: Connect Your API

**URL:** `/setup/step/1` (auto-redirect from `/`)

**Purpose:** Establish connection to the API Seller's backend.

#### UI Elements

| Element | Type | Validation | Notes |
|---------|------|------------|-------|
| Upstream URL | Text input | Valid URL, reachable | Auto-detect HTTPS |
| Test Connection | Button | - | Shows success/failure inline |
| Next | Button | Disabled until valid | Primary CTA |

#### User Actions

1. Enter their API's base URL (e.g., `https://api.example.com`)
2. Click "Test Connection"
3. See success confirmation
4. Click "Next"

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Empty form | Page load | `j1-setup/01-upstream-empty.png` |
| URL entered | After input | `j1-setup/01-upstream-filled.png` |
| Testing | During test | `j1-setup/01-upstream-testing.png` |
| Success | Test passed | `j1-setup/01-upstream-success.png` |
| Error | Test failed | `j1-setup/01-upstream-error.png` |

#### Potential Friction Points

| Issue | Cause | Mitigation |
|-------|-------|------------|
| URL format confusion | http vs https, trailing slash | Auto-normalize, show examples |
| Connection timeout | Slow backend, firewall | Increase timeout, clear error message |
| SSL errors | Self-signed certs | Option to skip verification (dev mode) |
| CORS errors | Backend rejects | Explain this is server-side, no CORS |

#### Error Recovery

**Connection Failed:**
```
Unable to connect to https://api.example.com

Possible causes:
• The URL might be incorrect
• Your API server might not be running
• A firewall might be blocking the connection

[Try Again] [Skip for Now]
```

---

### Step 2: Create Admin Account

**URL:** `/setup/step/2`

**Purpose:** Secure the platform with admin credentials.

#### UI Elements

| Element | Type | Validation | Notes |
|---------|------|------------|-------|
| Email | Email input | Valid email format | Used for login + notifications |
| Password | Password input | Min 8 chars, complexity | Show strength meter |
| Confirm Password | Password input | Must match | Real-time match indicator |
| Back | Button | - | Return to step 1 |
| Next | Button | Disabled until valid | Primary CTA |

#### User Actions

1. Enter email address
2. Create password (see strength indicator)
3. Confirm password
4. Click "Next"

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Empty form | Page load | `j1-setup/02-admin-empty.png` |
| Filled form | After input | `j1-setup/02-admin-filled.png` |
| Weak password | Validation | `j1-setup/02-admin-weak-password.png` |
| Strong password | Validation | `j1-setup/02-admin-strong-password.png` |
| Mismatch | Passwords differ | `j1-setup/02-admin-mismatch.png` |

#### Potential Friction Points

| Issue | Cause | Mitigation |
|-------|-------|------------|
| Password too complex | Strict requirements | Show requirements upfront, suggest generator |
| Forgot what they entered | Hidden password | Toggle visibility button |
| Email typo | Fast typing | Confirmation or email validation |

#### Security Considerations

- Password hashed with bcrypt (cost 12)
- No password recovery during setup (intentional)
- Admin account is the only way to access `/ui/`

---

### Step 3: Create Your First Plan

**URL:** `/setup/step/3`

**Purpose:** Define the initial pricing structure.

#### UI Elements

| Element | Type | Validation | Notes |
|---------|------|------------|-------|
| Plan Name | Text input | Required | Default: "Free" |
| Monthly Price | Number input | >= 0 | In dollars, converted to cents |
| Requests/Month | Number input | > 0 | Quota limit |
| Rate Limit | Number input | > 0 | Requests per minute |
| Back | Button | - | Return to step 2 |
| Complete Setup | Button | Disabled until valid | Final CTA |

#### User Actions

1. Name the plan (or accept "Free" default)
2. Set price (default: $0)
3. Set monthly request quota
4. Set rate limit
5. Click "Complete Setup"

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Empty form | Page load | `j1-setup/03-plan-empty.png` |
| Free plan | Default values | `j1-setup/03-plan-free.png` |
| Paid plan | Custom values | `j1-setup/03-plan-paid.png` |
| Validation error | Invalid input | `j1-setup/03-plan-error.png` |

#### Smart Defaults

| Field | Default | Rationale |
|-------|---------|-----------|
| Plan Name | "Free" | Most start with free tier |
| Price | $0 | Free tier is common starting point |
| Requests/Month | 1,000 | Reasonable for evaluation |
| Rate Limit | 10/min | Prevents abuse without hindering testing |

#### Guidance Text

> **Tip:** Start with a free plan to let developers try your API. You can add paid plans later from the dashboard.

---

### Step 4: Setup Complete

**URL:** `/ui/` (redirect after completion)

**Purpose:** Confirm success and guide next steps.

#### What Happens

1. Setup wizard data saved to database
2. Admin session created (JWT cookie)
3. Redirect to admin dashboard
4. Getting Started checklist displayed

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Dashboard | After redirect | `j1-setup/04-dashboard.png` |
| Checklist | Dashboard loaded | `j1-setup/04-checklist.png` |

#### Post-Setup Guidance

The Getting Started checklist shows:

- [x] Connect your API ✓
- [x] Create admin account ✓
- [x] Create pricing plan ✓
- [ ] Create an API key to test
- [ ] View your customer portal
- [ ] Share with your first customer

---

## UX Analysis

### Cognitive Load Assessment

| Step | Decisions Required | Cognitive Load | Notes |
|------|-------------------|----------------|-------|
| 1 | 1 (URL) | Low | Single, familiar input |
| 2 | 2 (email, password) | Low-Medium | Standard auth pattern |
| 3 | 4 (name, price, quota, limit) | Medium | Multiple values, but defaults help |

**Overall:** Low-Medium. Wizard pattern reduces overwhelm.

### Friction Analysis

```
Friction Score: 1 (Low) to 5 (High)

Step 1: Upstream URL        ████░░░░░░ 2/5
Step 2: Admin Account       ██░░░░░░░░ 1/5  (familiar pattern)
Step 3: Plan Creation       ███░░░░░░░ 3/5  (requires decisions)

Main Friction Point: Step 3 - Users unsure what values to choose
Mitigation: Smart defaults + guidance text
```

### Accessibility

| Requirement | Status | Notes |
|-------------|--------|-------|
| Keyboard navigation | ✓ | Tab through all fields |
| Screen reader | ✓ | ARIA labels on all inputs |
| Color contrast | ✓ | WCAG AA compliant |
| Error messages | ✓ | Associated with inputs |
| Focus management | ✓ | Focus moves to first field on step change |

### Mobile Experience

| Aspect | Status | Notes |
|--------|--------|-------|
| Responsive layout | ✓ | Single column on mobile |
| Touch targets | ✓ | 44px minimum |
| Keyboard handling | ✓ | Appropriate input types |
| Progress indicator | ✓ | Visible on small screens |

---

## Emotional Map

```
                     Emotional State During Setup

Delight  ─┐                                              ┌─ ●
          │                                            ╱
          │                                          ╱
Neutral  ─┼──────────●─────────────────●───────────●
          │        ╱                    ╲         ╱
          │      ╱                       ╲      ╱
Anxiety  ─┴────●───────────────────────────────
          │
          └────┬─────────┬─────────┬─────────┬─────────
             Start    Step 1    Step 2    Step 3    Done

Legend:
● = Emotional state at each point
```

### Emotional Journey

| Stage | Emotion | Trigger | Design Response |
|-------|---------|---------|-----------------|
| **Start** | Skeptical curiosity | Just downloaded | Clean, professional first screen |
| **Step 1** | Slight anxiety | "Will my API work?" | Quick test with clear feedback |
| **Step 1 Success** | Relief, building trust | Connection works | Celebratory checkmark |
| **Step 2** | Familiar, comfortable | Standard auth pattern | No surprises |
| **Step 3** | Slight uncertainty | "What values?" | Smart defaults reduce decisions |
| **Complete** | Achievement, excitement | "It worked!" | Dashboard with clear next steps |

### Delight Opportunities

1. **Instant validation** - Show success immediately when URL is valid
2. **Connection animation** - Brief animation when testing backend
3. **Confetti moment** - Subtle celebration on setup completion
4. **Personalized dashboard** - Use admin's name in welcome message

### Anxiety Reducers

1. **Progress indicator** - Always show "Step X of 3"
2. **Back buttons** - User can always go back
3. **Skip options** - For non-critical steps (if applicable)
4. **Persistent data** - Refresh doesn't lose progress
5. **Help links** - Documentation accessible throughout

---

## Metrics & KPIs

### Primary Metrics

| Metric | Definition | Target | Alert Threshold |
|--------|------------|--------|-----------------|
| **Setup Completion Rate** | Users who complete / Users who start | > 90% | < 80% |
| **Time to Complete** | Median time from start to dashboard | < 3 min | > 5 min |
| **Step Drop-off** | Users who abandon at each step | < 5% per step | > 10% |

### Step-by-Step Funnel

```
Track: setup_step_viewed, setup_step_completed, setup_abandoned

Expected Funnel:
  Start Setup    ████████████████████████████████████████ 100%
  Complete S1    ██████████████████████████████████████   95%
  Complete S2    ████████████████████████████████████     90%
  Complete S3    ██████████████████████████████████       85%
  View Dashboard █████████████████████████████████        85%
```

### Error Tracking

| Error | Metric | Alert |
|-------|--------|-------|
| Connection failures | setup_upstream_error | > 20% of attempts |
| Password validation | setup_password_error | > 15% of attempts |
| Plan validation | setup_plan_error | > 10% of attempts |

### Analytics Events

```javascript
// Step viewed
analytics.track('setup_step_viewed', {
  step: 1,
  referrer: 'direct'
});

// Step completed
analytics.track('setup_step_completed', {
  step: 1,
  duration_seconds: 45
});

// Setup abandoned
analytics.track('setup_abandoned', {
  step: 2,
  reason: 'closed_tab' // or 'error', 'back_navigation'
});

// Setup completed
analytics.track('setup_completed', {
  total_duration_seconds: 180,
  plan_type: 'free'
});
```

---

## Edge Cases & Errors

### Network Errors

| Scenario | Detection | User Message | Recovery |
|----------|-----------|--------------|----------|
| Offline | fetch fails | "No internet connection" | Auto-retry when online |
| Timeout | 30s elapsed | "Connection timed out" | Retry button |
| SSL error | cert invalid | "SSL certificate error" | Dev mode toggle |

### Backend Errors

| Scenario | Detection | User Message | Recovery |
|----------|-----------|--------------|----------|
| 404 | HTTP 404 | "URL not found" | Check URL guidance |
| 500 | HTTP 5xx | "API server error" | Try again later |
| Auth required | HTTP 401 | "API requires auth" | Advanced config |

### Validation Errors

| Field | Error | Message |
|-------|-------|---------|
| Email | Invalid format | "Please enter a valid email" |
| Password | Too short | "Password must be at least 8 characters" |
| Password | No uppercase | "Include at least one uppercase letter" |
| Passwords | Don't match | "Passwords do not match" |
| Plan name | Empty | "Plan name is required" |
| Rate limit | Zero | "Rate limit must be greater than 0" |

### Recovery Flows

**Browser Refresh During Setup:**
- Data persists in localStorage
- User returns to current step
- Show "Welcome back" message

**Session Timeout (shouldn't happen during setup):**
- Setup doesn't require auth
- Only dashboard requires session

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j1-first-time-setup
requires_fresh_db: true
viewport: 1280x720

steps:
  - name: terminal-start
    action: external
    notes: Capture terminal output manually

  - name: upstream-empty
    url: /setup/step/1
    wait: networkidle

  - name: upstream-filled
    url: /setup/step/1
    actions:
      - fill: input[name="upstream_url"]
        value: "https://api.example.com"

  - name: upstream-success
    url: /setup/step/1
    actions:
      - fill: input[name="upstream_url"]
        value: "https://httpbin.org"
      - click: button:has-text("Test")
      - wait: text=Connection successful

  - name: admin-empty
    url: /setup/step/2
    wait: networkidle

  - name: admin-filled
    url: /setup/step/2
    actions:
      - fill: input[name="email"]
        value: "admin@example.com"
      - fill: input[name="password"]
        value: "SecurePass123!"
      - fill: input[name="confirm_password"]
        value: "SecurePass123!"

  - name: plan-defaults
    url: /setup/step/3
    wait: networkidle

  - name: dashboard
    url: /ui/
    wait: networkidle
    notes: After completing setup
```

### GIF Sequence

**j1-setup-wizard.gif**
- Frame 1: Empty upstream form (2s)
- Frame 2: URL entered (1s)
- Frame 3: Testing... (1s)
- Frame 4: Success checkmark (1s)
- Frame 5: Admin form filled (2s)
- Frame 6: Plan form with defaults (2s)
- Frame 7: Dashboard with checklist (3s)

Total duration: ~12 seconds

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J2: Plan Management](j2-plan-management.md) | Next step: create more plans |
| [J4: Platform Config](j4-platform-config.md) | Configure payment/email |
| [J5: Customer Onboarding](../customer/j5-onboarding.md) | What your customers experience |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
