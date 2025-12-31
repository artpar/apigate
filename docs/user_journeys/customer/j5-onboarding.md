# J5: Customer Onboarding

> **The first impression of the API Seller's product - make it count.**

---

## Business Context

### Why This Journey Matters

Customer onboarding is the **top of the funnel** for API Sellers. Every customer they acquire goes through this flow. Friction here directly reduces the API Seller's customer base.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CUSTOMER ACQUISITION FUNNEL                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Visit Portal ──▶ Signup ──▶ Verify ──▶ Login ──▶ Dashboard       │
│      100%           60%       55%       50%        48%              │
│                                                                     │
│   Target: 70%+ of visitors who click signup complete onboarding    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact (for API Seller)

| Metric | Impact |
|--------|--------|
| **Signup conversion** | Each 1% improvement = more customers |
| **Time to signup** | Longer = higher drop-off |
| **First experience** | Sets tone for entire relationship |

### Business Success Criteria

- [ ] Signup completes in < 2 minutes
- [ ] No email verification required (reduces friction)
- [ ] Automatic plan assignment (no decision paralysis)
- [ ] Immediate access to dashboard
- [ ] Clear next steps after login

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Buyer (developer exploring the API) |
| **Prior Action** | Found the API through docs, recommendation, or search |
| **Mental State** | Curious, evaluating, slightly impatient |
| **Expectation** | "Let me try this API quickly" |

### What Triggered This Journey?

- Read about the API, wants to try it
- Colleague recommended the API
- Found in API marketplace/directory
- Searching for solution to a problem

### User Goals

1. **Primary:** Get access to the API as quickly as possible
2. **Secondary:** Understand what the API offers
3. **Tertiary:** Evaluate if it fits their needs

### User Questions at This Stage

- "Is there a free tier?"
- "How quickly can I test this?"
- "Do I need a credit card?"
- "What do I get with a free account?"

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Portal  │───▶│ Signup  │───▶│  Login  │───▶│Dashboard│
│  Home   │    │  Form   │    │  Form   │    │  (Done) │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     │               │              │              │
     ▼               ▼              ▼              ▼
  See value      Create       Authenticate     Start
  proposition    account       session        using API
```

---

### Step 1: Portal Home

**URL:** `/portal/`

**Purpose:** Show value proposition and clear CTA to sign up.

#### UI Elements

| Element | Type | Purpose |
|---------|------|---------|
| Hero section | Content | Value proposition |
| Sign Up button | Primary CTA | Start registration |
| Login link | Secondary | Existing users |
| Pricing/Plans | Link | View options |

#### Portal Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  [API Name]                                    [Login] [Sign Up]    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│                    Welcome to [API Name]                            │
│                                                                     │
│       [Brief description of what the API does and its value]        │
│                                                                     │
│                    [Get Started - Free]                             │
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │   Feature   │  │   Feature   │  │   Feature   │                 │
│  │     One     │  │     Two     │  │    Three    │                 │
│  └─────────────┘  └─────────────┘  └─────────────┘                 │
│                                                                     │
│             Already have an account? [Log in]                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Key Messages

- **No credit card required** (if true)
- **Free tier available** (if true)
- **Start in minutes**

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Portal home | Page load | `j5-onboarding/01-portal-home.png` |
| Mobile view | Responsive | `j5-onboarding/01-portal-mobile.png` |

---

### Step 2: Signup Form

**URL:** `/portal/signup`

**Purpose:** Collect minimum information to create account.

#### Form Fields

| Field | Type | Validation | Required |
|-------|------|------------|----------|
| Name | Text | Min 2 chars | Yes |
| Email | Email | Valid format, unique | Yes |
| Password | Password | Min 8, complexity | Yes |
| Terms | Checkbox | Must accept | Yes |

#### Form Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Create Your Account                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Name                                                               │
│  [_________________________________________________]               │
│                                                                     │
│  Email                                                              │
│  [_________________________________________________]               │
│                                                                     │
│  Password                                                           │
│  [_________________________________________________] [👁]          │
│  Must be 8+ characters with uppercase, lowercase, and number       │
│                                                                     │
│  [✓] I agree to the Terms of Service and Privacy Policy            │
│                                                                     │
│                    [Create Account]                                 │
│                                                                     │
│             Already have an account? [Log in]                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Password Requirements

| Requirement | Validation |
|-------------|------------|
| Minimum length | 8 characters |
| Uppercase | At least 1 |
| Lowercase | At least 1 |
| Number | At least 1 |

#### Real-time Validation

- Show password strength meter
- Validate email format on blur
- Highlight password requirements as met

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Empty form | Page load | `j5-onboarding/02-signup-empty.png` |
| Filled form | Data entered | `j5-onboarding/02-signup-filled.png` |
| Validation errors | Submit invalid | `j5-onboarding/02-signup-errors.png` |
| Password strength | Typing password | `j5-onboarding/02-password-strength.png` |

---

### Step 3: Account Created

**URL:** `/portal/signup` → redirect or `/portal/login`

**Purpose:** Confirm success and guide to login.

#### Success Flow Options

**Option A: Auto-login (Recommended)**
- Create account → automatically logged in → redirect to dashboard
- Best UX, reduces friction

**Option B: Manual login**
- Create account → success message → redirect to login
- More secure, requires verification

#### Success Message

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                    ✓ Account Created!                               │
│                                                                     │
│       Welcome to [API Name]. You've been signed up for the          │
│       Free plan with 1,000 requests per month.                      │
│                                                                     │
│                    [Go to Dashboard]                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Success message | Account created | `j5-onboarding/03-success.png` |

---

### Step 4: Login

**URL:** `/portal/login`

**Purpose:** Authenticate returning users.

#### Form Fields

| Field | Type | Validation |
|-------|------|------------|
| Email | Email | Required |
| Password | Password | Required |
| Remember me | Checkbox | Optional |

#### Form Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Welcome Back                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Email                                                              │
│  [_________________________________________________]               │
│                                                                     │
│  Password                                                           │
│  [_________________________________________________] [👁]          │
│                                                                     │
│  [✓] Remember me                        [Forgot password?]          │
│                                                                     │
│                        [Log In]                                     │
│                                                                     │
│              Don't have an account? [Sign up]                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Login form | Page load | `j5-onboarding/04-login.png` |
| Login error | Wrong credentials | `j5-onboarding/04-login-error.png` |

---

### Step 5: Customer Dashboard

**URL:** `/portal/dashboard` or `/portal/`

**Purpose:** Orient new user and guide to next steps.

#### Dashboard Elements

| Section | Content |
|---------|---------|
| Welcome banner | Personalized greeting, plan info |
| Plan summary | Current plan, usage, limits |
| Quick actions | Create API key, view docs |
| Getting started | Checklist for new users |

#### Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  [API Name] Portal                      [John ▼] [Settings] [Logout]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Welcome, John!                                                     │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Your Plan: Free                                                ││
│  │  Requests: 0 / 1,000 this month                                ││
│  │  Rate Limit: 10 requests/minute                                 ││
│  │                                              [Upgrade Plan]     ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Getting Started                                                    │
│  ─────────────────                                                  │
│  [ ] Create your first API key                                     │
│  [ ] Read the documentation                                        │
│  [ ] Make your first API call                                      │
│                                                                     │
│  Quick Links                                                        │
│  ───────────                                                        │
│  [Create API Key]  [Documentation]  [Examples]                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Dashboard | After login | `j5-onboarding/05-dashboard.png` |
| Welcome banner | First login | `j5-onboarding/05-welcome.png` |

---

## UX Analysis

### Cognitive Load Assessment

| Step | Decisions | Load | Notes |
|------|-----------|------|-------|
| Portal home | 1 (signup vs browse) | Low | Clear CTA |
| Signup form | 3 (name, email, password) | Low | Standard pattern |
| Login | 2 (email, password) | Very Low | Familiar |
| Dashboard | 2-3 (what to do next) | Low | Guided |

### Friction Analysis

```
Friction Score: 1 (Low) to 5 (High)

Portal Home       █░░░░░░░░░ 1/5
Signup Form       ██░░░░░░░░ 2/5  (password requirements)
Login             █░░░░░░░░░ 1/5
Dashboard         ██░░░░░░░░ 2/5  (new interface)

Main Friction Point: Password complexity requirements
Mitigation: Clear indicators, suggest passwords
```

### Accessibility

| Requirement | Implementation |
|-------------|----------------|
| Form labels | All inputs labeled |
| Error messages | Associated with inputs |
| Keyboard navigation | Full tab support |
| Screen reader | ARIA labels |
| Password visibility | Toggle button |

### Mobile Experience

| Aspect | Status |
|--------|--------|
| Responsive layout | Single column |
| Touch targets | 44px minimum |
| Form fields | Appropriate keyboards |
| Social login buttons | Full width |

---

## Emotional Map

```
                     Emotional State During Onboarding

Delight  ─┐                                              ┌─ ●
          │                                            ╱
          │                                          ╱
Neutral  ─┼────●───────────────────────────────────●
          │      ╲                               ╱
          │        ╲                           ╱
Anxiety  ─┴──────────●─────●─────────────────●
          │
          └────┬─────────┬─────────┬─────────┬─────────
            Portal   Signup    Login    Dashboard
```

### Emotional Journey

| Stage | Emotion | Trigger | Design Response |
|-------|---------|---------|-----------------|
| **Portal** | Curious | Exploring | Clear value prop |
| **Start signup** | Slight anxiety | Commitment | Reassurance (free, no CC) |
| **Password** | Frustration possible | Requirements | Real-time validation |
| **Success** | Relief | Account created | Celebration moment |
| **Dashboard** | Excitement | "I'm in!" | Clear next steps |

### Delight Opportunities

1. **Instant feedback** - Real-time validation
2. **Progress indication** - Show steps completed
3. **Welcome message** - Personalized greeting
4. **Quick win** - Guide to first API call

### Anxiety Reducers

1. **"Free, no credit card"** - Remove payment fear
2. **Password visibility toggle** - Reduce typo anxiety
3. **Clear requirements** - Know what's needed upfront
4. **Social proof** - Show other users (optional)

---

## Metrics & KPIs

### Funnel Metrics

| Stage | Metric | Target |
|-------|--------|--------|
| Portal → Signup | Click-through rate | > 40% |
| Signup → Created | Completion rate | > 80% |
| Created → Login | Activation rate | > 95% |
| Login → API Key | Engagement rate | > 60% |

### Time Metrics

| Metric | Target | Alert |
|--------|--------|-------|
| Time on signup form | < 90 sec | > 3 min |
| Time to first login | < 5 min | > 15 min |
| Total onboarding time | < 2 min | > 5 min |

### Analytics Events

```javascript
// Signup started
analytics.track('signup_started', {
  referrer: document.referrer,
  plan_viewed: 'free'
});

// Signup completed
analytics.track('signup_completed', {
  duration_seconds: 45,
  plan: 'free'
});

// First login
analytics.track('first_login', {
  time_since_signup_seconds: 120
});
```

---

## Edge Cases & Errors

### Signup Errors

| Error | Cause | Message |
|-------|-------|---------|
| Email exists | Duplicate email | "An account with this email already exists. [Log in instead]" |
| Invalid email | Bad format | "Please enter a valid email address" |
| Weak password | Doesn't meet requirements | "Password needs [specific requirement]" |
| Terms not accepted | Checkbox unchecked | "Please accept the Terms of Service" |

### Login Errors

| Error | Cause | Message |
|-------|-------|---------|
| Wrong credentials | Bad email/password | "Invalid email or password" |
| Account suspended | Admin action | "Your account has been suspended. Contact support." |
| Account deleted | User deleted | "Account not found" |

### Recovery Flows

**Forgot Password:**
1. Click "Forgot password?"
2. Enter email
3. Receive reset link
4. Create new password
5. Auto-login

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j5-onboarding
requires_fresh_user: true
viewport: 1280x720

steps:
  - name: portal-home
    url: /portal/
    wait: networkidle

  - name: signup-empty
    url: /portal/signup
    wait: networkidle

  - name: signup-filled
    url: /portal/signup
    actions:
      - fill: input[name="name"]
        value: "Sarah Developer"
      - fill: input[name="email"]
        value: "sarah@example.com"
      - fill: input[name="password"]
        value: "SecurePass123!"
      - check: input[name="terms"]

  - name: signup-errors
    url: /portal/signup
    actions:
      - fill: input[name="email"]
        value: "invalid-email"
      - fill: input[name="password"]
        value: "weak"
      - click: button[type="submit"]

  - name: login
    url: /portal/login
    wait: networkidle

  - name: dashboard
    notes: Capture after successful login
```

### GIF Sequence

**j5-signup.gif**
- Frame 1: Portal home (2s)
- Frame 2: Click Sign Up (1s)
- Frame 3: Fill form (3s)
- Frame 4: Submit (1s)
- Frame 5: Success/Dashboard (2s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J6: API Access](j6-api-access.md) | Next step after onboarding |
| [J9: Documentation](j9-documentation.md) | Learn about the API |
| [E1: Auth Errors](../errors/authentication-errors.md) | Login failures |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
