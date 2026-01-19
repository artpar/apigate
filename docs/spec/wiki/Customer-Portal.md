# Customer Portal

The **Customer Portal** is a self-service interface where your API customers manage their accounts.

---

## Overview

The portal enables customers to:
- View and manage API keys
- Monitor usage and quotas
- Access API documentation
- Manage billing and subscriptions

```
┌────────────────────────────────────────────────────────────────┐
│                      Customer Portal                            │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  🔑 API Keys    📊 Usage    📖 Docs    💳 Billing        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─────────────────────┐  ┌─────────────────────────────────┐  │
│  │                     │  │                                 │  │
│  │   Your API Keys     │  │   Usage This Month              │  │
│  │                     │  │                                 │  │
│  │   ak_abc... [Copy]  │  │   ████████░░ 8,234 / 10,000    │  │
│  │   ak_def... [Copy]  │  │                                 │  │
│  │                     │  │   Rate: 45 req/min (limit: 60)  │  │
│  │   [+ New Key]       │  │                                 │  │
│  │                     │  │                                 │  │
│  └─────────────────────┘  └─────────────────────────────────┘  │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Accessing the Portal

### URL

```
https://your-domain.com/portal
```

### Customer Registration

1. Visit `/portal/register`
2. Enter email and password
3. Verify email (if required)
4. Access portal dashboard

### Customer Login

1. Visit `/portal/login`
2. Enter credentials
3. Redirected to dashboard

---

## Portal Features

### API Key Management

Customers can:

| Action | Description |
|--------|-------------|
| **View keys** | See prefix and creation date |
| **Create key** | Generate new API key |
| **Name key** | Set descriptive name |
| **Revoke key** | Permanently disable key |

**Note**: Full key shown only at creation time.

### Usage Dashboard

Real-time usage metrics:

- **Current month requests**: X / quota
- **Rate limit status**: Current vs max
- **Usage history**: Daily/weekly charts
- **Top endpoints**: Most used routes

### API Documentation

Integrated documentation viewer:

- **API Reference**: All available endpoints
- **Try It**: Interactive API testing
- **Code Examples**: Copy-paste snippets
- **Authentication**: How to use API keys

### Billing & Subscription

If payment integration enabled:

- **Current plan**: Name, limits, price
- **Upgrade/Downgrade**: Change plans
- **Payment method**: Update card
- **Invoices**: Download history

---

## Portal Configuration

### Enable/Disable Portal

```bash
# Enable portal (default)
apigate settings set portal_enabled true

# Disable portal
apigate settings set portal_enabled false
```

### Customize Portal URL

```bash
apigate settings set portal_base_path "/developer"
# Now at: https://your-domain.com/developer
```

### Branding

```bash
# Company name
apigate settings set portal_company_name "Acme API"

# Logo URL
apigate settings set portal_logo_url "https://example.com/logo.png"

# Primary color
apigate settings set portal_primary_color "#3B82F6"

# Custom CSS
apigate settings set portal_custom_css_url "https://example.com/custom.css"
```

### Registration Settings

```bash
# Allow self-registration
apigate settings set portal_registration_enabled true

# Require email verification
apigate settings set require_email_verification true

# Default plan for new users
apigate plans update <plan-id> --default true
```

### Feature Toggles

```bash
# Enable/disable portal sections
apigate settings set portal_show_usage true
apigate settings set portal_show_docs true
apigate settings set portal_show_billing true
apigate settings set portal_allow_key_creation true
```

---

## Portal Pages

### Dashboard (`/portal`)

Overview with:
- Quick stats (requests, quota, keys)
- Recent activity
- Alerts (quota warnings)

### API Keys (`/portal/keys`)

- List all keys (prefix only)
- Create new key
- Revoke existing key
- Copy key to clipboard

### Usage (`/portal/usage`)

- Monthly usage chart
- Daily breakdown
- Per-endpoint breakdown
- Export usage data (CSV)

### Documentation (`/portal/docs`)

- API reference
- Authentication guide
- Code examples
- Changelog

### Settings (`/portal/settings`)

- Profile (name, email)
- Password change
- Notification preferences
- Delete account

### Billing (`/portal/billing`)

- Current plan details
- Upgrade/downgrade options
- Payment method
- Invoice history

---

## Customizing Portal Content

### Welcome Message

```bash
apigate settings set portal_welcome_message "Welcome to Acme API! Get started by creating an API key."
```

### Documentation Content

Place custom documentation in:
```
/data/docs/
├── getting-started.md
├── authentication.md
├── endpoints/
│   ├── users.md
│   └── orders.md
└── examples/
    ├── curl.md
    └── python.md
```

### Custom Pages

Add custom portal pages:

```bash
apigate portal pages create \
  --slug "changelog" \
  --title "Changelog" \
  --content-file ./changelog.md
```

---

## Portal Security

### Session Management

```bash
# Session duration (default: 24 hours)
apigate settings set session_duration_hours 24

# Session cookie settings
apigate settings set session_secure_cookie true
apigate settings set session_same_site "strict"
```

### Rate Limiting (Portal)

Portal has separate rate limits:

```bash
# Login attempts
apigate settings set portal_login_rate_limit 5  # per minute

# Password reset requests
apigate settings set portal_reset_rate_limit 3  # per hour
```

### CSRF Protection

Automatically enabled for all portal forms.

### Two-Factor Authentication

```bash
# Enable 2FA option for customers
apigate settings set portal_2fa_enabled true
```

---

## Embedding Portal

### iframe Embedding

```html
<iframe src="https://api.example.com/portal" width="100%" height="600"></iframe>
```

### Single Sign-On (SSO)

Integrate with your existing auth:

```bash
# Configure SSO provider
apigate settings set sso_provider "oauth2"
apigate settings set sso_client_id "your-client-id"
apigate settings set sso_client_secret "your-secret"
apigate settings set sso_authorize_url "https://auth.example.com/authorize"
apigate settings set sso_token_url "https://auth.example.com/token"
```

### API-Only (Headless)

Build custom portal using API:

```bash
# All portal actions available via API
GET  /api/portal/profile
GET  /api/portal/keys
POST /api/portal/keys
DELETE /api/portal/keys/:id
GET  /api/portal/usage
GET  /api/portal/billing
```

---

## Portal Notifications

### Email Notifications

Customers receive emails for:

| Event | Email |
|-------|-------|
| Registration | Welcome email |
| Key created | Key confirmation |
| Quota 80% | Warning |
| Quota 100% | Alert |
| Plan changed | Confirmation |
| Password reset | Reset link |

### In-Portal Notifications

Alerts shown in portal:
- Quota warnings
- Plan expiration
- New features

---

## Analytics

Track portal usage:

```bash
# Portal analytics
apigate analytics portal

# Output:
# - Active users (DAU/MAU)
# - Key creation rate
# - Docs views
# - Upgrade conversions
```

---

## Troubleshooting

### "Portal Not Loading"

1. Check `portal_enabled` setting
2. Verify web server is running
3. Check browser console for errors

### "Can't Create API Key"

1. Check `portal_allow_key_creation` setting
2. Verify user has active plan
3. Check if key limit reached (if configured)

### "Usage Not Updating"

1. Usage updates every minute
2. Check usage tracking is enabled
3. Verify database connection

---

## See Also

- [[API-Keys]] - API key management
- [[Users]] - User management
- [[Branding]] - Customization options
- [[SSO]] - Single sign-on setup
