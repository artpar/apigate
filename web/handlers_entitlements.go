package web

import (
	"net/http"
	"time"

	"github.com/artpar/apigate/domain/entitlement"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// EntitlementsPage renders the entitlements list page.
func (h *Handler) EntitlementsPage(w http.ResponseWriter, r *http.Request) {
	data := h.newPageData(r.Context(), "Entitlements")
	h.render(w, "entitlements", data)
}

// EntitlementNewPage renders the new entitlement form.
func (h *Handler) EntitlementNewPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		PageData
		Entitlement entitlement.Entitlement
		IsNew       bool
		Error       string
	}{
		PageData: h.newPageData(r.Context(), "Create Entitlement"),
		Entitlement: entitlement.Entitlement{
			Category:     entitlement.CategoryFeature,
			ValueType:    entitlement.ValueTypeBoolean,
			DefaultValue: "true",
			Enabled:      true,
		},
		IsNew: true,
	}
	h.render(w, "entitlement_form", data)
}

// EntitlementCreate handles the create entitlement form submission.
func (h *Handler) EntitlementCreate(w http.ResponseWriter, r *http.Request) {
	if h.entitlements == nil {
		http.Error(w, "Entitlement store not configured", http.StatusInternalServerError)
		return
	}

	ent := entitlement.Entitlement{
		ID:           uuid.New().String(),
		Name:         r.FormValue("name"),
		DisplayName:  r.FormValue("display_name"),
		Description:  r.FormValue("description"),
		Category:     entitlement.Category(r.FormValue("category")),
		ValueType:    entitlement.ValueType(r.FormValue("value_type")),
		DefaultValue: r.FormValue("default_value"),
		HeaderName:   r.FormValue("header_name"),
		Enabled:      r.FormValue("enabled") == "true",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if ent.Name == "" {
		h.renderEntitlementFormError(w, r, ent, true, "Name is required")
		return
	}

	if err := h.entitlements.Create(r.Context(), ent); err != nil {
		h.renderEntitlementFormError(w, r, ent, true, "Failed to create entitlement: "+err.Error())
		return
	}

	// Trigger reload
	if h.entitlementReloader != nil {
		h.entitlementReloader.ReloadEntitlements(r.Context())
	}

	http.Redirect(w, r, "/entitlements", http.StatusSeeOther)
}

// EntitlementEditPage renders the edit entitlement form.
func (h *Handler) EntitlementEditPage(w http.ResponseWriter, r *http.Request) {
	if h.entitlements == nil {
		http.Error(w, "Entitlement store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	ent, err := h.entitlements.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Entitlement not found", http.StatusNotFound)
		return
	}

	data := struct {
		PageData
		Entitlement entitlement.Entitlement
		IsNew       bool
		Error       string
	}{
		PageData:    h.newPageData(r.Context(), "Edit Entitlement"),
		Entitlement: ent,
		IsNew:       false,
	}
	h.render(w, "entitlement_form", data)
}

// EntitlementUpdate handles the update entitlement form submission.
func (h *Handler) EntitlementUpdate(w http.ResponseWriter, r *http.Request) {
	if h.entitlements == nil {
		http.Error(w, "Entitlement store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	ent, err := h.entitlements.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Entitlement not found", http.StatusNotFound)
		return
	}

	ent.Name = r.FormValue("name")
	ent.DisplayName = r.FormValue("display_name")
	ent.Description = r.FormValue("description")
	ent.Category = entitlement.Category(r.FormValue("category"))
	ent.ValueType = entitlement.ValueType(r.FormValue("value_type"))
	ent.DefaultValue = r.FormValue("default_value")
	ent.HeaderName = r.FormValue("header_name")
	ent.Enabled = r.FormValue("enabled") == "true"
	ent.UpdatedAt = time.Now()

	if err := h.entitlements.Update(r.Context(), ent); err != nil {
		h.renderEntitlementFormError(w, r, ent, false, "Failed to update: "+err.Error())
		return
	}

	// Trigger reload
	if h.entitlementReloader != nil {
		h.entitlementReloader.ReloadEntitlements(r.Context())
	}

	http.Redirect(w, r, "/entitlements", http.StatusSeeOther)
}

// EntitlementDelete handles the delete entitlement request.
func (h *Handler) EntitlementDelete(w http.ResponseWriter, r *http.Request) {
	if h.entitlements == nil {
		http.Error(w, "Entitlement store not configured", http.StatusInternalServerError)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.entitlements.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger reload
	if h.entitlementReloader != nil {
		h.entitlementReloader.ReloadEntitlements(r.Context())
	}

	w.Header().Set("HX-Redirect", "/entitlements")
	w.WriteHeader(http.StatusOK)
}

// PartialEntitlements renders the entitlements table partial.
func (h *Handler) PartialEntitlements(w http.ResponseWriter, r *http.Request) {
	if h.entitlements == nil {
		h.renderPartial(w, "entitlements-table", nil)
		return
	}

	ents, err := h.entitlements.List(r.Context())
	if err != nil {
		h.renderPartial(w, "entitlements-table", struct{ Error string }{Error: err.Error()})
		return
	}

	h.renderPartial(w, "entitlements-table", struct {
		Entitlements []entitlement.Entitlement
	}{
		Entitlements: ents,
	})
}

// PartialPlanEntitlements renders the plan entitlements table partial.
func (h *Handler) PartialPlanEntitlements(w http.ResponseWriter, r *http.Request) {
	if h.planEntitlements == nil {
		h.renderPartial(w, "plan-entitlements-table", nil)
		return
	}

	pes, err := h.planEntitlements.List(r.Context())
	if err != nil {
		h.renderPartial(w, "plan-entitlements-table", struct{ Error string }{Error: err.Error()})
		return
	}

	// Also get plans and entitlements for reference
	var plans []ports.Plan
	var ents []entitlement.Entitlement
	if h.plans != nil {
		plans, _ = h.plans.List(r.Context())
	}
	if h.entitlements != nil {
		ents, _ = h.entitlements.List(r.Context())
	}

	// Build lookup maps
	planMap := make(map[string]string)
	for _, p := range plans {
		planMap[p.ID] = p.Name
	}
	entMap := make(map[string]string)
	for _, e := range ents {
		entMap[e.ID] = e.Name
	}

	type PlanEntitlementRow struct {
		entitlement.PlanEntitlement
		PlanName        string
		EntitlementName string
	}

	rows := make([]PlanEntitlementRow, 0, len(pes))
	for _, pe := range pes {
		rows = append(rows, PlanEntitlementRow{
			PlanEntitlement: pe,
			PlanName:        planMap[pe.PlanID],
			EntitlementName: entMap[pe.EntitlementID],
		})
	}

	h.renderPartial(w, "plan-entitlements-table", struct {
		PlanEntitlements []PlanEntitlementRow
	}{
		PlanEntitlements: rows,
	})
}

func (h *Handler) renderEntitlementFormError(w http.ResponseWriter, r *http.Request, ent entitlement.Entitlement, isNew bool, errMsg string) {
	title := "Edit Entitlement"
	if isNew {
		title = "Create Entitlement"
	}
	data := struct {
		PageData
		Entitlement entitlement.Entitlement
		IsNew       bool
		Error       string
	}{
		PageData:    h.newPageData(r.Context(), title),
		Entitlement: ent,
		IsNew:       isNew,
		Error:       errMsg,
	}
	h.render(w, "entitlement_form", data)
}
