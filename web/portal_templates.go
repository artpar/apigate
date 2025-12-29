package web

import (
	"fmt"
	"strings"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/usage"
)

// Portal HTML templates - simple inline templates for the user portal.
// These are separate from the admin templates to keep the portal lightweight.

func (h *PortalHandler) renderSignupPage(email string, errors map[string]string) string {
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
            <form method="POST" action="/portal/signup" class="auth-form">
                <div class="form-group">
                    <label for="name">Name</label>
                    <input type="text" id="name" name="name" required autofocus>
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
                <button type="submit" class="btn btn-primary btn-block">Create Account</button>
            </form>
            <div class="auth-footer">
                <p>Already have an account? <a href="/portal/login">Log in</a></p>
            </div>
        </div>
    </div>
</body>
</html>`, h.appName, portalCSS, h.appName, errorHTML, email)
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

func (h *PortalHandler) renderDashboardPage(user *PortalUser, keyCount int, requestCount int64) string {
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
                <a href="/portal/settings" class="link-card">
                    <strong>Account Settings</strong>
                    <span>Update your account</span>
                </a>
            </div>
        </div>
    </main>
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), user.Name, keyCount, requestCount)
}

func (h *PortalHandler) renderAPIKeysPage(user *PortalUser, keys []key.Key) string {
	keyRows := ""
	for _, k := range keys {
		status := "Active"
		statusClass := "status-active"
		if k.RevokedAt != nil {
			status = "Revoked"
			statusClass = "status-revoked"
		}
		keyRows += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td><code>%s****</code></td>
                <td><span class="%s">%s</span></td>
                <td>%s</td>
                <td>
                    <form method="POST" action="/portal/api-keys/%s" style="display:inline">
                        <input type="hidden" name="_method" value="DELETE">
                        <button type="submit" class="btn btn-sm btn-danger">Revoke</button>
                    </form>
                </td>
            </tr>
        `, k.Name, k.Prefix, statusClass, status, k.CreatedAt.Format("Jan 2, 2006"), k.ID)
	}

	if keyRows == "" {
		keyRows = `<tr><td colspan="5" class="text-center">No API keys yet</td></tr>`
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
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), keyRows)
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

func (h *PortalHandler) renderAccountSettingsPage(user *PortalUser, errors map[string]string) string {
	errorHTML := ""
	if len(errors) > 0 {
		var msgs []string
		for field, msg := range errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
		}
		errorHTML = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, strings.Join(msgs, "<br>"))
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
        <div class="card">
            <h2>Profile</h2>
            <form method="POST" action="/portal/settings">
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
</html>`, h.appName, portalCSS, h.renderPortalNav(user), errorHTML, user.Email)
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
`
