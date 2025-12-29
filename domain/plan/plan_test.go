package plan_test

import (
	"testing"

	"github.com/artpar/apigate/domain/plan"
)

func TestGetCostMultiplier_DefaultMultiplier(t *testing.T) {
	endpoints := []plan.Endpoint{}
	result := plan.GetCostMultiplier(endpoints, "GET", "/api/users")
	if result != 1.0 {
		t.Errorf("expected 1.0, got %f", result)
	}
}

func TestGetCostMultiplier_ExactPathMatch(t *testing.T) {
	endpoints := []plan.Endpoint{
		{Path: "/api/expensive", CostMultiplier: 10.0},
		{Path: "/api/cheap", CostMultiplier: 0.5},
	}

	tests := []struct {
		path string
		want float64
	}{
		{"/api/expensive", 10.0},
		{"/api/cheap", 0.5},
		{"/api/other", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := plan.GetCostMultiplier(endpoints, "GET", tt.path)
			if result != tt.want {
				t.Errorf("got %f, want %f", result, tt.want)
			}
		})
	}
}

func TestGetCostMultiplier_PrefixMatch(t *testing.T) {
	endpoints := []plan.Endpoint{
		{Path: "/api/v1/*", CostMultiplier: 2.0},
		{Path: "/api/v2/*", CostMultiplier: 3.0},
	}

	tests := []struct {
		path string
		want float64
	}{
		{"/api/v1/users", 2.0},
		{"/api/v1/products", 2.0},
		{"/api/v1/", 2.0},
		{"/api/v2/data", 3.0},
		{"/api/v3/stuff", 1.0}, // no match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := plan.GetCostMultiplier(endpoints, "GET", tt.path)
			if result != tt.want {
				t.Errorf("got %f, want %f", result, tt.want)
			}
		})
	}
}

func TestGetCostMultiplier_MethodFiltering(t *testing.T) {
	endpoints := []plan.Endpoint{
		{Path: "/api/data", Method: "POST", CostMultiplier: 5.0},
		{Path: "/api/data", Method: "DELETE", CostMultiplier: 10.0},
	}

	tests := []struct {
		method string
		want   float64
	}{
		{"POST", 5.0},
		{"DELETE", 10.0},
		{"GET", 1.0},    // no match
		{"PUT", 1.0},    // no match
		{"PATCH", 1.0},  // no match
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := plan.GetCostMultiplier(endpoints, tt.method, "/api/data")
			if result != tt.want {
				t.Errorf("got %f, want %f", result, tt.want)
			}
		})
	}
}

func TestGetCostMultiplier_MethodWildcard(t *testing.T) {
	endpoints := []plan.Endpoint{
		{Path: "/api/data", Method: "", CostMultiplier: 7.0}, // empty = all methods
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			result := plan.GetCostMultiplier(endpoints, method, "/api/data")
			if result != 7.0 {
				t.Errorf("got %f, want 7.0", result)
			}
		})
	}
}

func TestGetCostMultiplier_FirstMatchWins(t *testing.T) {
	endpoints := []plan.Endpoint{
		{Path: "/api/*", CostMultiplier: 2.0},
		{Path: "/api/data", CostMultiplier: 5.0}, // More specific but comes second
	}

	result := plan.GetCostMultiplier(endpoints, "GET", "/api/data")
	if result != 2.0 {
		t.Errorf("expected first match (2.0), got %f", result)
	}
}

func TestFindPlan_Found(t *testing.T) {
	plans := []plan.Plan{
		{ID: "free", Name: "Free Plan", RequestsPerMonth: 1000},
		{ID: "pro", Name: "Pro Plan", RequestsPerMonth: 100000},
		{ID: "enterprise", Name: "Enterprise", RequestsPerMonth: -1},
	}

	tests := []struct {
		id   string
		want string
	}{
		{"free", "Free Plan"},
		{"pro", "Pro Plan"},
		{"enterprise", "Enterprise"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			p, found := plan.FindPlan(plans, tt.id)
			if !found {
				t.Fatalf("plan %s not found", tt.id)
			}
			if p.Name != tt.want {
				t.Errorf("got %s, want %s", p.Name, tt.want)
			}
		})
	}
}

func TestFindPlan_NotFound(t *testing.T) {
	plans := []plan.Plan{
		{ID: "free", Name: "Free Plan"},
	}

	_, found := plan.FindPlan(plans, "nonexistent")
	if found {
		t.Error("expected plan not found")
	}
}

func TestFindPlan_EmptyList(t *testing.T) {
	_, found := plan.FindPlan(nil, "any")
	if found {
		t.Error("expected not found for empty list")
	}
}

func TestIsUnlimited(t *testing.T) {
	tests := []struct {
		name   string
		plan   plan.Plan
		want   bool
	}{
		{
			"unlimited plan",
			plan.Plan{ID: "enterprise", RequestsPerMonth: -1},
			true,
		},
		{
			"limited plan",
			plan.Plan{ID: "free", RequestsPerMonth: 1000},
			false,
		},
		{
			"zero requests",
			plan.Plan{ID: "disabled", RequestsPerMonth: 0},
			false,
		},
		{
			"negative -2 (also unlimited)",
			plan.Plan{ID: "test", RequestsPerMonth: -2},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plan.IsUnlimited(tt.plan)
			if result != tt.want {
				t.Errorf("got %v, want %v", result, tt.want)
			}
		})
	}
}

func TestPlan_StructFields(t *testing.T) {
	p := plan.Plan{
		ID:                 "pro",
		Name:               "Pro Plan",
		RequestsPerMonth:   100000,
		RateLimitPerMinute: 1000,
		PriceMonthly:       2999, // $29.99 in cents
		OveragePrice:       1,    // $0.01 per request
		StripePriceID:      "price_abc123",
	}

	if p.ID != "pro" {
		t.Errorf("ID = %s, want pro", p.ID)
	}
	if p.Name != "Pro Plan" {
		t.Errorf("Name = %s, want Pro Plan", p.Name)
	}
	if p.RequestsPerMonth != 100000 {
		t.Errorf("RequestsPerMonth = %d, want 100000", p.RequestsPerMonth)
	}
	if p.RateLimitPerMinute != 1000 {
		t.Errorf("RateLimitPerMinute = %d, want 1000", p.RateLimitPerMinute)
	}
	if p.PriceMonthly != 2999 {
		t.Errorf("PriceMonthly = %d, want 2999", p.PriceMonthly)
	}
	if p.OveragePrice != 1 {
		t.Errorf("OveragePrice = %d, want 1", p.OveragePrice)
	}
	if p.StripePriceID != "price_abc123" {
		t.Errorf("StripePriceID = %s, want price_abc123", p.StripePriceID)
	}
}

func TestEndpoint_StructFields(t *testing.T) {
	e := plan.Endpoint{
		Path:           "/api/v1/*",
		Method:         "POST",
		CostMultiplier: 2.5,
	}

	if e.Path != "/api/v1/*" {
		t.Errorf("Path = %s, want /api/v1/*", e.Path)
	}
	if e.Method != "POST" {
		t.Errorf("Method = %s, want POST", e.Method)
	}
	if e.CostMultiplier != 2.5 {
		t.Errorf("CostMultiplier = %f, want 2.5", e.CostMultiplier)
	}
}
