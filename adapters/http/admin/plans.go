package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PlanResponse represents a plan in API responses.
type PlanResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	RequestsPerMonth   int64  `json:"requests_per_month"`
	PriceMonthly       int64  `json:"price_monthly"`
	OveragePrice       int64  `json:"overage_price"`
}

// CreatePlanRequest represents a request to create a plan.
type CreatePlanRequest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	RequestsPerMonth   int64  `json:"requests_per_month"`
	PriceMonthly       int64  `json:"price_monthly"`
	OveragePrice       int64  `json:"overage_price"`
}

// UpdatePlanRequest represents a request to update a plan.
type UpdatePlanRequest struct {
	Name               string `json:"name,omitempty"`
	RateLimitPerMinute *int   `json:"rate_limit_per_minute,omitempty"`
	RequestsPerMonth   *int64 `json:"requests_per_month,omitempty"`
	PriceMonthly       *int64 `json:"price_monthly,omitempty"`
	OveragePrice       *int64 `json:"overage_price,omitempty"`
}

// ListPlans returns all plans.
//
//	@Summary		List plans
//	@Description	Get all subscription plans
//	@Tags			Admin - Plans
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}	"Plans list"
//	@Security		AdminAuth
//	@Router			/admin/plans [get]
func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	// Plans come from config for now
	plans := make([]PlanResponse, len(h.config.Plans))
	for i, p := range h.config.Plans {
		plans[i] = PlanResponse{
			ID:                 p.ID,
			Name:               p.Name,
			RateLimitPerMinute: p.RateLimitPerMinute,
			RequestsPerMonth:   p.RequestsPerMonth,
			PriceMonthly:       p.PriceMonthly,
			OveragePrice:       p.OveragePrice,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plans": plans,
		"total": len(plans),
		"note":  "Plans are currently defined in config. Dynamic plans coming in future release.",
	})
}

// CreatePlan creates a new plan.
//
//	@Summary		Create plan
//	@Description	Create a new subscription plan (requires config reload)
//	@Tags			Admin - Plans
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreatePlanRequest	true	"Plan data"
//	@Success		501		{object}	ErrorResponse		"Not yet implemented"
//	@Security		AdminAuth
//	@Router			/admin/plans [post]
func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// For now, plans are defined in config
	// This will be implemented with a PlanStore in a future release
	writeError(w, http.StatusNotImplemented, "not_implemented",
		"Dynamic plan creation not yet available. Edit apigate.yaml and reload config.")
}

// UpdatePlan updates a plan.
//
//	@Summary		Update plan
//	@Description	Update a subscription plan (requires config reload)
//	@Tags			Admin - Plans
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Plan ID"
//	@Param			request	body		UpdatePlanRequest	true	"Update data"
//	@Success		501		{object}	ErrorResponse		"Not yet implemented"
//	@Security		AdminAuth
//	@Router			/admin/plans/{id} [put]
func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if plan exists
	found := false
	for _, p := range h.config.Plans {
		if p.ID == id {
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "not_found", "Plan not found")
		return
	}

	var req UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// For now, plans are defined in config
	writeError(w, http.StatusNotImplemented, "not_implemented",
		"Dynamic plan updates not yet available. Edit apigate.yaml and reload config.")
}
