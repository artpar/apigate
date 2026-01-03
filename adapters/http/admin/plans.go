package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
)

// PlanResponse represents a plan in API responses.
type PlanResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Description        string  `json:"description,omitempty"`
	RateLimitPerMinute int     `json:"rate_limit_per_minute"`
	RequestsPerMonth   int64   `json:"requests_per_month"`
	PriceMonthly       float64 `json:"price_monthly"`
	OveragePrice       float64 `json:"overage_price"`
	StripePriceID      string  `json:"stripe_price_id,omitempty"`
	PaddlePriceID      string  `json:"paddle_price_id,omitempty"`
	LemonVariantID     string  `json:"lemon_variant_id,omitempty"`
	IsDefault          bool    `json:"is_default"`
	Enabled            bool    `json:"enabled"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// CreatePlanRequest represents a request to create a plan.
type CreatePlanRequest struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Description        string  `json:"description,omitempty"`
	RateLimitPerMinute int     `json:"rate_limit_per_minute"`
	RequestsPerMonth   int64   `json:"requests_per_month"`
	PriceMonthly       float64 `json:"price_monthly"`
	OveragePrice       float64 `json:"overage_price"`
	StripePriceID      string  `json:"stripe_price_id,omitempty"`
	PaddlePriceID      string  `json:"paddle_price_id,omitempty"`
	LemonVariantID     string  `json:"lemon_variant_id,omitempty"`
	IsDefault          bool    `json:"is_default"`
	Enabled            bool    `json:"enabled"`
}

// UpdatePlanRequest represents a request to update a plan.
type UpdatePlanRequest struct {
	Name               string   `json:"name,omitempty"`
	Description        string   `json:"description,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
	RequestsPerMonth   *int64   `json:"requests_per_month,omitempty"`
	PriceMonthly       *float64 `json:"price_monthly,omitempty"`
	OveragePrice       *float64 `json:"overage_price,omitempty"`
	StripePriceID      *string  `json:"stripe_price_id,omitempty"`
	PaddlePriceID      *string  `json:"paddle_price_id,omitempty"`
	LemonVariantID     *string  `json:"lemon_variant_id,omitempty"`
	IsDefault          *bool    `json:"is_default,omitempty"`
	Enabled            *bool    `json:"enabled,omitempty"`
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
	ctx := r.Context()

	planList, err := h.plans.List(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list plans")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list plans")
		return
	}

	plans := make([]PlanResponse, len(planList))
	for i, p := range planList {
		plans[i] = planToResponse(p)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plans": plans,
		"total": len(plans),
	})
}

// GetPlan returns a single plan.
//
//	@Summary		Get plan
//	@Description	Get plan by ID
//	@Tags			Admin - Plans
//	@Produce		json
//	@Param			id	path		string			true	"Plan ID"
//	@Success		200	{object}	PlanResponse	"Plan data"
//	@Failure		404	{object}	ErrorResponse	"Plan not found"
//	@Security		AdminAuth
//	@Router			/admin/plans/{id} [get]
func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	plan, err := h.plans.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Plan not found")
		return
	}

	writeJSON(w, http.StatusOK, planToResponse(plan))
}

// CreatePlan creates a new plan.
//
//	@Summary		Create plan
//	@Description	Create a new subscription plan
//	@Tags			Admin - Plans
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreatePlanRequest	true	"Plan data"
//	@Success		201		{object}	PlanResponse		"Created plan"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Failure		409		{object}	ErrorResponse		"Plan already exists"
//	@Security		AdminAuth
//	@Router			/admin/plans [post]
func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Plan ID is required")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_name", "Plan name is required")
		return
	}

	// Check if plan already exists
	if _, err := h.plans.Get(ctx, req.ID); err == nil {
		writeError(w, http.StatusConflict, "plan_exists", "Plan with this ID already exists")
		return
	}

	now := time.Now().UTC()
	plan := ports.Plan{
		ID:                 req.ID,
		Name:               req.Name,
		Description:        req.Description,
		RateLimitPerMinute: req.RateLimitPerMinute,
		RequestsPerMonth:   req.RequestsPerMonth,
		PriceMonthly:       int64(req.PriceMonthly * 100), // Convert to cents
		OveragePrice:       int64(req.OveragePrice * 10000), // Convert to hundredths of cents
		StripePriceID:      req.StripePriceID,
		PaddlePriceID:      req.PaddlePriceID,
		LemonVariantID:     req.LemonVariantID,
		IsDefault:          req.IsDefault,
		Enabled:            req.Enabled,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Clear default flag on existing plans if creating a new default plan
	if req.IsDefault {
		existingPlans, err := h.plans.List(ctx)
		if err == nil {
			for _, p := range existingPlans {
				if p.IsDefault {
					p.IsDefault = false
					p.UpdatedAt = now
					_ = h.plans.Update(ctx, p)
				}
			}
		}
	}

	if err := h.plans.Create(ctx, plan); err != nil {
		h.logger.Error().Err(err).Msg("failed to create plan")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create plan")
		return
	}

	h.logger.Info().Str("plan_id", plan.ID).Msg("plan created via admin api")
	writeJSON(w, http.StatusCreated, planToResponse(plan))
}

// UpdatePlan updates a plan.
//
//	@Summary		Update plan
//	@Description	Update a subscription plan
//	@Tags			Admin - Plans
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Plan ID"
//	@Param			request	body		UpdatePlanRequest	true	"Update data"
//	@Success		200		{object}	PlanResponse		"Updated plan"
//	@Failure		404		{object}	ErrorResponse		"Plan not found"
//	@Security		AdminAuth
//	@Router			/admin/plans/{id} [put]
func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	plan, err := h.plans.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Plan not found")
		return
	}

	var req UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.Description != "" {
		plan.Description = req.Description
	}
	if req.RateLimitPerMinute != nil {
		plan.RateLimitPerMinute = *req.RateLimitPerMinute
	}
	if req.RequestsPerMonth != nil {
		plan.RequestsPerMonth = *req.RequestsPerMonth
	}
	if req.PriceMonthly != nil {
		plan.PriceMonthly = int64(*req.PriceMonthly * 100)
	}
	if req.OveragePrice != nil {
		plan.OveragePrice = int64(*req.OveragePrice * 10000) // Convert to hundredths of cents
	}
	if req.StripePriceID != nil {
		plan.StripePriceID = *req.StripePriceID
	}
	if req.PaddlePriceID != nil {
		plan.PaddlePriceID = *req.PaddlePriceID
	}
	if req.LemonVariantID != nil {
		plan.LemonVariantID = *req.LemonVariantID
	}
	if req.IsDefault != nil {
		plan.IsDefault = *req.IsDefault
		// Clear default flag on other plans if setting this plan as default
		if *req.IsDefault {
			existingPlans, err := h.plans.List(ctx)
			if err == nil {
				for _, p := range existingPlans {
					if p.IsDefault && p.ID != plan.ID {
						p.IsDefault = false
						p.UpdatedAt = time.Now().UTC()
						_ = h.plans.Update(ctx, p)
					}
				}
			}
		}
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	plan.UpdatedAt = time.Now().UTC()

	if err := h.plans.Update(ctx, plan); err != nil {
		h.logger.Error().Err(err).Msg("failed to update plan")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update plan")
		return
	}

	h.logger.Info().Str("plan_id", plan.ID).Msg("plan updated via admin api")
	writeJSON(w, http.StatusOK, planToResponse(plan))
}

// DeletePlan deletes a plan.
//
//	@Summary		Delete plan
//	@Description	Delete a subscription plan
//	@Tags			Admin - Plans
//	@Produce		json
//	@Param			id	path		string				true	"Plan ID"
//	@Success		200	{object}	map[string]string	"Deleted"
//	@Failure		404	{object}	ErrorResponse		"Plan not found"
//	@Failure		409	{object}	ErrorResponse		"Plan in use"
//	@Security		AdminAuth
//	@Router			/admin/plans/{id} [delete]
func (h *Handler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if _, err := h.plans.Get(ctx, id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Plan not found")
		return
	}

	// Check if any users are on this plan
	users, _ := h.users.List(ctx, 1000, 0)
	for _, u := range users {
		if u.PlanID == id {
			writeError(w, http.StatusConflict, "plan_in_use", "Cannot delete plan: users are assigned to it")
			return
		}
	}

	if err := h.plans.Delete(ctx, id); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete plan")
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete plan")
		return
	}

	h.logger.Info().Str("plan_id", id).Msg("plan deleted via admin api")
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func planToResponse(p ports.Plan) PlanResponse {
	return PlanResponse{
		ID:                 p.ID,
		Name:               p.Name,
		Description:        p.Description,
		RateLimitPerMinute: p.RateLimitPerMinute,
		RequestsPerMonth:   p.RequestsPerMonth,
		PriceMonthly:       float64(p.PriceMonthly) / 100,
		OveragePrice:       float64(p.OveragePrice) / 10000,
		StripePriceID:      p.StripePriceID,
		PaddlePriceID:      p.PaddlePriceID,
		LemonVariantID:     p.LemonVariantID,
		IsDefault:          p.IsDefault,
		Enabled:            p.Enabled,
		CreatedAt:          p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          p.UpdatedAt.Format(time.RFC3339),
	}
}
