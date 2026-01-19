# Admin Invites

Admin access in APIGate is managed through an invite system via the web UI.

---

## Overview

```
+-------------------------------------------------------------+
|                    Admin Invite Flow                        |
+-------------------------------------------------------------+
|                                                             |
|   1. Existing admin creates invite via web UI               |
|      Go to /invites in the admin dashboard                  |
|                     |                                       |
|                     v                                       |
|   2. Invite email sent with secure link (if email config)   |
|      https://api.example.com/admin/register/{token}         |
|                     |                                       |
|                     v                                       |
|   3. New admin clicks link, sets name and password          |
|                     |                                       |
|                     v                                       |
|   4. Invite consumed, admin account active                  |
|                                                             |
+-------------------------------------------------------------+
```

---

## First Admin

The first admin can be created via CLI:

```bash
# Create first admin via CLI
apigate admin create --email=admin@example.com

# You will be prompted for a password
# Then visit the web UI to log in
```

---

## Creating Invites

### Web UI (Recommended)

1. Log into the admin dashboard
2. Go to **Invites** (`/invites`)
3. Enter the invitee's email address
4. Click **Create Invite**
5. Share the invite link (shown after creation, and emailed if configured)

### CLI (Direct Admin Creation)

For immediate admin creation without the invite flow:

```bash
# Create admin directly
apigate admin create --email=newadmin@example.com

# With password (not recommended, prompts are safer)
apigate admin create --email=admin@example.com --password=secret
```

---

## Invite Properties

| Property | Description |
|----------|-------------|
| `email` | Invitee email address |
| `token_hash` | SHA-256 hash of secure random token |
| `created_by` | Admin user ID who created invite |
| `created_at` | When invite was created |
| `expires_at` | Expiration time (48 hours from creation) |
| `used_at` | When accepted (null if pending) |

---

## Managing Admin Users

### List Admin Users

```bash
apigate admin list
```

### Delete Admin User

```bash
apigate admin delete <email>
```

### Reset Admin Password

```bash
apigate admin reset-password <email>
```

---

## Security

- Invites expire after 48 hours
- Tokens are cryptographically random (32 bytes)
- Each invite can only be used once
- Tokens are stored as SHA-256 hashes

---

## Email Configuration

For invite emails to be sent automatically, configure SMTP:

```bash
# Email provider
APIGATE_EMAIL_PROVIDER=smtp

# SMTP settings
APIGATE_SMTP_HOST=smtp.example.com
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=user
APIGATE_SMTP_PASSWORD=secret
APIGATE_SMTP_FROM=noreply@example.com
APIGATE_SMTP_FROM_NAME=APIGate
APIGATE_SMTP_USE_TLS=true
```

If email is not configured, admins can manually share the invite link.

---

## See Also

- [[Authentication]] - Authentication overview
- [[Security]] - Security best practices
- [[Email-Configuration]] - Email setup for invites
