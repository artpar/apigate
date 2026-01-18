package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/core/terminology"
	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/entitlement"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// Portal HTML templates - simple inline templates for the user portal.
// These are separate from the admin templates to keep the portal lightweight.

// renderLandingPage renders the public landing page
func (h *PortalHandler) renderLandingPage() string {
	// Determine button text based on setup status
	adminButtonText := "Get Started"
	adminButtonHref := "/login"
	if h.isSetup != nil && h.isSetup() {
		adminButtonText = "Admin Dashboard"
	}
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #fff; min-height: 100vh; color: #111; }

        .header { padding: 16px 24px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #e5e5e5; }
        .logo { color: #111; font-size: 18px; font-weight: 600; text-decoration: none; letter-spacing: -0.02em; }
        .header-actions { display: flex; gap: 8px; }
        .header-actions a { padding: 8px 16px; border-radius: 4px; text-decoration: none; font-size: 14px; }
        .btn-login { color: #666; }
        .btn-login:hover { color: #111; }
        .btn-signup { background: #111; color: #fff; }
        .btn-signup:hover { background: #333; }

        .hero { max-width: 640px; margin: 120px auto 80px; text-align: center; padding: 0 24px; }
        .hero h1 { font-size: 40px; font-weight: 600; margin-bottom: 16px; line-height: 1.1; letter-spacing: -0.03em; }
        .hero p { color: #666; font-size: 18px; margin-bottom: 32px; line-height: 1.5; }
        .hero-actions { display: flex; gap: 12px; justify-content: center; }
        .hero-actions a { padding: 12px 24px; border-radius: 4px; text-decoration: none; font-size: 15px; }
        .btn-primary { background: #111; color: #fff; }
        .btn-primary:hover { background: #333; }
        .btn-secondary { color: #111; border: 1px solid #ddd; }
        .btn-secondary:hover { border-color: #111; }

        .features { max-width: 800px; margin: 0 auto 80px; padding: 0 24px; }
        .features-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 32px; }
        .feature { text-align: center; }
        .feature h3 { font-size: 15px; font-weight: 500; margin-bottom: 8px; color: #111; }
        .feature p { color: #666; font-size: 14px; line-height: 1.5; }

        .footer { text-align: center; padding: 32px 24px; color: #999; font-size: 13px; border-top: 1px solid #e5e5e5; }

        .seller-section { max-width: 640px; margin: 0 auto 80px; text-align: center; padding: 40px 24px; background: linear-gradient(135deg, #f8f9fa, #e9ecef); border-radius: 12px; }
        .seller-section h3 { font-size: 20px; font-weight: 600; margin-bottom: 12px; color: #111; }
        .seller-section p { color: #666; font-size: 15px; margin-bottom: 20px; line-height: 1.5; }
        .btn-admin { background: linear-gradient(135deg, #4f46e5, #7c3aed); color: #fff; padding: 12px 24px; border-radius: 4px; text-decoration: none; font-size: 15px; display: inline-block; }
        .btn-admin:hover { opacity: 0.9; }

        @media (max-width: 640px) {
            .hero h1 { font-size: 28px; }
            .features-grid { grid-template-columns: 1fr; gap: 24px; }
        }
    </style>
</head>
<body>
    <header class="header">
        <a href="/portal" class="logo">%s</a>
        <div class="header-actions">
            <a href="/portal/login" class="btn-login">Log in</a>
            <a href="/portal/signup" class="btn-signup">Get started</a>
        </div>
    </header>

    <section class="hero">
        <h1>Build with our API</h1>
        <p>Simple, reliable API access. Get your API key and start building in minutes.</p>
        <div class="hero-actions">
            <a href="/portal/signup" class="btn-primary">Get API key</a>
            <a href="/docs" class="btn-secondary">Documentation</a>
        </div>
    </section>

    <section class="features">
        <div class="features-grid">
            <div class="feature">
                <h3>Quick setup</h3>
                <p>Create an account, get your API key, and make your first request in under a minute.</p>
            </div>
            <div class="feature">
                <h3>Usage tracking</h3>
                <p>Monitor your API calls and data usage from your dashboard.</p>
            </div>
            <div class="feature">
                <h3>Flexible plans</h3>
                <p>Start free, upgrade when you need more. Pay only for what you use.</p>
            </div>
        </div>
    </section>

    <section class="seller-section">
        <h3>Are you an API provider?</h3>
        <p>Monetize your API with usage-based billing, rate limiting, and customer management. Self-hosted and open source.</p>
        <a href="%s" class="btn-admin">%s</a>
    </section>

    <footer class="footer">
        <p>%s</p>
    </footer>
</body>
</html>`, h.appName, h.appName, adminButtonHref, adminButtonText, h.appName)
}

func (h *PortalHandler) renderSignupPage(name, email string, errors map[string]string) string {
	return h.renderSignupPageWithPlan(name, email, nil, terminology.Default(), errors)
}

func (h *PortalHandler) renderSignupPageWithPlan(name, email string, defaultPlan *ports.Plan, labels terminology.Labels, errors map[string]string) string {
	errorHTML := ""
	if len(errors) > 0 {
		var msgs []string
		for _, msg := range errors {
			msgs = append(msgs, msg)
		}
		errorHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, strings.Join(msgs, "<br>"))
	}

	// Plan info section
	planInfoHTML := ""
	if defaultPlan != nil {
		priceDisplay := "Free"
		if defaultPlan.PriceMonthly > 0 {
			priceDisplay = fmt.Sprintf("$%.2f/month", float64(defaultPlan.PriceMonthly)/100)
		}
		quotaDisplay := fmt.Sprintf("Unlimited %s", labels.UsageUnitPlural)
		if defaultPlan.RequestsPerMonth > 0 {
			if defaultPlan.RequestsPerMonth >= 1000 {
				quotaDisplay = fmt.Sprintf("%.0fK %s/month", float64(defaultPlan.RequestsPerMonth)/1000, labels.UsageUnitPlural)
			} else {
				quotaDisplay = fmt.Sprintf("%d %s/month", defaultPlan.RequestsPerMonth, labels.UsageUnitPlural)
			}
		}
		planInfoHTML = fmt.Sprintf(`
            <div style="background: #f0f9ff; border: 1px solid #bae6fd; padding: 12px 16px; border-radius: 6px; margin-bottom: 20px;">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <div>
                        <strong style="color: #0369a1;">%s Plan</strong>
                        <span style="color: #0284c7; font-size: 13px; margin-left: 8px;">%s</span>
                    </div>
                    <span style="font-weight: 500; color: #0369a1;">%s</span>
                </div>
            </div>`, defaultPlan.Name, quotaDisplay, priceDisplay)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sign Up - %s</title>
    <style>%s</style>
</head>
<body>
    <div class="auth-container">
        <div class="auth-box">
            <div class="auth-header">
                <h1>%s</h1>
                <p>Create your account</p>
            </div>
            %s
            %s
            <form method="POST" action="/portal/signup" class="auth-form">
                <div class="form-group">
                    <label for="name">Name</label>
                    <input type="text" id="name" name="name" value="%s" required autofocus>
                </div>
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" value="%s" required>
                </div>
                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" name="password" required minlength="8">
                    <small>At least 8 characters with uppercase, lowercase, and number</small>
                </div>
                <div class="form-group" style="margin-top: 16px;">
                    <label style="display: flex; align-items: flex-start; gap: 8px; cursor: pointer; font-weight: normal;">
                        <input type="checkbox" name="agree_tos" required style="margin-top: 3px;">
                        <span style="font-size: 13px; color: #4b5563;">
                            I agree to the <a href="/terms" target="_blank" style="color: #2563eb;">Terms of Service</a>
                            and <a href="/privacy" target="_blank" style="color: #2563eb;">Privacy Policy</a>
                        </span>
                    </label>
                </div>
                <button type="submit" class="btn btn-primary btn-block">Create Account</button>
            </form>
            <div class="auth-footer">
                <p>Already have an account? <a href="/portal/login">Log in</a></p>
            </div>
        </div>
    </div>
    <script>
    (function() {
        var alert = document.querySelector('.alert-error');
        if (alert) {
            document.querySelectorAll('input').forEach(function(input) {
                input.addEventListener('input', function() {
                    alert.style.display = 'none';
                });
            });
        }
    })();
    </script>
</body>
</html>`, h.appName, portalCSS, h.appName, planInfoHTML, errorHTML, name, email)
}

func (h *PortalHandler) renderLoginPage(email, message, messageType string, errors map[string]string) string {
	alertHTML := ""
	if message != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-%s">%s</div>`, messageType, message)
	}
	if len(errors) > 0 {
		var msgs []string
		for _, msg := range errors {
			msgs = append(msgs, msg)
		}
		alertHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, strings.Join(msgs, "<br>"))
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Log In - %s</title>
    <style>%s</style>
</head>
<body>
    <div class="auth-container">
        <div class="auth-box">
            <div class="auth-header">
                <h1>%s</h1>
                <p>Log in to your account</p>
            </div>
            %s
            <form method="POST" action="/portal/login" class="auth-form">
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" value="%s" required autofocus>
                </div>
                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" name="password" required>
                </div>
                <button type="submit" class="btn btn-primary btn-block">Log In</button>
            </form>
            <div class="auth-footer">
                <p><a href="/portal/forgot-password">Forgot your password?</a></p>
                <p>Don't have an account? <a href="/portal/signup">Sign up</a></p>
            </div>
        </div>
    </div>
    <script>
    (function() {
        var alert = document.querySelector('.alert-error');
        if (alert) {
            document.querySelectorAll('input').forEach(function(input) {
                input.addEventListener('input', function() {
                    alert.style.display = 'none';
                });
            });
        }
    })();
    </script>
</body>
</html>`, h.appName, portalCSS, h.appName, alertHTML, email)
}

func (h *PortalHandler) renderForgotPasswordPage(email, message, messageType string) string {
	alertHTML := ""
	if message != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-%s">%s</div>`, messageType, message)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Password - %s</title>
    <style>%s</style>
</head>
<body>
    <div class="auth-container">
        <div class="auth-box">
            <div class="auth-header">
                <h1>%s</h1>
                <p>Reset your password</p>
            </div>
            %s
            <form method="POST" action="/portal/forgot-password" class="auth-form">
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" value="%s" required autofocus>
                </div>
                <button type="submit" class="btn btn-primary btn-block">Send Reset Link</button>
            </form>
            <div class="auth-footer">
                <p><a href="/portal/login">Back to login</a></p>
            </div>
        </div>
    </div>
</body>
</html>`, h.appName, portalCSS, h.appName, alertHTML, email)
}

func (h *PortalHandler) renderResetPasswordPage(token string, errors map[string]string) string {
	errorHTML := ""
	if len(errors) > 0 {
		var msgs []string
		for _, msg := range errors {
			msgs = append(msgs, msg)
		}
		errorHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, strings.Join(msgs, "<br>"))
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Set New Password - %s</title>
    <style>%s</style>
</head>
<body>
    <div class="auth-container">
        <div class="auth-box">
            <div class="auth-header">
                <h1>%s</h1>
                <p>Set your new password</p>
            </div>
            %s
            <form method="POST" action="/portal/reset-password" class="auth-form">
                <input type="hidden" name="token" value="%s">
                <div class="form-group">
                    <label for="password">New Password</label>
                    <input type="password" id="password" name="password" required minlength="8">
                    <small>At least 8 characters with uppercase, lowercase, and number</small>
                </div>
                <div class="form-group">
                    <label for="confirm_password">Confirm Password</label>
                    <input type="password" id="confirm_password" name="confirm_password" required>
                </div>
                <button type="submit" class="btn btn-primary btn-block">Reset Password</button>
            </form>
        </div>
    </div>
</body>
</html>`, h.appName, portalCSS, h.appName, errorHTML, token)
}

func (h *PortalHandler) renderDashboardPage(user *PortalUser, keyCount int, requestCount int64, planName string, requestsPerMonth int64, rateLimitPerMinute int, userEntitlements []entitlement.UserEntitlement, labels terminology.Labels) string {
	// Show getting started section for new users with no API keys
	gettingStartedSection := ""
	if keyCount == 0 {
		gettingStartedSection = `
        <div class="card" style="margin-bottom: 24px;">
            <h2 style="margin: 0 0 16px 0; font-size: 16px; font-weight: 500;">Get started</h2>
            <div style="display: flex; flex-direction: column; gap: 12px; margin-bottom: 20px;">
                <div style="display: flex; align-items: baseline; gap: 12px;">
                    <span style="color: #666; font-size: 14px; min-width: 16px;">1.</span>
                    <span style="font-size: 14px;"><strong>Create an API key</strong> to authenticate your requests</span>
                </div>
                <div style="display: flex; align-items: baseline; gap: 12px;">
                    <span style="color: #666; font-size: 14px; min-width: 16px;">2.</span>
                    <span style="font-size: 14px;"><strong>Read the documentation</strong> to learn the API</span>
                </div>
                <div style="display: flex; align-items: baseline; gap: 12px;">
                    <span style="color: #666; font-size: 14px; min-width: 16px;">3.</span>
                    <span style="font-size: 14px;"><strong>Make your first request</strong></span>
                </div>
            </div>
            <div style="display: flex; gap: 8px;">
                <a href="/portal/api-keys" class="btn btn-primary">Create API key</a>
                <a href="/docs" target="_blank" class="btn btn-secondary">Documentation</a>
            </div>
        </div>`
	}

	// Build entitlements section
	entitlementsSection := ""
	if len(userEntitlements) > 0 {
		entitlementItems := ""
		for _, ue := range userEntitlements {
			displayName := ue.DisplayName
			if displayName == "" {
				displayName = ue.Name
			}
			valueDisplay := ue.Value
			if ue.ValueType == "boolean" {
				if ue.Value == "true" {
					valueDisplay = `<span style="color: #15803d;">Enabled</span>`
				} else {
					valueDisplay = `<span style="color: #b91c1c;">Disabled</span>`
				}
			}
			entitlementItems += fmt.Sprintf(`
				<div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f0f0f0;">
					<span style="color: #333;">%s</span>
					<span style="font-weight: 500;">%s</span>
				</div>`, displayName, valueDisplay)
		}
		entitlementsSection = fmt.Sprintf(`
        <div class="card" style="margin-bottom: 16px;">
            <h2 style="margin: 0 0 16px 0; font-size: 16px; font-weight: 500;">Your Features</h2>
            <div style="font-size: 14px;">
                %s
            </div>
        </div>`, entitlementItems)
	}

	// Calculate quota usage
	quotaSection := ""
	if requestsPerMonth > 0 {
		usagePercent := float64(requestCount) / float64(requestsPerMonth) * 100
		if usagePercent > 100 {
			usagePercent = 100
		}
		progressColor := "#111"
		if usagePercent > 90 {
			progressColor = "#b91c1c"
		}
		quotaSection = fmt.Sprintf(`
        <div class="card" style="margin-bottom: 16px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                <div style="font-size: 14px;">
                    <strong>%s</strong>
                    <span style="color: #666;"> · %d / %d %s</span>
                </div>
                <div style="font-size: 13px; color: #666;">
                    %.0f%% used
                </div>
            </div>
            <div style="background: #e5e5e5; border-radius: 2px; height: 4px; overflow: hidden;">
                <div style="background: %s; height: 100%%; width: %.1f%%;"></div>
            </div>
            <div style="display: flex; justify-content: space-between; margin-top: 8px; font-size: 13px; color: #666;">
                <span>%d %s rate limit</span>
                <span>%d remaining</span>
            </div>
        </div>`, planName, requestCount, requestsPerMonth, labels.UsageUnitPlural, usagePercent, progressColor, usagePercent, rateLimitPerMinute, labels.RateLimitLabel, requestsPerMonth-requestCount)
	} else if planName != "" {
		quotaSection = fmt.Sprintf(`
        <div class="card" style="margin-bottom: 16px;">
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div style="font-size: 14px;">
                    <strong>%s</strong>
                    <span style="color: #666;"> · Unlimited %s</span>
                </div>
                <div style="font-size: 13px; color: #666;">
                    %d %s rate limit
                </div>
            </div>
        </div>`, planName, labels.UsageUnitPlural, rateLimitPerMinute, labels.RateLimitLabel)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Dashboard</h1>
            <p>Welcome back, %s!</p>
        </div>
        %s
        %s
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">API Keys</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">%s This Month</div>
            </div>
        </div>
        %s
        <div class="quick-links">
            <h2>Quick Actions</h2>
            <div class="link-grid">
                <a href="/portal/api-keys" class="link-card">
                    <strong>Manage API Keys</strong>
                    <span>Create and revoke API keys</span>
                </a>
                <a href="/portal/usage" class="link-card">
                    <strong>View Usage</strong>
                    <span>Monitor your API usage</span>
                </a>
                <a href="/docs" class="link-card" target="_blank">
                    <strong>API Documentation</strong>
                    <span>Learn how to use the API</span>
                </a>
                <a href="/portal/settings" class="link-card">
                    <strong>Account Settings</strong>
                    <span>Update your account</span>
                </a>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), user.Name, quotaSection, gettingStartedSection, keyCount, requestCount, labels.QuotaLabel, entitlementsSection)
}

func (h *PortalHandler) renderAPIKeysPage(user *PortalUser, keys []key.Key, revokedMsg bool) string {
	keyRows := h.renderAPIKeysTableRows(keys)

	if keyRows == "" {
		keyRows = `<tr><td colspan="6" class="text-center">No API keys yet</td></tr>`
	}

	successMsg := ""
	if revokedMsg {
		successMsg = `<div class="alert alert-success" style="background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 12px 16px; border-radius: 6px; margin-bottom: 16px;">API key has been revoked successfully.</div>`
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Keys - %s</title>
    <style>%s</style>
    <script src="/static/js/htmx.min.js"></script>
</head>
<body>
    %s
    <main class="main-content">
        %s
        <div class="page-header">
            <h1>API Keys</h1>
            <button class="btn btn-primary" onclick="document.getElementById('create-modal').style.display='block'">Create New Key</button>
        </div>
        <div class="card">
            <table class="table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Key</th>
                        <th>Status</th>
                        <th>Last Used</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody id="api-keys-table" hx-get="/portal/api-keys/partial" hx-trigger="every 30s" hx-swap="innerHTML">
                    %s
                </tbody>
            </table>
        </div>
    </main>

    <!-- Create Key Modal -->
    <div id="create-modal" class="modal-overlay" style="display:none">
        <div class="modal-box">
            <div class="modal-header">
                <h3>Create API Key</h3>
                <button onclick="document.getElementById('create-modal').style.display='none'" class="modal-close">&times;</button>
            </div>
            <form action="/portal/api-keys" method="POST">
                <div class="form-group">
                    <label for="key-name">Key Name (optional)</label>
                    <input type="text" id="key-name" name="name" placeholder="e.g., Production API Key">
                    <small>A friendly name to identify this key</small>
                </div>
                <div class="modal-actions">
                    <button type="button" onclick="document.getElementById('create-modal').style.display='none'" class="btn btn-secondary">Cancel</button>
                    <button type="submit" class="btn btn-primary">Create Key</button>
                </div>
            </form>
        </div>
    </div>
    %s
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), successMsg, keyRows, portalConfirmJS)
}

// renderAPIKeysTableRows renders just the table rows for API keys (used for HTMX partial updates).
func (h *PortalHandler) renderAPIKeysTableRows(keys []key.Key) string {
	if len(keys) == 0 {
		return ""
	}

	var rows string
	for _, k := range keys {
		status := "Active"
		statusClass := "status-active"
		revokeBtn := ""
		if k.RevokedAt != nil {
			status = "Revoked"
			statusClass = "status-revoked"
			revokeBtn = "-"
		} else {
			revokeBtn = fmt.Sprintf(`<form method="POST" action="/portal/api-keys/%s/revoke" style="display:inline" onsubmit="showConfirmModal(this, 'Are you sure you want to revoke this API key? This cannot be undone.', 'Revoke API Key'); return false;"><button type="submit" class="btn btn-sm btn-danger">Revoke</button></form>`, k.ID)
		}

		lastUsed := "Never"
		if k.LastUsed != nil {
			lastUsed = timeAgo(*k.LastUsed)
		}

		rows += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td><code>%s****</code></td>
                <td><span class="%s">%s</span></td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>
        `, k.Name, k.Prefix, statusClass, status, lastUsed, k.CreatedAt.Format("Jan 2, 2006"), revokeBtn)
	}
	return rows
}

// timeAgo returns a human-readable time ago string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func (h *PortalHandler) renderKeyCreatedPage(w http.ResponseWriter, r *http.Request, user *PortalUser, rawKey, keyName string) {
	displayName := keyName
	if displayName == "" {
		displayName = "API Key"
	}

	// Get the server base URL from the request
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Find a real documented endpoint to use as an example
	exampleEndpoint := "/your-endpoint"
	exampleMethod := "GET"
	helpText := `<p style="margin: 12px 0 0 0; color: #6c757d; font-size: 13px;">Replace <code style="background: #e9ecef; padding: 2px 6px; border-radius: 4px;">your-endpoint</code> with the API path you want to call.</p>`

	if h.openAPIService != nil {
		spec := h.openAPIService.GetCustomerSpec(r.Context(), baseURL)
		if spec != nil {
			// Look for a concrete endpoint (not a wildcard) to use as an example
			for path, pathItem := range spec.Paths {
				// Skip wildcard/proxy routes
				if strings.Contains(path, "{path}") || strings.Contains(path, "*") {
					continue
				}

				// Prefer GET endpoints for examples
				if pathItem.Get != nil {
					exampleEndpoint = path
					exampleMethod = "GET"
					break
				}
				// Fall back to POST
				if pathItem.Post != nil {
					exampleEndpoint = path
					exampleMethod = "POST"
				}
			}

			// Update help text based on whether we found a real endpoint
			if exampleEndpoint != "/your-endpoint" {
				helpText = fmt.Sprintf(`<p style="margin: 12px 0 0 0; color: #6c757d; font-size: 13px;">This example uses the <code style="background: #e9ecef; padding: 2px 6px; border-radius: 4px;">%s %s</code> endpoint. <a href="/docs/api-reference">See all available endpoints →</a></p>`, exampleMethod, exampleEndpoint)
			} else {
				helpText = `<p style="margin: 12px 0 0 0; color: #6c757d; font-size: 13px;">Check the <a href="/docs/api-reference">API Reference</a> for available endpoints.</p>`
			}
		}
	}

	// Build the curl example based on method
	curlExample := fmt.Sprintf(`curl -H "X-API-Key: <span style="color: #ce9178;">%s</span>" \`, rawKey)
	if exampleMethod == "POST" {
		curlExample = fmt.Sprintf(`curl -X POST -H "X-API-Key: <span style="color: #ce9178;">%s</span>" \`, rawKey)
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Key Created - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="card">
            <div class="alert alert-success">
                <strong>Success!</strong> Your API key has been created.
            </div>
            <h2>%s</h2>
            <p>Copy your API key now. You won't be able to see it again!</p>
            <div class="key-display">
                <code id="api-key">%s</code>
            </div>
            <button class="btn btn-primary" onclick="navigator.clipboard.writeText(document.getElementById('api-key').textContent)">
                Copy to Clipboard
            </button>
            <p class="key-warning">
                ⚠️ Store this key securely. It provides access to your API and cannot be recovered if lost.
            </p>

            <div style="margin-top: 24px; padding: 16px; background: #f8f9fa; border-radius: 8px; border: 1px solid #e9ecef;">
                <h3 style="margin: 0 0 12px 0; font-size: 16px;">How to Use Your API Key</h3>
                <p style="margin: 0 0 12px 0; color: #6c757d; font-size: 14px;">Include your API key in the <code style="background: #e9ecef; padding: 2px 6px; border-radius: 4px;">X-API-Key</code> header with every request:</p>
                <div style="background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 6px; font-family: monospace; font-size: 13px; overflow-x: auto;">
                    <div style="color: #6a9955;">## Example request with curl</div>
                    <div>%s</div>
                    <div style="padding-left: 20px;">%s%s</div>
                </div>
                %s
            </div>

            <div style="margin-top: 20px;">
                <a href="/portal/api-keys" class="btn btn-secondary">Back to API Keys</a>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), displayName, rawKey, curlExample, baseURL, exampleEndpoint, helpText)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *PortalHandler) renderUsagePage(user *PortalUser, summary usage.Summary, labels terminology.Labels) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Usage - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Usage</h1>
            <p>Current billing period</p>
        </div>
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Total %s</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%d</div>
                <div class="stat-label">Errors</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%.2f KB</div>
                <div class="stat-label">Data In</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">%.2f KB</div>
                <div class="stat-label">Data Out</div>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), summary.RequestCount, labels.QuotaLabel, summary.ErrorCount, float64(summary.BytesIn)/1024, float64(summary.BytesOut)/1024)
}

func (h *PortalHandler) renderAccountSettingsPage(user *PortalUser, errors map[string]string, success string) string {
	errorHTML := ""
	if len(errors) > 0 {
		var msgs []string
		for field, msg := range errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
		}
		errorHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, strings.Join(msgs, "<br>"))
	}
	successHTML := ""
	if success != "" {
		successHTML = fmt.Sprintf(`<div class="alert alert-success">%s</div>`, success)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Settings - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Account Settings</h1>
        </div>
        %s
        %s
        <div class="card">
            <h2>Profile</h2>
            <form method="POST" action="/portal/settings">
                <div class="form-group">
                    <label for="name">Name</label>
                    <input type="text" id="name" name="name" value="%s" required minlength="2" maxlength="100">
                </div>
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" value="%s" disabled>
                    <small>Contact support to change your email</small>
                </div>
                <button type="submit" class="btn btn-primary">Save Changes</button>
            </form>
        </div>

        <div class="card">
            <h2>Change Password</h2>
            <form method="POST" action="/portal/settings/password">
                <div class="form-group">
                    <label for="current_password">Current Password</label>
                    <input type="password" id="current_password" name="current_password" required>
                </div>
                <div class="form-group">
                    <label for="new_password">New Password</label>
                    <input type="password" id="new_password" name="new_password" required minlength="8">
                </div>
                <div class="form-group">
                    <label for="confirm_password">Confirm New Password</label>
                    <input type="password" id="confirm_password" name="confirm_password" required>
                </div>
                <button type="submit" class="btn btn-primary">Change Password</button>
            </form>
        </div>

        <div class="card card-danger">
            <h2>Danger Zone</h2>
            <p>Closing your account will revoke all API keys and delete your data.</p>
            <form method="POST" action="/portal/settings/close-account" onsubmit="showConfirmModal(this, 'Are you sure you want to close your account? This will revoke all API keys and delete your data. This cannot be undone.', 'Close Account'); return false;">
                <div class="form-group">
                    <label for="password">Confirm with your password</label>
                    <input type="password" id="password" name="password" required>
                </div>
                <button type="submit" class="btn btn-danger">Close Account</button>
            </form>
        </div>
    </main>
    %s
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), successHTML, errorHTML, user.Name, user.Email, portalConfirmJS)
}

func (h *PortalHandler) renderErrorPage(message string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error - %s</title>
    <style>%s</style>
</head>
<body>
    <div class="auth-container">
        <div class="auth-box">
            <div class="auth-header">
                <h1>%s</h1>
            </div>
            <div class="alert alert-error">%s</div>
            <div class="auth-footer">
                <p><a href="/portal/login">Back to login</a></p>
            </div>
        </div>
    </div>
</body>
</html>`, h.appName, portalCSS, h.appName, message)
}

func (h *PortalHandler) renderPortalNav(user *PortalUser) string {
	return fmt.Sprintf(`
    <nav class="portal-nav">
        <div class="nav-brand">
            <a href="/portal/dashboard">%s</a>
            <span class="role-badge" style="background: linear-gradient(135deg, #059669, #10b981); color: white; padding: 2px 8px; font-size: 0.65rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 4px; margin-left: 8px;">Developer Portal</span>
        </div>
        <div class="nav-links">
            <a href="/portal/dashboard">Dashboard</a>
            <a href="/portal/api-keys">API Keys</a>
            <a href="/portal/usage">Usage</a>
            <a href="/portal/plans">Plans</a>
            <a href="/portal/webhooks">Webhooks</a>
            <a href="/docs" target="_blank">Docs</a>
            <a href="/portal/settings">Settings</a>
        </div>
        <div class="nav-user">
            <span>%s</span>
            <form method="POST" action="/portal/logout" style="display:inline">
                <button type="submit" class="btn btn-sm">Logout</button>
            </form>
        </div>
    </nav>
`, h.appName, user.Email)
}

func (h *PortalHandler) renderPlansPage(user *PortalUser, plans []ports.Plan, currentPlan *ports.Plan, success, errorMsg string, hasStripeSubscription bool, labels terminology.Labels) string {
	alertHTML := ""
	if success != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-success">%s</div>`, success)
	}
	if errorMsg != "" {
		alertHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, errorMsg)
	}

	currentPlanID := ""
	if currentPlan != nil {
		currentPlanID = currentPlan.ID
	}

	planCards := ""
	for _, p := range plans {
		isCurrent := p.ID == currentPlanID

		// Format price
		priceDisplay := "Free"
		if p.PriceMonthly > 0 {
			priceDisplay = fmt.Sprintf("$%.2f/mo", float64(p.PriceMonthly)/100)
		}

		// Format quota
		quotaDisplay := fmt.Sprintf("Unlimited %s", labels.UsageUnitPlural)
		if p.RequestsPerMonth > 0 {
			if p.RequestsPerMonth >= 1000000 {
				quotaDisplay = fmt.Sprintf("%.1fM %s/mo", float64(p.RequestsPerMonth)/1000000, labels.UsageUnitPlural)
			} else if p.RequestsPerMonth >= 1000 {
				quotaDisplay = fmt.Sprintf("%.0fK %s/mo", float64(p.RequestsPerMonth)/1000, labels.UsageUnitPlural)
			} else {
				quotaDisplay = fmt.Sprintf("%d %s/mo", p.RequestsPerMonth, labels.UsageUnitPlural)
			}
		}

		// Format rate limit
		rateDisplay := fmt.Sprintf("%d %s", p.RateLimitPerMinute, labels.RateLimitLabel)

		// Format overage
		overageDisplay := fmt.Sprintf("%s blocked at limit", labels.UsageUnitPlural)
		if p.OveragePrice > 0 {
			overageDisplay = fmt.Sprintf("$%.4f per extra %s", float64(p.OveragePrice)/10000, labels.UsageUnit)
		}

		// Current plan badge
		currentBadge := ""
		if isCurrent {
			currentBadge = `<span style="display: inline-block; background: #111; color: white; padding: 3px 8px; border-radius: 3px; font-size: 11px; font-weight: 500; margin-left: 8px;">Current</span>`
		}

		// Trial badge
		trialBadge := ""
		if p.TrialDays > 0 {
			trialBadge = fmt.Sprintf(`<div style="color: #666; font-size: 13px; margin-top: 4px;">%d-day trial</div>`, p.TrialDays)
		}

		// Action button
		actionBtn := ""
		if isCurrent {
			actionBtn = `<button class="btn" disabled style="background: #e5e7eb; color: #9ca3af; cursor: not-allowed;">Current Plan</button>`
		} else if p.PriceMonthly > 0 {
			// Paid plan - show upgrade button with trial info
			buttonText := "Upgrade"
			if p.TrialDays > 0 {
				buttonText = fmt.Sprintf("Start %d-Day Trial", p.TrialDays)
			}
			actionBtn = fmt.Sprintf(`
				<form method="POST" action="/portal/plans/change" onsubmit="showConfirmModal(this, 'Upgrade to the %s plan?', 'Confirm Plan Change'); return false;">
					<input type="hidden" name="plan_id" value="%s">
					<button type="submit" class="btn btn-primary">%s</button>
				</form>`, p.Name, p.ID, buttonText)
		} else {
			// Free plan - show downgrade button
			actionBtn = fmt.Sprintf(`
				<form method="POST" action="/portal/plans/change" onsubmit="showConfirmModal(this, 'Switch to the %s plan? You will lose access to higher limits.', 'Confirm Plan Change'); return false;">
					<input type="hidden" name="plan_id" value="%s">
					<button type="submit" class="btn btn-secondary">Switch Plan</button>
				</form>`, p.Name, p.ID)
		}

		// Card highlight for current plan
		cardStyle := "border: 1px solid #e5e5e5;"
		if isCurrent {
			cardStyle = "border: 1px solid #111;"
		}

		planCards += fmt.Sprintf(`
			<div class="plan-card" style="background: white; padding: 20px; border-radius: 6px; %s">
				<div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px;">
					<div>
						<h3 style="margin: 0; font-size: 16px; font-weight: 500;">%s%s</h3>
						<p style="margin: 4px 0 0 0; color: #666; font-size: 13px;">%s</p>
					</div>
					<div style="text-align: right;">
						<div style="font-size: 20px; font-weight: 600; color: #111;">%s</div>
						%s
					</div>
				</div>
				<div style="border-top: 1px solid #e5e5e5; padding-top: 16px; margin-bottom: 16px;">
					<div style="display: grid; gap: 6px; font-size: 14px;">
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #666;">-</span>
							<span>%s</span>
						</div>
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #666;">-</span>
							<span>%s rate limit</span>
						</div>
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #666;">-</span>
							<span style="color: #666; font-size: 13px;">%s</span>
						</div>
					</div>
				</div>
				<div>%s</div>
			</div>
		`, cardStyle, p.Name, currentBadge, p.Description, priceDisplay, trialBadge, quotaDisplay, rateDisplay, overageDisplay, actionBtn)
	}

	if planCards == "" {
		planCards = `<div class="card" style="text-align: center; padding: 40px;"><p style="color: #6b7280;">No plans available at this time.</p></div>`
	}

	// Subscription management section for paid subscribers
	subscriptionSection := ""
	if hasStripeSubscription {
		subscriptionSection = `
        <div style="margin-top: 32px; padding: 24px; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
            <h3 style="margin: 0 0 16px 0; font-size: 18px;">Subscription Management</h3>
            <p style="margin: 0 0 16px 0; color: #6b7280;">Manage your billing, payment methods, and invoices through the customer portal.</p>
            <div style="display: flex; gap: 12px; flex-wrap: wrap;">
                <a href="/portal/subscription/manage" class="btn btn-primary" style="text-decoration: none;">Manage Billing</a>
                <form method="POST" action="/portal/subscription/cancel" onsubmit="showConfirmModal(this, 'Are you sure you want to cancel your subscription? You will lose access to your current plan features at the end of the billing period.', 'Cancel Subscription'); return false;">
                    <button type="submit" class="btn btn-secondary" style="background: #fef2f2; color: #dc2626; border: 1px solid #fecaca;">Cancel Subscription</button>
                </form>
            </div>
        </div>`
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Plans - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Plans & Pricing</h1>
            <p>Choose the plan that fits your needs</p>
        </div>
        %s
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 24px;">
            %s
        </div>
        %s
        <div style="margin-top: 24px; padding: 16px; background: #f3f4f6; border-radius: 8px;">
            <p style="margin: 0; color: #6b7280; font-size: 14px;">
                Need a custom plan with higher limits? <a href="mailto:support@example.com" style="color: #3b82f6;">Contact us</a>
            </p>
        </div>
    </main>
    %s
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), alertHTML, planCards, subscriptionSection, portalConfirmJS)
}

func (h *PortalHandler) renderBillingPage(user *PortalUser, subscription *billing.Subscription, plan *ports.Plan, invoices []billing.Invoice, successMsg, errorMsg string) string {
	// Alert messages
	alertHTML := ""
	if successMsg != "" {
		alertHTML = fmt.Sprintf(`
			<div style="background: #dcfce7; border: 1px solid #86efac; color: #166534; padding: 16px; border-radius: 8px; margin-bottom: 24px; display: flex; align-items: center; gap: 12px;">
				<svg style="width: 20px; height: 20px; flex-shrink: 0;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
				</svg>
				<span>%s</span>
			</div>`, successMsg)
	}
	if errorMsg != "" {
		alertHTML = fmt.Sprintf(`
			<div style="background: #fee2e2; border: 1px solid #fecaca; color: #b91c1c; padding: 16px; border-radius: 8px; margin-bottom: 24px; display: flex; align-items: center; gap: 12px;">
				<svg style="width: 20px; height: 20px; flex-shrink: 0;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
				</svg>
				<span>%s</span>
			</div>`, errorMsg)
	}

	// Subscription status section
	subscriptionHTML := ""
	if subscription != nil {
		statusBadge := ""
		switch subscription.Status {
		case billing.SubscriptionStatusActive:
			statusBadge = `<span style="background: #dcfce7; color: #15803d; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">Active</span>`
		case billing.SubscriptionStatusTrialing:
			statusBadge = `<span style="background: #dbeafe; color: #1d4ed8; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">Trial</span>`
		case billing.SubscriptionStatusPastDue:
			statusBadge = `<span style="background: #fef3c7; color: #b45309; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">Past Due</span>`
		case billing.SubscriptionStatusCancelled:
			statusBadge = `<span style="background: #fee2e2; color: #b91c1c; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">Cancelled</span>`
		default:
			statusBadge = fmt.Sprintf(`<span style="background: #f3f4f6; color: #6b7280; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">%s</span>`, subscription.Status)
		}

		cancelNotice := ""
		if subscription.CancelAtPeriodEnd {
			cancelNotice = fmt.Sprintf(`
				<div style="margin-top: 16px; padding: 12px; background: #fef3c7; border-radius: 8px; display: flex; align-items: center; gap: 8px;">
					<span style="color: #b45309;">&#9888;</span>
					<span style="color: #92400e;">Your subscription will end on %s</span>
				</div>`, subscription.CurrentPeriodEnd.Format("January 2, 2006"))
		}

		planName := "Unknown Plan"
		if plan != nil {
			planName = plan.Name
		}

		subscriptionHTML = fmt.Sprintf(`
			<div class="card" style="margin-bottom: 24px;">
				<div style="padding: 24px; border-bottom: 1px solid #e5e7eb;">
					<div style="display: flex; justify-content: space-between; align-items: center;">
						<h2 style="margin: 0; font-size: 18px;">Current Subscription</h2>
						%s
					</div>
				</div>
				<div style="padding: 24px;">
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 24px;">
						<div>
							<div style="color: #6b7280; font-size: 14px; margin-bottom: 4px;">Plan</div>
							<div style="font-size: 18px; font-weight: 600;">%s</div>
						</div>
						<div>
							<div style="color: #6b7280; font-size: 14px; margin-bottom: 4px;">Current Period</div>
							<div>%s - %s</div>
						</div>
						<div>
							<div style="color: #6b7280; font-size: 14px; margin-bottom: 4px;">Next Billing Date</div>
							<div>%s</div>
						</div>
					</div>
					%s
					<div style="margin-top: 24px; display: flex; gap: 12px; flex-wrap: wrap;">
						<a href="/portal/plans" class="btn btn-secondary" style="text-decoration: none;">Change Plan</a>
						<a href="/portal/subscription/manage" class="btn btn-primary" style="text-decoration: none;">Manage Payment</a>
						%s
					</div>
				</div>
			</div>`,
			statusBadge,
			planName,
			subscription.CurrentPeriodStart.Format("Jan 2, 2006"),
			subscription.CurrentPeriodEnd.Format("Jan 2, 2006"),
			subscription.CurrentPeriodEnd.Format("January 2, 2006"),
			cancelNotice,
			func() string {
				if subscription.Status == billing.SubscriptionStatusActive || subscription.Status == billing.SubscriptionStatusTrialing {
					if subscription.CancelAtPeriodEnd {
						return ""
					}
					return `<a href="/portal/subscription/cancel" style="padding: 8px 16px; border: 1px solid #dc2626; border-radius: 6px; color: #dc2626; text-decoration: none; font-size: 14px;">Cancel Subscription</a>`
				}
				return ""
			}(),
		)
	} else if plan != nil {
		// User has a plan but no subscription record (likely free plan or local-only)
		subscriptionHTML = fmt.Sprintf(`
			<div class="card" style="margin-bottom: 24px;">
				<div style="padding: 24px; border-bottom: 1px solid #e5e7eb;">
					<div style="display: flex; justify-content: space-between; align-items: center;">
						<h2 style="margin: 0; font-size: 18px;">Current Plan</h2>
						<span style="background: #dcfce7; color: #15803d; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 500;">Active</span>
					</div>
				</div>
				<div style="padding: 24px;">
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 24px;">
						<div>
							<div style="color: #6b7280; font-size: 14px; margin-bottom: 4px;">Plan</div>
							<div style="font-size: 18px; font-weight: 600;">%s</div>
						</div>
						<div>
							<div style="color: #6b7280; font-size: 14px; margin-bottom: 4px;">Price</div>
							<div>%s/month</div>
						</div>
					</div>
					<div style="margin-top: 24px;">
						<a href="/portal/plans" class="btn btn-primary" style="text-decoration: none;">Upgrade Plan</a>
					</div>
				</div>
			</div>`, plan.Name, billing.FormatAmount(plan.PriceMonthly))
	} else {
		subscriptionHTML = `
			<div class="card" style="margin-bottom: 24px;">
				<div style="padding: 24px; text-align: center;">
					<p style="color: #6b7280; margin: 0 0 16px 0;">No active subscription found.</p>
					<a href="/portal/plans" class="btn btn-primary" style="text-decoration: none;">View Plans</a>
				</div>
			</div>`
	}

	// Invoices table
	invoicesHTML := ""
	if len(invoices) > 0 {
		invoiceRows := ""
		for _, inv := range invoices {
			statusBadge := ""
			switch inv.Status {
			case billing.InvoiceStatusPaid:
				statusBadge = `<span style="background: #dcfce7; color: #15803d; padding: 2px 8px; border-radius: 4px; font-size: 12px;">Paid</span>`
			case billing.InvoiceStatusOpen:
				statusBadge = `<span style="background: #dbeafe; color: #1d4ed8; padding: 2px 8px; border-radius: 4px; font-size: 12px;">Open</span>`
			case billing.InvoiceStatusDraft:
				statusBadge = `<span style="background: #f3f4f6; color: #6b7280; padding: 2px 8px; border-radius: 4px; font-size: 12px;">Draft</span>`
			case billing.InvoiceStatusVoid:
				statusBadge = `<span style="background: #fee2e2; color: #b91c1c; padding: 2px 8px; border-radius: 4px; font-size: 12px;">Void</span>`
			default:
				statusBadge = fmt.Sprintf(`<span style="background: #f3f4f6; color: #6b7280; padding: 2px 8px; border-radius: 4px; font-size: 12px;">%s</span>`, inv.Status)
			}

			downloadLink := ""
			if inv.InvoiceURL != "" {
				downloadLink = fmt.Sprintf(`<a href="%s" target="_blank" style="color: #3b82f6; text-decoration: none; font-size: 14px;">Download</a>`, inv.InvoiceURL)
			}

			invoiceRows += fmt.Sprintf(`
				<tr>
					<td style="padding: 12px 16px; border-bottom: 1px solid #e5e7eb;">%s</td>
					<td style="padding: 12px 16px; border-bottom: 1px solid #e5e7eb;">%s - %s</td>
					<td style="padding: 12px 16px; border-bottom: 1px solid #e5e7eb;">%s</td>
					<td style="padding: 12px 16px; border-bottom: 1px solid #e5e7eb;">%s</td>
					<td style="padding: 12px 16px; border-bottom: 1px solid #e5e7eb;">%s</td>
				</tr>`,
				inv.CreatedAt.Format("Jan 2, 2006"),
				inv.PeriodStart.Format("Jan 2"),
				inv.PeriodEnd.Format("Jan 2, 2006"),
				billing.FormatAmount(inv.Total),
				statusBadge,
				downloadLink,
			)
		}

		invoicesHTML = fmt.Sprintf(`
			<div class="card">
				<div style="padding: 24px; border-bottom: 1px solid #e5e7eb;">
					<h2 style="margin: 0; font-size: 18px;">Billing History</h2>
				</div>
				<div style="overflow-x: auto;">
					<table style="width: 100%%; border-collapse: collapse;">
						<thead>
							<tr style="background: #f9fafb;">
								<th style="padding: 12px 16px; text-align: left; font-weight: 500; color: #6b7280; font-size: 14px;">Date</th>
								<th style="padding: 12px 16px; text-align: left; font-weight: 500; color: #6b7280; font-size: 14px;">Period</th>
								<th style="padding: 12px 16px; text-align: left; font-weight: 500; color: #6b7280; font-size: 14px;">Amount</th>
								<th style="padding: 12px 16px; text-align: left; font-weight: 500; color: #6b7280; font-size: 14px;">Status</th>
								<th style="padding: 12px 16px; text-align: left; font-weight: 500; color: #6b7280; font-size: 14px;"></th>
							</tr>
						</thead>
						<tbody>
							%s
						</tbody>
					</table>
				</div>
			</div>`, invoiceRows)
	} else {
		invoicesHTML = `
			<div class="card">
				<div style="padding: 24px; border-bottom: 1px solid #e5e7eb;">
					<h2 style="margin: 0; font-size: 18px;">Billing History</h2>
				</div>
				<div style="padding: 40px; text-align: center;">
					<p style="color: #6b7280; margin: 0;">No invoices yet.</p>
				</div>
			</div>`
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Billing - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Billing</h1>
            <p>Manage your subscription and view invoices</p>
        </div>
        %s
        %s
        %s
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), alertHTML, subscriptionHTML, invoicesHTML)
}

// renderCancelSubscriptionPage renders the cancel subscription confirmation page
func (h *PortalHandler) renderCancelSubscriptionPage(user *PortalUser, subscription *billing.Subscription, plan *ports.Plan) string {
	periodEndDate := subscription.CurrentPeriodEnd.Format("January 2, 2006")
	planName := "Current Plan"
	if plan != nil {
		planName = plan.Name
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cancel Subscription - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Cancel Subscription</h1>
            <p>We're sorry to see you go</p>
        </div>

        <div class="card" style="max-width: 600px; margin: 0 auto;">
            <div style="padding: 32px;">
                <div style="text-align: center; margin-bottom: 32px;">
                    <div style="width: 64px; height: 64px; background: #fef2f2; border-radius: 50%%; margin: 0 auto 16px; display: flex; align-items: center; justify-content: center;">
                        <svg style="width: 32px; height: 32px; color: #dc2626;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                        </svg>
                    </div>
                    <h2 style="margin: 0 0 8px; font-size: 20px;">Cancel %s?</h2>
                    <p style="color: #6b7280; margin: 0;">Please review what happens when you cancel.</p>
                </div>

                <div style="background: #f9fafb; border-radius: 8px; padding: 20px; margin-bottom: 24px;">
                    <h3 style="margin: 0 0 16px; font-size: 16px; color: #111827;">What happens when you cancel</h3>
                    <ul style="margin: 0; padding-left: 20px; color: #4b5563; line-height: 1.8;">
                        <li>You'll lose access to premium features</li>
                        <li>Your API quota will be reduced to the free plan limits</li>
                        <li>Any unused quota will not be refunded</li>
                        <li>Your API keys will continue to work (with free plan limits)</li>
                        <li>You can resubscribe anytime</li>
                    </ul>
                </div>

                <form method="POST" action="/portal/subscription/cancel">
                    <div style="background: #fff; border: 1px solid #e5e7eb; border-radius: 8px; margin-bottom: 24px;">
                        <label style="display: flex; align-items: start; padding: 16px; cursor: pointer; border-bottom: 1px solid #e5e7eb;">
                            <input type="radio" name="cancel_mode" value="end_of_period" checked style="margin-right: 12px; margin-top: 4px;">
                            <div>
                                <div style="font-weight: 500; color: #111827;">Cancel at end of billing period</div>
                                <div style="color: #6b7280; font-size: 14px; margin-top: 4px;">
                                    Keep access until <strong>%s</strong>, then downgrade to free plan
                                </div>
                            </div>
                        </label>
                        <label style="display: flex; align-items: start; padding: 16px; cursor: pointer;">
                            <input type="radio" name="cancel_mode" value="immediately" style="margin-right: 12px; margin-top: 4px;">
                            <div>
                                <div style="font-weight: 500; color: #111827;">Cancel immediately</div>
                                <div style="color: #6b7280; font-size: 14px; margin-top: 4px;">
                                    Lose access right now, no prorated refund
                                </div>
                            </div>
                        </label>
                    </div>

                    <div style="display: flex; gap: 12px; justify-content: flex-end;">
                        <a href="/portal/billing" style="padding: 10px 20px; border: 1px solid #e5e7eb; border-radius: 6px; color: #374151; text-decoration: none; font-weight: 500;">
                            Keep Subscription
                        </a>
                        <button type="submit" style="padding: 10px 20px; background: #dc2626; color: white; border: none; border-radius: 6px; font-weight: 500; cursor: pointer;">
                            Cancel Subscription
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), planName, periodEndDate)
}

// Portal CSS styles
const portalCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #fafafa; color: #111; line-height: 1.5; }

.auth-container { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px; background: #fff; }
.auth-box { width: 100%; max-width: 360px; }
.auth-header { margin-bottom: 32px; }
.auth-header h1 { font-size: 18px; font-weight: 600; margin-bottom: 4px; letter-spacing: -0.02em; }
.auth-header p { color: #666; font-size: 14px; }
.auth-form { margin-bottom: 24px; }
.auth-footer { text-align: center; }
.auth-footer p { margin: 8px 0; color: #666; font-size: 14px; }
.auth-footer a { color: #111; text-decoration: underline; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.form-group input { width: 100%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
.form-group input:focus { border-color: #111; outline: none; }
.form-group small { display: block; margin-top: 4px; color: #888; font-size: 12px; }

.btn { display: inline-block; padding: 10px 16px; border: none; border-radius: 4px; font-size: 14px; cursor: pointer; text-decoration: none; font-weight: 500; }
.btn-block { width: 100%; }
.btn-primary { background: #111; color: #fff; }
.btn-primary:hover { background: #333; }
.btn-secondary { background: #fff; color: #111; border: 1px solid #ddd; }
.btn-secondary:hover { border-color: #111; }
.btn-danger { background: #fff; color: #b91c1c; border: 1px solid #fca5a5; }
.btn-danger:hover { background: #fef2f2; }
.btn-sm { padding: 6px 12px; font-size: 13px; }

.alert { padding: 12px 16px; border-radius: 4px; margin-bottom: 16px; font-size: 14px; }
.alert-success { background: #f0fdf4; color: #166534; border: 1px solid #bbf7d0; }
.alert-error { background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; }
.alert-warning { background: #fffbeb; color: #92400e; border: 1px solid #fed7aa; }
.alert-info { background: #f0f9ff; color: #075985; border: 1px solid #bae6fd; }

.portal-nav { background: #fff; padding: 12px 24px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #e5e5e5; }
.nav-brand a { font-size: 16px; font-weight: 600; color: #111; text-decoration: none; letter-spacing: -0.02em; }
.nav-links { display: flex; gap: 24px; }
.nav-links a { color: #666; text-decoration: none; font-size: 14px; }
.nav-links a:hover { color: #111; }
.nav-user { display: flex; align-items: center; gap: 12px; }
.nav-user span { color: #666; font-size: 13px; }

.main-content { max-width: 960px; margin: 0 auto; padding: 32px 24px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.page-header h1 { font-size: 20px; font-weight: 600; letter-spacing: -0.02em; }
.page-header p { color: #666; font-size: 14px; }

.card { background: #fff; padding: 24px; border-radius: 6px; border: 1px solid #e5e5e5; margin-bottom: 16px; }
.card h2 { font-size: 16px; font-weight: 500; margin-bottom: 16px; }
.card-danger { border-color: #fca5a5; }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; margin-bottom: 24px; }
.stat-card { background: #fff; padding: 20px; border-radius: 6px; border: 1px solid #e5e5e5; }
.stat-value { font-size: 28px; font-weight: 600; color: #111; letter-spacing: -0.02em; }
.stat-label { color: #666; margin-top: 4px; font-size: 13px; }

.quick-links h2 { margin-bottom: 12px; font-size: 16px; font-weight: 500; }
.link-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }
.link-card { display: block; padding: 16px; background: #fff; border-radius: 6px; border: 1px solid #e5e5e5; text-decoration: none; color: #111; }
.link-card:hover { border-color: #111; }
.link-card strong { display: block; margin-bottom: 4px; font-size: 14px; }
.link-card span { color: #666; font-size: 13px; }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #e5e5e5; font-size: 14px; }
.table th { font-weight: 500; color: #666; font-size: 13px; }
.text-center { text-align: center; }

.status-active { color: #166534; }
.status-revoked { color: #991b1b; }

code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-family: ui-monospace, monospace; font-size: 13px; }

.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-box { background: #fff; padding: 24px; border-radius: 6px; width: 100%; max-width: 400px; }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.modal-header h3 { font-size: 16px; font-weight: 500; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #666; }
.modal-close:hover { color: #111; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }

.key-display { background: #f5f5f5; border: 1px solid #e5e5e5; padding: 12px; border-radius: 4px; margin: 12px 0; }
.key-display code { background: none; padding: 0; font-size: 13px; word-break: break-all; }
.key-warning { color: #92400e; font-size: 13px; margin-top: 8px; }

.confirm-modal-message { color: #333; font-size: 14px; margin-bottom: 20px; line-height: 1.6; }
.confirm-modal-actions { display: flex; gap: 8px; justify-content: flex-end; }
`

// portalConfirmJS provides custom confirm dialog functionality to replace native browser confirms
const portalConfirmJS = `
<div id="confirm-modal" class="modal-overlay" style="display:none">
    <div class="modal-box">
        <div class="modal-header">
            <h3 id="confirm-modal-title">Confirm</h3>
            <button onclick="closeConfirmModal()" class="modal-close">&times;</button>
        </div>
        <p id="confirm-modal-message" class="confirm-modal-message"></p>
        <div class="confirm-modal-actions">
            <button type="button" onclick="closeConfirmModal()" class="btn btn-secondary">Cancel</button>
            <button type="button" id="confirm-modal-ok" class="btn btn-danger">Confirm</button>
        </div>
    </div>
</div>
<script>
var pendingConfirmForm = null;
function showConfirmModal(form, message, title) {
    pendingConfirmForm = form;
    document.getElementById('confirm-modal-title').textContent = title || 'Confirm';
    document.getElementById('confirm-modal-message').textContent = message;
    document.getElementById('confirm-modal').style.display = 'flex';
}
function closeConfirmModal() {
    document.getElementById('confirm-modal').style.display = 'none';
    pendingConfirmForm = null;
}
function confirmAndSubmit() {
    if (pendingConfirmForm) {
        pendingConfirmForm.submit();
    }
    closeConfirmModal();
}
document.getElementById('confirm-modal-ok').onclick = confirmAndSubmit;
document.getElementById('confirm-modal').onclick = function(e) {
    if (e.target === this) closeConfirmModal();
};
</script>
`
