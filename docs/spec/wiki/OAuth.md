# OAuth Authentication

APIGate supports OAuth 2.0 / OpenID Connect for user authentication, allowing users to sign in with external identity providers.

---

## Overview

OAuth enables "Sign in with Google/GitHub" functionality:

```
┌────────────────────────────────────────────────────────────────┐
│                      OAuth Flow                                 │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. User clicks "Sign in with Google"                           │
│                    │                                            │
│                    ▼                                            │
│  2. APIGate redirects to Google                                 │
│     └─▶ /auth/oauth/google                                      │
│                    │                                            │
│                    ▼                                            │
│  3. User authorizes at Google                                   │
│                    │                                            │
│                    ▼                                            │
│  4. Google redirects back with code                             │
│     └─▶ /auth/oauth/google/callback?code=xxx                    │
│                    │                                            │
│                    ▼                                            │
│  5. APIGate exchanges code for tokens                           │
│                    │                                            │
│                    ▼                                            │
│  6. APIGate fetches user profile                                │
│                    │                                            │
│                    ▼                                            │
│  7. User logged in (or created if new)                          │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## Supported Providers

### Google

OpenID Connect with Google accounts.

| Setting | Value |
|---------|-------|
| Provider Name | `google` |
| Auth URL | `https://accounts.google.com/o/oauth2/auth` |
| Token URL | `https://oauth2.googleapis.com/token` |
| Scopes | `openid email profile` |

### GitHub

OAuth 2.0 with GitHub accounts.

| Setting | Value |
|---------|-------|
| Provider Name | `github` |
| Auth URL | `https://github.com/login/oauth/authorize` |
| Token URL | `https://github.com/login/oauth/access_token` |
| Scopes | `user:email` |

### Generic OIDC

Any OpenID Connect compatible provider.

| Setting | Value |
|---------|-------|
| Provider Name | `oidc` |
| Issuer URL | Your provider's issuer URL |
| Discovery | `{issuer}/.well-known/openid-configuration` |

---

## Configuration

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create OAuth 2.0 credentials
3. Set redirect URI: `https://your-domain.com/auth/oauth/google/callback`
4. Configure in APIGate:

```bash
# Environment variables
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your-client-id.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
```

Or via CLI:

```bash
apigate settings set oauth.google.enabled true
apigate settings set oauth.google.client_id "your-client-id"
apigate settings set oauth.google.client_secret "your-secret"
```

### GitHub OAuth Setup

1. Go to GitHub Settings > Developer Settings > OAuth Apps
2. Create new OAuth App
3. Set callback URL: `https://your-domain.com/auth/oauth/github/callback`
4. Configure in APIGate:

```bash
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=your-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-client-secret
```

### Generic OIDC Setup

```bash
OAUTH_OIDC_ENABLED=true
OAUTH_OIDC_ISSUER=https://your-idp.com
OAUTH_OIDC_CLIENT_ID=your-client-id
OAUTH_OIDC_CLIENT_SECRET=your-client-secret
```

---

## OAuth Endpoints

### Start OAuth Flow

```
GET /auth/oauth/{provider}
```

Redirects user to the OAuth provider for authentication.

**Example**:
```
https://api.example.com/auth/oauth/google
```

### OAuth Callback

```
GET /auth/oauth/{provider}/callback
```

Handles the callback from the OAuth provider after authentication.

**Query Parameters**:
| Parameter | Description |
|-----------|-------------|
| `code` | Authorization code from provider |
| `state` | CSRF protection token |
| `error` | Error code (if auth failed) |

---

## OAuth Identity

When a user signs in via OAuth, an OAuth Identity record is created:

### Identity Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `user_id` | string | Linked APIGate user |
| `provider` | string | Provider name (google, github, oidc) |
| `provider_user_id` | string | User ID from the provider |
| `email` | string | Email from provider |
| `name` | string | Display name from provider |
| `avatar_url` | string | Profile picture URL |
| `access_token` | string | OAuth access token (encrypted) |
| `refresh_token` | string | OAuth refresh token (encrypted) |
| `token_expires_at` | timestamp | Token expiration time |
| `created_at` | timestamp | When identity was linked |

### List User's OAuth Identities

```bash
# CLI
apigate oauth-identities list --user "user-id"

# API
curl http://localhost:8080/admin/oauth/identities/user/<user-id>
```

### Unlink OAuth Identity

```bash
# CLI
apigate oauth-identities unlink <identity-id>

# API
curl -X DELETE http://localhost:8080/admin/oauth/identities/<id>
```

---

## OAuth State

CSRF protection via state tokens:

### State Properties

| Property | Type | Description |
|----------|------|-------------|
| `state` | string | Random CSRF token |
| `provider` | string | OAuth provider |
| `redirect_uri` | string | Where to redirect after auth |
| `code_verifier` | string | PKCE code verifier |
| `nonce` | string | OIDC nonce for ID token |
| `expires_at` | timestamp | State expiration (10 min) |

States are automatically cleaned up after use or expiration.

---

## User Linking

### Auto-Linking by Email

When a user signs in via OAuth:

1. If email matches existing user: Link identity to existing user
2. If no match: Create new user with OAuth email

```bash
# Configure auto-linking
apigate settings set oauth.auto_link_by_email true
```

### Manual Linking

Existing users can link additional OAuth providers:

1. User logs in with email/password
2. Goes to Settings > Connected Accounts
3. Clicks "Connect Google/GitHub"
4. Completes OAuth flow
5. Identity linked to existing account

---

## Security Features

### PKCE Support

Proof Key for Code Exchange prevents authorization code interception:

```bash
# Enable PKCE (recommended)
apigate settings set oauth.use_pkce true
```

### State Validation

All OAuth flows include state parameter validation to prevent CSRF attacks.

### Token Encryption

OAuth tokens (access_token, refresh_token) are encrypted at rest using the application secret.

### Token Refresh

When access tokens expire, APIGate automatically uses the refresh token to get new tokens.

---

## Login Page Integration

OAuth buttons appear on the login page when enabled:

```html
<!-- Rendered when Google OAuth enabled -->
<a href="/auth/oauth/google" class="oauth-button">
  Sign in with Google
</a>

<!-- Rendered when GitHub OAuth enabled -->
<a href="/auth/oauth/github" class="oauth-button">
  Sign in with GitHub
</a>
```

---

## Troubleshooting

### Invalid Redirect URI

**Error**: `redirect_uri_mismatch`

**Solution**: Ensure the callback URL in your OAuth provider settings exactly matches:
```
https://your-domain.com/auth/oauth/{provider}/callback
```

### Missing Email Scope

**Error**: User created without email

**Solution**: Ensure your OAuth app requests email scope:
- Google: `openid email profile`
- GitHub: `user:email`

### State Mismatch

**Error**: `invalid_state`

**Cause**: State token expired (>10 min) or browser cookies disabled

**Solution**:
- Complete OAuth flow within 10 minutes
- Ensure cookies are enabled

### Access Denied

**Error**: `access_denied`

**Cause**: User cancelled OAuth flow or denied permissions

**Solution**: Redirect back to login page with error message

---

## Best Practices

### 1. Enable Multiple Providers

Give users choice:

```bash
apigate settings set oauth.google.enabled true
apigate settings set oauth.github.enabled true
```

### 2. Use PKCE

Always enable PKCE for enhanced security:

```bash
apigate settings set oauth.use_pkce true
```

### 3. Set Redirect URIs Carefully

Use exact HTTPS URLs:

```
# Good
https://api.example.com/auth/oauth/google/callback

# Bad
http://localhost:8080/auth/oauth/google/callback  # HTTP
https://api.example.com/auth/oauth/google  # Missing /callback
```

### 4. Handle Account Linking

Allow users to link multiple providers to one account for flexibility.

---

## CLI Commands

```bash
# List OAuth identities
apigate oauth-identities list
apigate oauth-identities list --user "user-id"

# Get identity details
apigate oauth-identities get <id>

# Unlink identity
apigate oauth-identities unlink <id>

# Cleanup expired states
apigate oauth-states cleanup
```

---

## See Also

- [[Users]] - User management
- [[Configuration]] - OAuth settings
- [[Customer-Portal]] - Portal login integration
