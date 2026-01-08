package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PortalWebhooksPage renders the user's webhooks list.
func (h *PortalHandler) PortalWebhooksPage(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	var webhooks []webhook.Webhook
	if h.webhooks != nil {
		webhooks, _ = h.webhooks.ListByUser(r.Context(), user.ID)
	}

	html := h.renderWebhooksPage(user, webhooks)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// PortalWebhookNewPage renders the new webhook form.
func (h *PortalHandler) PortalWebhookNewPage(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	wh := webhook.Webhook{
		RetryCount: 3,
		TimeoutMS:  30000,
		Enabled:    true,
		Secret:     webhook.GenerateSecret(),
	}

	html := h.renderWebhookFormPage(user, wh, true, "", nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// PortalWebhookCreate handles the create webhook form submission.
func (h *PortalHandler) PortalWebhookCreate(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	events := parsePortalWebhookEvents(r)

	retryCount, _ := strconv.Atoi(r.FormValue("retry_count"))
	if retryCount == 0 {
		retryCount = 3
	}
	timeoutMS, _ := strconv.Atoi(r.FormValue("timeout_ms"))
	if timeoutMS == 0 {
		timeoutMS = 30000
	}

	wh := webhook.Webhook{
		ID:          uuid.New().String(),
		UserID:      user.ID,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		URL:         r.FormValue("url"),
		Secret:      r.FormValue("secret"),
		Events:      events,
		RetryCount:  retryCount,
		TimeoutMS:   timeoutMS,
		Enabled:     r.FormValue("enabled") == "true",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate
	if wh.Name == "" {
		html := h.renderWebhookFormPage(user, wh, true, "Name is required", nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}
	if valid, msg := webhook.ValidateURL(wh.URL); !valid {
		html := h.renderWebhookFormPage(user, wh, true, msg, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}
	if valid, msg := webhook.ValidateEvents(wh.Events); !valid {
		html := h.renderWebhookFormPage(user, wh, true, msg, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	if wh.Secret == "" {
		wh.Secret = webhook.GenerateSecret()
	}

	if err := h.webhooks.Create(r.Context(), wh); err != nil {
		html := h.renderWebhookFormPage(user, wh, true, "Failed to create webhook: "+err.Error(), nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	http.Redirect(w, r, "/portal/webhooks", http.StatusSeeOther)
}

// PortalWebhookEditPage renders the edit webhook form.
func (h *PortalHandler) PortalWebhookEditPage(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	wh, err := h.webhooks.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	if wh.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var deliveries []webhook.Delivery
	if h.deliveries != nil {
		deliveries, _ = h.deliveries.List(r.Context(), id, 20)
	}

	html := h.renderWebhookFormPage(user, wh, false, "", deliveries)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// PortalWebhookUpdate handles the update webhook form submission.
func (h *PortalHandler) PortalWebhookUpdate(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	wh, err := h.webhooks.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	if wh.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	events := parsePortalWebhookEvents(r)

	retryCount, _ := strconv.Atoi(r.FormValue("retry_count"))
	if retryCount == 0 {
		retryCount = 3
	}
	timeoutMS, _ := strconv.Atoi(r.FormValue("timeout_ms"))
	if timeoutMS == 0 {
		timeoutMS = 30000
	}

	wh.Name = r.FormValue("name")
	wh.Description = r.FormValue("description")
	wh.URL = r.FormValue("url")
	wh.Events = events
	wh.RetryCount = retryCount
	wh.TimeoutMS = timeoutMS
	wh.Enabled = r.FormValue("enabled") == "true"
	wh.UpdatedAt = time.Now()

	if valid, msg := webhook.ValidateURL(wh.URL); !valid {
		html := h.renderWebhookFormPage(user, wh, false, msg, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}
	if valid, msg := webhook.ValidateEvents(wh.Events); !valid {
		html := h.renderWebhookFormPage(user, wh, false, msg, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	if err := h.webhooks.Update(r.Context(), wh); err != nil {
		html := h.renderWebhookFormPage(user, wh, false, "Failed to update: "+err.Error(), nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
		return
	}

	http.Redirect(w, r, "/portal/webhooks", http.StatusSeeOther)
}

// PortalWebhookDelete handles the delete webhook request.
func (h *PortalHandler) PortalWebhookDelete(w http.ResponseWriter, r *http.Request) {
	user := getPortalUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/portal/login", http.StatusSeeOther)
		return
	}

	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	wh, err := h.webhooks.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	if wh.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.webhooks.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/portal/webhooks", http.StatusSeeOther)
}

func (h *PortalHandler) renderWebhooksPage(user *PortalUser, webhooks []webhook.Webhook) string {
	webhookRows := ""
	for _, wh := range webhooks {
		status := "Active"
		statusClass := "status-active"
		if !wh.Enabled {
			status = "Disabled"
			statusClass = "status-revoked"
		}

		webhookRows += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td><code style="font-size: 12px; word-break: break-all;">%s</code></td>
                <td>%d events</td>
                <td><span class="%s">%s</span></td>
                <td>
                    <a href="/portal/webhooks/%s" class="btn btn-sm btn-secondary">Edit</a>
                </td>
            </tr>
        `, wh.Name, wh.URL, len(wh.Events), statusClass, status, wh.ID)
	}

	if webhookRows == "" {
		webhookRows = `<tr><td colspan="5" class="text-center">No webhooks configured</td></tr>`
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Webhooks - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>Webhooks</h1>
            <a href="/portal/webhooks/new" class="btn btn-primary">Create Webhook</a>
        </div>
        <div class="card">
            <table class="table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>URL</th>
                        <th>Events</th>
                        <th>Status</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
        </div>

        <div class="card" style="margin-top: 24px;">
            <h3 style="margin-bottom: 16px;">About Webhooks</h3>
            <p style="color: #666; margin-bottom: 12px;">Webhooks allow your systems to receive real-time notifications when events occur.</p>
            <p style="color: #666; font-size: 14px;">All payloads are signed with HMAC-SHA256. Verify the <code>X-Webhook-Signature</code> header with your webhook secret.</p>
        </div>
    </main>
    %s
</body>
</html>`, h.appName, portalCSS, h.renderPortalNav(user), webhookRows, portalConfirmJS)
}

func (h *PortalHandler) renderWebhookFormPage(user *PortalUser, wh webhook.Webhook, isNew bool, errorMsg string, deliveries []webhook.Delivery) string {
	title := "Edit Webhook"
	submitBtn := "Save Changes"
	if isNew {
		title = "Create Webhook"
		submitBtn = "Create Webhook"
	}

	errorHTML := ""
	if errorMsg != "" {
		errorHTML = fmt.Sprintf(`<div class="alert alert-error" style="background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; padding: 12px 16px; border-radius: 6px; margin-bottom: 16px;">%s</div>`, errorMsg)
	}

	// Build event checkboxes
	eventsHTML := ""
	allEvents := webhook.AllEventTypes()
	for _, et := range allEvents {
		checked := ""
		for _, we := range wh.Events {
			if we == et {
				checked = "checked"
				break
			}
		}
		eventsHTML += fmt.Sprintf(`
            <label style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                <input type="checkbox" name="events" value="%s" %s>
                <span>%s</span>
            </label>
        `, et, checked, et)
	}

	enabledChecked := ""
	if wh.Enabled {
		enabledChecked = "checked"
	}

	deleteBtn := ""
	if !isNew {
		deleteBtn = fmt.Sprintf(`<form method="POST" action="/portal/webhooks/%s" style="display:inline" onsubmit="if(!confirm('Are you sure you want to delete this webhook?')) return false;">
            <input type="hidden" name="_method" value="DELETE">
            <button type="submit" class="btn btn-danger">Delete</button>
        </form>`, wh.ID)
	}

	// Build deliveries table
	deliveriesHTML := ""
	if !isNew && len(deliveries) > 0 {
		deliveryRows := ""
		for _, d := range deliveries {
			statusClass := "status-active"
			if d.Status == webhook.DeliveryFailed {
				statusClass = "status-revoked"
			} else if d.Status == webhook.DeliveryRetrying {
				statusClass = "status-pending"
			}
			deliveryRows += fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td><span class="%s">%s</span></td>
                    <td>%d/%d</td>
                    <td>%d</td>
                    <td>%s</td>
                </tr>
            `, d.EventType, statusClass, d.Status, d.Attempt, d.MaxAttempts, d.StatusCode, d.CreatedAt.Format("Jan 2 15:04"))
		}
		deliveriesHTML = fmt.Sprintf(`
        <div class="card" style="margin-top: 24px;">
            <h3 style="margin-bottom: 16px;">Recent Deliveries</h3>
            <table class="table">
                <thead>
                    <tr>
                        <th>Event</th>
                        <th>Status</th>
                        <th>Attempt</th>
                        <th>Response</th>
                        <th>Time</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
        </div>
        `, deliveryRows)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="main-content">
        <div class="page-header">
            <h1>%s</h1>
        </div>
        %s
        <div class="card">
            <form method="POST" action="/portal/webhooks">
                <div class="form-group">
                    <label for="name">Name</label>
                    <input type="text" id="name" name="name" value="%s" required placeholder="e.g., Usage Alerts">
                </div>

                <div class="form-group">
                    <label for="description">Description</label>
                    <textarea id="description" name="description" rows="2" placeholder="What this webhook is used for">%s</textarea>
                </div>

                <div class="form-group">
                    <label for="url">Endpoint URL</label>
                    <input type="url" id="url" name="url" value="%s" required placeholder="https://example.com/webhooks">
                </div>

                <div class="form-group">
                    <label for="secret">Signing Secret</label>
                    <input type="text" id="secret" name="secret" value="%s" readonly style="font-family: monospace; background: #f5f5f5;">
                    <small>Use this secret to verify webhook signatures</small>
                </div>

                <div class="form-group">
                    <label>Events</label>
                    <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; margin-top: 8px;">
                        %s
                    </div>
                </div>

                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                    <div class="form-group">
                        <label for="retry_count">Max Retries</label>
                        <input type="number" id="retry_count" name="retry_count" value="%d" min="0" max="10">
                    </div>
                    <div class="form-group">
                        <label for="timeout_ms">Timeout (ms)</label>
                        <input type="number" id="timeout_ms" name="timeout_ms" value="%d" min="1000" max="60000">
                    </div>
                </div>

                <div class="form-group">
                    <label style="display: flex; align-items: center; gap: 8px;">
                        <input type="checkbox" name="enabled" value="true" %s>
                        <span>Enabled</span>
                    </label>
                </div>

                <div style="display: flex; gap: 12px; margin-top: 24px;">
                    <button type="submit" class="btn btn-primary">%s</button>
                    <a href="/portal/webhooks" class="btn btn-secondary">Cancel</a>
                    %s
                </div>
            </form>
        </div>
        %s
    </main>
    %s
</body>
</html>`, title, h.appName, portalCSS, h.renderPortalNav(user), title, errorHTML,
		wh.Name, wh.Description, wh.URL, wh.Secret, eventsHTML,
		wh.RetryCount, wh.TimeoutMS, enabledChecked, submitBtn, deleteBtn, deliveriesHTML, portalConfirmJS)
}

func parsePortalWebhookEvents(r *http.Request) []webhook.EventType {
	r.ParseForm()
	eventStrs := r.Form["events"]
	var events []webhook.EventType
	for _, e := range eventStrs {
		e = strings.TrimSpace(e)
		if e != "" {
			events = append(events, webhook.EventType(e))
		}
	}
	return events
}
