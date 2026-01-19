# Single Sign-On (SSO)

APIGate supports Single Sign-On through OAuth 2.0 and OpenID Connect.

---

## Overview

SSO allows users to sign in using their existing identity provider:

- Google Workspace
- GitHub
- Azure AD
- Okta
- Any OIDC provider

---

## Supported Providers

### Google

```bash
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=xxx.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=xxx
```

### GitHub

```bash
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=xxx
OAUTH_GITHUB_CLIENT_SECRET=xxx
```

### Generic OIDC

For enterprise IdPs (Okta, Azure AD, etc.):

```bash
OAUTH_OIDC_ENABLED=true
OAUTH_OIDC_ISSUER=https://your-idp.com
OAUTH_OIDC_CLIENT_ID=xxx
OAUTH_OIDC_CLIENT_SECRET=xxx
```

---

## Configuration

See [[OAuth]] for detailed setup instructions.

---

## User Linking

SSO identities are linked to APIGate users:

- By email (automatic if `oauth.auto_link_by_email=true`)
- Manually via account settings

---

## See Also

- [[OAuth]] - OAuth configuration
- [[Authentication]] - Auth overview
- [[Security]] - Security features
