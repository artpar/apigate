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
# Via CLI (stored in database)
apigate settings set oauth.google.enabled true
apigate settings set oauth.google.client_id "your-client-id.googleusercontent.com"
apigate settings set oauth.google.client_secret "your-client-secret" --encrypted
```

### GitHub OAuth Setup

1. Go to GitHub Settings > Developer Settings > OAuth Apps
2. Create new OAuth App
3. Set callback URL: `https://your-domain.com/auth/oauth/github/callback`
4. Configure in APIGate:

```bash
apigate settings set oauth.github.enabled true
apigate settings set oauth.github.client_id "your-client-id"
apigate settings set oauth.github.client_secret "your-client-secret" --encrypted
```

### Generic OIDC Setup

```bash
apigate settings set oauth.oidc.enabled true
apigate settings set oauth.oidc.name "My IdP"
apigate settings set oauth.oidc.issuer_url "https://your-idp.com"
apigate settings set oauth.oidc.client_id "your-client-id"
apigate settings set oauth.oidc.client_secret "your-client-secret" --encrypted
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

## User Linking

### Auto-Linking by Email

When a user signs in via OAuth:

1. If email matches existing user: Link identity to existing user
2. If no match: Create new user with OAuth email

### Manual Linking

Existing users can link additional OAuth providers:

1. User logs in with email/password
2. Goes to Settings > Connected Accounts
3. Clicks "Connect Google/GitHub"
4. Completes OAuth flow
5. Identity linked to existing account

---

## Security Features

### State Validation

All OAuth flows include state parameter validation to prevent CSRF attacks.

### Token Encryption

OAuth tokens (access_token, refresh_token) are encrypted at rest using the application secret.

### PKCE Support

Proof Key for Code Exchange prevents authorization code interception. PKCE is automatically used when supported by the provider.

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

### 2. Set Redirect URIs Carefully

Use exact HTTPS URLs:

```
# Good
https://api.example.com/auth/oauth/google/callback

# Bad
http://localhost:8080/auth/oauth/google/callback  # HTTP in production
https://api.example.com/auth/oauth/google  # Missing /callback
```

### 3. Handle Account Linking

Allow users to link multiple providers to one account for flexibility.

---

## See Also

- [[Users]] - User management
- [[Configuration]] - Full configuration reference
- [[Customer-Portal]] - Portal login integration
