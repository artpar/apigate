# Email Configuration

APIGate sends emails for password reset, email verification, and welcome messages.

---

## Providers

### SMTP

Standard email via any SMTP server:

```bash
APIGATE_EMAIL_PROVIDER=smtp
APIGATE_SMTP_HOST=smtp.example.com
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=apigate@example.com
APIGATE_SMTP_PASSWORD=xxx
APIGATE_SMTP_FROM=noreply@example.com
APIGATE_SMTP_FROM_NAME="APIGate"
APIGATE_SMTP_USE_TLS=true
```

### Mock (Development)

Stores emails in memory for testing. Does not send actual emails:

```bash
APIGATE_EMAIL_PROVIDER=mock
```

### None (Default)

Disables email sending. Password reset and email verification will not work:

```bash
APIGATE_EMAIL_PROVIDER=none
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APIGATE_EMAIL_PROVIDER` | `none` | Provider: `smtp`, `mock`, `none` |
| `APIGATE_SMTP_HOST` | - | SMTP server hostname |
| `APIGATE_SMTP_PORT` | `587` | SMTP server port |
| `APIGATE_SMTP_USERNAME` | - | SMTP authentication username |
| `APIGATE_SMTP_PASSWORD` | - | SMTP authentication password |
| `APIGATE_SMTP_FROM` | - | From email address |
| `APIGATE_SMTP_FROM_NAME` | (app name) | Sender display name |
| `APIGATE_SMTP_USE_TLS` | `false` | Enable TLS for SMTP connection |

---

## Email Types

APIGate sends these email types:

| Type | Trigger | Description |
|------|---------|-------------|
| **Verification** | User registration | Verify email address |
| **Password Reset** | Forgot password | Reset password link |
| **Welcome** | Account activation | Welcome message |

---

## Configuration via Settings

Email can also be configured via the settings system:

```bash
# Using CLI
apigate settings set email.provider smtp
apigate settings set email.smtp.host smtp.example.com
apigate settings set email.smtp.port 587
apigate settings set email.smtp.username user
apigate settings set email.smtp.password secret --encrypted
apigate settings set email.from_address noreply@example.com
apigate settings set email.from_name "APIGate"
apigate settings set email.smtp.use_tls true
```

---

## Common SMTP Configurations

### Gmail

```bash
APIGATE_SMTP_HOST=smtp.gmail.com
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=your-email@gmail.com
APIGATE_SMTP_PASSWORD=app-password  # Use App Password, not regular password
APIGATE_SMTP_USE_TLS=true
```

### Amazon SES (via SMTP)

```bash
APIGATE_SMTP_HOST=email-smtp.us-east-1.amazonaws.com
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=AKIA...
APIGATE_SMTP_PASSWORD=xxx
APIGATE_SMTP_USE_TLS=true
```

### Mailgun

```bash
APIGATE_SMTP_HOST=smtp.mailgun.org
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=postmaster@your-domain.mailgun.org
APIGATE_SMTP_PASSWORD=xxx
APIGATE_SMTP_USE_TLS=true
```

### Postmark

```bash
APIGATE_SMTP_HOST=smtp.postmarkapp.com
APIGATE_SMTP_PORT=587
APIGATE_SMTP_USERNAME=your-server-token
APIGATE_SMTP_PASSWORD=your-server-token
APIGATE_SMTP_USE_TLS=true
```

---

## Troubleshooting

### Emails Not Sending

1. Verify `APIGATE_EMAIL_PROVIDER` is set to `smtp`
2. Check that SMTP host and credentials are correct
3. Check application logs for email errors
4. Verify SMTP port is not blocked by firewall

### Emails Going to Spam

- Configure SPF, DKIM, and DMARC records for your domain
- Use a reputable SMTP provider
- Ensure From address matches your domain

### Connection Issues

- Verify SMTP host resolves correctly
- Check that outbound port 587 (or 465 for SSL) is allowed
- Try with TLS disabled to isolate connection issues

---

## See Also

- [[Configuration]] - Full configuration reference
