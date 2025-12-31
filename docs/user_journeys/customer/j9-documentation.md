# J9: Self-Service Documentation

> **Empowering developers to succeed without support tickets.**

---

## Business Context

### Why This Journey Matters

Documentation is the **self-service support layer**. Good docs reduce support costs, accelerate adoption, and build developer trust. Poor docs create frustration and churn.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    DOCUMENTATION VALUE                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Question ──▶ Find Docs ──▶ Get Answer ──▶ Continue Building      │
│      │              │             │                                 │
│      │              │             └── Self-served (low cost)       │
│      │              │                                               │
│      │              └── Can't find ──▶ Support ticket (high cost)  │
│      │                                                              │
│      └── No docs ──▶ Give up ──▶ Churn (lost revenue)             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact (for API Seller)

| Docs Quality | Support Cost | Adoption | Retention |
|--------------|--------------|----------|-----------|
| Excellent | Very Low | Fast | High |
| Good | Low | Moderate | Good |
| Poor | High | Slow | Low |
| None | Very High | Very Slow | Very Low |

### Business Success Criteria

- [ ] Developers find answers within 2 minutes
- [ ] Try It feature works on first attempt
- [ ] Code examples are copy-paste ready
- [ ] Multiple language examples available
- [ ] Search returns relevant results

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Buyer (developer integrating the API) |
| **Prior Action** | Has API key, ready to build |
| **Mental State** | Task-focused, slightly impatient |
| **Expectation** | "Show me how to do X" |

### What Triggered This Journey?

- First time using the API
- Specific task needs reference
- Troubleshooting an error
- Exploring API capabilities
- Before committing to use

### User Goals

1. **Primary:** Find specific technical information quickly
2. **Secondary:** Understand how to use the API correctly
3. **Tertiary:** Test before implementing

---

## The Journey

### Overview

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Entry  │───▶│ Quick   │───▶│Reference│───▶│Examples │───▶│ Try It  │
│  Point  │    │ Start   │    │  Docs   │    │         │    │         │
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

---

### Step 1: Documentation Home

**URL:** `/docs/` or `/portal/docs`

**Purpose:** Orient users and provide quick access to all doc sections.

#### Docs Home Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  [API Name] Documentation                        [Search...]        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Welcome to [API Name]                                              │
│  Everything you need to integrate our API into your application.    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  🚀 Quickstart                                                  ││
│  │  Get up and running in 5 minutes                                ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │ 🔑 Auth       │  │ 📖 Reference  │  │ 💻 Examples   │           │
│  │ API keys and  │  │ All endpoints │  │ Code samples  │           │
│  │ authentication│  │ and methods   │  │ in 5 languages│           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
│                                                                     │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │ 🧪 Try It     │  │ ❓ FAQ        │  │ ⚠️ Errors     │           │
│  │ Test API live │  │ Common        │  │ Error codes   │           │
│  │               │  │ questions     │  │ and handling  │           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Documentation Sections

| Section | Purpose | Content |
|---------|---------|---------|
| **Quickstart** | Get started fast | 5-minute guide |
| **Authentication** | How to auth | API key usage |
| **Reference** | All endpoints | Full API spec |
| **Examples** | Code samples | Multiple languages |
| **Try It** | Live testing | Interactive tester |
| **FAQ** | Common questions | Q&A format |
| **Errors** | Error handling | Error codes, recovery |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Docs home | Page load | `j9-docs/01-docs-home.png` |
| Search | Focus search | `j9-docs/01-docs-search.png` |

---

### Step 2: Quickstart Guide

**URL:** `/docs/quickstart`

**Purpose:** Get developers to first successful API call.

#### Quickstart Content

```markdown
# Quickstart

Get your first API response in under 5 minutes.

## Step 1: Get Your API Key

1. Log in to your account at [portal](/portal)
2. Go to API Keys
3. Click "Create Key"
4. Copy your key (starts with `ak_`)

## Step 2: Make Your First Request

```bash
curl -H "X-API-Key: YOUR_API_KEY" \
  https://api.example.com/health
```

## Step 3: Check the Response

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Congratulations! You've made your first API call.

## Next Steps

- [Authentication Guide](/docs/authentication) - Learn about auth options
- [API Reference](/docs/reference) - Explore all endpoints
- [Code Examples](/docs/examples) - Copy-paste samples
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Quickstart page | Page load | `j9-docs/02-quickstart.png` |
| Code block | Scroll to code | `j9-docs/02-quickstart-code.png` |

---

### Step 3: Authentication Guide

**URL:** `/docs/authentication`

**Purpose:** Explain all authentication methods.

#### Authentication Content

```markdown
# Authentication

All API requests require authentication using an API key.

## Using Your API Key

### Option 1: X-API-Key Header (Recommended)

```bash
curl -H "X-API-Key: ak_your_key_here" \
  https://api.example.com/endpoint
```

### Option 2: Authorization Bearer

```bash
curl -H "Authorization: Bearer ak_your_key_here" \
  https://api.example.com/endpoint
```

## Security Best Practices

- Never expose your API key in client-side code
- Use environment variables
- Rotate keys periodically
- Use different keys for dev/prod

## Rate Limiting

Your plan determines your rate limit:
- Free: 10 requests/minute
- Pro: 600 requests/minute
- Enterprise: 6,000 requests/minute

When rate limited, you'll receive a `429` response with:
- `Retry-After` header indicating when to retry
- `X-RateLimit-Reset` with reset timestamp
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Auth guide | Page load | `j9-docs/03-authentication.png` |
| Best practices | Scroll | `j9-docs/03-auth-best-practices.png` |

---

### Step 4: API Reference

**URL:** `/docs/reference`

**Purpose:** Complete endpoint documentation.

#### Reference Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  API Reference                                      [Search...]     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Endpoints                                                          │
│  ─────────                                                          │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ GET  /health                                                    ││
│  │ Check API health status                                         ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ GET  /api/data                                                  ││
│  │ Retrieve data from the API                                      ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ POST /api/data                                                  ││
│  │ Create new data                                                 ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  GET /health                                                        │
│  ─────────────                                                      │
│  Check the health status of the API.                               │
│                                                                     │
│  Request                                                            │
│  curl -H "X-API-Key: YOUR_KEY" https://api.example.com/health      │
│                                                                     │
│  Response                                                           │
│  {                                                                  │
│    "status": "healthy",                                             │
│    "timestamp": "2024-01-15T10:30:00Z"                             │
│  }                                                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Endpoint Documentation Structure

| Section | Content |
|---------|---------|
| Method + Path | `GET /api/data` |
| Description | What it does |
| Parameters | Query params, path params |
| Request Body | For POST/PUT |
| Response | Success response |
| Errors | Possible error responses |
| Example | cURL + code samples |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Reference index | Page load | `j9-docs/04-reference-index.png` |
| Endpoint detail | Click endpoint | `j9-docs/04-endpoint-detail.png` |

---

### Step 5: Code Examples

**URL:** `/docs/examples`

**Purpose:** Provide copy-paste ready code.

#### Language Tabs

```
┌─────────────────────────────────────────────────────────────────────┐
│  Code Examples                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [cURL] [JavaScript] [Python] [Go] [Ruby]                          │
│  ──────────────────────────────────────────                        │
│                                                                     │
│  Making a GET Request                                               │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ const response = await fetch('https://api.example.com/data', { ││
│  │   headers: {                                                    ││
│  │     'X-API-Key': process.env.API_KEY                           ││
│  │   }                                                             ││
│  │ });                                                             ││
│  │                                                                 ││
│  │ const data = await response.json();                            ││
│  │ console.log(data);                                             ││
│  │                                               [Copy]           ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Making a POST Request                                              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ const response = await fetch('https://api.example.com/data', { ││
│  │   method: 'POST',                                               ││
│  │   headers: {                                                    ││
│  │     'X-API-Key': process.env.API_KEY,                          ││
│  │     'Content-Type': 'application/json'                         ││
│  │   },                                                            ││
│  │   body: JSON.stringify({ name: 'example' })                    ││
│  │ });                                                             ││
│  │                                               [Copy]           ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Supported Languages

| Language | Library Used |
|----------|--------------|
| cURL | Native |
| JavaScript | Fetch API |
| Python | requests |
| Go | net/http |
| Ruby | net/http |
| PHP | curl |
| Java | HttpClient |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Examples - JS | Default view | `j9-docs/05-examples-js.png` |
| Examples - Python | Click Python | `j9-docs/05-examples-python.png` |
| Copy code | Click copy | `j9-docs/05-examples-copy.png` |

---

### Step 6: Try It (Interactive Tester)

**URL:** `/docs/try-it`

**Purpose:** Test API without writing code.

#### Try It Interface

```
┌─────────────────────────────────────────────────────────────────────┐
│  Try It - Interactive API Tester                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  API Key                                                            │
│  [ak_xxxx...____________________________________] [Paste from Keys] │
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  Request                                                            │
│                                                                     │
│  Method        Endpoint                                             │
│  [GET ▼]      [/health________________________________]             │
│                                                                     │
│  Headers                                           [+ Add Header]   │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ (API key header added automatically)                            ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Body (for POST/PUT)                                               │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ {                                                               ││
│  │   "name": "example"                                             ││
│  │ }                                                               ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│                           [Send Request]                            │
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│                                                                     │
│  Response                                            Status: 200 OK │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ {                                                               ││
│  │   "status": "healthy",                                          ││
│  │   "timestamp": "2024-01-15T10:30:00Z"                          ││
│  │ }                                                               ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  Response Headers                                                   │
│  X-RateLimit-Limit: 10                                             │
│  X-RateLimit-Remaining: 9                                          │
│  Content-Type: application/json                                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Try It empty | Page load | `j9-docs/06-try-it-empty.png` |
| Request ready | Filled form | `j9-docs/06-try-it-ready.png` |
| Success response | After send | `j9-docs/06-try-it-success.png` |
| Error response | Bad request | `j9-docs/06-try-it-error.png` |

---

## UX Analysis

### Information Architecture

```
Docs Home (entry point)
├── Quickstart (get started)
├── Authentication (how to auth)
├── Reference (all endpoints)
│   ├── Endpoint 1
│   ├── Endpoint 2
│   └── ...
├── Examples (code samples)
│   ├── By language
│   └── By use case
├── Try It (interactive)
├── FAQ
└── Errors
```

### Search Behavior

Developers expect:
- Instant results as they type
- Search across all sections
- Code-aware (find endpoint by name)
- Recent searches

### Copy-Paste Optimization

All code blocks should be:
- One-click copyable
- Self-contained (include imports)
- Environment-variable aware
- Syntax highlighted

### Accessibility

| Requirement | Implementation |
|-------------|----------------|
| Code blocks | Proper syntax, keyboard nav |
| Navigation | Skip links, landmarks |
| Search | ARIA live regions |
| Tab panels | ARIA tablist |

---

## Metrics & KPIs

### Documentation Effectiveness

| Metric | Definition | Target |
|--------|------------|--------|
| **Time to answer** | Time on docs before success | < 2 min |
| **Search success** | Searches with click | > 70% |
| **Try It usage** | Users who test | > 50% |
| **Copy rate** | Code blocks copied | > 40% |

### Support Reduction

| Metric | Definition | Target |
|--------|------------|--------|
| **Docs-deflected tickets** | Questions answered by docs | > 80% |
| **Self-service rate** | Users who never contact support | > 90% |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j9-documentation
requires_auth: false  # Docs may be public
viewport: 1280x720

steps:
  - name: docs-home
    url: /docs/
    wait: networkidle

  - name: quickstart
    url: /docs/quickstart
    wait: networkidle

  - name: authentication
    url: /docs/authentication
    wait: networkidle

  - name: reference
    url: /docs/reference
    wait: networkidle

  - name: examples-js
    url: /docs/examples
    wait: networkidle

  - name: examples-python
    actions:
      - click: button:has-text("Python")

  - name: try-it
    url: /docs/try-it
    wait: networkidle
```

### GIF Sequence

**j9-docs-tour.gif**
- Frame 1: Docs home (2s)
- Frame 2: Quickstart (2s)
- Frame 3: Reference index (2s)
- Frame 4: Examples (2s)
- Frame 5: Try It with response (3s)

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J5: Onboarding](j5-onboarding.md) | First docs visit |
| [J6: API Access](j6-api-access.md) | Using docs to test |
| [E1-E3: Errors](../errors/) | Error documentation |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
