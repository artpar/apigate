package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/domain/usage"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// mockUsageStore implements ports.UsageStore for testing
type mockUsageStore struct {
	events []usage.Event
}

func (m *mockUsageStore) RecordBatch(ctx context.Context, events []usage.Event) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *mockUsageStore) GetSummary(ctx context.Context, userID string, start, end time.Time) (usage.Summary, error) {
	return usage.Summary{UserID: userID}, nil
}

func (m *mockUsageStore) GetHistory(ctx context.Context, userID string, periods int) ([]usage.Summary, error) {
	return nil, nil
}

func (m *mockUsageStore) GetRecentRequests(ctx context.Context, userID string, limit int) ([]usage.Event, error) {
	var result []usage.Event
	for _, e := range m.events {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

// mockUserStore implements ports.UserStore for testing
type mockUserStore struct {
	users map[string]ports.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users: make(map[string]ports.User),
	}
}

func (m *mockUserStore) Get(ctx context.Context, id string) (ports.User, error) {
	user, ok := m.users[id]
	if !ok {
		return ports.User{}, ports.ErrNotFound
	}
	return user, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return ports.User{}, ports.ErrNotFound
}

func (m *mockUserStore) GetByStripeID(ctx context.Context, stripeID string) (ports.User, error) {
	for _, u := range m.users {
		if u.StripeID == stripeID {
			return u, nil
		}
	}
	return ports.User{}, ports.ErrNotFound
}

func (m *mockUserStore) Create(ctx context.Context, u ports.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserStore) Update(ctx context.Context, u ports.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserStore) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	var result []ports.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

func (m *mockUserStore) Count(ctx context.Context) (int, error) {
	return len(m.users), nil
}

// Helper to get meta value from response
func getMetaValue(result map[string]any, key string) any {
	meta, ok := result["meta"].(map[string]any)
	if !ok {
		return nil
	}
	return meta[key]
}

// Helper to get error code from response
func getMeterErrorCode(result map[string]any) string {
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		return ""
	}
	errData, ok := errors[0].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := errData["code"].(string)
	return code
}

func TestMeterHandler_SubmitEvents_Success(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{
		ID:    "usr_123",
		Email: "test@example.com",
	})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "usr_123",
					"event_type": "deployment.started",
					"resource_id": "depl_456",
					"resource_type": "deployment",
					"quantity": 1,
					"metadata": {"region": "us-east-1"}
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rec.Code)
		t.Logf("Response: %s", rec.Body.String())
	}

	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)

	accepted := getMetaValue(result, "accepted")
	if accepted != float64(1) {
		t.Errorf("Expected accepted=1, got %v", accepted)
	}

	rejected := getMetaValue(result, "rejected")
	if rejected != float64(0) {
		t.Errorf("Expected rejected=0, got %v", rejected)
	}

	// Verify event was stored
	if len(usageStore.events) != 1 {
		t.Errorf("Expected 1 event stored, got %d", len(usageStore.events))
	}
}

func TestMeterHandler_SubmitEvents_MultipleEvents(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{ID: "usr_123"})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "usr_123",
					"event_type": "deployment.started"
				}
			},
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_002",
					"user_id": "usr_123",
					"event_type": "deployment.stopped"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rec.Code)
	}

	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)

	if getMetaValue(result, "accepted") != float64(2) {
		t.Errorf("Expected accepted=2, got %v", getMetaValue(result, "accepted"))
	}
}

func TestMeterHandler_SubmitEvents_InvalidEventType(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{ID: "usr_123"})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "usr_123",
					"event_type": "invalid.event.type"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	// All events rejected returns 422
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rec.Code)
	}
}

func TestMeterHandler_SubmitEvents_CustomEventType(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{ID: "usr_123"})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	// Custom event types (prefixed with "custom.") should be allowed
	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "usr_123",
					"event_type": "custom.my_event"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rec.Code)
	}

	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)

	if getMetaValue(result, "accepted") != float64(1) {
		t.Errorf("Expected accepted=1, got %v", getMetaValue(result, "accepted"))
	}
}

func TestMeterHandler_SubmitEvents_DuplicateEvent(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{ID: "usr_123"})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_dup",
					"user_id": "usr_123",
					"event_type": "deployment.started"
				}
			}
		]
	}`

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req1.Header.Set("Content-Type", "application/vnd.api+json")
	rec1 := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusAccepted {
		t.Errorf("First request: expected 202, got %d", rec1.Code)
	}

	// Second request with same event ID
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req2.Header.Set("Content-Type", "application/vnd.api+json")
	rec2 := httptest.NewRecorder()
	handler.Router().ServeHTTP(rec2, req2)

	// Should still return 422 since all events rejected
	if rec2.Code != http.StatusUnprocessableEntity {
		t.Errorf("Second request: expected 422, got %d", rec2.Code)
	}
}

func TestMeterHandler_SubmitEvents_UserNotFound(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	// Don't create user

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "nonexistent_user",
					"event_type": "deployment.started"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rec.Code)
	}
}

func TestMeterHandler_SubmitEvents_MissingRequiredFields(t *testing.T) {
	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Logger: zerolog.Nop(),
	})

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing id",
			body: `{
				"data": [{
					"type": "usage_events",
					"attributes": {
						"user_id": "usr_123",
						"event_type": "deployment.started"
					}
				}]
			}`,
		},
		{
			name: "missing user_id",
			body: `{
				"data": [{
					"type": "usage_events",
					"attributes": {
						"id": "evt_001",
						"event_type": "deployment.started"
					}
				}]
			}`,
		},
		{
			name: "missing event_type",
			body: `{
				"data": [{
					"type": "usage_events",
					"attributes": {
						"id": "evt_001",
						"user_id": "usr_123"
					}
				}]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/vnd.api+json")
			rec := httptest.NewRecorder()

			handler.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("Expected status 422, got %d", rec.Code)
			}
		})
	}
}

func TestMeterHandler_SubmitEvents_EmptyData(t *testing.T) {
	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Logger: zerolog.Nop(),
	})

	body := `{"data": []}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rec.Code)
	}
}

func TestMeterHandler_SubmitEvents_InvalidJSON(t *testing.T) {
	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Logger: zerolog.Nop(),
	})

	body := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestMeterHandler_SubmitEvents_PartialSuccess(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()
	userStore.Create(context.Background(), ports.User{ID: "usr_123"})

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	body := `{
		"data": [
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_001",
					"user_id": "usr_123",
					"event_type": "deployment.started"
				}
			},
			{
				"type": "usage_events",
				"attributes": {
					"id": "evt_002",
					"user_id": "usr_123",
					"event_type": "invalid.type"
				}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	// Partial success should return 202
	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rec.Code)
	}

	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)

	if getMetaValue(result, "accepted") != float64(1) {
		t.Errorf("Expected accepted=1, got %v", getMetaValue(result, "accepted"))
	}
	if getMetaValue(result, "rejected") != float64(1) {
		t.Errorf("Expected rejected=1, got %v", getMetaValue(result, "rejected"))
	}
}

func TestMeterHandler_ListEvents(t *testing.T) {
	usageStore := &mockUsageStore{}
	userStore := newMockUserStore()

	// Add some events
	usageStore.events = []usage.Event{
		{
			ID:        "evt_001",
			UserID:    "usr_123",
			EventType: "deployment.started",
			Timestamp: time.Now(),
		},
		{
			ID:        "evt_002",
			UserID:    "usr_123",
			EventType: "deployment.stopped",
			Timestamp: time.Now(),
		},
	}

	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  usageStore,
		Users:  userStore,
		Logger: zerolog.Nop(),
	})

	req := httptest.NewRequest(http.MethodGet, "/?user_id=usr_123", nil)
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
		t.Logf("Response: %s", rec.Body.String())
	}

	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)

	data, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("Expected data array in response")
	}

	if len(data) != 2 {
		t.Errorf("Expected 2 events, got %d", len(data))
	}
}

func TestMeterHandler_ListEvents_MissingUserID(t *testing.T) {
	handler := admin.NewMeterHandler(admin.MeterHandlerConfig{
		Usage:  &mockUsageStore{},
		Logger: zerolog.Nop(),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", rec.Code)
	}
}

func TestUsage_IsValidEventType(t *testing.T) {
	tests := []struct {
		eventType string
		expected  bool
	}{
		{"api.request", true},
		{"deployment.started", true},
		{"deployment.stopped", true},
		{"deployment.created", true},
		{"deployment.deleted", true},
		{"compute.minutes", true},
		{"storage.gb_hours", true},
		{"bandwidth.gb", true},
		{"custom.anything", true},
		{"custom.my_special_event", true},
		{"invalid.type", false},
		{"random", false},
		{"", false},
		{"custom", false}, // "custom" alone is not valid, needs custom.*
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := usage.IsValidEventType(tt.eventType)
			if got != tt.expected {
				t.Errorf("IsValidEventType(%q) = %v, want %v", tt.eventType, got, tt.expected)
			}
		})
	}
}

func TestUsage_NewExternalEvent(t *testing.T) {
	metadata := map[string]string{"key": "value"}
	timestamp := time.Now().UTC()

	event := usage.NewExternalEvent(
		"evt_001",
		"usr_123",
		"deployment.started",
		"depl_456",
		"deployment",
		"test-service",
		2.5,
		metadata,
		timestamp,
	)

	if event.ID != "evt_001" {
		t.Errorf("ID = %q, want %q", event.ID, "evt_001")
	}
	if event.UserID != "usr_123" {
		t.Errorf("UserID = %q, want %q", event.UserID, "usr_123")
	}
	if event.EventType != "deployment.started" {
		t.Errorf("EventType = %q, want %q", event.EventType, "deployment.started")
	}
	if event.ResourceID != "depl_456" {
		t.Errorf("ResourceID = %q, want %q", event.ResourceID, "depl_456")
	}
	if event.ResourceType != "deployment" {
		t.Errorf("ResourceType = %q, want %q", event.ResourceType, "deployment")
	}
	if event.SourceName != "test-service" {
		t.Errorf("SourceName = %q, want %q", event.SourceName, "test-service")
	}
	if event.Quantity != 2.5 {
		t.Errorf("Quantity = %f, want %f", event.Quantity, 2.5)
	}
	if event.Source != usage.SourceExternal {
		t.Errorf("Source = %q, want %q", event.Source, usage.SourceExternal)
	}
	if event.IsExternal() != true {
		t.Errorf("IsExternal() = false, want true")
	}
}

func TestUsage_NewExternalEvent_DefaultQuantity(t *testing.T) {
	event := usage.NewExternalEvent(
		"evt_001",
		"usr_123",
		"deployment.started",
		"", "", "",
		0, // Should default to 1.0
		nil,
		time.Time{}, // Should default to now
	)

	if event.Quantity != 1.0 {
		t.Errorf("Quantity = %f, want 1.0 (default)", event.Quantity)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero (should default to now)")
	}
}

func TestUsage_EffectiveQuantity(t *testing.T) {
	// External event with quantity
	external := usage.NewExternalEvent("evt_001", "usr_123", "compute.minutes", "", "", "", 5.0, nil, time.Now())
	if external.EffectiveQuantity() != 5.0 {
		t.Errorf("External event EffectiveQuantity() = %f, want 5.0", external.EffectiveQuantity())
	}

	// Proxy event (should always be 1.0)
	proxy := usage.NewProxyEvent("evt_002", "key_123", "usr_123", "GET", "/api/test", 200, 100, 50, 100, 1.0, "127.0.0.1", "test-agent", time.Now())
	if proxy.EffectiveQuantity() != 1.0 {
		t.Errorf("Proxy event EffectiveQuantity() = %f, want 1.0", proxy.EffectiveQuantity())
	}
}

func TestUsage_EffectiveCost(t *testing.T) {
	// External event with quantity and cost multiplier
	event := usage.NewExternalEvent("evt_001", "usr_123", "compute.minutes", "", "", "", 5.0, nil, time.Now())
	event.CostMultiplier = 0.1 // 10 minutes = 1 request equivalent

	if event.EffectiveCost() != 0.5 {
		t.Errorf("EffectiveCost() = %f, want 0.5", event.EffectiveCost())
	}
}

func TestRequireMeterScope_Authorized(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := admin.RequireMeterScope(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := context.WithValue(req.Context(), "scopes", []string{"meter:write"})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireMeterScope_Forbidden(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := admin.RequireMeterScope(next)

	tests := []struct {
		name   string
		scopes []string
	}{
		{"empty scopes", []string{}},
		{"wrong scope", []string{"read", "write"}},
		{"no scopes in context", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.scopes != nil {
				ctx := context.WithValue(req.Context(), "scopes", tt.scopes)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if called {
				t.Error("Handler should not have been called")
			}
			if rec.Code != http.StatusForbidden {
				t.Errorf("Expected status 403, got %d", rec.Code)
			}
		})
	}
}
