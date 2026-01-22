# APIGate User Guide

## Overview

APIGate is a self-hosted API monetization platform that allows you to easily sell and monetize any API. Whether you're an API provider looking to generate revenue from your services, or a developer wanting to use APIs, APIGate provides a complete solution with minimal setup.

---

## Part 1: For API Providers (Admins)

### Getting Started

#### First-Time Setup

When you first access APIGate, you'll be guided through a simple 3-step setup wizard:

**Step 1: Connect Your API**
- Enter the base URL of your API (e.g., `https://api.example.com`)
- APIGate will verify the connection
- All requests from your customers will be proxied to this URL

**Step 2: Create Admin Account**
- Enter your email address
- Create a secure password
- This becomes your admin account for managing the platform

**Step 3: Set Up Your First Plan**
- Name your plan (e.g., "Free", "Starter", "Pro")
- Set the monthly price (in dollars, 0 for free plans)
- Configure rate limits (requests per minute)
- Set monthly quota (total requests per month)

After completing these steps, you'll be taken to the Admin Dashboard.

---

### Admin Dashboard

The dashboard provides an overview of your API business:

#### Key Metrics
- **Total Users**: Number of registered customers
- **Active API Keys**: Currently active keys in use
- **Requests Today**: API calls made today
- **Monthly Revenue**: Recurring revenue from subscriptions

#### Getting Started Checklist
Follow the checklist to complete your setup:
1. Connect your API (done during setup)
2. Create your admin account (done during setup)
3. Create your first pricing plan (done during setup)
4. Create an API key to test
5. View your documentation portal

---

### Managing Plans

Plans define how customers can access your API and at what price.

#### Creating a Plan

1. Navigate to **Plans** in the admin menu
2. Click **Create New Plan**
3. Fill in the details:

| Field | Description | Example |
|-------|-------------|---------|
| Name | Plan identifier | "Pro" |
| Description | What's included | "For production applications" |
| Monthly Price | Price in dollars | $29.00 |
| Rate Limit | Requests per minute | 600 |
| Monthly Quota | Requests per month | 100,000 |
| Trial Days | Free trial period | 14 |

4. Click **Create Plan**

#### Plan Types

- **Free Plan**: $0/month, limited requests - great for developers to try your API
- **Starter Plan**: Low price, moderate limits - for small projects
- **Pro Plan**: Higher price, generous limits - for production use
- **Enterprise**: Custom pricing, unlimited or very high limits

#### Setting a Default Plan

The default plan is automatically assigned to new users:
1. Go to Plans
2. Click the menu on the desired plan
3. Select "Set as Default"

---

### Managing Users

View and manage your customers:

#### User List
- See all registered users
- View their plan, status, and creation date
- Filter by status (active, suspended, cancelled)

#### User Actions
- **Activate**: Enable a suspended account
- **Suspend**: Temporarily disable access
- **Cancel**: Permanently close the account
- **Change Plan**: Move user to a different plan

---

### Managing API Keys

Monitor API keys across all users:

- View all active keys
- See usage statistics per key
- Revoke keys if needed
- Track last usage time

---

### Configuring Payment Providers

To accept payments for paid plans, configure a payment provider:

#### Supported Providers
- **Stripe**: Industry-standard payment processing
- **Paddle**: All-in-one with tax handling
- **LemonSqueezy**: Simple setup, developer-friendly

#### Stripe Setup
1. Go to **Settings** > **Payment**
2. Select "Stripe" as provider
3. Enter your Stripe Secret Key
4. Enter your Stripe Webhook Secret
5. Save settings

For each paid plan, you'll also need to enter the Stripe Price ID.

---

### Documentation Portal

APIGate automatically generates a documentation portal for your customers at `/docs`.

#### Portal Sections
- **Home**: Overview and quick links
- **Quickstart**: Step-by-step getting started guide
- **Authentication**: How to use API keys
- **API Reference**: Endpoint documentation
- **Examples**: Code samples in multiple languages
- **Try It**: Interactive API tester

#### Customization
The documentation uses your API's OpenAPI/Swagger spec if available. Customize branding in Settings.

---

### Settings

Configure global platform settings:

#### General
- Site name and branding
- Contact email
- Support URL

#### Security
- JWT token expiration
- Password requirements
- Session timeout

#### Email
- SMTP configuration for notifications
- Welcome email templates
- Password reset emails

#### Payment
- Payment provider selection
- API keys and webhooks
- Currency settings

#### Web UI
- Enable/disable admin web UI
- Configure custom base path for UI mounting
- API-only mode for headless deployments

**Configuration Options:**

| Setting | Default | Description |
|---------|---------|-------------|
| `webui.enabled` | `true` | Enable or disable the admin web UI entirely |
| `webui.base_path` | `""` | Custom path to mount the UI (e.g., `/admin-ui`) |

**Environment Variables:**

```bash
export APIGATE_WEBUI_ENABLED=true
export APIGATE_WEBUI_BASE_PATH="/admin-ui"
```

**Use Cases:**

- **Standard Deployment**: Leave defaults for UI at root path
- **Custom Frontend Integration**: Mount UI at custom path (e.g., `/admin-ui`) to serve your own frontend at root
- **Headless/API-Only**: Disable UI entirely and manage via API or CLI only

**Note**: Admin JSON API at `/admin/*` is always accessible regardless of UI settings.

---

## Part 2: For API Customers (Users)

### Getting Started

#### Creating an Account

1. Visit the API portal (provided by your API provider)
2. Click **Sign Up**
3. Enter your details:
   - Name
   - Email address
   - Password (min 8 characters with uppercase, lowercase, number)
4. Agree to Terms of Service
5. Click **Create Account**

You'll receive a confirmation and be redirected to login.

---

### Customer Dashboard

After logging in, you'll see your personalized dashboard:

#### Your Plan
- Current plan name and limits
- Requests used this month
- Requests remaining
- Rate limit (requests per minute)

#### Quick Actions
- Create API Key
- View Documentation
- Check Usage
- Account Settings

#### Getting Started Steps
1. Create an API Key
2. Read the Documentation
3. Make your first API request

---

### Creating API Keys

API keys authenticate your requests to the API.

#### Creating a New Key

1. Go to **API Keys** or click "Create API Key"
2. Click **Create New Key**
3. Enter a name (optional but recommended, e.g., "Production App")
4. Click **Create Key**

**IMPORTANT**: Your API key is shown only once! Copy it immediately and store it securely.

Example key format:
```
ak_81e1ee17656b2cca60f4b6775a3bb39f42a09eaf4291744e983fce244125f702
```

#### Security Best Practices
- Never share your API key publicly
- Don't commit keys to version control
- Use environment variables in your code
- Rotate keys periodically
- Create separate keys for different environments

#### Managing Keys
- View all your keys (masked for security)
- See last usage time
- Revoke keys you no longer need

---

### Using the API

#### Authentication

Include your API key in every request using the `X-API-Key` header:

**cURL**
```bash
curl -H "X-API-Key: YOUR_API_KEY" https://api.example.com/endpoint
```

**JavaScript**
```javascript
fetch('https://api.example.com/endpoint', {
  headers: {
    'X-API-Key': 'YOUR_API_KEY'
  }
})
```

**Python**
```python
import requests

response = requests.get(
    'https://api.example.com/endpoint',
    headers={'X-API-Key': 'YOUR_API_KEY'}
)
```

#### Alternative: Bearer Token

You can also use standard Bearer authentication:
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.example.com/endpoint
```

---

### Monitoring Usage

Track your API consumption in the **Usage** section:

#### Current Period Stats
- **Total Requests**: API calls made this billing period
- **Errors**: Failed requests
- **Data In**: Request data transferred
- **Data Out**: Response data transferred

#### Quota Warnings
- You'll see warnings when approaching your limit
- At 80%: Approaching limit notification
- At 95%: Critical - consider upgrading
- At 100%: Requests may be blocked (depends on plan)

---

### Managing Your Plan

#### Viewing Available Plans

1. Go to **Plans**
2. See all available plans with:
   - Monthly price
   - Requests included
   - Rate limits
   - Additional features

#### Upgrading Your Plan

1. Find the plan you want
2. Click **Upgrade**
3. Complete payment (if applicable)
4. Your new limits are effective immediately

#### Downgrading

Contact support to downgrade to a lower plan. Changes typically take effect at the next billing cycle.

---

### Account Settings

Manage your account in **Settings**:

#### Profile
- Update your display name
- View your email (contact support to change)

#### Password
- Change your password
- Requires current password for security

#### Danger Zone
- Close your account
- **Warning**: This revokes all API keys and deletes your data

---

### Documentation

The **Docs** section provides everything you need:

#### Quickstart
Step-by-step guide to make your first API call.

#### Authentication
Detailed explanation of authentication methods.

#### API Reference
Complete endpoint documentation with:
- HTTP methods
- URL paths
- Request parameters
- Response formats
- Error codes

#### Examples
Ready-to-use code samples in:
- cURL
- JavaScript
- Python
- More languages as available

#### Try It
Interactive API tester:
1. Enter your API key
2. Select HTTP method
3. Enter endpoint path
4. Add request body (for POST/PUT)
5. Click **Send Request**
6. View the response

---

### Getting Help

#### Common Issues

**"Invalid API Key" Error**
- Check that you're using the correct key
- Ensure the key hasn't been revoked
- Verify the X-API-Key header is correct

**"Rate Limit Exceeded" Error**
- You've exceeded requests per minute
- Wait a moment and retry
- Consider upgrading your plan

**"Quota Exceeded" Error**
- You've used all monthly requests
- Wait for the next billing cycle
- Upgrade to a higher plan

#### Contact Support
- Email: Listed in the portal
- Documentation: /docs
- For enterprise inquiries: Contact the API provider directly

---

## Appendix: Quick Reference

### API Key Header
```
X-API-Key: your_api_key_here
```

### Response Headers
- `X-RateLimit-Limit`: Requests allowed per minute
- `X-RateLimit-Remaining`: Requests remaining this minute
- `X-RateLimit-Reset`: When the limit resets (Unix timestamp)

### Error Codes
| Code | Meaning |
|------|---------|
| 401 | Invalid or missing API key |
| 403 | Access denied (account suspended) |
| 429 | Rate limit exceeded |
| 503 | Quota exceeded |

### URLs
- Customer Portal: `/portal`
- Documentation: `/docs`
- Admin Dashboard: `/ui` (admin only)
