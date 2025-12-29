package modules

import (
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

func TestParseAllModules(t *testing.T) {
	modules, err := schema.ParseDir(".")
	if err != nil {
		t.Fatalf("ParseDir failed: %v", err)
	}

	if len(modules) == 0 {
		t.Error("No modules found")
	}

	expectedModules := map[string]bool{
		"user":     false,
		"plan":     false,
		"api_key":  false,
		"route":    false,
		"upstream": false,
		"setting":  false,
	}

	for _, mod := range modules {
		t.Logf("Loaded module: %s with %d fields, %d actions",
			mod.Name, len(mod.Schema), len(mod.Actions))

		expectedModules[mod.Name] = true

		// Derive to verify convention derivation works
		derived := convention.Derive(mod)
		t.Logf("  Derived: table=%s, plural=%s, %d paths",
			derived.Table, derived.Plural, len(derived.Paths))

		// Check for HTTP paths
		httpCount := 0
		cliCount := 0
		for _, p := range derived.Paths {
			if p.Type == schema.PathTypeHTTP {
				httpCount++
			}
			if p.Type == schema.PathTypeCLI {
				cliCount++
			}
		}
		t.Logf("  Paths: %d HTTP, %d CLI", httpCount, cliCount)
	}

	// Verify all expected modules were found
	for name, found := range expectedModules {
		if !found {
			t.Errorf("Expected module %q not found", name)
		}
	}
}

func TestUserModuleSchema(t *testing.T) {
	mod, err := schema.ParseFile("user.yaml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if mod.Name != "user" {
		t.Errorf("Name = %q, want %q", mod.Name, "user")
	}

	// Check required fields exist
	requiredFields := []string{"email", "password_hash", "name", "stripe_id", "plan_id", "status"}
	for _, field := range requiredFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("Missing field %q", field)
		}
	}

	// Check email field properties
	email := mod.Schema["email"]
	if email.Type != schema.FieldTypeEmail {
		t.Errorf("email.Type = %q, want %q", email.Type, schema.FieldTypeEmail)
	}
	if !email.Unique {
		t.Error("email should be unique")
	}
	if !email.Lookup {
		t.Error("email should be lookup")
	}

	// Check actions exist
	requiredActions := []string{"activate", "suspend", "cancel"}
	for _, action := range requiredActions {
		if _, ok := mod.Actions[action]; !ok {
			t.Errorf("Missing action %q", action)
		}
	}
}

func TestPlanModuleSchema(t *testing.T) {
	mod, err := schema.ParseFile("plan.yaml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if mod.Name != "plan" {
		t.Errorf("Name = %q, want %q", mod.Name, "plan")
	}

	// Check pricing fields
	pricingFields := []string{"rate_limit_per_minute", "requests_per_month", "price_monthly", "overage_price"}
	for _, field := range pricingFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("Missing field %q", field)
		}
	}

	// Check payment provider fields
	providerFields := []string{"stripe_price_id", "paddle_price_id", "lemon_variant_id"}
	for _, field := range providerFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("Missing field %q", field)
		}
	}

	// Check enable/disable actions
	if _, ok := mod.Actions["enable"]; !ok {
		t.Error("Missing action 'enable'")
	}
	if _, ok := mod.Actions["disable"]; !ok {
		t.Error("Missing action 'disable'")
	}
}

func TestRouteModuleSchema(t *testing.T) {
	mod, err := schema.ParseFile("route.yaml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if mod.Name != "route" {
		t.Errorf("Name = %q, want %q", mod.Name, "route")
	}

	// Check routing fields
	routingFields := []string{"path_pattern", "match_type", "methods", "upstream_id"}
	for _, field := range routingFields {
		if _, ok := mod.Schema[field]; !ok {
			t.Errorf("Missing field %q", field)
		}
	}

	// Check match_type is an enum
	matchType := mod.Schema["match_type"]
	if matchType.Type != schema.FieldTypeEnum {
		t.Errorf("match_type.Type = %q, want %q", matchType.Type, schema.FieldTypeEnum)
	}
	if len(matchType.Values) != 3 {
		t.Errorf("match_type.Values has %d values, want 3", len(matchType.Values))
	}
}

func TestUpstreamModuleSchema(t *testing.T) {
	mod, err := schema.ParseFile("upstream.yaml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if mod.Name != "upstream" {
		t.Errorf("Name = %q, want %q", mod.Name, "upstream")
	}

	// Check base_url exists
	if _, ok := mod.Schema["base_url"]; !ok {
		t.Error("Missing field 'base_url'")
	}

	// Check auth_type is an enum
	authType := mod.Schema["auth_type"]
	if authType.Type != schema.FieldTypeEnum {
		t.Errorf("auth_type.Type = %q, want %q", authType.Type, schema.FieldTypeEnum)
	}
}
