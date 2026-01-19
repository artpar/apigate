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
- **Authentication**: How to use API keys
- **Code Examples**: Copy-paste snippets

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
# Enable portal (default is true)
apigate settings set portal.enabled true

# Disable portal
apigate settings set portal.enabled false
```

### Set App Name

```bash
apigate settings set portal.app_name "Acme API"
```

### Set Portal Base URL

```bash
apigate settings set portal.base_url "https://api.example.com"
```

---

## Customization

### Welcome Message

```bash
apigate settings set custom.portal_welcome_html "<h2>Welcome to Acme API!</h2><p>Get started by creating an API key.</p>"
```

### Custom CSS

```bash
apigate settings set custom.portal_css ".header { background: #3B82F6; }"
```

### Branding

See [[Branding]] for full customization options including:
- Logo URL (`custom.logo_url`)
- Primary color (`custom.primary_color`)
- Support email (`custom.support_email`)

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

### Documentation (`/portal/docs`)

- API reference
- Authentication guide
- Code examples

### Settings (`/portal/settings`)

- Profile (name, email)
- Password change
- Notification preferences

### Billing (`/portal/billing`)

- Current plan details
- Upgrade/downgrade options
- Payment method
- Invoice history

---

## Portal Security

### Session Management

Portal uses secure session cookies with automatic expiration.

### CSRF Protection

Automatically enabled for all portal forms.

### Rate Limiting

The portal is protected by the same rate limiting that applies to API requests.

---

## Embedding Portal

### iframe Embedding

```html
<iframe src="https://api.example.com/portal" width="100%" height="600"></iframe>
```

### Single Sign-On (SSO)

Integrate with your existing auth using OAuth providers:

```bash
apigate settings set oauth.enabled true
apigate settings set oauth.google.enabled true
apigate settings set oauth.google.client_id "your-client-id"
apigate settings set oauth.google.client_secret "your-secret" --encrypted
```

See [[OAuth]] for full SSO configuration.

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

---

## Troubleshooting

### "Portal Not Loading"

1. Check `portal.enabled` setting:
   ```bash
   apigate settings get portal.enabled
   ```
2. Verify web server is running
3. Check browser console for errors

### "Can't Create API Key"

1. Verify user has active plan
2. Check if key limit reached (if configured)

### "Usage Not Updating"

1. Usage updates periodically
2. Check database connection

---

## See Also

- [[API-Keys]] - API key management
- [[Users]] - User management
- [[Branding]] - Customization options
- [[OAuth]] - OAuth/SSO setup
