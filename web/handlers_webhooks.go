package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WebhooksPage renders the webhooks list page.
func (h *Handler) WebhooksPage(w http.ResponseWriter, r *http.Request) {
	data := h.newPageData(r.Context(), "Webhooks")
	h.render(w, "webhooks", data)
}

// WebhookNewPage renders the new webhook form.
func (h *Handler) WebhookNewPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		Webhook    webhook.Webhook
		EventTypes []webhook.EventType
		IsNew      bool
		Error      string
	}{
		PageData: h.newPageData(r.Context(), "Create Webhook"),
		Webhook: webhook.Webhook{
			RetryCount: 3,
			TimeoutMS:  30000,
			Enabled:    true,
			Secret:     webhook.GenerateSecret(),
		},
		EventTypes: webhook.AllEventTypes(),
		IsNew:      true,
	}
	h.render(w, "webhook_form", data)
}

// WebhookCreate handles the create webhook form submission.
func (h *Handler) WebhookCreate(w http.ResponseWriter, r *http.Request) {
	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	// Parse events from form
	events := parseWebhookEvents(r)

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
		UserID:      r.FormValue("user_id"),
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
		h.renderWebhookFormError(w, r, wh, true, "Name is required")
		return
	}
	if valid, msg := webhook.ValidateURL(wh.URL); !valid {
		h.renderWebhookFormError(w, r, wh, true, msg)
		return
	}
	if valid, msg := webhook.ValidateEvents(wh.Events); !valid {
		h.renderWebhookFormError(w, r, wh, true, msg)
		return
	}

	// Generate secret if empty
	if wh.Secret == "" {
		wh.Secret = webhook.GenerateSecret()
	}

	if err := h.webhooks.Create(r.Context(), wh); err != nil {
		h.renderWebhookFormError(w, r, wh, true, "Failed to create webhook: "+err.Error())
		return
	}

	http.Redirect(w, r, "/webhooks", http.StatusSeeOther)
}

// WebhookEditPage renders the edit webhook form.
func (h *Handler) WebhookEditPage(w http.ResponseWriter, r *http.Request) {
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

	data := struct {
		PageData
		Webhook    webhook.Webhook
		EventTypes []webhook.EventType
		IsNew      bool
		Error      string
	}{
		PageData:   h.newPageData(r.Context(), "Edit Webhook"),
		Webhook:    wh,
		EventTypes: webhook.AllEventTypes(),
		IsNew:      false,
	}
	h.render(w, "webhook_form", data)
}

// WebhookUpdate handles the update webhook form submission.
func (h *Handler) WebhookUpdate(w http.ResponseWriter, r *http.Request) {
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

	// Parse events from form
	events := parseWebhookEvents(r)

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

	// Validate
	if valid, msg := webhook.ValidateURL(wh.URL); !valid {
		h.renderWebhookFormError(w, r, wh, false, msg)
		return
	}
	if valid, msg := webhook.ValidateEvents(wh.Events); !valid {
		h.renderWebhookFormError(w, r, wh, false, msg)
		return
	}

	if err := h.webhooks.Update(r.Context(), wh); err != nil {
		h.renderWebhookFormError(w, r, wh, false, "Failed to update: "+err.Error())
		return
	}

	http.Redirect(w, r, "/webhooks", http.StatusSeeOther)
}

// WebhookDelete handles the delete webhook request.
func (h *Handler) WebhookDelete(w http.ResponseWriter, r *http.Request) {
	if h.webhooks == nil {
		http.Error(w, "Webhook store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.webhooks.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/webhooks")
	w.WriteHeader(http.StatusOK)
}

// WebhookTest sends a test event to a webhook.
func (h *Handler) WebhookTest(w http.ResponseWriter, r *http.Request) {
	if h.webhooks == nil || h.webhookService == nil {
		http.Error(w, "Webhook service not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.webhookService.TestWebhook(r.Context(), id); err != nil {
		http.Error(w, "Failed to send test: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "webhook-tested")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Test event sent"))
}

// PartialWebhooks renders the webhooks table partial.
func (h *Handler) PartialWebhooks(w http.ResponseWriter, r *http.Request) {
	if h.webhooks == nil {
		h.renderPartial(w, "webhooks-table", nil)
		return
	}

	webhooks, err := h.webhooks.List(r.Context())
	if err != nil {
		h.renderPartial(w, "webhooks-table", struct{ Error string }{Error: err.Error()})
		return
	}

	h.renderPartial(w, "webhooks-table", struct {
		Webhooks []webhook.Webhook
	}{
		Webhooks: webhooks,
	})
}

// PartialWebhookDeliveries renders the webhook deliveries table partial.
func (h *Handler) PartialWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	if h.deliveries == nil {
		h.renderPartial(w, "webhook-deliveries-table", nil)
		return
	}

	webhookID := chi.URLParam(r, "id")
	deliveries, err := h.deliveries.List(r.Context(), webhookID, 50)
	if err != nil {
		h.renderPartial(w, "webhook-deliveries-table", struct{ Error string }{Error: err.Error()})
		return
	}

	h.renderPartial(w, "webhook-deliveries-table", struct {
		Deliveries []webhook.Delivery
	}{
		Deliveries: deliveries,
	})
}

func (h *Handler) renderWebhookFormError(w http.ResponseWriter, r *http.Request, wh webhook.Webhook, isNew bool, errMsg string) {
	title := "Edit Webhook"
	if isNew {
		title = "Create Webhook"
	}
	data := struct {
		PageData
		Webhook    webhook.Webhook
		EventTypes []webhook.EventType
		IsNew      bool
		Error      string
	}{
		PageData:   h.newPageData(r.Context(), title),
		Webhook:    wh,
		EventTypes: webhook.AllEventTypes(),
		IsNew:      isNew,
		Error:      errMsg,
	}
	h.render(w, "webhook_form", data)
}

func parseWebhookEvents(r *http.Request) []webhook.EventType {
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
