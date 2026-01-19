# Tutorial: Custom Portal

Customize the customer portal for your brand.

---

## Prerequisites

- APIGate running
- Access to admin settings

---

## Step 1: Basic Branding

Set your logo and primary color:

```bash
# Logo
apigate settings set custom.logo_url "https://example.com/logo.png"

# Primary brand color (hex)
apigate settings set custom.primary_color "#4F46E5"
```

---

## Step 2: Contact Information

```bash
apigate settings set custom.support_email "support@acme.com"
apigate settings set custom.support_url "https://acme.com/support"
```

---

## Step 3: Portal Content

Customize portal and docs content:

```bash
# Docs hero section
apigate settings set custom.docs_hero_title "Acme API Documentation"
apigate settings set custom.docs_hero_subtitle "The fastest way to integrate with Acme services."

# Portal welcome section (HTML supported)
apigate settings set custom.portal_welcome_html "<h2>Welcome to Acme API</h2><p>Manage your API keys and usage.</p>"
```

---

## Step 4: Custom CSS

Add custom styling for portal and docs pages:

```bash
# Portal CSS
apigate settings set custom.portal_css "
  :root {
    --primary: #4F46E5;
    --radius: 8px;
  }

  .header {
    background: linear-gradient(135deg, #4F46E5, #7C3AED);
  }

  .button {
    border-radius: var(--radius);
    font-weight: 600;
  }
"

# Docs CSS
apigate settings set custom.docs_css "
  .docs-header {
    background: #4F46E5;
  }
"
```

---

## Step 5: Custom Footer

Add custom footer content:

```bash
apigate settings set custom.footer_html '<p>© 2024 Acme Corp. <a href="/terms">Terms</a> | <a href="/privacy">Privacy</a></p>'
```

---

## Step 6: Custom Domain with TLS

Point your domain to APIGate and configure TLS:

```bash
# Enable TLS with ACME (Let's Encrypt)
apigate settings set tls.enabled "true"
apigate settings set tls.mode "acme"
apigate settings set tls.domain "api.acme.com"
apigate settings set tls.acme_email "admin@acme.com"

# Or use manual certificates
apigate settings set tls.mode "manual"
apigate settings set tls.cert_path "/path/to/cert.pem"
apigate settings set tls.key_path "/path/to/key.pem"
```

---

## Step 7: Test

Visit your portal:
- Main portal: `https://api.acme.com/portal`
- Documentation: `https://api.acme.com/docs`
- Login: `https://api.acme.com/portal/login`

---

## Advanced: Full Docs Page Override

For complete control over the docs home page:

```bash
apigate settings set custom.docs_home_html '<!DOCTYPE html>
<html>
<head><title>API Docs</title></head>
<body>
  <h1>Acme API Documentation</h1>
  <!-- Your custom content -->
</body>
</html>'
```

---

## Available Custom Settings

| Setting | Description |
|---------|-------------|
| `custom.logo_url` | Logo URL for header |
| `custom.primary_color` | Primary brand color (hex) |
| `custom.support_email` | Support email address |
| `custom.support_url` | Support/help URL |
| `custom.footer_html` | Custom footer HTML |
| `custom.docs_css` | CSS injected into docs pages |
| `custom.portal_css` | CSS injected into portal pages |
| `custom.docs_hero_title` | Docs hero section title |
| `custom.docs_hero_subtitle` | Docs hero section subtitle |
| `custom.portal_welcome_html` | Portal welcome section HTML |
| `custom.docs_home_html` | Full HTML override for docs home |

---

## See Also

- [[Customer-Portal]] - Portal configuration
- [[Branding]] - Branding options
- [[Certificates]] - SSL setup
