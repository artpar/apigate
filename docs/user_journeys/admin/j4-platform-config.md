# J4: Platform Configuration

> **The wiring behind the scenes - payment, email, and backend settings.**

---

## Business Context

### Why This Journey Matters

Configuration is the **infrastructure layer** that enables monetization. Without proper payment setup, API Sellers can't collect revenue. Without email, they can't communicate with customers.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CONFIGURATION DEPENDENCIES                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Payment Config  ──▶  Paid Plans Work  ──▶  Revenue Flows         │
│   Email Config    ──▶  Notifications    ──▶  User Engagement       │
│   Upstream Config ──▶  API Proxying     ──▶  Core Functionality    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Revenue Impact

| Configuration | When Missing | Business Impact |
|---------------|--------------|-----------------|
| **Payment** | Can't accept payments | $0 revenue |
| **Email** | No transactional emails | Reduced engagement |
| **Upstream** | API doesn't work | Total failure |

### Business Success Criteria

- [ ] Payment provider connects in < 5 minutes
- [ ] Test payment processes successfully
- [ ] Transactional emails deliver reliably
- [ ] Upstream changes take effect immediately
- [ ] Settings persist across restarts

---

## User Context

### Who Is This User?

| Attribute | Description |
|-----------|-------------|
| **Persona** | API Seller (configuring for launch or maintenance) |
| **Prior Action** | Setup complete, preparing for real customers |
| **Mental State** | Technical, detail-oriented, careful |
| **Expectation** | "I need to connect my payment provider" |

### What Triggered This Journey?

- Preparing to launch paid plans
- Setting up email notifications
- Changing backend API URL
- Troubleshooting configuration issues

### User Goals

1. **Primary:** Connect payment provider to accept subscriptions
2. **Secondary:** Configure email for customer notifications
3. **Tertiary:** Fine-tune advanced settings

---

## The Journey

### Overview

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Settings │───▶│ Payment  │───▶│  Email   │───▶│ Upstream │
│   Hub    │    │  Config  │    │  Config  │    │  Config  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
```

---

### Step 1: Settings Hub

**URL:** `/ui/setting`

**Purpose:** Central location for all platform configuration.

#### Settings Categories

| Category | Description | Priority |
|----------|-------------|----------|
| **Payment** | Stripe/Paddle/LemonSqueezy | Required for paid plans |
| **Email** | SMTP/SendGrid configuration | Recommended |
| **Upstream** | Backend API URL | Required (set in setup) |
| **Security** | JWT expiration, session settings | Advanced |
| **Branding** | Site name, logo, colors | Optional |

#### Settings Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  Settings                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Payment Provider                               [Not Configured]││
│  │  Accept payments for your paid plans                            ││
│  │                                                    [Configure →]││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Email                                          [Not Configured]││
│  │  Send transactional emails to customers                         ││
│  │                                                    [Configure →]││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Upstream API                                      [Configured] ││
│  │  https://api.example.com                                        ││
│  │                                                         [Edit →]││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Advanced Settings                                              ││
│  │  Security, sessions, and other options                          ││
│  │                                                         [View →]││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Settings hub | Page load | `j4-config/01-settings-hub.png` |
| All configured | Complete setup | `j4-config/01-settings-complete.png` |

---

### Step 2: Payment Configuration

**URL:** `/ui/setting/payment` or inline section

**Purpose:** Connect payment provider for subscription billing.

#### Supported Providers

| Provider | Best For | Setup Complexity |
|----------|----------|------------------|
| **Stripe** | Global, full-featured | Medium |
| **Paddle** | EU compliance, MoR | Low |
| **LemonSqueezy** | Indie hackers | Low |

#### Stripe Configuration

| Field | Description | Required |
|-------|-------------|----------|
| Secret Key | `sk_live_...` or `sk_test_...` | Yes |
| Webhook Secret | `whsec_...` | Yes |
| Webhook URL | Your endpoint for Stripe events | Display only |

#### Setup Flow

```
1. Select Provider
   ┌─────────────────────────────────────────────────────────────────┐
   │  Select Payment Provider                                        │
   │                                                                 │
   │  ( ) Stripe - Industry standard, global coverage               │
   │  ( ) Paddle - All-in-one with tax handling                     │
   │  ( ) LemonSqueezy - Simple setup for indie developers          │
   │                                                                 │
   │                                            [Next]               │
   └─────────────────────────────────────────────────────────────────┘

2. Enter Credentials
   ┌─────────────────────────────────────────────────────────────────┐
   │  Stripe Configuration                                           │
   │                                                                 │
   │  Secret Key:                                                    │
   │  [sk_live_________________________________]                     │
   │  Get this from Stripe Dashboard → Developers → API keys        │
   │                                                                 │
   │  Webhook Secret:                                                │
   │  [whsec_________________________________]                       │
   │  Create a webhook at stripe.com/webhooks pointing to:          │
   │  https://your-domain.com/api/webhooks/stripe                   │
   │                                                                 │
   │  [Test Connection]                         [Save]               │
   └─────────────────────────────────────────────────────────────────┘

3. Test Connection
   ┌─────────────────────────────────────────────────────────────────┐
   │  ✓ Connected to Stripe successfully                            │
   │                                                                 │
   │  Account: ACME Inc                                              │
   │  Mode: Live (use sk_test_... for testing)                      │
   │                                                                 │
   │                                            [Done]               │
   └─────────────────────────────────────────────────────────────────┘
```

#### Webhook Configuration Guide

```markdown
## Setting up Stripe Webhooks

1. Go to [Stripe Dashboard](https://dashboard.stripe.com/webhooks)
2. Click "Add endpoint"
3. Enter URL: `https://your-domain.com/api/webhooks/stripe`
4. Select events:
   - `checkout.session.completed`
   - `customer.subscription.created`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
   - `invoice.paid`
   - `invoice.payment_failed`
5. Click "Add endpoint"
6. Copy the "Signing secret" (starts with `whsec_`)
7. Paste it in the Webhook Secret field above
```

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Provider selection | Open payment config | `j4-config/02-payment-provider.png` |
| Stripe form | Select Stripe | `j4-config/02-stripe-form.png` |
| Test success | Test connection | `j4-config/02-stripe-success.png` |
| Test failure | Bad credentials | `j4-config/02-stripe-error.png` |

---

### Step 3: Email Configuration

**URL:** `/ui/setting/email` or inline section

**Purpose:** Enable transactional email sending.

#### Supported Providers

| Provider | Best For | Setup Complexity |
|----------|----------|------------------|
| **SMTP** | Self-hosted, any provider | Medium |
| **SendGrid** | Easy setup, good deliverability | Low |
| **Mailgun** | Developer-friendly | Low |

#### SMTP Configuration

| Field | Description | Required |
|-------|-------------|----------|
| Host | SMTP server hostname | Yes |
| Port | 587 (TLS) or 465 (SSL) | Yes |
| Username | SMTP username | Yes |
| Password | SMTP password | Yes |
| From Address | Sender email | Yes |
| From Name | Sender display name | Yes |

#### Email Types Sent

| Email | Trigger | Template |
|-------|---------|----------|
| Welcome | User signup | welcome.html |
| Password Reset | Reset request | password-reset.html |
| Key Created | API key generated | key-created.html |
| Usage Warning | 80% quota | usage-warning.html |
| Quota Exceeded | 100% quota | quota-exceeded.html |
| Payment Receipt | Subscription charge | receipt.html |
| Payment Failed | Charge failed | payment-failed.html |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Email config | Open email settings | `j4-config/03-email-config.png` |
| SMTP form | Select SMTP | `j4-config/03-smtp-form.png` |
| Test email sent | Send test | `j4-config/03-email-test.png` |

---

### Step 4: Upstream Configuration

**URL:** `/ui/setting/upstream` or `/ui/upstream`

**Purpose:** Configure the backend API that APIGate proxies to.

#### Configuration Options

| Field | Description | Required |
|-------|-------------|----------|
| Base URL | Backend API URL | Yes |
| Timeout | Request timeout (seconds) | Yes (default: 30) |
| Retry Count | Retries on failure | Yes (default: 0) |
| Health Check Path | Endpoint to check health | No |
| Headers | Custom headers to inject | No |

#### Advanced Options

| Option | Description |
|--------|-------------|
| Skip SSL Verify | For self-signed certs (dev only) |
| Custom CA | For internal CAs |
| Basic Auth | If backend requires auth |
| Rate Limit Upstream | Protect backend from overload |

#### Screenshot Points

| Screenshot | Trigger | File |
|------------|---------|------|
| Upstream settings | Open upstream config | `j4-config/04-upstream.png` |
| Advanced options | Expand advanced | `j4-config/04-upstream-advanced.png` |

---

## UX Analysis

### Cognitive Load by Section

| Section | Complexity | User Skill Required |
|---------|------------|---------------------|
| Payment | High | Stripe dashboard experience |
| Email | Medium | SMTP knowledge |
| Upstream | Low | Basic URL understanding |
| Security | High | Security concepts |

### Friction Points

| Step | Friction | Mitigation |
|------|----------|------------|
| Finding Stripe keys | High | Direct link to Stripe dashboard |
| Webhook setup | High | Step-by-step guide with screenshots |
| SMTP credentials | Medium | Provider-specific instructions |
| Testing | Low | One-click test with clear feedback |

### Help Strategy

Each configuration section includes:
1. **What it does** - Plain English explanation
2. **Why you need it** - Business context
3. **How to set it up** - Step-by-step
4. **How to test** - Verification steps

---

## Emotional Map

```
                     Emotional State During Configuration

Delight  ─┐                                              ┌─ ●
          │                                            ╱
          │                                          ╱
Neutral  ─┼────●─────────────────────────────●─────●
          │      ╲                         ╱
          │        ╲                     ╱
Anxiety  ─┴──────────●─────●───────────●
          │
          └────┬─────────┬─────────┬─────────┬─────────
            Start   Get Keys  Configure   Test!
```

### Emotional Triggers

| Stage | Emotion | Design Response |
|-------|---------|-----------------|
| Start | Neutral | Clear categorization |
| Finding keys | Anxiety | Direct links, guides |
| Enter credentials | Uncertainty | Input validation |
| Test connection | Anticipation | Quick feedback |
| Success | Relief/Delight | Clear success message |

---

## Metrics & KPIs

### Configuration Completion

| Metric | Definition | Target |
|--------|------------|--------|
| **Payment setup rate** | Sellers who configure payment | > 60% |
| **Email setup rate** | Sellers who configure email | > 40% |
| **Time to configure** | Minutes to complete | < 10 min |

### Support Reduction

| Metric | Definition | Target |
|--------|------------|--------|
| **Config-related tickets** | Support requests about setup | < 10% |
| **Self-service success** | Complete without help | > 80% |

---

## Edge Cases & Errors

### Payment Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| Invalid API key | Wrong key copied | Show format hint |
| Webhook failure | Wrong URL | Show expected URL |
| Test charge failed | No payment method | Use Stripe test mode |

### Email Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| Connection refused | Wrong host/port | Check provider docs |
| Auth failed | Wrong credentials | Verify username/password |
| Test email not received | Spam filter | Check spam folder |

---

## Screenshot Automation

### Capture Configuration

```yaml
journey: j4-platform-config
requires_auth: admin
viewport: 1280x720

steps:
  - name: settings-hub
    url: /ui/setting
    wait: networkidle

  - name: payment-provider
    url: /ui/setting
    actions:
      - click: text=Payment Provider
      - wait: text=Select Payment Provider

  - name: stripe-form
    actions:
      - click: text=Stripe
      - wait: input[name="secret_key"]

  - name: email-config
    url: /ui/setting
    actions:
      - click: text=Email
      - wait: form

  - name: upstream
    url: /ui/upstream
    wait: networkidle
```

---

## Related Journeys

| Journey | Relationship |
|---------|-------------|
| [J1: Setup](j1-first-time-setup.md) | Initial upstream config |
| [J2: Plans](j2-plan-management.md) | Stripe Price IDs |
| [J8: Upgrade](../customer/j8-plan-upgrade.md) | Payment flow |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2024-01-XX | Initial documentation | Claude |
