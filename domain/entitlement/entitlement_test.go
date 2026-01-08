package entitlement

import (
	"testing"
	"time"
)

func TestGetValue(t *testing.T) {
	ent := Entitlement{
		ID:           "ent-1",
		Name:         "api.streaming",
		DefaultValue: "true",
	}

	tests := []struct {
		name     string
		pe       PlanEntitlement
		expected string
	}{
		{
			name: "uses override value when set",
			pe: PlanEntitlement{
				EntitlementID: "ent-1",
				Value:         "false",
			},
			expected: "false",
		},
		{
			name: "uses default value when override empty",
			pe: PlanEntitlement{
				EntitlementID: "ent-1",
				Value:         "",
			},
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValue(tt.pe, ent)
			if result != tt.expected {
				t.Errorf("GetValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindEntitlement(t *testing.T) {
	entitlements := []Entitlement{
		{ID: "ent-1", Name: "api.streaming"},
		{ID: "ent-2", Name: "api.batch"},
		{ID: "ent-3", Name: "support.priority"},
	}

	tests := []struct {
		name    string
		id      string
		wantOK  bool
		wantEnt string
	}{
		{"finds existing entitlement", "ent-2", true, "api.batch"},
		{"returns false for missing", "ent-999", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ent, ok := FindEntitlement(entitlements, tt.id)
			if ok != tt.wantOK {
				t.Errorf("FindEntitlement() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && ent.Name != tt.wantEnt {
				t.Errorf("FindEntitlement() name = %q, want %q", ent.Name, tt.wantEnt)
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	entitlements := []Entitlement{
		{ID: "ent-1", Name: "api.streaming"},
		{ID: "ent-2", Name: "api.batch"},
	}

	tests := []struct {
		name   string
		search string
		wantOK bool
		wantID string
	}{
		{"finds by name", "api.batch", true, "ent-2"},
		{"returns false for missing", "api.nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ent, ok := FindByName(entitlements, tt.search)
			if ok != tt.wantOK {
				t.Errorf("FindByName() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && ent.ID != tt.wantID {
				t.Errorf("FindByName() id = %q, want %q", ent.ID, tt.wantID)
			}
		})
	}
}

func TestResolveForPlan(t *testing.T) {
	now := time.Now()

	entitlements := []Entitlement{
		{ID: "ent-1", Name: "api.streaming", DefaultValue: "true", ValueType: ValueTypeBoolean, Enabled: true, HeaderName: "X-Has-Streaming"},
		{ID: "ent-2", Name: "api.batch", DefaultValue: "true", ValueType: ValueTypeBoolean, Enabled: true},
		{ID: "ent-3", Name: "api.rate_limit", DefaultValue: "100", ValueType: ValueTypeNumber, Enabled: true, HeaderName: "X-Rate-Limit"},
		{ID: "ent-4", Name: "disabled.feature", DefaultValue: "true", ValueType: ValueTypeBoolean, Enabled: false},
	}

	planEntitlements := []PlanEntitlement{
		{ID: "pe-1", PlanID: "plan-pro", EntitlementID: "ent-1", Value: "", Enabled: true, CreatedAt: now},
		{ID: "pe-2", PlanID: "plan-pro", EntitlementID: "ent-2", Value: "", Enabled: true, CreatedAt: now},
		{ID: "pe-3", PlanID: "plan-pro", EntitlementID: "ent-3", Value: "500", Enabled: true, CreatedAt: now}, // Override
		{ID: "pe-4", PlanID: "plan-pro", EntitlementID: "ent-4", Value: "", Enabled: true, CreatedAt: now},   // Disabled entitlement
		{ID: "pe-5", PlanID: "plan-free", EntitlementID: "ent-1", Value: "", Enabled: true, CreatedAt: now},  // Different plan
		{ID: "pe-6", PlanID: "plan-pro", EntitlementID: "ent-2", Value: "", Enabled: false, CreatedAt: now},  // Disabled PE
	}

	result := ResolveForPlan("plan-pro", entitlements, planEntitlements)

	// Should have 3 entitlements: ent-1, ent-2, ent-3 (ent-4 is disabled, pe-6 is disabled)
	if len(result) != 3 {
		t.Errorf("ResolveForPlan() returned %d entitlements, want 3", len(result))
	}

	// Check streaming entitlement
	if !HasEntitlement(result, "api.streaming") {
		t.Error("ResolveForPlan() missing api.streaming")
	}

	// Check rate limit override value
	val, ok := GetEntitlementValue(result, "api.rate_limit")
	if !ok {
		t.Error("ResolveForPlan() missing api.rate_limit")
	}
	if val != "500" {
		t.Errorf("ResolveForPlan() api.rate_limit = %q, want %q", val, "500")
	}
}

func TestHasEntitlement(t *testing.T) {
	userEnts := []UserEntitlement{
		{Name: "api.streaming", Value: "true"},
		{Name: "api.batch", Value: "true"},
	}

	tests := []struct {
		name   string
		search string
		want   bool
	}{
		{"has existing", "api.streaming", true},
		{"missing entitlement", "api.nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasEntitlement(userEnts, tt.search); got != tt.want {
				t.Errorf("HasEntitlement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEntitlementValue(t *testing.T) {
	userEnts := []UserEntitlement{
		{Name: "api.streaming", Value: "true"},
		{Name: "api.rate_limit", Value: "500"},
	}

	tests := []struct {
		name      string
		search    string
		wantValue string
		wantOK    bool
	}{
		{"gets existing value", "api.rate_limit", "500", true},
		{"returns false for missing", "api.nonexistent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := GetEntitlementValue(userEnts, tt.search)
			if ok != tt.wantOK {
				t.Errorf("GetEntitlementValue() ok = %v, want %v", ok, tt.wantOK)
			}
			if val != tt.wantValue {
				t.Errorf("GetEntitlementValue() = %q, want %q", val, tt.wantValue)
			}
		})
	}
}

func TestToHeaders(t *testing.T) {
	userEnts := []UserEntitlement{
		{Name: "api.streaming", Value: "true", HeaderName: "X-Has-Streaming"},
		{Name: "api.batch", Value: "true", HeaderName: ""}, // No header
		{Name: "api.rate_limit", Value: "500", HeaderName: "X-Rate-Limit"},
	}

	headers := ToHeaders(userEnts)

	if len(headers) != 2 {
		t.Errorf("ToHeaders() returned %d headers, want 2", len(headers))
	}

	if headers["X-Has-Streaming"] != "true" {
		t.Errorf("ToHeaders()[X-Has-Streaming] = %q, want %q", headers["X-Has-Streaming"], "true")
	}

	if headers["X-Rate-Limit"] != "500" {
		t.Errorf("ToHeaders()[X-Rate-Limit] = %q, want %q", headers["X-Rate-Limit"], "500")
	}

	if _, exists := headers[""]; exists {
		t.Error("ToHeaders() should not include empty header names")
	}
}

func TestFilterByCategory(t *testing.T) {
	entitlements := []Entitlement{
		{ID: "1", Name: "api.streaming", Category: CategoryAPI},
		{ID: "2", Name: "api.batch", Category: CategoryAPI},
		{ID: "3", Name: "support.priority", Category: CategorySupport},
		{ID: "4", Name: "feature.dark_mode", Category: CategoryFeature},
	}

	apiEnts := FilterByCategory(entitlements, CategoryAPI)
	if len(apiEnts) != 2 {
		t.Errorf("FilterByCategory(API) returned %d, want 2", len(apiEnts))
	}

	supportEnts := FilterByCategory(entitlements, CategorySupport)
	if len(supportEnts) != 1 {
		t.Errorf("FilterByCategory(Support) returned %d, want 1", len(supportEnts))
	}

	integrationEnts := FilterByCategory(entitlements, CategoryIntegration)
	if len(integrationEnts) != 0 {
		t.Errorf("FilterByCategory(Integration) returned %d, want 0", len(integrationEnts))
	}
}
