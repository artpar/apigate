package usage

import (
	"testing"
	"time"
)

func TestEvent_IsExternal(t *testing.T) {
	tests := []struct {
		name   string
		source EventSource
		want   bool
	}{
		{"proxy source", SourceProxy, false},
		{"external source", SourceExternal, true},
		{"empty source", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Source: tt.source}
			got := e.IsExternal()
			if got != tt.want {
				t.Errorf("IsExternal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_EffectiveQuantity(t *testing.T) {
	tests := []struct {
		name     string
		source   EventSource
		quantity float64
		want     float64
	}{
		{"proxy event returns 1", SourceProxy, 10.0, 1.0},
		{"external event with quantity", SourceExternal, 5.5, 5.5},
		{"external event with zero quantity", SourceExternal, 0, 1.0},
		{"external event with negative quantity", SourceExternal, -5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Source: tt.source, Quantity: tt.quantity}
			got := e.EffectiveQuantity()
			if got != tt.want {
				t.Errorf("EffectiveQuantity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_EffectiveCost(t *testing.T) {
	tests := []struct {
		name           string
		source         EventSource
		quantity       float64
		costMultiplier float64
		want           float64
	}{
		{"proxy event default cost", SourceProxy, 1.0, 0, 1.0},
		{"proxy event with multiplier", SourceProxy, 1.0, 2.5, 2.5},
		{"external event with quantity and multiplier", SourceExternal, 10.0, 2.0, 20.0},
		{"external event no multiplier", SourceExternal, 5.0, 0, 5.0},
		{"external event negative quantity becomes 1", SourceExternal, -1.0, 2.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{
				Source:         tt.source,
				Quantity:       tt.quantity,
				CostMultiplier: tt.costMultiplier,
			}
			got := e.EffectiveCost()
			if got != tt.want {
				t.Errorf("EffectiveCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewProxyEvent(t *testing.T) {
	now := time.Now()
	e := NewProxyEvent(
		"evt-123",
		"key-456",
		"user-789",
		"POST",
		"/api/users",
		201,
		50,
		1024,
		2048,
		1.5,
		"192.168.1.1",
		"test-agent",
		now,
	)

	if e.ID != "evt-123" {
		t.Errorf("ID = %v, want evt-123", e.ID)
	}
	if e.KeyID != "key-456" {
		t.Errorf("KeyID = %v, want key-456", e.KeyID)
	}
	if e.UserID != "user-789" {
		t.Errorf("UserID = %v, want user-789", e.UserID)
	}
	if e.Method != "POST" {
		t.Errorf("Method = %v, want POST", e.Method)
	}
	if e.Path != "/api/users" {
		t.Errorf("Path = %v, want /api/users", e.Path)
	}
	if e.StatusCode != 201 {
		t.Errorf("StatusCode = %v, want 201", e.StatusCode)
	}
	if e.LatencyMs != 50 {
		t.Errorf("LatencyMs = %v, want 50", e.LatencyMs)
	}
	if e.RequestBytes != 1024 {
		t.Errorf("RequestBytes = %v, want 1024", e.RequestBytes)
	}
	if e.ResponseBytes != 2048 {
		t.Errorf("ResponseBytes = %v, want 2048", e.ResponseBytes)
	}
	if e.CostMultiplier != 1.5 {
		t.Errorf("CostMultiplier = %v, want 1.5", e.CostMultiplier)
	}
	if e.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %v, want 192.168.1.1", e.IPAddress)
	}
	if e.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %v, want test-agent", e.UserAgent)
	}
	if e.Source != SourceProxy {
		t.Errorf("Source = %v, want proxy", e.Source)
	}
	if e.Quantity != 1.0 {
		t.Errorf("Quantity = %v, want 1.0", e.Quantity)
	}
	if !e.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, now)
	}
}

func TestNewExternalEvent(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		now := time.Now()
		metadata := map[string]string{"env": "production"}
		e := NewExternalEvent(
			"evt-ext-123",
			"user-456",
			"compute.minutes",
			"res-789",
			"deployment",
			"hoster-service",
			15.5,
			metadata,
			now,
		)

		if e.ID != "evt-ext-123" {
			t.Errorf("ID = %v, want evt-ext-123", e.ID)
		}
		if e.UserID != "user-456" {
			t.Errorf("UserID = %v, want user-456", e.UserID)
		}
		if e.EventType != "compute.minutes" {
			t.Errorf("EventType = %v, want compute.minutes", e.EventType)
		}
		if e.ResourceID != "res-789" {
			t.Errorf("ResourceID = %v, want res-789", e.ResourceID)
		}
		if e.ResourceType != "deployment" {
			t.Errorf("ResourceType = %v, want deployment", e.ResourceType)
		}
		if e.SourceName != "hoster-service" {
			t.Errorf("SourceName = %v, want hoster-service", e.SourceName)
		}
		if e.Quantity != 15.5 {
			t.Errorf("Quantity = %v, want 15.5", e.Quantity)
		}
		if e.Source != SourceExternal {
			t.Errorf("Source = %v, want external", e.Source)
		}
		if e.Metadata["env"] != "production" {
			t.Errorf("Metadata[env] = %v, want production", e.Metadata["env"])
		}
		if !e.Timestamp.Equal(now) {
			t.Errorf("Timestamp = %v, want %v", e.Timestamp, now)
		}
	})

	t.Run("defaults zero quantity to 1", func(t *testing.T) {
		e := NewExternalEvent("id", "user", "type", "", "", "", 0, nil, time.Now())
		if e.Quantity != 1.0 {
			t.Errorf("Quantity = %v, want 1.0 for zero input", e.Quantity)
		}
	})

	t.Run("defaults negative quantity to 1", func(t *testing.T) {
		e := NewExternalEvent("id", "user", "type", "", "", "", -5.0, nil, time.Now())
		if e.Quantity != 1.0 {
			t.Errorf("Quantity = %v, want 1.0 for negative input", e.Quantity)
		}
	})

	t.Run("defaults zero timestamp to now", func(t *testing.T) {
		before := time.Now().UTC()
		e := NewExternalEvent("id", "user", "type", "", "", "", 1.0, nil, time.Time{})
		after := time.Now().UTC()

		if e.Timestamp.Before(before) || e.Timestamp.After(after) {
			t.Errorf("Timestamp = %v, expected between %v and %v", e.Timestamp, before, after)
		}
	})
}

func TestIsValidEventType(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		// Valid built-in types
		{"api.request", true},
		{"deployment.created", true},
		{"deployment.started", true},
		{"deployment.stopped", true},
		{"deployment.deleted", true},
		{"compute.minutes", true},
		{"storage.gb_hours", true},
		{"bandwidth.gb", true},

		// Valid custom types
		{"custom.my_event", true},
		{"custom.another.event", true},

		// Invalid types
		{"unknown.type", false},
		{"", false},
		{"custom", false},    // Not long enough for custom.
		{"custom.", false},   // Exactly 7 chars, needs > 7 for custom. prefix
		{"api.invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := IsValidEventType(tt.eventType)
			if got != tt.want {
				t.Errorf("IsValidEventType(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
