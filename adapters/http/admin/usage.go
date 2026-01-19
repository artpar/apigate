package admin

import (
	"net/http"
	"time"

	"github.com/artpar/apigate/pkg/jsonapi"
)

// UsageResponse represents usage statistics.
type UsageResponse struct {
	Period    string              `json:"period"`
	StartDate string              `json:"start_date"`
	EndDate   string              `json:"end_date"`
	Summary   UsageSummary        `json:"summary"`
	ByUser    []UserUsageSummary  `json:"by_user,omitempty"`
	ByPlan    []PlanUsageSummary  `json:"by_plan,omitempty"`
}

// UsageSummary represents aggregate usage.
type UsageSummary struct {
	TotalRequests   int64 `json:"total_requests"`
	TotalUsers      int   `json:"total_users"`
	TotalKeys       int   `json:"total_keys"`
	RequestsToday   int64 `json:"requests_today"`
	RequestsWeek    int64 `json:"requests_week"`
	RequestsMonth   int64 `json:"requests_month"`
}

// UserUsageSummary represents usage for a single user.
type UserUsageSummary struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	PlanID        string `json:"plan_id"`
	Requests      int64  `json:"requests"`
	BytesIn       int64  `json:"bytes_in"`
	BytesOut      int64  `json:"bytes_out"`
	LastRequestAt string `json:"last_request_at,omitempty"`
}

// PlanUsageSummary represents usage by plan.
type PlanUsageSummary struct {
	PlanID    string `json:"plan_id"`
	PlanName  string `json:"plan_name"`
	UserCount int    `json:"user_count"`
	Requests  int64  `json:"requests"`
}

// GetUsage returns usage statistics.
//
//	@Summary		Get usage statistics
//	@Description	Get usage statistics with optional filters
//	@Tags			Admin - Usage
//	@Produce		json
//	@Param			user_id		query		string			false	"Filter by user ID"
//	@Param			period		query		string			false	"Period: day, week, month"	default(month)
//	@Param			start_date	query		string			false	"Start date (RFC3339)"
//	@Param			end_date	query		string			false	"End date (RFC3339)"
//	@Success		200			{object}	UsageResponse	"Usage statistics"
//	@Security		AdminAuth
//	@Router			/admin/usage [get]
func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}

	userID := r.URL.Query().Get("user_id")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// Calculate date range
	now := time.Now().UTC()
	var startDate, endDate time.Time

	if startDateStr != "" && endDateStr != "" {
		var err error
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			jsonapi.WriteValidationError(w, "start_date", "Invalid date format, expected RFC3339")
			return
		}
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			jsonapi.WriteValidationError(w, "end_date", "Invalid date format, expected RFC3339")
			return
		}
	} else {
		endDate = now
		switch period {
		case "day":
			startDate = now.AddDate(0, 0, -1)
		case "week":
			startDate = now.AddDate(0, 0, -7)
		case "month":
			startDate = now.AddDate(0, -1, 0)
		default:
			startDate = now.AddDate(0, -1, 0)
		}
	}

	response := UsageResponse{
		Period:    period,
		StartDate: startDate.Format(time.RFC3339),
		EndDate:   endDate.Format(time.RFC3339),
	}

	// Get user count
	userCount, _ := h.users.Count(r.Context())

	// If filtering by user
	if userID != "" {
		if h.usage != nil {
			summary, err := h.usage.GetSummary(r.Context(), userID, startDate, endDate)
			if err == nil {
				user, _ := h.users.Get(r.Context(), userID)
				response.ByUser = []UserUsageSummary{{
					UserID:   userID,
					Email:    user.Email,
					PlanID:   user.PlanID,
					Requests: summary.RequestCount,
					BytesIn:  summary.BytesIn,
					BytesOut: summary.BytesOut,
				}}
				response.Summary.TotalRequests = summary.RequestCount
			}
		}
	} else {
		// Aggregate stats
		response.Summary = UsageSummary{
			TotalUsers: userCount,
		}

		// Get usage for all users
		if h.usage != nil {
			users, _ := h.users.List(r.Context(), 1000, 0)
			var byUser []UserUsageSummary
			var totalRequests int64

			planStats := make(map[string]*PlanUsageSummary)

			for _, u := range users {
				summary, err := h.usage.GetSummary(r.Context(), u.ID, startDate, endDate)
				if err == nil && summary.RequestCount > 0 {
					byUser = append(byUser, UserUsageSummary{
						UserID:   u.ID,
						Email:    u.Email,
						PlanID:   u.PlanID,
						Requests: summary.RequestCount,
						BytesIn:  summary.BytesIn,
						BytesOut: summary.BytesOut,
					})
					totalRequests += summary.RequestCount

					// Aggregate by plan
					if _, ok := planStats[u.PlanID]; !ok {
						planName := u.PlanID
						if h.plans != nil {
							if plan, err := h.plans.Get(r.Context(), u.PlanID); err == nil {
								planName = plan.Name
							}
						}
						planStats[u.PlanID] = &PlanUsageSummary{
							PlanID:   u.PlanID,
							PlanName: planName,
						}
					}
					planStats[u.PlanID].UserCount++
					planStats[u.PlanID].Requests += summary.RequestCount
				}

				// Count keys
				keys, _ := h.keys.ListByUser(r.Context(), u.ID)
				response.Summary.TotalKeys += len(keys)
			}

			response.Summary.TotalRequests = totalRequests
			response.ByUser = byUser

			// Convert plan stats map to slice
			for _, ps := range planStats {
				response.ByPlan = append(response.ByPlan, *ps)
			}
		}
	}

	// Return as JSON:API meta response (usage stats aren't a typical resource)
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
		"period":     response.Period,
		"start_date": response.StartDate,
		"end_date":   response.EndDate,
		"summary":    response.Summary,
		"by_user":    response.ByUser,
		"by_plan":    response.ByPlan,
	})
}
