# Branding

Customize the look and feel of APIGate's customer-facing pages.

---

## Overview

Branding customizations apply to:
- Customer portal
- API documentation pages
- Public pages (login, signup)

All branding settings use the `custom.*` namespace in the settings system.

---

## Available Customization Settings

| Setting Key | Description |
|-------------|-------------|
| `custom.logo_url` | Custom logo URL |
| `custom.primary_color` | Primary brand color (hex, e.g., `#4F46E5`) |
| `custom.support_email` | Support email shown in docs/portal |
| `custom.support_url` | Support URL/documentation link |
| `custom.footer_html` | Custom footer HTML |
| `custom.docs_home_html` | Full HTML override for docs home page |
| `custom.docs_css` | Custom CSS injected into all docs pages |
| `custom.docs_hero_title` | Custom docs hero title |
| `custom.docs_hero_subtitle` | Custom docs hero subtitle |
| `custom.portal_welcome_html` | Custom welcome section HTML for portal |
| `custom.portal_css` | Custom CSS injected into all portal pages |

---

## Setting Branding Options

### Via CLI

```bash
# Logo
apigate settings set custom.logo_url "https://example.com/logo.png"

# Colors
apigate settings set custom.primary_color "#4F46E5"

# Support contact
apigate settings set custom.support_email "support@acme.com"
apigate settings set custom.support_url "https://docs.acme.com"
```

### Via Web UI

Navigate to **Settings** in the admin dashboard to configure branding options through the UI.

---

## Custom CSS

Inject custom CSS into docs or portal pages:

```bash
# For documentation pages
apigate settings set custom.docs_css ".header { background: linear-gradient(...); }"

# For portal pages
apigate settings set custom.portal_css ".button { border-radius: 8px; }"
```

---

## Custom HTML

Add custom HTML to pages:

```bash
# Custom footer
apigate settings set custom.footer_html "<p>© 2024 Acme Corp</p>"

# Custom docs welcome section
apigate settings set custom.docs_home_html "<div class='welcome'>...</div>"

# Custom portal welcome
apigate settings set custom.portal_welcome_html "<div class='intro'>...</div>"
```

---

## See Also

- [[Configuration]] - All settings
- [[Email-Configuration]] - Email provider setup
