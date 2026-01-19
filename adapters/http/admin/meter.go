// Package admin provides HTTP handlers for the Admin API.
package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/pkg/jsonapi"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// JSON:API resource type constant for usage events
const TypeUsageEvent = "usage_events"

// MeterHandler provides metering API endpoints.
type MeterHandler struct {
	usage  ports.UsageStore
	users  ports.UserStore
	logger zerolog.Logger
	// idempotencyStore tracks processed event IDs to prevent duplicates.
	// In production, this should be backed by a database or cache.
	idempotencyStore map[string]time.Time
}

// MeterHandlerConfig contains dependencies for the meter handler.
type MeterHandlerConfig struct {
	Usage  ports.UsageStore
	Users  ports.UserStore
	Logger zerolog.Logger
}

// NewMeterHandler creates a new metering API handler.
func NewMeterHandler(cfg MeterHandlerConfig) *MeterHandler {
	return &MeterHandler{
		usage:            cfg.Usage,
		users:            cfg.Users,
		logger:           cfg.Logger,
		idempotencyStore: make(map[string]time.Time),
	}
}

// Router returns the metering API router.
func (h *MeterHandler) Router() chi.Router {
	r := chi.NewRouter()

	// POST /api/v1/meter - Submit usage events (requires meter:write scope)
	r.Post("/", h.SubmitEvents)

	// GET /api/v1/meter - Query usage events (admin only)
	r.Get("/", h.ListEvents)

	return r
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// UsageEventInput represents a single usage event in the request.
type UsageEventInput struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	EventType    string            `json:"event_type"`
	ResourceID   string            `json:"resource_id,omitempty"`
	ResourceType string            `json:"resource_type,omitempty"`
	Quantity     float64           `json:"quantity,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    string            `json:"timestamp,omitempty"`
}

// SubmitEventsRequest represents the request body for submitting events.
type SubmitEventsRequest struct {
	Data []struct {
		Type       string          `json:"type"`
		Attributes UsageEventInput `json:"attributes"`
	} `json:"data"`
}

// EventError represents an error for a specific event in the batch.
type EventError struct {
	Index  int    `json:"index"`
	ID     string `json:"id,omitempty"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// -----------------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------------

// SubmitEvents handles POST /api/v1/meter
//
//	@Summary		Submit usage events
//	@Description	Submit one or more usage events for billing
//	@Tags			Metering
//	@Accept			json
//	@Produce		json
//	@Param			events	body		SubmitEventsRequest	true	"Usage events"
//	@Success		202		{object}	object				"Events accepted"
//	@Failure		400		{object}	jsonapi.Document	"Bad request"
//	@Failure		403		{object}	jsonapi.Document	"Forbidden - insufficient scope"
//	@Failure		422		{object}	jsonapi.Document	"Validation failed"
//	@Security		ServiceAPIKey
//	@Router			/api/v1/meter [post]
func (h *MeterHandler) SubmitEvents(w http.ResponseWriter, r *http.Request) {
	// Get source name from context (set by auth middleware)
	sourceName := r.Header.Get("X-Service-Name")
	if sourceName == "" {
		sourceName = "external"
	}

	// Parse request body
	var req SubmitEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonapi.WriteBadRequest(w, "Invalid JSON body: "+err.Error())
		return
	}

	if len(req.Data) == 0 {
		jsonapi.WriteValidationError(w, "data", "At least one event is required")
		return
	}

	// Limit batch size
	if len(req.Data) > 1000 {
		jsonapi.WriteValidationError(w, "data", "Maximum 1000 events per batch")
		return
	}

	// Process events
	var (
		accepted     int
		rejected     int
		eventErrors  []EventError
		eventsToSave []usage.Event
	)

	now := time.Now().UTC()
	maxTimestampAge := 7 * 24 * time.Hour // 7 days

	for i, item := range req.Data {
		input := item.Attributes

		// Validate required fields
		if input.ID == "" {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				Code:   "validation_error",
				Detail: "id is required",
			})
			rejected++
			continue
		}

		if input.UserID == "" {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				ID:     input.ID,
				Code:   "validation_error",
				Detail: "user_id is required",
			})
			rejected++
			continue
		}

		if input.EventType == "" {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				ID:     input.ID,
				Code:   "validation_error",
				Detail: "event_type is required",
			})
			rejected++
			continue
		}

		// Validate event type
		if !usage.IsValidEventType(input.EventType) {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				ID:     input.ID,
				Code:   "invalid_event_type",
				Detail: fmt.Sprintf("Event type '%s' is not recognized", input.EventType),
			})
			rejected++
			continue
		}

		// Check for duplicate (idempotency)
		if _, exists := h.idempotencyStore[input.ID]; exists {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				ID:     input.ID,
				Code:   "duplicate_event",
				Detail: "Event with this ID already processed",
			})
			rejected++
			continue
		}

		// Validate user exists
		if h.users != nil {
			if _, err := h.users.Get(r.Context(), input.UserID); err != nil {
				if err == ports.ErrNotFound {
					eventErrors = append(eventErrors, EventError{
						Index:  i,
						ID:     input.ID,
						Code:   "user_not_found",
						Detail: fmt.Sprintf("User '%s' does not exist", input.UserID),
					})
					rejected++
					continue
				}
				// Log other errors but don't reject
				h.logger.Error().Err(err).Str("user_id", input.UserID).Msg("Error checking user")
			}
		}

		// Parse timestamp
		var eventTimestamp time.Time
		if input.Timestamp != "" {
			var err error
			eventTimestamp, err = time.Parse(time.RFC3339, input.Timestamp)
			if err != nil {
				eventErrors = append(eventErrors, EventError{
					Index:  i,
					ID:     input.ID,
					Code:   "invalid_timestamp",
					Detail: "Timestamp must be in RFC3339 format",
				})
				rejected++
				continue
			}

			// Check timestamp is not in future
			if eventTimestamp.After(now.Add(time.Minute)) {
				eventErrors = append(eventErrors, EventError{
					Index:  i,
					ID:     input.ID,
					Code:   "invalid_timestamp",
					Detail: "Timestamp cannot be in the future",
				})
				rejected++
				continue
			}

			// Check timestamp is not too old
			if now.Sub(eventTimestamp) > maxTimestampAge {
				eventErrors = append(eventErrors, EventError{
					Index:  i,
					ID:     input.ID,
					Code:   "invalid_timestamp",
					Detail: fmt.Sprintf("Timestamp cannot be older than %d days", int(maxTimestampAge.Hours()/24)),
				})
				rejected++
				continue
			}
		} else {
			eventTimestamp = now
		}

		// Validate quantity
		quantity := input.Quantity
		if quantity < 0 {
			eventErrors = append(eventErrors, EventError{
				Index:  i,
				ID:     input.ID,
				Code:   "invalid_quantity",
				Detail: "Quantity must be >= 0",
			})
			rejected++
			continue
		}
		if quantity == 0 {
			quantity = 1.0
		}

		// Create event
		event := usage.NewExternalEvent(
			input.ID,
			input.UserID,
			input.EventType,
			input.ResourceID,
			input.ResourceType,
			sourceName,
			quantity,
			input.Metadata,
			eventTimestamp,
		)

		eventsToSave = append(eventsToSave, event)
		h.idempotencyStore[input.ID] = now // Mark as processed
		accepted++
	}

	// Save events to storage
	if len(eventsToSave) > 0 && h.usage != nil {
		if err := h.usage.RecordBatch(r.Context(), eventsToSave); err != nil {
			h.logger.Error().Err(err).Int("count", len(eventsToSave)).Msg("Failed to save usage events")
			// Don't fail the whole request - events are marked as processed
		}
	}

	// Return response
	// If all events rejected with errors, return 422
	if accepted == 0 && rejected > 0 {
		jsonapi.WriteError(w, jsonapi.ErrValidation("data", "All events failed validation"))
		return
	}

	// Otherwise return 202 Accepted with summary
	jsonapi.WriteAccepted(w, jsonapi.Meta{
		"accepted": accepted,
		"rejected": rejected,
		"errors":   eventErrors,
	})
}

// ListEvents handles GET /api/v1/meter
//
//	@Summary		Query usage events
//	@Description	Query submitted usage events (admin only)
//	@Tags			Metering
//	@Produce		json
//	@Param			user_id		query		string	false	"Filter by user ID"
//	@Param			event_type	query		string	false	"Filter by event type"
//	@Param			start_date	query		string	false	"Events after this time (RFC3339)"
//	@Param			end_date	query		string	false	"Events before this time (RFC3339)"
//	@Param			page[number]	query	int		false	"Page number"	default(1)
//	@Param			page[size]		query	int		false	"Page size"		default(50)
//	@Success		200		{object}	jsonapi.Document	"Usage events"
//	@Security		AdminAuth
//	@Router			/api/v1/meter [get]
func (h *MeterHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	userID := r.URL.Query().Get("user_id")
	// eventType := r.URL.Query().Get("event_type")
	// startDateStr := r.URL.Query().Get("start_date")
	// endDateStr := r.URL.Query().Get("end_date")

	// For now, just return recent events for the user
	if h.usage == nil {
		jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
			"events": []interface{}{},
			"total":  0,
		})
		return
	}

	if userID == "" {
		jsonapi.WriteValidationError(w, "user_id", "user_id query parameter is required")
		return
	}

	// Get recent events
	events, err := h.usage.GetRecentRequests(r.Context(), userID, 50)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get usage events")
		jsonapi.WriteInternalError(w, "Failed to retrieve usage events")
		return
	}

	// Convert to JSON:API resources
	resources := make([]jsonapi.Resource, 0, len(events))
	for _, e := range events {
		attrs := map[string]any{
			"user_id":    e.UserID,
			"event_type": e.EventType,
			"timestamp":  e.Timestamp.Format(time.RFC3339),
		}

		// Add proxy-specific fields if present
		if e.Method != "" {
			attrs["method"] = e.Method
			attrs["path"] = e.Path
			attrs["status_code"] = e.StatusCode
			attrs["latency_ms"] = e.LatencyMs
		}

		// Add external event fields if present
		if e.ResourceID != "" {
			attrs["resource_id"] = e.ResourceID
			attrs["resource_type"] = e.ResourceType
		}
		if e.Quantity > 0 {
			attrs["quantity"] = e.Quantity
		}
		if e.SourceName != "" {
			attrs["source"] = e.SourceName
		}
		if len(e.Metadata) > 0 {
			attrs["metadata"] = e.Metadata
		}

		resources = append(resources, jsonapi.Resource{
			Type:       TypeUsageEvent,
			ID:         e.ID,
			Attributes: attrs,
		})
	}

	jsonapi.WriteCollection(w, http.StatusOK, resources, nil)
}

// -----------------------------------------------------------------------------
// Middleware helpers
// -----------------------------------------------------------------------------

// RequireMeterScope is middleware that checks for meter:write scope.
// This should be applied to the metering endpoints.
func RequireMeterScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for meter:write scope in context
		// This would be set by the API key auth middleware
		scopes, ok := r.Context().Value("scopes").([]string)
		if !ok {
			jsonapi.WriteError(w, jsonapi.ErrInsufficientScope("meter:write"))
			return
		}

		if !slices.Contains(scopes, "meter:write") {
			jsonapi.WriteError(w, jsonapi.ErrInsufficientScope("meter:write"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
