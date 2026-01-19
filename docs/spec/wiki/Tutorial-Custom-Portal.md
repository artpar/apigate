# Tutorial: Custom Portal

Customize the customer portal for your brand.

---

## Prerequisites

- APIGate running
- Access to admin settings

---

## Step 1: Basic Branding

Set your logo and colors:

```bash
# Logo
apigate settings set branding.logo_url "https://example.com/logo.png"
apigate settings set branding.favicon_url "https://example.com/favicon.ico"

# Colors
apigate settings set branding.primary_color "#4F46E5"
apigate settings set branding.secondary_color "#10B981"
```

---

## Step 2: Company Information

```bash
apigate settings set branding.company_name "Acme API"
apigate settings set branding.support_email "support@acme.com"
apigate settings set branding.website_url "https://acme.com"
```

---

## Step 3: Portal Text

Customize portal content:

```bash
# Welcome message
apigate settings set portal.welcome_title "Welcome to Acme API"
apigate settings set portal.welcome_text "The fastest way to integrate with Acme services."

# Documentation intro
apigate settings set docs.intro "Acme API provides RESTful access to our platform."
```

---

## Step 4: Custom CSS

Add custom styling:

```bash
apigate settings set branding.custom_css "
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
```

---

## Step 5: Custom HTML

Add tracking or custom elements:

```bash
# Analytics in head
apigate settings set branding.head_html '<script async src="https://analytics.example.com/script.js"></script>'

# Footer content
apigate settings set branding.footer_html '<p>© 2024 Acme Corp. <a href="/terms">Terms</a> | <a href="/privacy">Privacy</a></p>'
```

---

## Step 6: Custom Domain

Point your domain to APIGate and configure:

```bash
# Enable custom domain
apigate settings set portal.custom_domain "api.acme.com"

# Obtain certificate
apigate certificates obtain --domain api.acme.com
```

---

## Step 7: Test

Visit your portal:
- Main portal: `https://api.acme.com/portal`
- Documentation: `https://api.acme.com/docs`
- Login: `https://api.acme.com/portal/login`

---

## Advanced Customization

### Email Templates

Copy and modify email templates:

```bash
cp -r templates/email custom-templates/email
# Edit templates...
apigate settings set email.template_dir "./custom-templates/email"
```

### Full Theme Override

For complete customization, override the theme directory:

```bash
cp -r web/static/themes/default custom-theme
# Edit templates...
apigate settings set portal.theme_dir "./custom-theme"
```

---

## See Also

- [[Customer-Portal]] - Portal configuration
- [[Branding]] - Branding options
- [[Certificates]] - SSL setup
