package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
)

// Portal HTML templates - simple inline templates for the user portal.
// These are separate from the admin templates to keep the portal lightweight.

func (h *PortalHandler) renderSignupPage(name, email string, errors map[string]string) string {
	return h.renderSignupPageWithPlan(name, email, nil, errors)
}

func (h *PortalHandler) renderSignupPageWithPlan(name, email string, defaultPlan *ports.Plan, errors map[string]string) string {
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
		quotaDisplay := "Unlimited requests"
		if defaultPlan.RequestsPerMonth > 0 {
			if defaultPlan.RequestsPerMonth >= 1000 {
				quotaDisplay = fmt.Sprintf("%.0fK requests/month", float64(defaultPlan.RequestsPerMonth)/1000)
			} else {
				quotaDisplay = fmt.Sprintf("%d requests/month", defaultPlan.RequestsPerMonth)
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

func (h *PortalHandler) renderDashboardPage(user *PortalUser, keyCount int, requestCount int64, planName string, requestsPerMonth int64, rateLimitPerMinute int) string {
	// Show getting started section for new users with no API keys
	gettingStartedSection := ""
	if keyCount == 0 {
		gettingStartedSection = `
        <div class="card" style="margin-bottom: 24px; padding: 24px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; border-radius: 12px;">
            <h2 style="margin: 0 0 12px 0; font-size: 20px;">Welcome! Let's get you started</h2>
            <p style="margin: 0 0 16px 0; opacity: 0.9;">Follow these simple steps to start using the API:</p>
            <div style="display: flex; flex-direction: column; gap: 12px;">
                <div style="display: flex; align-items: center; gap: 12px;">
                    <span style="background: rgba(255,255,255,0.2); width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: bold;">1</span>
                    <span><strong>Create an API Key</strong> - Click the button below to generate your first key</span>
                </div>
                <div style="display: flex; align-items: center; gap: 12px;">
                    <span style="background: rgba(255,255,255,0.2); width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: bold;">2</span>
                    <span><strong>Read the Docs</strong> - Learn how to authenticate and make API calls</span>
                </div>
                <div style="display: flex; align-items: center; gap: 12px;">
                    <span style="background: rgba(255,255,255,0.2); width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: bold;">3</span>
                    <span><strong>Make your first request</strong> - Use your key to call the API</span>
                </div>
            </div>
            <div style="margin-top: 20px; display: flex; gap: 12px;">
                <a href="/portal/api-keys" style="background: white; color: #667eea; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 600;">Create API Key</a>
                <a href="/docs" target="_blank" style="background: rgba(255,255,255,0.2); color: white; padding: 10px 20px; border-radius: 6px; text-decoration: none; font-weight: 600;">View Documentation</a>
            </div>
        </div>`
	}

	// Calculate quota usage
	quotaSection := ""
	if requestsPerMonth > 0 {
		usagePercent := float64(requestCount) / float64(requestsPerMonth) * 100
		if usagePercent > 100 {
			usagePercent = 100
		}
		progressColor := "#22c55e" // green
		if usagePercent > 80 {
			progressColor = "#f59e0b" // amber
		}
		if usagePercent > 95 {
			progressColor = "#ef4444" // red
		}
		quotaSection = fmt.Sprintf(`
        <div class="card" style="margin-bottom: 24px; padding: 20px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                <div>
                    <strong>Monthly Quota</strong>
                    <span style="color: #666; font-size: 14px;"> - %s Plan</span>
                </div>
                <div style="font-size: 14px; color: #666;">
                    %d / %d requests (%.1f%%)
                </div>
            </div>
            <div style="background: #e5e7eb; border-radius: 4px; height: 8px; overflow: hidden;">
                <div style="background: %s; height: 100%%; width: %.1f%%; transition: width 0.3s;"></div>
            </div>
            <div style="display: flex; justify-content: space-between; margin-top: 12px; font-size: 13px; color: #666;">
                <span>Rate limit: %d requests/minute</span>
                <span>%d requests remaining</span>
            </div>
        </div>`, planName, requestCount, requestsPerMonth, usagePercent, progressColor, usagePercent, rateLimitPerMinute, requestsPerMonth-requestCount)
	} else if planName != "" {
		quotaSection = fmt.Sprintf(`
        <div class="card" style="margin-bottom: 24px; padding: 20px;">
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <div>
                    <strong>%s Plan</strong>
                    <span style="color: #22c55e; font-size: 14px;"> - Unlimited requests</span>
                </div>
                <div style="font-size: 14px; color: #666;">
                    Rate limit: %d requests/minute
                </div>
            </div>
        </div>`, planName, rateLimitPerMinute)
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
                <div class="stat-label">Requests This Month</div>
            </div>
        </div>
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
</html>`, h.appName, portalCSS, h.renderPortalNav(user), user.Name, quotaSection, gettingStartedSection, keyCount, requestCount)
}

func (h *PortalHandler) renderAPIKeysPage(user *PortalUser, keys []key.Key, revokedMsg bool) string {
	keyRows := ""
	for _, k := range keys {
		status := "Active"
		statusClass := "status-active"
		revokeBtn := ""
		if k.RevokedAt != nil {
			status = "Revoked"
			statusClass = "status-revoked"
			revokeBtn = "-" // Already revoked
		} else {
			revokeBtn = fmt.Sprintf(`<form method="POST" action="/portal/api-keys/%s/revoke" style="display:inline" onsubmit="return confirm('Are you sure you want to revoke this API key? This cannot be undone.')"><button type="submit" class="btn btn-sm btn-danger">Revoke</button></form>`, k.ID)
		}
		keyRows += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td><code>%s****</code></td>
                <td><span class="%s">%s</span></td>
                <td>%s</td>
                <td>%s</td>
            </tr>
        `, k.Name, k.Prefix, statusClass, status, k.CreatedAt.Format("Jan 2, 2006"), revokeBtn)
	}

	if keyRows == "" {
		keyRows = `<tr><td colspan="5" class="text-center">No API keys yet</td></tr>`
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
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
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
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), successMsg, keyRows)
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
                    <div>curl -H "X-API-Key: <span style="color: #ce9178;">%s</span>" \</div>
                    <div style="padding-left: 20px;">%s/your-endpoint</div>
                </div>
                <p style="margin: 12px 0 0 0; color: #6c757d; font-size: 13px;">Replace <code style="background: #e9ecef; padding: 2px 6px; border-radius: 4px;">your-endpoint</code> with the API path you want to call.</p>
            </div>

            <div style="margin-top: 20px;">
                <a href="/portal/api-keys" class="btn btn-secondary">Back to API Keys</a>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), displayName, rawKey, rawKey, baseURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *PortalHandler) renderUsagePage(user *PortalUser, summary usage.Summary) string {
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
                <div class="stat-label">Total Requests</div>
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
</html>`, h.appName, portalCSS, h.renderPortalNav(user), summary.RequestCount, summary.ErrorCount, float64(summary.BytesIn)/1024, float64(summary.BytesOut)/1024)
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
            <form method="POST" action="/portal/settings/close-account" onsubmit="return confirm('Are you sure? This cannot be undone.')">
                <div class="form-group">
                    <label for="password">Confirm with your password</label>
                    <input type="password" id="password" name="password" required>
                </div>
                <button type="submit" class="btn btn-danger">Close Account</button>
            </form>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), successHTML, errorHTML, user.Name, user.Email)
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
        </div>
        <div class="nav-links">
            <a href="/portal/dashboard">Dashboard</a>
            <a href="/portal/api-keys">API Keys</a>
            <a href="/portal/usage">Usage</a>
            <a href="/portal/plans">Plans</a>
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

func (h *PortalHandler) renderPlansPage(user *PortalUser, plans []ports.Plan, currentPlan *ports.Plan, success, errorMsg string, hasStripeSubscription bool) string {
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
		quotaDisplay := "Unlimited"
		if p.RequestsPerMonth > 0 {
			if p.RequestsPerMonth >= 1000000 {
				quotaDisplay = fmt.Sprintf("%.1fM requests/mo", float64(p.RequestsPerMonth)/1000000)
			} else if p.RequestsPerMonth >= 1000 {
				quotaDisplay = fmt.Sprintf("%.0fK requests/mo", float64(p.RequestsPerMonth)/1000)
			} else {
				quotaDisplay = fmt.Sprintf("%d requests/mo", p.RequestsPerMonth)
			}
		}

		// Format rate limit
		rateDisplay := fmt.Sprintf("%d req/min", p.RateLimitPerMinute)

		// Format overage
		overageDisplay := "Requests blocked"
		if p.OveragePrice > 0 {
			overageDisplay = fmt.Sprintf("$%.4f per extra request", float64(p.OveragePrice)/100)
		}

		// Current plan badge
		currentBadge := ""
		if isCurrent {
			currentBadge = `<span style="display: inline-block; background: #22c55e; color: white; padding: 4px 10px; border-radius: 12px; font-size: 12px; font-weight: 500; margin-left: 8px;">Current Plan</span>`
		}

		// Trial badge
		trialBadge := ""
		if p.TrialDays > 0 {
			trialBadge = fmt.Sprintf(`<div style="color: #7c3aed; font-size: 13px; font-weight: 500; margin-top: 4px;">%d-day free trial</div>`, p.TrialDays)
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
				<form method="POST" action="/portal/plans/change" onsubmit="return confirm('Change to %s plan?')">
					<input type="hidden" name="plan_id" value="%s">
					<button type="submit" class="btn btn-primary">%s</button>
				</form>`, p.Name, p.ID, buttonText)
		} else {
			// Free plan - show downgrade button
			actionBtn = fmt.Sprintf(`
				<form method="POST" action="/portal/plans/change" onsubmit="return confirm('Change to %s plan? You will lose access to higher limits.')">
					<input type="hidden" name="plan_id" value="%s">
					<button type="submit" class="btn btn-secondary">Switch Plan</button>
				</form>`, p.Name, p.ID)
		}

		// Card highlight for current plan
		cardStyle := ""
		if isCurrent {
			cardStyle = "border: 2px solid #22c55e;"
		}

		planCards += fmt.Sprintf(`
			<div class="plan-card" style="background: white; padding: 24px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); %s">
				<div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px;">
					<div>
						<h3 style="margin: 0; font-size: 20px;">%s%s</h3>
						<p style="margin: 4px 0 0 0; color: #6b7280; font-size: 14px;">%s</p>
					</div>
					<div style="text-align: right;">
						<div style="font-size: 28px; font-weight: bold; color: #111827;">%s</div>
						%s
					</div>
				</div>
				<div style="border-top: 1px solid #e5e7eb; padding-top: 16px; margin-bottom: 16px;">
					<div style="display: grid; gap: 8px;">
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #22c55e;">&#10003;</span>
							<span>%s</span>
						</div>
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #22c55e;">&#10003;</span>
							<span>%s rate limit</span>
						</div>
						<div style="display: flex; align-items: center; gap: 8px;">
							<span style="color: #6b7280;">&#8226;</span>
							<span style="color: #6b7280; font-size: 13px;">Over quota: %s</span>
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
                <form method="POST" action="/portal/subscription/cancel" onsubmit="return confirm('Are you sure you want to cancel your subscription?')">
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
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), alertHTML, planCards, subscriptionSection)
}

// Portal CSS styles
const portalCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }

.auth-container { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
.auth-box { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); width: 100%; max-width: 400px; }
.auth-header { text-align: center; margin-bottom: 30px; }
.auth-header h1 { color: #007bff; font-size: 24px; margin-bottom: 10px; }
.auth-header p { color: #666; }
.auth-form { margin-bottom: 20px; }
.auth-footer { text-align: center; }
.auth-footer p { margin: 10px 0; color: #666; }
.auth-footer a { color: #007bff; text-decoration: none; }

.form-group { margin-bottom: 20px; }
.form-group label { display: block; margin-bottom: 5px; font-weight: 500; }
.form-group input { width: 100%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 16px; }
.form-group input:focus { border-color: #007bff; outline: none; }
.form-group small { display: block; margin-top: 5px; color: #666; font-size: 12px; }

.btn { display: inline-block; padding: 10px 20px; border: none; border-radius: 4px; font-size: 14px; cursor: pointer; text-decoration: none; }
.btn-block { width: 100%; }
.btn-primary { background: #007bff; color: white; }
.btn-primary:hover { background: #0056b3; }
.btn-danger { background: #dc3545; color: white; }
.btn-danger:hover { background: #c82333; }
.btn-sm { padding: 5px 10px; font-size: 12px; }

.alert { padding: 15px; border-radius: 4px; margin-bottom: 20px; }
.alert-success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
.alert-error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
.alert-warning { background: #fff3cd; color: #856404; border: 1px solid #ffeeba; }
.alert-info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }

.portal-nav { background: white; padding: 15px 30px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #ddd; }
.nav-brand a { font-size: 20px; font-weight: bold; color: #007bff; text-decoration: none; }
.nav-links { display: flex; gap: 20px; }
.nav-links a { color: #333; text-decoration: none; }
.nav-links a:hover { color: #007bff; }
.nav-user { display: flex; align-items: center; gap: 15px; }
.nav-user span { color: #666; }

.main-content { max-width: 1200px; margin: 0 auto; padding: 30px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.page-header h1 { font-size: 28px; }
.page-header p { color: #666; }

.card { background: white; padding: 25px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin-bottom: 20px; }
.card h2 { font-size: 18px; margin-bottom: 20px; }
.card-danger { border-left: 4px solid #dc3545; }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
.stat-card { background: white; padding: 25px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); text-align: center; }
.stat-value { font-size: 32px; font-weight: bold; color: #007bff; }
.stat-label { color: #666; margin-top: 5px; }

.quick-links h2 { margin-bottom: 15px; }
.link-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 15px; }
.link-card { display: block; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); text-decoration: none; color: #333; transition: box-shadow 0.2s; }
.link-card:hover { box-shadow: 0 3px 10px rgba(0,0,0,0.15); }
.link-card strong { display: block; margin-bottom: 5px; }
.link-card span { color: #666; font-size: 14px; }

.table { width: 100%; border-collapse: collapse; }
.table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
.table th { background: #f9f9f9; font-weight: 500; }
.text-center { text-align: center; }

.status-active { color: #28a745; }
.status-revoked { color: #dc3545; }

code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: monospace; }

.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal-box { background: white; padding: 30px; border-radius: 8px; width: 100%; max-width: 450px; box-shadow: 0 4px 20px rgba(0,0,0,0.15); }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.modal-header h3 { font-size: 20px; }
.modal-close { background: none; border: none; font-size: 24px; cursor: pointer; color: #666; }
.modal-close:hover { color: #333; }
.modal-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
.btn-secondary { background: #6c757d; color: white; }
.btn-secondary:hover { background: #545b62; }

.key-display { background: #f8f9fa; border: 1px solid #dee2e6; padding: 15px; border-radius: 4px; margin: 15px 0; }
.key-display code { background: none; padding: 0; font-size: 14px; word-break: break-all; }
.key-warning { color: #856404; font-size: 13px; margin-top: 10px; }
`
