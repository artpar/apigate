# J6: Get API Access

> **The activation moment - where curiosity becomes usage.**

---

## Business Context

### Why This Journey Matters

Creating an API key is the **activation event** that transforms a signup into an actual user. Until a customer has a working API key, they're not really using the product.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ACTIVATION FUNNEL                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Signup ──▶ Create Key ──▶ Copy Key ──▶ First Call ──▶ Regular Use│
│    100%       70%           68%          50%           40%          │
│                                                                     │
│   Target: 80%+ of signups create an API key within 24 hours        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact (for API Seller)

| Metric | Impact |
|--------|--------|
| **Key creation rate** | Strong predictor of conversion to paid |
| **Time to first call** | Faster = higher engagement |
| **First call success** | Failed first calls = churn risk |

### Business Success Criteria

- [ ] Key creation takes < 30 seconds
- [ ] Key is shown clearly for copy
- [ ] Test API call succeeds on first try
- [ ] Rate limit headers visible in response
- [ ] Clear path to documentation

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Buyer (just signed up, ready to build) |
| **Prior Action** | Completed signup, viewing dashboard |
| **Mental State** | Motivated, wants to test quickly |
| **Expectation** | "Give me an API key so I can start" |

### What Triggered This Journey?

- Just completed signup
- Following getting started checklist
- Ready to integrate API into their app
- Testing API capabilities

### User Goals

1. **Primary:** Get a working API key
2. **Secondary:** Verify the key works with a test call
3. **Tertiary:** Understand rate limits and usage

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│Dashboard│───▶│API Keys │───▶│ Create  │───▶│  Copy   │───▶│  Test   │
│         │    │  Page   │    │   Key   │    │   Key   │    │   API   │
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

---

### Step 1: Navigate to API Keys

**URL:** `/portal/keys` or via dashboard

**Entry Points:**
- Dashboard "Create API Key" button
- Navigation menu "API Keys"
- Getting started checklist item

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Keys page empty | No keys yet | `j6-api-access/01-keys-empty.png` |
| Keys page with keys | Has existing keys | `j6-api-access/01-keys-list.png` |

---

### Step 2: Create API Key

**URL:** `/portal/keys/new` or modal

**Purpose:** Generate a new API key.

#### Creation Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Create API Key                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Name (optional)                                                    │
│  [_________________________________________________]               │
│  Give your key a name to identify it later                         │
│                                                                     │
│  Example: "Production", "Development", "Testing"                    │
│                                                                     │
│                [Cancel]  [Create Key]                               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Key Generation

- System generates cryptographically secure key
- Key format: `ak_` + 64 hex characters
- Key is hashed before storage (bcrypt)
- Only shown once at creation time

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Create dialog | Click create | `j6-api-access/02-create-dialog.png` |
| Name entered | Fill name | `j6-api-access/02-name-entered.png` |

---

### Step 3: Key Revealed

**URL:** Same page/modal

**Purpose:** Show the key for the user to copy.

#### Key Display

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Your API Key                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ⚠️  Copy this key now - you won't be able to see it again!        │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ ak_81e1ee17656b2cca60f4b6775a3bb39f42a09eaf4291744e983fce24411 ││
│  │                                                        [Copy]  ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Quick Start                                                        │
│  ───────────                                                        │
│  curl -H "X-API-Key: YOUR_KEY" https://api.example.com/endpoint    │
│                                                                     │
│                           [Done]                                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Security Warning

The key is:
- Shown only once
- Never stored in plaintext
- Cannot be recovered if lost
- Can be revoked and regenerated

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Key shown | Key generated | `j6-api-access/03-key-shown.png` |
| Copy confirmation | Click copy | `j6-api-access/03-key-copied.png` |

---

### Step 4: Key in List

**URL:** `/portal/keys`

**Purpose:** Manage existing keys.

#### Key List Table

| Column | Description |
|--------|-------------|
| Name | User-provided name |
| Key (masked) | `ak_81e1...f702` |
| Created | Creation date |
| Last Used | Most recent request |
| Status | Active/Revoked |
| Actions | Revoke button |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Key in list | After creation | `j6-api-access/04-key-in-list.png` |
| Multiple keys | Several keys | `j6-api-access/04-multiple-keys.png` |

---

### Step 5: Test API

**URL:** `/docs/try-it` or `/portal/test`

**Purpose:** Verify the key works.

#### Try It Interface

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Try Your API                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  API Key                                                            │
│  [ak_81e1ee17656b2cca60f4b6775a3bb39f42a09eaf...________] [Paste]  │
│                                                                     │
│  Method    Endpoint                                                 │
│  [GET ▼]  [/health________________________________]                 │
│                                                                     │
│                    [Send Request]                                   │
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  Response                                              Status: 200  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ {                                                               ││
│  │   "status": "healthy",                                          ││
│  │   "timestamp": "2024-01-15T10:30:00Z"                          ││
│  │ }                                                               ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Headers                                                            │
│  X-RateLimit-Limit: 10                                             │
│  X-RateLimit-Remaining: 9                                          │
│  X-RateLimit-Reset: 1705315860                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Try It empty | Page load | `j6-api-access/05-try-it-empty.png` |
| Key entered | Paste key | `j6-api-access/05-key-entered.png` |
| Successful response | Send request | `j6-api-access/05-success.png` |
| Error response | Bad key | `j6-api-access/05-error.png` |

---

## UX Analysis

### Cognitive Load Assessment

| Step | Decisions | Load |
|------|-----------|------|
| Navigate | 0-1 | Very Low |
| Create key | 1 (name) | Very Low |
| Copy key | 0 | Low |
| Test API | 2-3 | Low-Medium |

### Friction Points

| Step | Friction | Mitigation |
|------|----------|------------|
| Finding create button | Low | Prominent placement |
| Key shown once | Medium | Clear warning |
| Forgetting to copy | High | Auto-copy option |
| Testing | Low | Pre-filled example |

### Critical UX: Key Display

The key reveal moment is critical:

1. **Visibility** - Key must be obviously visible
2. **Copy support** - One-click copy to clipboard
3. **Warning** - Clear "shown once only" message
4. **Confirmation** - Show copied confirmation
5. **Next steps** - Guide to testing

### Accessibility

| Requirement | Implementation |
|-------------|----------------|
| Copy button | Keyboard accessible |
| Key text | Selectable |
| Success feedback | Visual + screen reader |
| Warning | High contrast, icon |

---

## Emotional Map

```
                     Emotional State During API Access

Delight  ─┐                                              ┌─ ●
          │                                            ╱
          │                   ●─────────────●────────●
Neutral  ─┼────●───────●───╱
          │          ╱
          │        ╱
Anxiety  ─┴──────●
          │
          └────┬─────────┬─────────┬─────────┬─────────
            Start   Create    Copy     Test!
```

### Emotional Journey

| Stage | Emotion | Trigger | Design Response |
|-------|---------|---------|-----------------|
| **Start** | Neutral | Navigating | Clear navigation |
| **Create** | Slight anticipation | About to get key | Simple form |
| **Key revealed** | Anxiety (don't lose it!) | Shown once warning | Clear copy button |
| **Copied** | Relief | Confirmation | "Copied!" feedback |
| **Test works** | Delight | Successful call | Show all response details |

### Delight Opportunities

1. **Auto-copy option** - Copy key automatically
2. **Quick start snippet** - Ready-to-run curl command
3. **Response highlighting** - Syntax-colored JSON
4. **Rate limit visibility** - Show headers clearly

---

## Metrics & KPIs

### Activation Metrics

| Metric | Definition | Target |
|--------|------------|--------|
| **Key creation rate** | Users who create a key | > 80% |
| **Time to key** | Signup to first key | < 5 min |
| **Copy rate** | Users who copy the key | > 95% |
| **First call rate** | Keys used within 24h | > 70% |

### Engagement Metrics

| Metric | Definition | Target |
|--------|------------|--------|
| **Keys per user** | Average keys created | 1.5 |
| **Try It usage** | Users who test | > 50% |
| **Test success rate** | First tests that work | > 90% |

### Analytics Events

```javascript
// Key creation started
analytics.track('api_key_create_started');

// Key created
analytics.track('api_key_created', {
  key_name: 'Production',
  time_since_signup_seconds: 120
});

// Key copied
analytics.track('api_key_copied', {
  copy_method: 'button' // or 'select'
});

// First API call
analytics.track('first_api_call', {
  success: true,
  endpoint: '/health',
  time_since_key_creation_seconds: 30
});
```

---

## Edge Cases & Errors

### Creation Errors

| Error | Cause | Message |
|-------|-------|---------|
| Key limit reached | Too many keys | "Maximum keys reached. Revoke unused keys." |
| Server error | System issue | "Unable to create key. Please try again." |

### Copy Failures

| Scenario | Detection | Fallback |
|----------|-----------|----------|
| Clipboard blocked | API fails | Manual selection |
| Mobile browser | Varies | Long-press to copy |

### Test Errors

| Error | Status | Response | Action |
|-------|--------|----------|--------|
| Invalid key | 401 | `missing_api_key` | Check key format |
| Wrong key | 401 | `invalid_api_key` | Verify correct key |
| Rate limited | 429 | `rate_limit_exceeded` | Wait and retry |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j6-api-access
requires_auth: customer
viewport: 1280x720

steps:
  - name: keys-empty
    url: /portal/keys
    wait: networkidle

  - name: create-dialog
    url: /portal/keys
    actions:
      - click: button:has-text("Create")
      - wait: dialog

  - name: name-entered
    actions:
      - fill: input[name="name"]
        value: "Production"

  - name: key-shown
    actions:
      - click: button:has-text("Create")
      - wait: text=ak_

  - name: key-copied
    actions:
      - click: button:has-text("Copy")
      - wait: text=Copied

  - name: try-it
    url: /docs/try-it
    wait: networkidle

  - name: success
    actions:
      - fill: input[name="api_key"]
        value: "${CREATED_KEY}"
      - click: button:has-text("Send")
      - wait: text=200
```

### GIF Sequence

**j6-create-key.gif**
- Frame 1: Keys page empty (1s)
- Frame 2: Click Create (1s)
- Frame 3: Enter name (1s)
- Frame 4: Key revealed (2s)
- Frame 5: Click Copy (1s)
- Frame 6: Copied confirmation (1s)

**j6-test-api.gif**
- Frame 1: Try It page (1s)
- Frame 2: Paste key (1s)
- Frame 3: Click Send (1s)
- Frame 4: Success response (2s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J5: Onboarding](j5-onboarding.md) | Previous step |
| [J7: Usage](j7-usage-monitoring.md) | After using key |
| [J9: Documentation](j9-documentation.md) | How to use API |
| [E1: Auth Errors](../errors/authentication-errors.md) | Key errors |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
